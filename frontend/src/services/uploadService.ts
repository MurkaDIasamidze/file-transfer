// uploadService.ts — WebSocket-based chunked upload

const WS_BASE    = (import.meta.env.VITE_WS_URL ?? 'ws://localhost:8081').replace(/\/$/, '');
const CHUNK_SIZE = parseInt(import.meta.env.VITE_CHUNK_SIZE ?? '262144', 10); // 256 KB

// ─── Public types ─────────────────────────────────────────────────────────────

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
  relPath: string; // '' for plain file, 'folder/sub/file.txt' for folder uploads
}

// ─── FileSystem API helpers (for drag-drop folder support) ────────────────────
//
// When a folder is dragged from the OS, DataTransfer.files is flat (top-level
// only) and webkitRelativePath is NOT set. The only correct way to get the full
// recursive file tree with relative paths is webkitGetAsEntry() / FileSystem API.
//
// CRITICAL: webkitGetAsEntry() MUST be called synchronously inside the drop
// event handler before it returns — the DataTransfer becomes invalid afterwards.
// We therefore split the work: collect entries sync, then read files async.

function readAllEntries(reader: FileSystemDirectoryReader): Promise<FileSystemEntry[]> {
  // readEntries() yields at most 100 items per call — must loop until empty batch.
  return new Promise((resolve, reject) => {
    const all: FileSystemEntry[] = [];
    function next() {
      reader.readEntries(batch => {
        if (batch.length === 0) { resolve(all); return; }
        all.push(...batch);
        next();
      }, reject);
    }
    next();
  });
}

async function entryToTasks(entry: FileSystemEntry, prefix: string): Promise<UploadTask[]> {
  if (entry.isFile) {
    const file = await new Promise<File>((resolve, reject) =>
      (entry as FileSystemFileEntry).file(resolve, reject)
    );
    const relPath = prefix ? `${prefix}/${entry.name}` : entry.name;
    return [{ file, relPath }];
  }

  if (entry.isDirectory) {
    const dirPrefix = prefix ? `${prefix}/${entry.name}` : entry.name;
    const reader    = (entry as FileSystemDirectoryEntry).createReader();
    const children  = await readAllEntries(reader);
    const nested    = await Promise.all(children.map(c => entryToTasks(c, dirPrefix)));
    return nested.flat();
  }

  return [];
}

/**
 * Convert a DataTransfer to UploadTask[].
 *
 * Handles:
 *   - Plain file drops  (one or many files)
 *   - Folder drops      (recursive, preserving relative paths)
 *   - Mixed drops       (files + folders in one drop)
 *
 * MUST receive the entries array that was collected SYNCHRONOUSLY during the
 * drop event (see collectEntriesSync below).
 */
export async function entriesToTasks(entries: FileSystemEntry[]): Promise<UploadTask[]> {
  const groups = await Promise.all(entries.map(e => entryToTasks(e, '')));
  return groups.flat();
}

/**
 * Call this SYNCHRONOUSLY inside the drop event handler to snapshot the
 * FileSystemEntry objects before the DataTransfer becomes invalid.
 */
export function collectEntriesSync(dt: DataTransfer): FileSystemEntry[] {
  const entries: FileSystemEntry[] = [];
  if (dt.items) {
    for (let i = 0; i < dt.items.length; i++) {
      const entry = dt.items[i].webkitGetAsEntry?.();
      if (entry) entries.push(entry);
    }
  }
  return entries;
}

/**
 * Convert a FileList (from <input> elements) to UploadTask[].
 * webkitRelativePath is set when the user picks via <input webkitdirectory>.
 */
export function filesToTasks(files: FileList): UploadTask[] {
  return Array.from(files).map(f => ({
    file:    f,
    relPath: (f as File & { webkitRelativePath?: string }).webkitRelativePath ?? '',
  }));
}

// ─── WebSocket upload helpers ─────────────────────────────────────────────────

async function sha256hex(buf: ArrayBuffer): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(digest))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

