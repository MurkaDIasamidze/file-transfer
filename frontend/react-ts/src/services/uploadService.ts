import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';
const CHUNK_SIZE = 1024 * 1024; // 1MB chunks

export interface UploadProgress {
  uploadedChunks: number;
  totalChunks: number;
  percentage: number;
  status: string;
}

async function calculateChecksum(data: ArrayBuffer): Promise<string> {
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

export class FileUploadService {
  private onProgress?: (progress: UploadProgress) => void;
  private fileUploadId?: number;

  constructor(onProgress?: (progress: UploadProgress) => void) {
    this.onProgress = onProgress;
  }

  async uploadFile(file: File): Promise<void> {
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
    
    // Calculate file checksum
    const fileBuffer = await file.arrayBuffer();
    const fileChecksum = await calculateChecksum(fileBuffer);

    // Initialize upload
    const initResponse = await axios.post(`${API_URL}/api/upload/init`, {
      file_name: file.name,
      file_type: file.type,
      file_size: file.size,
      total_chunks: totalChunks,
      checksum: fileChecksum,
    });

    this.fileUploadId = initResponse.data.file_upload_id;

    // Upload chunks
    await this.uploadChunks(file, totalChunks);

    // Complete upload
    await this.completeUpload();
  }

  private async uploadChunks(file: File, totalChunks: number): Promise<void> {
    const uploadedChunks = new Set<number>();
    let retryCount = 0;
    const maxRetries = 3;

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

          if (this.onProgress) {
            this.onProgress({
              uploadedChunks: uploadedChunks.size,
              totalChunks,
              percentage: Math.round((uploadedChunks.size / totalChunks) * 100),
              status: 'uploading',
            });
          }

          // Periodic verification check every 10 chunks
          if (uploadedChunks.size % 10 === 0) {
            await this.verifyUploadedChunks(uploadedChunks);
          }

        } catch (error) {
          console.error(`Error uploading chunk ${i}:`, error);
          // Chunk will be retried in next iteration
        }
      }

      if (uploadedChunks.size < totalChunks) {
        retryCount++;
        console.log(`Retry attempt ${retryCount} of ${maxRetries}`);
        await new Promise(resolve => setTimeout(resolve, 1000)); // Wait 1s before retry
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
      
      // Check if all our uploaded chunks are on the server
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
      const response = await axios.post(`${API_URL}/api/upload/complete`, {
        file_upload_id: this.fileUploadId,
      });

      if (this.onProgress) {
        this.onProgress({
          uploadedChunks: response.data.total_chunks,
          totalChunks: response.data.total_chunks,
          percentage: 100,
          status: 'completed',
        });
      }
    } catch (error: any) {
      if (this.onProgress) {
        this.onProgress({
          uploadedChunks: 0,
          totalChunks: 0,
          percentage: 0,
          status: 'failed',
        });
      }
      throw error;
    }
  }
}