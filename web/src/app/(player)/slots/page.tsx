'use client';

import { useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Game {
  id: string;
  name: string;
  provider: string;
  rtp: number;
}

interface SpinResult {
  symbols: string[];
  payout: number;
  win: boolean;
}

export default function SlotsPage() {
  const token = useAuthStore((s) => s.token)!;
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [activeGame, setActiveGame] = useState<string | null>(null);
  const [betAmount, setBetAmount] = useState('1.00');
  const [spinning, setSpinning] = useState(false);
  const [result, setResult] = useState<SpinResult | null>(null);
  const [spinError, setSpinError] = useState('');

  useEffect(() => {
    api<Game[]>('/slots/games', { token })
      .then(setGames)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  async function handleSpin() {
    if (!activeGame) return;
    setSpinError('');
    setResult(null);
    setSpinning(true);
    try {
      const res = await api<SpinResult>('/slots/spin', {
        method: 'POST',
        token,
        body: {
          game_id: activeGame,
          bet: Math.round(parseFloat(betAmount) * 100),
        },
      });
      setResult(res);
    } catch (err) {
      setSpinError(err instanceof ApiError ? err.message : 'Spin failed');
    } finally {
      setSpinning(false);
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-gray-900">Slots</h2>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        {games.map((game) => (
          <button
            key={game.id}
            onClick={() => {
              setActiveGame(activeGame === game.id ? null : game.id);
              setResult(null);
              setSpinError('');
            }}
            className={`rounded-lg p-4 text-left shadow-sm transition-shadow ${
              activeGame === game.id
                ? 'bg-indigo-50 border-2 border-indigo-400'
                : 'bg-white border border-gray-200 hover:shadow-md'
            }`}
          >
            <p className="font-medium text-gray-900">{game.name}</p>
            <p className="text-xs text-gray-500">{game.provider}</p>
            <p className="mt-1 text-xs text-gray-400">RTP: {(game.rtp * 100).toFixed(1)}%</p>
          </button>
        ))}
      </div>

      {activeGame && (
        <div className="rounded-lg bg-white p-6 shadow-sm">
          <h3 className="mb-4 text-sm font-semibold text-gray-900">
            {games.find((g) => g.id === activeGame)?.name}
          </h3>
          <ErrorMessage message={spinError} />

          {result && (
            <div
              className={`mb-4 rounded-lg p-4 text-center ${
                result.win ? 'bg-green-50 border border-green-200' : 'bg-gray-50 border border-gray-200'
              }`}
            >
              <div className="mb-2 flex justify-center gap-2 text-3xl">
                {result.symbols.map((s, i) => (
                  <span key={i} className="rounded-md bg-white px-3 py-1 shadow-sm">
                    {s}
                  </span>
                ))}
              </div>
              {result.win ? (
                <p className="text-lg font-bold text-green-600">Win! +${formatCents(result.payout)}</p>
              ) : (
                <p className="text-sm text-gray-500">No win this time</p>
              )}
            </div>
          )}

          <div className="flex items-end gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-700">Bet ($)</label>
              <input
                type="number"
                step="0.01"
                min="0.01"
                value={betAmount}
                onChange={(e) => setBetAmount(e.target.value)}
                className="mt-1 w-28 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <button
              onClick={handleSpin}
              disabled={spinning}
              className="rounded-md bg-indigo-600 px-6 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {spinning ? 'Spinning...' : 'Spin'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
