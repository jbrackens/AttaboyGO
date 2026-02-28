'use client';

import { create } from 'zustand';

export interface BetSelection {
  selectionId: string;
  eventId: string;
  marketId: string;
  eventName: string;
  marketName: string;
  selectionName: string;
  odds: number; // decimal, e.g. 1.75
  stake: number;
}

type BetMode = 'single' | 'parlay';

interface BetslipState {
  selections: BetSelection[];
  mode: BetMode;
  parlayStake: number;
  addSelection: (sel: Omit<BetSelection, 'stake'>) => void;
  removeSelection: (selectionId: string) => void;
  toggleSelection: (sel: Omit<BetSelection, 'stake'>) => void;
  setStake: (selectionId: string, stake: number) => void;
  setParlayStake: (stake: number) => void;
  setMode: (mode: BetMode) => void;
  clear: () => void;
  hasSelection: (selectionId: string) => boolean;
}

export const useBetslipStore = create<BetslipState>()((set, get) => ({
  selections: [],
  mode: 'single',
  parlayStake: 0,

  addSelection: (sel) =>
    set((s) => {
      if (s.selections.some((x) => x.selectionId === sel.selectionId)) return s;
      return { selections: [...s.selections, { ...sel, stake: 0 }] };
    }),

  removeSelection: (selectionId) =>
    set((s) => ({ selections: s.selections.filter((x) => x.selectionId !== selectionId) })),

  toggleSelection: (sel) => {
    const exists = get().selections.some((x) => x.selectionId === sel.selectionId);
    if (exists) {
      get().removeSelection(sel.selectionId);
    } else {
      get().addSelection(sel);
    }
  },

  setStake: (selectionId, stake) =>
    set((s) => ({
      selections: s.selections.map((x) => (x.selectionId === selectionId ? { ...x, stake } : x)),
    })),

  setParlayStake: (stake) => set({ parlayStake: stake }),
  setMode: (mode) => set({ mode }),
  clear: () => set({ selections: [], parlayStake: 0 }),

  hasSelection: (selectionId) => get().selections.some((x) => x.selectionId === selectionId),
}));

export function getCombinedOdds(selections: BetSelection[]): number {
  return selections.reduce((acc, s) => acc * s.odds, 1);
}

export function getParlayReturn(selections: BetSelection[], stake: number): number {
  return stake * getCombinedOdds(selections);
}

export function hasDuplicateEvents(selections: BetSelection[]): boolean {
  const eventIds = selections.map((s) => s.eventId);
  return new Set(eventIds).size !== eventIds.length;
}
