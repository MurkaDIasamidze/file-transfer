import axios from 'axios';
import type { FileItem, Folder, User } from '../types';

const BASE = (import.meta.env.VITE_API_URL ?? 'http://localhost:8081').replace(/\/$/, '');

export const api = axios.create({ baseURL: BASE });

api.interceptors.request.use((cfg) => {
  const token = localStorage.getItem('token');
  if (token) cfg.headers.Authorization = `Bearer ${token}`;
  return cfg;
});

api.interceptors.response.use(
  r => r,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
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

  me: () =>
    api.get<User>('/api/me'),
};

// ── Account ───────────────────────────────────────────────
export const accountApi = {
  updateProfile: (name: string) =>
    api.patch<User>('/api/me', { name }),

  changePassword: (currentPassword: string, newPassword: string) =>
    api.post('/api/me/password', {
      current_password: currentPassword,
      new_password:     newPassword,
    }),
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
};