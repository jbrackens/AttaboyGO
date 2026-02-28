'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate } from '@/lib/format';

interface Outcome { id: string; label: string; probability?: number; }
interface Market { id: string; title: string; status: string; category: string; close_at: string; created_at: string; outcomes?: Outcome[]; }
interface Position { id: string; market_id: string; outcome_id: string; outcome_label?: string; amount: number; created_at: string; }

export default function PredictionsPage() {
  const token = useAuthStore((s) => s.token)!;
  const [markets, setMarkets] = useState<Market[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'markets' | 'positions'>('markets');
  const [categoryFilter, setCategoryFilter] = useState('all');

  const [stakeMarket, setStakeMarket] = useState<string | null>(null);
  const [stakeOutcomeId, setStakeOutcomeId] = useState('');
  const [stakeAmount, setStakeAmount] = useState('');
  const [stakeError, setStakeError] = useState('');
  const [stakeLoading, setStakeLoading] = useState(false);

  useEffect(() => {
    Promise.all([
      api<Market[]>('/predictions/markets', { token }),
      api<Position[]>('/predictions/positions', { token }).catch(() => []),
    ])
      .then(([m, p]) => { setMarkets(m || []); setPositions(p || []); })
      .finally(() => setLoading(false));
  }, [token]);

  const categories = ['all', ...Array.from(new Set(markets.map((m) => m.category || 'general')))];
  const filteredMarkets = categoryFilter === 'all' ? markets : markets.filter((m) => (m.category || 'general') === categoryFilter);

  async function handleStake(e: React.FormEvent) {
    e.preventDefault();
    if (!stakeMarket || !stakeOutcomeId) return;
    setStakeError('');
    setStakeLoading(true);
    try {
      await api(`/predictions/markets/${stakeMarket}/stake`, { method: 'POST', token, body: { outcome_id: stakeOutcomeId, amount: Math.round(parseFloat(stakeAmount) * 100) } });
      setStakeMarket(null);
      setStakeAmount('');
      setStakeOutcomeId('');
      const p = await api<Position[]>('/predictions/positions', { token }).catch(() => []);
      setPositions(p || []);
    } catch (err) {
      setStakeError(err instanceof ApiError ? err.message : 'Stake failed');
    } finally {
      setStakeLoading(false);
    }
  }

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-4xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Predictions</h1>

      <div className="flex gap-2">
        <button onClick={() => setTab('markets')} className={`rounded-lg px-4 py-2 text-sm font-medium transition-all ${tab === 'markets' ? 'bg-brand-400 text-surface-900' : 'bg-surface-200 text-text-secondary border border-surface-50'}`}>Markets</button>
        <button onClick={() => setTab('positions')} className={`rounded-lg px-4 py-2 text-sm font-medium transition-all ${tab === 'positions' ? 'bg-brand-400 text-surface-900' : 'bg-surface-200 text-text-secondary border border-surface-50'}`}>My Positions</button>
      </div>

      {tab === 'markets' && (
        <div className="space-y-4">
          {/* Category filter */}
          <div className="flex gap-2 overflow-x-auto">
            {categories.map((c) => (
              <button key={c} onClick={() => setCategoryFilter(c)} className={`whitespace-nowrap rounded-full px-3 py-1 text-xs font-medium transition-colors ${categoryFilter === c ? 'bg-electric-purple text-white' : 'bg-surface-200 text-text-muted border border-surface-50'}`}>
                {c === 'all' ? 'All' : c}
              </button>
            ))}
          </div>

          {filteredMarkets.length === 0 ? (
            <p className="text-sm text-text-muted text-center py-8">No prediction markets available</p>
          ) : (
            filteredMarkets.map((market) => {
              const outcomes = market.outcomes || [];
              return (
                <div key={market.id} className="card">
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <Link href={`/predictions/${market.id}`} className="font-medium hover:text-brand-400 transition-colors">{market.title}</Link>
                      <p className="text-xs text-text-muted mt-1">{market.status} | Closes {formatDate(market.close_at)}</p>
                    </div>
                  </div>

                  {/* Outcome buttons */}
                  <div className="grid grid-cols-2 gap-2">
                    {outcomes.map((outcome) => (
                      <button
                        key={outcome.id}
                        onClick={() => { setStakeMarket(market.id); setStakeOutcomeId(outcome.id); setStakeError(''); }}
                        className={`rounded-lg border px-4 py-3 text-sm font-medium text-left transition-all ${
                          stakeMarket === market.id && stakeOutcomeId === outcome.id
                            ? 'border-electric-purple bg-electric-purple/15 text-electric-purple'
                            : 'border-surface-50 text-text-secondary hover:border-electric-purple/30'
                        }`}
                      >
                        <span>{outcome.label}</span>
                        {outcome.probability != null && (
                          <span className="ml-2 text-xs text-text-muted num">{outcome.probability}%</span>
                        )}
                      </button>
                    ))}
                  </div>

                  {stakeMarket === market.id && stakeOutcomeId && (
                    <form onSubmit={handleStake} className="mt-3 space-y-2">
                      {stakeError && <p className="text-xs text-electric-magenta">{stakeError}</p>}
                      <div className="flex gap-2">
                        <input type="number" step="0.01" min="0.01" required placeholder="Amount ($)" value={stakeAmount} onChange={(e) => setStakeAmount(e.target.value)} className="input-field flex-1" />
                        <button type="submit" disabled={stakeLoading} className="btn-primary">{stakeLoading ? 'Staking...' : 'Stake'}</button>
                      </div>
                    </form>
                  )}
                </div>
              );
            })
          )}
        </div>
      )}

      {tab === 'positions' && (
        <div className="space-y-3">
          {positions.length === 0 ? (
            <div className="card-glass text-center py-12"><p className="text-text-muted">No positions yet</p></div>
          ) : (
            positions.map((pos) => (
              <div key={pos.id} className="card flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">{pos.outcome_label || pos.outcome_id}</p>
                  <p className="text-xs text-text-muted">{formatDate(pos.created_at)}</p>
                </div>
                <span className="text-sm font-semibold text-electric-purple num">{formatCents(pos.amount)}</span>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
