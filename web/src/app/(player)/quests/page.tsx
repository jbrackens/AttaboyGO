'use client';

import { useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Quest {
  id: string;
  name: string;
  description: string;
  reward_cents: number;
  min_score: number;
  status: string;
}

interface Engagement {
  score: number;
  level: string;
}

export default function QuestsPage() {
  const token = useAuthStore((s) => s.token)!;
  const [quests, setQuests] = useState<Quest[]>([]);
  const [engagement, setEngagement] = useState<Engagement | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [claimError, setClaimError] = useState('');
  const [claiming, setClaiming] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      api<Quest[]>('/quests', { token }),
      api<Engagement>('/engagement/me', { token }).catch(() => null),
    ])
      .then(([q, e]) => {
        setQuests(q);
        setEngagement(e);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  async function handleClaim(questId: string) {
    setClaimError('');
    setClaiming(questId);
    try {
      await api(`/quests/${questId}/claim`, { method: 'POST', token });
      setQuests((prev) =>
        prev.map((q) => (q.id === questId ? { ...q, status: 'claimed' } : q)),
      );
    } catch (err) {
      setClaimError(err instanceof ApiError ? err.message : 'Claim failed');
    } finally {
      setClaiming(null);
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      {engagement && (
        <div className="rounded-lg bg-white p-5 shadow-sm">
          <div className="flex items-baseline gap-3">
            <span className="text-sm text-gray-500">Your Engagement Score:</span>
            <span className="text-2xl font-bold text-indigo-600">{engagement.score}</span>
            <span className="rounded-full bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700">
              {engagement.level}
            </span>
          </div>
        </div>
      )}

      <ErrorMessage message={claimError} />

      <div className="space-y-4">
        {quests.length === 0 ? (
          <p className="text-sm text-gray-400">No quests available</p>
        ) : (
          quests.map((quest) => {
            const score = engagement?.score ?? 0;
            const canClaim = quest.status !== 'claimed' && score >= quest.min_score;
            const progress = quest.min_score > 0 ? Math.min(100, (score / quest.min_score) * 100) : 100;

            return (
              <div key={quest.id} className="rounded-lg bg-white p-5 shadow-sm">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="font-medium text-gray-900">{quest.name}</p>
                    <p className="mt-1 text-sm text-gray-500">{quest.description}</p>
                    <p className="mt-1 text-xs text-gray-400">
                      Reward: ${(quest.reward_cents / 100).toFixed(2)} | Min score: {quest.min_score}
                    </p>
                  </div>
                  {quest.status === 'claimed' ? (
                    <span className="rounded-full bg-green-100 px-3 py-1 text-xs font-medium text-green-700">
                      Claimed
                    </span>
                  ) : (
                    <button
                      onClick={() => handleClaim(quest.id)}
                      disabled={!canClaim || claiming === quest.id}
                      className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                    >
                      {claiming === quest.id ? 'Claiming...' : 'Claim'}
                    </button>
                  )}
                </div>
                {quest.status !== 'claimed' && quest.min_score > 0 && (
                  <div className="mt-3">
                    <div className="h-2 rounded-full bg-gray-100">
                      <div
                        className="h-2 rounded-full bg-indigo-500 transition-all"
                        style={{ width: `${progress}%` }}
                      />
                    </div>
                    <p className="mt-1 text-xs text-gray-400">
                      {score} / {quest.min_score}
                    </p>
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
