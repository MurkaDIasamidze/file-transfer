import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081';
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8081';
const CHUNK_SIZE = parseInt(import.meta.env.VITE_CHUNK_SIZE || '1048576'); // 1MB default

export interface UploadProgress {
  uploadedChunks: number;
  totalChunks: number;
  percentage: number;
  status: string;
}

interface IUploadService {
  uploadFile(file: File): Promise<void>;
  connectWebSocket(fileUploadId: number): void;
  disconnectWebSocket(): void;
}

async function calculateChecksum(data: ArrayBuffer): Promise<string> {
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

export class FileUploadService implements IUploadService {
  private onProgress?: (progress: UploadProgress) => void;
  private fileUploadId?: number;
  private ws?: WebSocket;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;

  constructor(onProgress?: (progress: UploadProgress) => void) {
    this.onProgress = onProgress;
  }

  async uploadFile(file: File): Promise<void> {
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
    
    const fileBuffer = await file.arrayBuffer();
    const fileChecksum = await calculateChecksum(fileBuffer);

    const initResponse = await axios.post(`${API_URL}/api/upload/init`, {
      file_name: file.name,
      file_type: file.type,
      file_size: file.size,
      total_chunks: totalChunks,
      checksum: fileChecksum,
    });

    this.fileUploadId = initResponse.data.file_upload_id;

    // Connect WebSocket with proper ID
    if (this.fileUploadId) {
      this.connectWebSocket(this.fileUploadId);
    }

    await this.uploadChunks(file, totalChunks);

    await this.completeUpload();
  }

  connectWebSocket(fileUploadId: number): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }

    const wsUrl = `${WS_URL}/ws/upload/${fileUploadId}`;
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'progress' && this.onProgress) {
          this.onProgress({
            uploadedChunks: data.uploaded_chunks,
            totalChunks: data.total_chunks,
            percentage: Math.round(data.progress_percent),
            status: data.status,
          });
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      if (this.reconnectAttempts < this.maxReconnectAttempts) {
        this.reconnectAttempts++;
        setTimeout(() => {
          if (this.fileUploadId) {
            this.connectWebSocket(this.fileUploadId);
          }
        }, 2000);
      }
    };
  }

  disconnectWebSocket(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = undefined;
    }
  }

  private async uploadChunks(file: File, totalChunks: number): Promise<void> {
    const uploadedChunks = new Set<number>();
    let retryCount = 0;
    const maxRetries = parseInt(import.meta.env.VITE_MAX_RETRIES || '3');

    while (uploadedChunks.size < totalChunks && retryCount < maxRetries) {
      for (let i = 0; i < totalChunks; i++) {
        if (uploadedChunks.has(i)) continue;

        try {
          const start = i * CHUNK_SIZE;
          const end = Math.min(start + CHUNK_SIZE, file.size);
          const chunk = file.slice(start, end);
          const chunkBuffer = await chunk.arrayBuffer();
          const chunkChecksum = await calculateChecksum(chunkBuffer);

          const formData = new FormData();
          formData.append('file_upload_id', this.fileUploadId!.toString());
          formData.append('chunk_index', i.toString());
          formData.append('checksum', chunkChecksum);
          formData.append('chunk', new Blob([chunkBuffer]));

          await axios.post(`${API_URL}/api/upload/chunk`, formData, {
            headers: {
              'Content-Type': 'multipart/form-data',
            },
          });

          uploadedChunks.add(i);

          const verifyInterval = parseInt(import.meta.env.VITE_VERIFY_INTERVAL || '10');
          if (uploadedChunks.size % verifyInterval === 0) {
            await this.verifyUploadedChunks(uploadedChunks);
          }

        } catch (error) {
          console.error(`Error uploading chunk ${i}:`, error);
        }
      }

      if (uploadedChunks.size < totalChunks) {
        retryCount++;
        console.log(`Retry attempt ${retryCount} of ${maxRetries}`);
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
    }

    if (uploadedChunks.size < totalChunks) {
      throw new Error(`Failed to upload all chunks after ${maxRetries} retries`);
    }
  }

  private async verifyUploadedChunks(uploadedChunks: Set<number>): Promise<void> {
    try {
      const response = await axios.get(
        `${API_URL}/api/upload/verify/${this.fileUploadId}`
      );

      const serverChunks = new Set(response.data.uploaded_chunks);
      
      for (const chunkIndex of uploadedChunks) {
        if (!serverChunks.has(chunkIndex)) {
          console.warn(`Chunk ${chunkIndex} missing on server, will retry`);
          uploadedChunks.delete(chunkIndex);
        }
      }
    } catch (error) {
      console.error('Error verifying chunks:', error);
    }
  }

  private async completeUpload(): Promise<void> {
    try {
      await axios.post(`${API_URL}/api/upload/complete`, {
        file_upload_id: this.fileUploadId,
      });

      setTimeout(() => {
        this.disconnectWebSocket();
      }, 1000);

    } catch (error: unknown) {
      this.disconnectWebSocket();
      throw error;
    }
  }
}