'use client';

import { create } from 'zustand';

interface BalanceState {
  balance: number;
  bonusBalance: number;
  reservedBalance: number;
  currency: string;
  setBalances: (b: { balance: number; bonusBalance: number; reservedBalance: number; currency: string }) => void;
}

export const useBalanceStore = create<BalanceState>()((set) => ({
  balance: 0,
  bonusBalance: 0,
  reservedBalance: 0,
  currency: 'USD',
  setBalances: (b) => set(b),
}));
