import axios from 'axios';
import type { FileItem, Folder, User } from '../types';

const BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8081';

export const api = axios.create({ baseURL: BASE });

api.interceptors.request.use((cfg) => {
  const token = localStorage.getItem('token');
  if (token) cfg.headers.Authorization = `Bearer ${token}`;
  return cfg;
});

api.interceptors.response.use(
  (r) => r,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  },
);

// ── Auth ──────────────────────────────────────────────────
export const authApi = {
  register: (name: string, email: string, password: string) =>
    api.post('/api/auth/register', { name, email, password }),

  login: (email: string, password: string) =>
    api.post<{ token: string; user: User }>('/api/auth/login', { email, password }),
};

// ── Folders ───────────────────────────────────────────────
export const foldersApi = {
  list: (parentId?: number | null) =>
    api.get<Folder[]>('/api/folders', {
      params: parentId != null ? { parent_id: parentId } : {},
    }),

  listTrashed: () =>
    api.get<Folder[]>('/api/folders/trash'),

  create: (name: string, parentId?: number | null) =>
    api.post<Folder>('/api/folders', { name, parent_id: parentId ?? null }),

  trash: (id: number) =>
    api.patch(`/api/folders/${id}/trash`),

  restore: (id: number) =>
    api.patch(`/api/folders/${id}/restore`),

  delete: (id: number) =>
    api.delete(`/api/folders/${id}`),
};

// ── Files ─────────────────────────────────────────────────
export const filesApi = {
  list: (folderId?: number | null) =>
    api.get<FileItem[]>('/api/files', {
      params: folderId != null ? { folder_id: folderId } : {},
    }),

  recent: () =>
    api.get<FileItem[]>('/api/files/recent'),

  starred: () =>
    api.get<FileItem[]>('/api/files/starred'),

  trash: () =>
    api.get<FileItem[]>('/api/files/trash'),

  move: (fileId: number, folderId: number | null) =>
    api.patch<FileItem>(`/api/files/${fileId}/move`, { folder_id: folderId }),

  toggleStar: (id: number) =>
    api.patch<FileItem>(`/api/files/${id}/star`),

  moveToTrash: (id: number) =>
    api.patch<FileItem>(`/api/files/${id}/trash`),

  restore: (id: number) =>
    api.patch<FileItem>(`/api/files/${id}/restore`),

  delete: (id: number) =>
    api.delete(`/api/files/${id}`),

  initUpload: (payload: {
    file_name: string;
    file_type: string;
    file_size: number;
    total_chunks: number;
    checksum: string;
    folder_id?: number | null;
  }) => api.post<{ file_upload_id: number }>('/api/upload/init', payload),

  uploadChunk: (form: FormData) =>
    api.post('/api/upload/chunk', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),

  complete: (fileUploadId: number) =>
    api.post<{ file: FileItem }>('/api/upload/complete', {
      file_upload_id: fileUploadId,
    }),

  verify: (id: number) =>
    api.get<{ uploaded_chunks: number[]; total: number }>(`/api/upload/verify/${id}`),
};