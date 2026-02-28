'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate } from '@/lib/format';

interface Outcome { id: string; label: string; odds?: number; probability?: number; }
interface MarketMeta { image?: string; volume_total?: number; end_time?: number; }
interface Market {
  id: string; title: string; description?: string; status: string; category: string;
  close_at: string; created_at: string; outcomes?: Outcome[]; source?: string;
  metadata?: MarketMeta; tags?: string[];
}
interface Position { id: string; market_id: string; market_title: string; outcome_id: string; stake_amount: number; status: string; placed_at: string; }

function formatVolume(v: number): string {
  if (v >= 1_000_000) return `$${(v / 1_000_000).toFixed(1)}M`;
  if (v >= 1_000) return `$${(v / 1_000).toFixed(0)}K`;
  return `$${v.toFixed(0)}`;
}

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
          <div className="flex gap-2 overflow-x-auto pb-1">
            {categories.map((c) => (
              <button key={c} onClick={() => setCategoryFilter(c)} className={`whitespace-nowrap rounded-full px-3 py-1 text-xs font-medium transition-colors ${categoryFilter === c ? 'bg-electric-purple text-white' : 'bg-surface-200 text-text-muted border border-surface-50'}`}>
                {c === 'all' ? 'All' : c.charAt(0).toUpperCase() + c.slice(1)}
              </button>
            ))}
          </div>

          {filteredMarkets.length === 0 ? (
            <p className="text-sm text-text-muted text-center py-8">No prediction markets available</p>
          ) : (
            filteredMarkets.map((market) => {
              const outcomes = market.outcomes || [];
              const volume = market.metadata?.volume_total;
              const image = market.metadata?.image;

              return (
                <div key={market.id} className="card">
                  <div className="flex items-start gap-4 mb-3">
                    {/* Thumbnail */}
                    {image && (
                      <div className="shrink-0 w-12 h-12 rounded-lg overflow-hidden bg-surface-200">
                        <img src={image} alt="" className="w-full h-full object-cover" />
                      </div>
                    )}

                    <div className="flex-1 min-w-0">
                      <Link href={`/predictions/${market.id}`} className="font-medium hover:text-brand-400 transition-colors line-clamp-2">{market.title}</Link>
                      <div className="flex items-center gap-2 mt-1 text-xs text-text-muted">
                        <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${market.status === 'open' ? 'bg-brand-400/15 text-brand-400' : 'bg-surface-200 text-text-muted'}`}>
                          {market.status}
                        </span>
                        <span>{market.category}</span>
                        {market.source && (
                          <span className="rounded bg-electric-cyan/10 text-electric-cyan px-1.5 py-0.5 text-[10px] font-medium">{market.source}</span>
                        )}
                        {volume != null && volume > 0 && (
                          <span className="num">{formatVolume(volume)} vol</span>
                        )}
                        {market.close_at && <span>Closes {formatDate(market.close_at)}</span>}
                      </div>
                    </div>
                  </div>

                  {/* Outcome buttons with probability bars */}
                  <div className="space-y-2">
                    {outcomes.map((outcome) => {
                      const prob = outcome.odds ?? outcome.probability ?? 0;
                      return (
                        <button
                          key={outcome.id}
                          onClick={() => { setStakeMarket(market.id); setStakeOutcomeId(outcome.id); setStakeError(''); }}
                          className={`relative w-full rounded-lg border px-4 py-3 text-sm font-medium text-left transition-all overflow-hidden ${
                            stakeMarket === market.id && stakeOutcomeId === outcome.id
                              ? 'border-electric-purple bg-electric-purple/15 text-electric-purple'
                              : 'border-surface-50 text-text-secondary hover:border-electric-purple/30'
                          }`}
                        >
                          {/* Probability fill bar */}
                          <div
                            className="absolute inset-y-0 left-0 bg-brand-400/8 transition-all"
                            style={{ width: `${Math.min(prob, 100)}%` }}
                          />
                          <div className="relative flex items-center justify-between">
                            <span>{outcome.label}</span>
                            <span className="num text-xs font-semibold">{prob.toFixed(1)}%</span>
                          </div>
                        </button>
                      );
                    })}
                  </div>

                  {/* Stake form */}
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
                  <p className="text-sm font-medium">{pos.market_title}</p>
                  <p className="text-xs text-text-muted">{pos.outcome_id} &middot; {formatDate(pos.placed_at)}</p>
                </div>
                <span className="text-sm font-semibold text-electric-purple num">${formatCents(pos.stake_amount)}</span>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
