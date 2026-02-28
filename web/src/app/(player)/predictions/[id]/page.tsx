'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/format';

interface Market { id: string; question: string; outcome_a: string; outcome_b: string; status: string; created_at: string; }

export default function PredictionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const token = useAuthStore((s) => s.token)!;

  const [market, setMarket] = useState<Market | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedOutcome, setSelectedOutcome] = useState('');
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
            const m = markets.find((m) => m.id === id);
            if (m) setMarket(m);
          });
      })
      .finally(() => setLoading(false));
  }, [id, token]);

  async function handleStake(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedOutcome) { setError('Select an outcome'); return; }
    setError('');
    setSuccess('');
    setSubmitting(true);
    try {
      await api(`/markets/${id}/stake`, { method: 'POST', token, body: { outcome: selectedOutcome, amount: Math.round(parseFloat(stakeAmount) * 100) } });
      setSuccess(`Staked $${stakeAmount} on "${selectedOutcome}"`);
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

  return (
    <div className="mx-auto max-w-2xl space-y-6 animate-fade-in">
      <button onClick={() => router.back()} className="text-sm text-brand-400 hover:underline">&larr; Back to predictions</button>

      <div className="card">
        <h1 className="font-display text-xl font-bold mb-2">{market.question}</h1>
        <p className="text-xs text-text-muted">{market.status} | {formatDate(market.created_at)}</p>
      </div>

      {/* Outcome selection */}
      <div className="grid grid-cols-2 gap-4">
        <button
          onClick={() => setSelectedOutcome(market.outcome_a)}
          className={`card text-center transition-all ${selectedOutcome === market.outcome_a ? 'border-electric-purple shadow-glow-brand' : 'hover:border-electric-purple/30'}`}
        >
          <span className="font-display text-lg font-semibold">{market.outcome_a}</span>
        </button>
        <button
          onClick={() => setSelectedOutcome(market.outcome_b)}
          className={`card text-center transition-all ${selectedOutcome === market.outcome_b ? 'border-electric-purple shadow-glow-brand' : 'hover:border-electric-purple/30'}`}
        >
          <span className="font-display text-lg font-semibold">{market.outcome_b}</span>
        </button>
      </div>

      {/* Stake form */}
      {selectedOutcome && (
        <form onSubmit={handleStake} className="card space-y-4">
          <h3 className="text-sm font-semibold text-text-muted">
            Staking on: <span className="text-electric-purple">{selectedOutcome}</span>
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
