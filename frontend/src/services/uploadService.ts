import { filesApi } from './api';

export interface UploadProgress {
  fileUploadId:   number;
  fileName:       string;
  uploadedChunks: number;
  totalChunks:    number;
  percentage:     number;
  status:         string;
}

const CHUNK_SIZE = parseInt(import.meta.env.VITE_CHUNK_SIZE ?? '1048576');
const WS_BASE    = import.meta.env.VITE_WS_URL ?? 'ws://localhost:8081';
const MAX_RETRY  = 3;

async function sha256hex(buf: ArrayBuffer): Promise<string> {
  const hash = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(hash))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

export class UploadService {
  private ws?:        WebSocket;
  private reconnects  = 0;
  private done        = false; // guard — stops reconnect loop after upload completes

  constructor(
    private onProgress: (p: UploadProgress) => void,
    private onComplete: () => void,
    private onError:    (msg: string) => void,
  ) {}

  async upload(file: File, folderId?: number | null): Promise<void> {
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
    const fileBuf     = await file.arrayBuffer();
    const checksum    = await sha256hex(fileBuf);

    const { data } = await filesApi.initUpload({
      file_name:    file.name,
      file_type:    file.type,
      file_size:    file.size,
      total_chunks: totalChunks,
      checksum,
      folder_id:    folderId ?? null,
    });

    const fileId = data.file_upload_id;
    this.openWS(fileId, file.name, totalChunks);

    await this.sendChunks(file, fileId, totalChunks);
    await filesApi.complete(fileId);

    // Mark done BEFORE closing so the onclose handler doesn't reconnect
    this.done = true;
    this.ws?.close();

    this.onProgress({
      fileUploadId:   fileId,
      fileName:       file.name,
      uploadedChunks: totalChunks,
      totalChunks,
      percentage:     100,
      status:         'completed',
    });

    this.onComplete();
  }

  private async sendChunks(file: File, fileId: number, total: number): Promise<void> {
    const done    = new Set<number>();
    let   retries = 0;

    while (done.size < total) {
      for (let i = 0; i < total; i++) {
        if (done.has(i)) continue;
        try {
          const start = i * CHUNK_SIZE;
          const slice = file.slice(start, Math.min(start + CHUNK_SIZE, file.size));
          const buf   = await slice.arrayBuffer();
          const cs    = await sha256hex(buf);

          const form = new FormData();
          form.append('file_upload_id', String(fileId));
          form.append('chunk_index',    String(i));
          form.append('checksum',       cs);
          form.append('chunk',          new Blob([buf]));

          await filesApi.uploadChunk(form);
          done.add(i);

          // Emit progress from client side immediately after each chunk
          // (don't wait for WS — it may be slightly behind)
          this.onProgress({
            fileUploadId:   fileId,
            fileName:       file.name,
            uploadedChunks: done.size,
            totalChunks:    total,
            percentage:     Math.round((done.size / total) * 100),
            status:         'uploading',
          });

          // Server-side verification every 10 chunks
          if (done.size % 10 === 0) {
            await this.verifyWithServer(fileId, done);
          }
        } catch (err) {
          console.warn(`chunk ${i} failed`, err);
        }
      }

      if (done.size < total) {
        retries++;
        if (retries >= MAX_RETRY) {
          this.onError('Upload failed after max retries');
          throw new Error('max retries exceeded');
        }
        await sleep(1000 * retries);
      }
    }
  }

  private async verifyWithServer(fileId: number, done: Set<number>): Promise<void> {
    try {
      const { data } = await filesApi.verify(fileId);
      const srv = new Set(data.uploaded_chunks);
      for (const i of [...done]) {
        if (!srv.has(i)) done.delete(i);
      }
    } catch {
      // non-fatal — retry will catch missing chunks
    }
  }

  private openWS(fileId: number, fileName: string, totalChunks: number): void {
    try {
      this.ws = new WebSocket(`${WS_BASE}/ws/upload/${fileId}`);
    } catch {
      // WS is optional — upload proceeds via HTTP chunks regardless
      return;
    }

    this.ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string);
        if (msg.type === 'progress') {
          this.onProgress({
            fileUploadId:   msg.file_upload_id,
            fileName:       msg.file_name ?? fileName,
            uploadedChunks: msg.uploaded_chunks,
            totalChunks:    msg.total_chunks ?? totalChunks,
            percentage:     Math.round(msg.progress_percent),
            status:         msg.status,
          });
        }
      } catch {
        // ignore malformed WS messages
      }
    };

    this.ws.onerror = () => {
      // WS error is non-fatal — chunks still upload via HTTP
      console.warn('WebSocket error — progress updates may be delayed');
    };

    this.ws.onclose = () => {
      // Only reconnect if upload is still in progress and under retry limit
      if (!this.done && this.reconnects < 3) {
        this.reconnects++;
        setTimeout(() => {
          if (!this.done) this.openWS(fileId, fileName, totalChunks);
        }, 2000);
      }
    };
  }
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));