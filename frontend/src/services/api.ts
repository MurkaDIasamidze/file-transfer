import axios from 'axios';
import type { FileItem, Folder } from '../types';
import type { AuthUser } from '../store/authStore';

const BASE = (import.meta.env.VITE_API_URL ?? 'http://localhost:8081').replace(/\/$/, '');

export const api = axios.create({ baseURL: BASE });

// Attach JWT from localStorage before every request
api.interceptors.request.use((cfg) => {
  const token = localStorage.getItem('token');
  if (token) cfg.headers.Authorization = `Bearer ${token}`;
  return cfg;
});

// On 401 — clear storage and redirect to login
api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  },
);

// ── Auth ──────────────────────────────────────────────────────────────────────
export const authApi = {
  register: (name: string, email: string, password: string) =>
    api.post('/api/auth/register', { name, email, password }),

  login: (email: string, password: string) =>
    api.post<{ token: string; user: AuthUser }>('/api/auth/login', { email, password }),

  me: () =>
    api.get<AuthUser>('/api/me'),

  updateProfile: (name: string) =>
    api.patch<AuthUser>('/api/me', { name }),

  changePassword: (current_password: string, new_password: string) =>
    api.post('/api/me/password', { current_password, new_password }),
};

// ── Files ─────────────────────────────────────────────────────────────────────
export const filesApi = {
  list:        (folderId: number | null) =>
    api.get<FileItem[]>('/api/files', { params: folderId != null ? { folder_id: folderId } : {} }),
  recent:      () => api.get<FileItem[]>('/api/files/recent'),
  starred:     () => api.get<FileItem[]>('/api/files/starred'),
  trash:       () => api.get<FileItem[]>('/api/files/trash'),
  move:        (id: number, folderId: number | null) =>
    api.patch<FileItem>(`/api/files/${id}/move`, { folder_id: folderId }),
  toggleStar:  (id: number) => api.patch<FileItem>(`/api/files/${id}/star`),
  moveToTrash: (id: number) => api.patch(`/api/files/${id}/trash`),
  restore:     (id: number) => api.patch(`/api/files/${id}/restore`),
  delete:      (id: number) => api.delete(`/api/files/${id}`),
};

// ── Folders ───────────────────────────────────────────────────────────────────
export const foldersApi = {
  list:        (parentId: number | null) =>
    api.get<Folder[]>('/api/folders', { params: parentId != null ? { parent_id: parentId } : {} }),
  listTrashed: () => api.get<Folder[]>('/api/folders/trash'),
  create:      (name: string, parentId: number | null) =>
    api.post<Folder>('/api/folders', { name, parent_id: parentId }),
  trash:       (id: number) => api.patch(`/api/folders/${id}/trash`),
  restore:     (id: number) => api.patch(`/api/folders/${id}/restore`),
  delete:      (id: number) => api.delete(`/api/folders/${id}`),
};

export default api;