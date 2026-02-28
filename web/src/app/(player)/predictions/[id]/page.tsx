'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/format';

interface Outcome { id: string; label: string; probability?: number; }
interface Market { id: string; title: string; description?: string; category: string; status: string; close_at: string; created_at: string; outcomes?: Outcome[]; }

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
        <h1 className="font-display text-xl font-bold mb-2">{market.title}</h1>
        {market.description && <p className="text-sm text-text-secondary mb-2">{market.description}</p>}
        <p className="text-xs text-text-muted">{market.status} | Closes {formatDate(market.close_at)}</p>
      </div>

      {/* Outcome selection */}
      <div className="grid grid-cols-2 gap-4">
        {outcomes.map((outcome) => (
          <button
            key={outcome.id}
            onClick={() => setSelectedOutcomeId(outcome.id)}
            className={`card text-center transition-all ${selectedOutcomeId === outcome.id ? 'border-electric-purple shadow-glow-brand' : 'hover:border-electric-purple/30'}`}
          >
            <span className="font-display text-lg font-semibold">{outcome.label}</span>
            {outcome.probability != null && (
              <p className="text-xs text-text-muted mt-1 num">{outcome.probability}%</p>
            )}
          </button>
        ))}
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
