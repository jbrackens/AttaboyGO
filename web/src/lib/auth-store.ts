'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface Player {
  id: string;
  email: string;
  currency: string;
  first_name?: string;
  last_name?: string;
}

interface AuthState {
  token: string | null;
  playerId: string | null;
  email: string | null;
  player: Player | null;
  setAuth: (token: string, playerId: string, email: string) => void;
  setPlayer: (player: Player) => void;
  logout: () => void;
  hydrate: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      playerId: null,
      email: null,
      player: null,
      setAuth: (token, playerId, email) => set({ token, playerId, email }),
      setPlayer: (player) => set({ player }),
      logout: () => set({ token: null, playerId: null, email: null, player: null }),
      hydrate: () => {
        // Zustand persist auto-hydrates; this is a no-op hook for manual trigger
        const state = get();
        if (state.token && !state.player && state.playerId) {
          // Player data will be fetched by pages that need it
        }
      },
    }),
    { name: 'attaboy-auth' },
  ),
);
