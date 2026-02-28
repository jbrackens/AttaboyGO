'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Market {
  id: string;
  name: string;
  status: string;
}

interface Selection {
  id: string;
  name: string;
  odds: number;
}

export default function EventDetailPage() {
  const { eventId } = useParams<{ eventId: string }>();
  const router = useRouter();
  const token = useAuthStore((s) => s.token)!;

  const [markets, setMarkets] = useState<Market[]>([]);
  const [selections, setSelections] = useState<Record<string, Selection[]>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [selectedSelection, setSelectedSelection] = useState<Selection | null>(null);
  const [stake, setStake] = useState('');
  const [betError, setBetError] = useState('');
  const [betLoading, setBetLoading] = useState(false);
  const [betSuccess, setBetSuccess] = useState('');

  useEffect(() => {
    api<Market[]>(`/events/${eventId}/markets`, { token })
      .then(async (mkts) => {
        setMarkets(mkts);
        const selMap: Record<string, Selection[]> = {};
        await Promise.all(
          mkts.map(async (m) => {
            const sels = await api<Selection[]>(`/markets/${m.id}/selections`, { token }).catch(() => []);
            selMap[m.id] = sels || [];
          }),
        );
        setSelections(selMap);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [eventId, token]);

  async function handlePlaceBet(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedSelection) return;
    setBetError('');
    setBetSuccess('');
    setBetLoading(true);
    try {
      await api('/sportsbook/bets', {
        method: 'POST',
        token,
        body: {
          selection_id: selectedSelection.id,
          stake: Math.round(parseFloat(stake) * 100),
        },
      });
      setBetSuccess(`Bet placed: $${stake} on ${selectedSelection.name}`);
      setStake('');
      setSelectedSelection(null);
    } catch (err) {
      setBetError(err instanceof ApiError ? err.message : 'Failed to place bet');
    } finally {
      setBetLoading(false);
    }
  }

  const payout = selectedSelection && stake ? (parseFloat(stake) * selectedSelection.odds).toFixed(2) : null;

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      <button onClick={() => router.back()} className="text-sm text-indigo-600 hover:underline">
        &larr; Back to sportsbook
      </button>

      {markets.length === 0 ? (
        <p className="text-sm text-gray-400">No markets available for this event</p>
      ) : (
        markets.map((market) => (
          <div key={market.id} className="rounded-lg bg-white p-5 shadow-sm">
            <h3 className="mb-3 text-sm font-semibold text-gray-900">{market.name}</h3>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              {(selections[market.id] || []).map((sel) => (
                <button
                  key={sel.id}
                  onClick={() => setSelectedSelection(sel)}
                  className={`rounded-md border px-3 py-2 text-sm transition-colors ${
                    selectedSelection?.id === sel.id
                      ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
                      : 'border-gray-200 hover:border-indigo-300'
                  }`}
                >
                  <span className="block font-medium">{sel.name}</span>
                  <span className="text-indigo-600 font-semibold">{sel.odds}</span>
                </button>
              ))}
            </div>
          </div>
        ))
      )}

      {selectedSelection && (
        <div className="rounded-lg bg-indigo-50 p-5 border border-indigo-200">
          <h3 className="mb-3 text-sm font-semibold text-indigo-900">Bet Slip</h3>
          <p className="mb-2 text-sm text-indigo-700">
            {selectedSelection.name} @ <span className="font-semibold">{selectedSelection.odds}</span>
          </p>
          <ErrorMessage message={betError} />
          {betSuccess && (
            <div className="mb-2 rounded-md bg-green-50 border border-green-200 px-4 py-2 text-sm text-green-700">
              {betSuccess}
            </div>
          )}
          <form onSubmit={handlePlaceBet} className="flex items-end gap-3">
            <div className="flex-1">
              <label className="block text-xs font-medium text-indigo-700">Stake ($)</label>
              <input
                type="number"
                step="0.01"
                min="0.01"
                required
                value={stake}
                onChange={(e) => setStake(e.target.value)}
                className="mt-1 block w-full rounded-md border border-indigo-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            {payout && (
              <p className="pb-2 text-sm text-indigo-700">
                Payout: <span className="font-semibold">${payout}</span>
              </p>
            )}
            <button
              type="submit"
              disabled={betLoading}
              className="rounded-md bg-indigo-600 px-5 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {betLoading ? 'Placing...' : 'Place Bet'}
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
