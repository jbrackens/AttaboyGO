'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/format';

interface Outcome { id: string; label: string; odds?: number; probability?: number; }
interface MarketMeta { image?: string; volume_total?: number; }
interface Market { id: string; title: string; description?: string; category: string; status: string; close_at: string; created_at: string; outcomes?: Outcome[]; source?: string; metadata?: MarketMeta; }

export default function PredictionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const token = useAuthStore((s) => s.token)!;

  const [market, setMarket] = useState<Market | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedOutcomeId, setSelectedOutcomeId] = useState('');
  const [stakeAmount, setStakeAmount] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    api<Market>(`/predictions/markets/${id}`, { token })
      .then(setMarket)
      .catch(() => {
        // Fallback: try fetching all markets and finding this one
        api<Market[]>('/predictions/markets', { token })
          .then((markets) => {
            const m = (markets || []).find((m) => m.id === id);
            if (m) setMarket(m);
          });
      })
      .finally(() => setLoading(false));
  }, [id, token]);

  async function handleStake(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedOutcomeId) { setError('Select an outcome'); return; }
    setError('');
    setSuccess('');
    setSubmitting(true);
    try {
      await api(`/predictions/markets/${id}/stake`, { method: 'POST', token, body: { outcome_id: selectedOutcomeId, amount: Math.round(parseFloat(stakeAmount) * 100) } });
      const selectedLabel = (market?.outcomes || []).find((o) => o.id === selectedOutcomeId)?.label || selectedOutcomeId;
      setSuccess(`Staked $${stakeAmount} on "${selectedLabel}"`);
      setStakeAmount('');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Stake failed');
    } finally {
      setSubmitting(false);
    }
  }

  const potentialPayout = stakeAmount ? (parseFloat(stakeAmount) * 2).toFixed(2) : null;

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;
  if (!market) return <p className="text-center text-text-muted py-12">Market not found</p>;

  const outcomes = market.outcomes || [];

  return (
    <div className="mx-auto max-w-2xl space-y-6 animate-fade-in">
      <button onClick={() => router.back()} className="text-sm text-brand-400 hover:underline">&larr; Back to predictions</button>

      <div className="card">
        <div className="flex items-start gap-4">
          {market.metadata?.image && (
            <div className="shrink-0 w-16 h-16 rounded-lg overflow-hidden bg-surface-200">
              <img src={market.metadata.image} alt="" className="w-full h-full object-cover" />
            </div>
          )}
          <div className="flex-1">
            <h1 className="font-display text-xl font-bold mb-2">{market.title}</h1>
            {market.description && <p className="text-sm text-text-secondary mb-2">{market.description}</p>}
            <div className="flex items-center gap-2 text-xs text-text-muted">
              <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${market.status === 'open' ? 'bg-brand-400/15 text-brand-400' : 'bg-surface-200 text-text-muted'}`}>
                {market.status}
              </span>
              <span>{market.category}</span>
              {market.source && <span className="rounded bg-electric-cyan/10 text-electric-cyan px-1.5 py-0.5 text-[10px] font-medium">{market.source}</span>}
              {market.close_at && <span>Closes {formatDate(market.close_at)}</span>}
            </div>
          </div>
        </div>
      </div>

      {/* Outcome selection with probability bars */}
      <div className="space-y-3">
        {outcomes.map((outcome) => {
          const prob = outcome.odds ?? outcome.probability ?? 0;
          return (
            <button
              key={outcome.id}
              onClick={() => setSelectedOutcomeId(outcome.id)}
              className={`relative w-full rounded-xl border px-5 py-4 text-left transition-all overflow-hidden ${selectedOutcomeId === outcome.id ? 'border-electric-purple bg-electric-purple/10 shadow-glow-brand' : 'border-surface-50 hover:border-electric-purple/30'}`}
            >
              <div className="absolute inset-y-0 left-0 bg-brand-400/8 transition-all" style={{ width: `${Math.min(prob, 100)}%` }} />
              <div className="relative flex items-center justify-between">
                <span className="font-display text-lg font-semibold">{outcome.label}</span>
                <span className="num text-sm font-bold text-brand-400">{prob.toFixed(1)}%</span>
              </div>
            </button>
          );
        })}
      </div>

      {/* Stake form */}
      {selectedOutcomeId && (
        <form onSubmit={handleStake} className="card space-y-4">
          <h3 className="text-sm font-semibold text-text-muted">
            Staking on: <span className="text-electric-purple">{outcomes.find((o) => o.id === selectedOutcomeId)?.label}</span>
          </h3>
          {error && <p className="text-xs text-electric-magenta">{error}</p>}
          {success && <p className="text-xs text-brand-400">{success}</p>}
          <input type="number" step="0.01" min="0.01" required placeholder="Amount ($)" value={stakeAmount} onChange={(e) => setStakeAmount(e.target.value)} className="input-field" />
          {potentialPayout && (
            <div className="flex items-center justify-between text-sm">
              <span className="text-text-muted">Potential payout</span>
              <span className="text-brand-400 font-semibold num">${potentialPayout}</span>
            </div>
          )}
          <button type="submit" disabled={submitting} className="btn-primary w-full">{submitting ? 'Staking...' : 'Place Stake'}</button>
        </form>
      )}
    </div>
  );
}
