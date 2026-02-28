'use client';

import { useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Market {
  id: string;
  question: string;
  outcome_a: string;
  outcome_b: string;
  status: string;
  created_at: string;
}

interface Position {
  id: string;
  market_id: string;
  outcome: string;
  amount: number;
  created_at: string;
}

export default function PredictionsPage() {
  const token = useAuthStore((s) => s.token)!;
  const [markets, setMarkets] = useState<Market[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'markets' | 'positions'>('markets');

  const [stakeMarket, setStakeMarket] = useState<string | null>(null);
  const [stakeOutcome, setStakeOutcome] = useState('');
  const [stakeAmount, setStakeAmount] = useState('');
  const [stakeError, setStakeError] = useState('');
  const [stakeLoading, setStakeLoading] = useState(false);

  useEffect(() => {
    Promise.all([
      api<Market[]>('/predictions/markets', { token }),
      api<Position[]>('/predictions/positions', { token }).catch(() => []),
    ])
      .then(([m, p]) => {
        setMarkets(m);
        setPositions(p || []);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  async function handleStake(e: React.FormEvent) {
    e.preventDefault();
    if (!stakeMarket) return;
    setStakeError('');
    setStakeLoading(true);
    try {
      await api(`/markets/${stakeMarket}/stake`, {
        method: 'POST',
        token,
        body: {
          outcome: stakeOutcome,
          amount: Math.round(parseFloat(stakeAmount) * 100),
        },
      });
      setStakeMarket(null);
      setStakeAmount('');
      const p = await api<Position[]>('/predictions/positions', { token }).catch(() => []);
      setPositions(p || []);
    } catch (err) {
      setStakeError(err instanceof ApiError ? err.message : 'Stake failed');
    } finally {
      setStakeLoading(false);
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      <div className="flex gap-2">
        <button
          onClick={() => setTab('markets')}
          className={`rounded-md px-4 py-2 text-sm font-medium ${tab === 'markets' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 shadow-sm'}`}
        >
          Markets
        </button>
        <button
          onClick={() => setTab('positions')}
          className={`rounded-md px-4 py-2 text-sm font-medium ${tab === 'positions' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 shadow-sm'}`}
        >
          My Positions
        </button>
      </div>

      {tab === 'markets' && (
        <div className="space-y-4">
          {markets.length === 0 ? (
            <p className="text-sm text-gray-400">No prediction markets available</p>
          ) : (
            markets.map((market) => (
              <div key={market.id} className="rounded-lg bg-white p-5 shadow-sm">
                <p className="font-medium text-gray-900">{market.question}</p>
                <p className="mt-1 text-xs text-gray-400">
                  {market.status} | {formatDate(market.created_at)}
                </p>
                <div className="mt-3 flex gap-2">
                  <button
                    onClick={() => {
                      setStakeMarket(market.id);
                      setStakeOutcome(market.outcome_a);
                      setStakeError('');
                    }}
                    className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium ${
                      stakeMarket === market.id && stakeOutcome === market.outcome_a
                        ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
                        : 'border-gray-200 text-gray-700 hover:bg-gray-50'
                    }`}
                  >
                    {market.outcome_a}
                  </button>
                  <button
                    onClick={() => {
                      setStakeMarket(market.id);
                      setStakeOutcome(market.outcome_b);
                      setStakeError('');
                    }}
                    className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium ${
                      stakeMarket === market.id && stakeOutcome === market.outcome_b
                        ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
                        : 'border-gray-200 text-gray-700 hover:bg-gray-50'
                    }`}
                  >
                    {market.outcome_b}
                  </button>
                </div>

                {stakeMarket === market.id && (
                  <form onSubmit={handleStake} className="mt-3 space-y-2">
                    <ErrorMessage message={stakeError} />
                    <div className="flex gap-2">
                      <input
                        type="number"
                        step="0.01"
                        min="0.01"
                        required
                        placeholder="Amount ($)"
                        value={stakeAmount}
                        onChange={(e) => setStakeAmount(e.target.value)}
                        className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                      <button
                        type="submit"
                        disabled={stakeLoading}
                        className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                      >
                        {stakeLoading ? 'Staking...' : 'Stake'}
                      </button>
                    </div>
                  </form>
                )}
              </div>
            ))
          )}
        </div>
      )}

      {tab === 'positions' && (
        <div className="space-y-2">
          {positions.length === 0 ? (
            <p className="text-sm text-gray-400">No positions yet</p>
          ) : (
            positions.map((pos) => (
              <div key={pos.id} className="flex items-center justify-between rounded-lg bg-white p-4 shadow-sm">
                <div>
                  <p className="text-sm font-medium text-gray-900">{pos.outcome}</p>
                  <p className="text-xs text-gray-400">{formatDate(pos.created_at)}</p>
                </div>
                <span className="text-sm font-semibold text-indigo-600">${formatCents(pos.amount)}</span>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
