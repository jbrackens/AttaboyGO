'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents } from '@/lib/utils';

interface Balance {
  balance: number;
  bonus_balance: number;
  reserved_balance: number;
  currency: string;
}

export function BalanceBar() {
  const token = useAuthStore((s) => s.token);
  const [bal, setBal] = useState<Balance | null>(null);

  useEffect(() => {
    if (!token) return;
    api<Balance>('/wallet/balance', { token }).then(setBal).catch(() => {});
  }, [token]);

  if (!bal) return null;

  return (
    <div className="flex items-center gap-6 border-b border-gray-200 bg-white px-6 py-3 text-sm">
      <span className="text-gray-500">
        Real: <span className="font-semibold text-gray-900">${formatCents(bal.balance)}</span>
      </span>
      <span className="text-gray-500">
        Bonus: <span className="font-semibold text-amber-600">${formatCents(bal.bonus_balance)}</span>
      </span>
      <span className="text-gray-500">
        Reserved: <span className="font-semibold text-gray-400">${formatCents(bal.reserved_balance)}</span>
      </span>
      <span className="ml-auto text-xs text-gray-400 uppercase">{bal.currency}</span>
    </div>
  );
}
