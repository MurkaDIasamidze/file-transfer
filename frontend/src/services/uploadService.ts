// uploadService.ts — WebSocket-based upload (files + folders)

const WS_BASE   = (import.meta.env.VITE_WS_URL ?? 'ws://localhost:8081').replace(/\/$/, '');
const CHUNK_SIZE = parseInt(import.meta.env.VITE_CHUNK_SIZE ?? '262144'); // 256 KB default

export interface UploadProgress {
  fileUploadId:   number;
  fileName:       string;
  relPath:        string;
  uploadedChunks: number;
  totalChunks:    number;
  percentage:     number;
  status:         'pending' | 'uploading' | 'completed' | 'failed';
}

export interface UploadTask {
  file:    File;
  relPath: string; // '' for single file, 'dir/sub/file.txt' for folder
}

async function sha256hex(buf: ArrayBuffer): Promise<string> {
  const hash = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(hash))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

// ─── Single connection shared across an upload session ───────────────────────

export class UploadSession {
  private ws!: WebSocket;
  private ready  = false;
  private queue: Array<() => void> = [];
  private messageHandlers = new Map<number, (msg: Record<string, unknown>) => void>();

  constructor(
    private token: string,
    private onProgress: (p: UploadProgress) => void,
    private onError: (msg: string) => void,
  ) {}

  async open(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(`${WS_BASE}/ws/upload?token=${this.token}`);
      this.ws.binaryType = 'arraybuffer';

      this.ws.onopen = () => {
        this.ready = true;
        this.queue.forEach(fn => fn());
        this.queue = [];
        resolve();
      };

      this.ws.onerror = () => reject(new Error('WebSocket connection failed'));

      this.ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string) as Record<string, unknown>;
          const type = msg.type as string;

          if (type === 'error') {
            this.onError(msg.message as string);
          }

          // Extract file_upload_id — it lives at the top level for most messages
          // (progress, error) but is nested inside `file.id` for `done` messages
          // because the server sends: { type: "done", file: { id: X, ... } }
          let fileId = msg.file_upload_id as number | undefined;
          if (!fileId && type === 'done') {
            const fileObj = msg.file as Record<string, unknown> | undefined;
            fileId = fileObj?.id as number | undefined;
          }

          if (fileId && this.messageHandlers.has(fileId)) {
            this.messageHandlers.get(fileId)!(msg);
          } else if (type === 'init_ack') {
            // init_ack routes via temp key -1 (we don't know the ID until we receive it)
            this.messageHandlers.get(-1)?.(msg);
          }
        } catch { /* ignore malformed */ }
      };

      this.ws.onclose = () => {
        this.ready = false;
      };
    });
  }

  close() {
    this.ws?.close();
  }

  private send(data: unknown) {
    const fn = () => this.ws.send(JSON.stringify(data));
    if (this.ready) fn();
    else this.queue.push(fn);
  }

  // ── Upload a single file task ─────────────────────────────────────────────

  async uploadFile(
    task: UploadTask,
    folderId: number | null,
    onFileProgress: (p: UploadProgress) => void,
  ): Promise<void> {
    const { file, relPath } = task;
    const fileBuf     = await file.arrayBuffer();
    const checksum    = await sha256hex(fileBuf);
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE) || 1;

    // ── 1. Init ─────────────────────────────────────────────────────────────
    const fileUploadId: number = await new Promise((resolve, reject) => {
      // Temporary handler for init_ack — keyed by -1 since we don't have the
      // file_upload_id yet. Cleared immediately upon receipt.
      this.messageHandlers.set(-1, (msg) => {
        if (msg.type === 'init_ack') {
          this.messageHandlers.delete(-1);
          resolve(msg.file_upload_id as number);
        } else if (msg.type === 'error') {
          this.messageHandlers.delete(-1);
          reject(new Error(msg.message as string));
        }
      });

      this.send({
        type: 'init',
        data: {
          file_name:  file.name,
          file_type:  file.type,
          file_size:  file.size,
          checksum,
          folder_id:  folderId,
          rel_path:   relPath,
        },
      });
    });

    // ── 2. Chunks ────────────────────────────────────────────────────────────
    let uploadedCount = 0;
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE;
      const slice = file.slice(start, Math.min(start + CHUNK_SIZE, file.size));
      const buf   = await slice.arrayBuffer();
      const cs    = await sha256hex(buf);

      // Convert to base64 for JSON transport
      const bytes   = new Uint8Array(buf);
      let   binary  = '';
      for (let b = 0; b < bytes.byteLength; b++) binary += String.fromCharCode(bytes[b]);
      const base64 = btoa(binary);

      await new Promise<void>((resolve, reject) => {
        this.messageHandlers.set(fileUploadId, (msg) => {
          if (msg.type === 'progress') {
            uploadedCount++;
            // FIX: emit progress with the task's relPath (not the server's file_name)
            // so the UploadModal queue matcher can correctly identify this task.
            onFileProgress({
              fileUploadId,
              fileName:       file.name,
              relPath,                          // ← use task relPath, not server field
              uploadedChunks: uploadedCount,
              totalChunks,
              percentage:     Math.round((uploadedCount / totalChunks) * 100),
              status:         'uploading',
            });
            this.messageHandlers.delete(fileUploadId);
            resolve();
          } else if (msg.type === 'error') {
            this.messageHandlers.delete(fileUploadId);
            reject(new Error(msg.message as string));
          }
        });

        this.send({
          type: 'chunk',
          data: {
            file_upload_id: fileUploadId,
            chunk_index:    i,
            total_chunks:   totalChunks,
            checksum:       cs,
            data:           base64,
          },
        });
      });
    }

    // ── 3. Complete ──────────────────────────────────────────────────────────
    await new Promise<void>((resolve, reject) => {
      this.messageHandlers.set(fileUploadId, (msg) => {
        if (msg.type === 'done') {
          // FIX: emit completed progress with the task's relPath
          onFileProgress({
            fileUploadId,
            fileName:       file.name,
            relPath,                            // ← use task relPath
            uploadedChunks: totalChunks,
            totalChunks,
            percentage:     100,
            status:         'completed',
          });
          this.messageHandlers.delete(fileUploadId);
          resolve();
        } else if (msg.type === 'error') {
          this.messageHandlers.delete(fileUploadId);
          reject(new Error(msg.message as string));
        }
      });

      this.send({
        type: 'complete',
        data: { file_upload_id: fileUploadId },
      });
    });
  }
}

// ─── Convenience: upload a batch of tasks over one WS session ────────────────

export async function uploadBatch(
  tasks: UploadTask[],
  folderId: number | null,
  token: string,
  onProgress: (p: UploadProgress) => void,
  onError: (msg: string) => void,
): Promise<void> {
  const session = new UploadSession(token, onProgress, onError);
  await session.open();

  try {
    // Upload sequentially — parallel support can be added later
    for (const task of tasks) {
      await session.uploadFile(task, folderId, onProgress);
    }
  } finally {
    session.close();
  }
}

// ─── Helper: expand a FileList into tasks (handles folder drag-drop) ─────────

export function filesToTasks(files: FileList): UploadTask[] {
  return Array.from(files).map(f => ({
    file:    f,
    // webkitRelativePath is set when user picks a folder via <input webkitdirectory>
    relPath: (f as File & { webkitRelativePath?: string }).webkitRelativePath ?? '',
  }));
}