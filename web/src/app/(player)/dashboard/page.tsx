'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Player {
  id: string;
  email: string;
  currency: string;
  created_at: string;
}

interface Engagement {
  player_id: string;
  score: number;
  level: string;
}

interface Transaction {
  id: string;
  type: string;
  amount: number;
  created_at: string;
}

export default function DashboardPage() {
  const token = useAuthStore((s) => s.token)!;
  const [player, setPlayer] = useState<Player | null>(null);
  const [engagement, setEngagement] = useState<Engagement | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api<Player>('/players/me', { token }),
      api<Engagement>('/engagement/me', { token }).catch(() => null),
      api<Transaction[]>('/wallet/transactions?limit=5', { token }).catch(() => []),
    ])
      .then(([p, e, t]) => {
        setPlayer(p);
        setEngagement(e);
        setTransactions(t || []);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;
  if (!player) return null;

  return (
    <div className="space-y-6">
      <div className="rounded-lg bg-white p-6 shadow-sm">
        <h2 className="text-xl font-semibold text-gray-900">Welcome back</h2>
        <p className="mt-1 text-sm text-gray-500">{player.email}</p>
        <p className="text-xs text-gray-400">Member since {formatDate(player.created_at)}</p>
      </div>

      {engagement && (
        <div className="rounded-lg bg-white p-6 shadow-sm">
          <h3 className="text-sm font-medium text-gray-500">Engagement</h3>
          <div className="mt-2 flex items-baseline gap-3">
            <span className="text-3xl font-bold text-indigo-600">{engagement.score}</span>
            <span className="rounded-full bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700">
              {engagement.level}
            </span>
          </div>
        </div>
      )}

      <div className="rounded-lg bg-white p-6 shadow-sm">
        <h3 className="mb-4 text-sm font-medium text-gray-500">Recent Transactions</h3>
        {transactions.length === 0 ? (
          <p className="text-sm text-gray-400">No transactions yet</p>
        ) : (
          <div className="space-y-2">
            {transactions.map((tx) => (
              <div key={tx.id} className="flex items-center justify-between rounded-md border border-gray-100 px-4 py-2">
                <div>
                  <span className="text-sm font-medium text-gray-700">{tx.type}</span>
                  <p className="text-xs text-gray-400">{formatDate(tx.created_at)}</p>
                </div>
                <span
                  className={`text-sm font-semibold ${tx.amount >= 0 ? 'text-green-600' : 'text-red-600'}`}
                >
                  {tx.amount >= 0 ? '+' : ''}${formatCents(tx.amount)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
