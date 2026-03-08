// store/authStore.ts
// Token is stored in a cookie instead of localStorage.
// Cookies are read synchronously so `hydrated` is always true immediately —
// no async flash or loading state needed.

import { create } from 'zustand';
import Cookies from 'js-cookie';

const COOKIE_NAME = 'auth_token';

const COOKIE_OPTS: Cookies.CookieAttributes = {
  expires:  7,        // 7 days — matches JWT_EXPIRY_HOURS=168 on the backend
  sameSite: 'Strict',
  secure:   window.location.protocol === 'https:',
  path:     '/',
};

export interface AuthUser {
  id:          number;
  name:        string;
  email:       string;
  avatar_url?: string;
}

interface AuthState {
  token:    string | null;
  user:     AuthUser | null;
  hydrated: true;           // always true — cookies are synchronous
  setAuth:  (token: string, user: AuthUser) => void;
  setUser:  (user: AuthUser) => void;
  logout:   () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token:    Cookies.get(COOKIE_NAME) ?? null,
  user:     null,
  hydrated: true,           // no async hydration needed for cookies

  setAuth: (token, user) => {
    Cookies.set(COOKIE_NAME, token, COOKIE_OPTS);
    set({ token, user });
  },

  setUser: (user) => {
    set({ user });
  },

  logout: () => {
    Cookies.remove(COOKIE_NAME, { path: '/' });
    set({ token: null, user: null });
  },
}));