'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate, txTypeLabel } from '@/lib/format';

interface Player { id: string; email: string; currency: string; created_at: string; }
interface Engagement { player_id: string; score: number; level: string; }
interface Transaction { id: string; type: string; amount: number; created_at: string; }

export default function DashboardPage() {
  const token = useAuthStore((s) => s.token)!;
  const [player, setPlayer] = useState<Player | null>(null);
  const [engagement, setEngagement] = useState<Engagement | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api<Player>('/players/me', { token }),
      api<Engagement>('/engagement/me', { token }).catch(() => null),
      api<{ transactions: Transaction[] }>('/wallet/transactions?limit=5', { token }).catch(() => ({ transactions: [] })),
    ])
      .then(([p, e, t]) => { setPlayer(p); setEngagement(e); setTransactions(t?.transactions || []); })
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;
  if (!player) return null;

  return (
    <div className="mx-auto max-w-4xl space-y-6 animate-fade-in">
      {/* Welcome */}
      <div className="card">
        <h2 className="font-display text-2xl font-bold">Welcome back</h2>
        <p className="mt-1 text-sm text-text-secondary">{player.email}</p>
        <p className="text-xs text-text-muted">Member since {formatDate(player.created_at)}</p>
      </div>

      {/* Engagement */}
      {engagement && (
        <div className="card">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-text-muted">Engagement Score</h3>
          <div className="mt-3 flex items-baseline gap-3">
            <span className="font-display text-4xl font-bold text-brand-400 text-glow num">{engagement.score}</span>
            <span className="rounded-full bg-brand-400/15 px-3 py-1 text-xs font-semibold text-brand-400">
              {engagement.level}
            </span>
          </div>
        </div>
      )}

      {/* Quick Actions */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        {[
          { href: '/sportsbook', label: 'Sports', color: 'text-brand-400' },
          { href: '/slots', label: 'Slots', color: 'text-electric-cyan' },
          { href: '/predictions', label: 'Predict', color: 'text-electric-purple' },
          { href: '/quests', label: 'Quests', color: 'text-electric-magenta' },
        ].map((a) => (
          <Link key={a.href} href={a.href} className="card-glass text-center hover:border-brand-400/20 transition-colors">
            <span className={`font-display text-lg font-semibold ${a.color}`}>{a.label}</span>
          </Link>
        ))}
      </div>

      {/* Recent Transactions */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-text-muted">Recent Transactions</h3>
          <Link href="/history" className="text-xs text-brand-400 hover:underline">View all</Link>
        </div>
        {transactions.length === 0 ? (
          <p className="text-sm text-text-muted">No transactions yet</p>
        ) : (
          <div className="space-y-2">
            {transactions.map((tx) => (
              <div key={tx.id} className="flex items-center justify-between rounded-lg bg-surface-300 px-4 py-2.5">
                <div>
                  <span className="text-sm font-medium">{txTypeLabel(tx.type)}</span>
                  <p className="text-xs text-text-muted">{formatDate(tx.created_at)}</p>
                </div>
                <span className={`text-sm font-semibold num ${tx.amount >= 0 ? 'text-brand-400' : 'text-electric-magenta'}`}>
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
