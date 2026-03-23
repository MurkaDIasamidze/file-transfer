// store/authStore.ts
import { create } from 'zustand';

export interface AuthUser {
  id:          number;
  name:        string;
  email:       string;
  avatar_url?: string;
  created_at?: string;  // returned by /api/me and /api/auth/login
}

interface AuthState {
  token:           string | null;
  user:            AuthUser | null;
  setAuth:         (token: string, user: AuthUser) => void;
  setUser:         (user: AuthUser) => void;
  logout:          () => void;
  isAuthenticated: () => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem('token') ?? null,
  user:  (() => {
    try {
      const raw = localStorage.getItem('user');
      return raw ? (JSON.parse(raw) as AuthUser) : null;
    } catch { return null; }
  })(),

  setAuth: (token, user) => {
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
    set({ token, user });
  },

  setUser: (user) => {
    localStorage.setItem('user', JSON.stringify(user));
    set({ user });
  },

  logout: () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    set({ token: null, user: null });
  },

  isAuthenticated: () => Boolean(get().token),
}));