function toBase64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let bin = '';
  for (let i = 0; i < bytes.byteLength; i++) bin += String.fromCharCode(bytes[i]);
  return btoa(bin);
}

function openWS(token: string): Promise<WebSocket> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`${WS_BASE}/ws/upload?token=${token}`);
    ws.onopen  = () => resolve(ws);
    ws.onerror = ()  => reject(new Error('WebSocket connection failed'));
  });
}

/**
 * Send one JSON message over ws and wait for the next server message that
 * matches one of acceptTypes. Rejects immediately on 'error' messages.
 */
function sendAndWait(
  ws: WebSocket,
  payload: unknown,
  acceptTypes: string[],
): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    const handler = (ev: MessageEvent) => {
      let msg: Record<string, unknown>;
      try { msg = JSON.parse(ev.data as string); }
      catch { return; }

      const t = msg.type as string;
      if (t === 'error') {
        ws.removeEventListener('message', handler);
        reject(new Error((msg.message as string) || 'server error'));
        return;
      }
      if (acceptTypes.includes(t)) {
        ws.removeEventListener('message', handler);
        resolve(msg);
      }
    };

    ws.addEventListener('message', handler);
    ws.send(JSON.stringify(payload));
  });
}

// ─── Single-file upload over a dedicated WebSocket ───────────────────────────
//
// One WS per file keeps the protocol trivially simple: strict request/response,
// no multiplexing, no ID routing, no shared state between files.

async function uploadOneFile(
  task: UploadTask,
  folderId: number | null,
  token: string,
  onProgress: (p: UploadProgress) => void,
): Promise<void> {
  const { file, relPath } = task;

  const fileBuf     = await file.arrayBuffer();
  const checksum    = await sha256hex(fileBuf);
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE) || 1;

  const ws = await openWS(token);
  try {
    // 1. Init
    const initAck = await sendAndWait(
      ws,
      {
        type: 'init',
        data: {
          file_name: file.name,
          file_type: file.type || 'application/octet-stream',
          file_size: file.size,
          checksum,
          folder_id: folderId,
          rel_path:  relPath,
        },
      },
      ['init_ack'],
    );
    const fileUploadId = initAck.file_upload_id as number;

    // 2. Chunks — send one, wait for progress ack, then send next
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE;
      const slice = file.slice(start, Math.min(start + CHUNK_SIZE, file.size));
      const buf   = await slice.arrayBuffer();
      const cs    = await sha256hex(buf);
      const b64   = toBase64(buf);

      await sendAndWait(
        ws,
        {
          type: 'chunk',
          data: {
            file_upload_id: fileUploadId,
            chunk_index:    i,
            total_chunks:   totalChunks,
            checksum:       cs,
            data:           b64,
          },
        },
        ['progress'],
      );

      onProgress({
        fileUploadId,
        fileName:       file.name,
        relPath,
        uploadedChunks: i + 1,
        totalChunks,
        percentage:     Math.round(((i + 1) / totalChunks) * 100),
        status:         'uploading',
      });
    }

    // 3. Complete
    await sendAndWait(
      ws,
      { type: 'complete', data: { file_upload_id: fileUploadId } },
      ['done'],
    );

    onProgress({
      fileUploadId,
      fileName:       file.name,
      relPath,
      uploadedChunks: totalChunks,
      totalChunks,
      percentage:     100,
      status:         'completed',
    });
  } finally {
    ws.close();
  }
}

// ─── Public batch API ─────────────────────────────────────────────────────────

export async function uploadBatch(
  tasks: UploadTask[],
  folderId: number | null,
  token: string,
  onProgress: (p: UploadProgress) => void,
  onError: (msg: string) => void,
): Promise<void> {
  for (const task of tasks) {
    try {
      await uploadOneFile(task, folderId, token, onProgress);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      console.error('[upload] failed:', task.relPath || task.file.name, msg);
      onError(`${task.relPath || task.file.name}: ${msg}`);
      // Continue uploading remaining files even when one fails.
    }
  }
}