'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  token: string | null;
  playerId: string | null;
  email: string | null;
  setAuth: (token: string, playerId: string, email: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      playerId: null,
      email: null,
      setAuth: (token, playerId, email) => set({ token, playerId, email }),
      logout: () => set({ token: null, playerId: null, email: null }),
    }),
    { name: 'attaboy-auth' },
  ),
);
