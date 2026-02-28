'use client';

import { useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents } from '@/lib/format';

interface Quest {
  id: string;
  name: string;
  description: string;
  type: string;
  target_progress: number;
  reward_amount: number;
  reward_currency: string;
  min_score: number;
  progress: number;
  status: string;
}
interface Engagement { score: number; level: string; }

export default function QuestsPage() {
  const token = useAuthStore((s) => s.token)!;
  const [quests, setQuests] = useState<Quest[]>([]);
  const [engagement, setEngagement] = useState<Engagement | null>(null);
  const [loading, setLoading] = useState(true);
  const [claimError, setClaimError] = useState('');
  const [claiming, setClaiming] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      api<Quest[]>('/quests/', { token }),
      api<Engagement>('/engagement/me', { token }).catch(() => null),
    ])
      .then(([q, e]) => { setQuests(q || []); setEngagement(e); })
      .finally(() => setLoading(false));
  }, [token]);

  async function handleClaim(questId: string) {
    setClaimError('');
    setClaiming(questId);
    try {
      await api(`/quests/${questId}/claim`, { method: 'POST', token });
      setQuests((prev) => prev.map((q) => (q.id === questId ? { ...q, status: 'claimed' } : q)));
    } catch (err) {
      setClaimError(err instanceof ApiError ? err.message : 'Claim failed');
    } finally {
      setClaiming(null);
    }
  }

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  const score = engagement?.score ?? 0;

  return (
    <div className="mx-auto max-w-4xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Quests</h1>

      {/* Engagement Score Display */}
      {engagement && (
        <div className="card">
          <div className="flex items-baseline gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-wider text-text-muted">Engagement Score</p>
              <p className="font-display text-4xl font-bold text-brand-400 text-glow num mt-1">{engagement.score}</p>
            </div>
            <span className="rounded-full bg-brand-400/15 px-3 py-1 text-xs font-semibold text-brand-400">{engagement.level}</span>
          </div>
        </div>
      )}

      {claimError && <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">{claimError}</div>}

      {/* Quest Cards */}
      {quests.length === 0 ? (
        <div className="card-glass text-center py-12"><p className="text-text-muted">No quests available</p></div>
      ) : (
        <div className="space-y-4">
          {quests.map((quest) => {
            const canClaim = quest.status !== 'claimed' && quest.progress >= quest.target_progress && score >= quest.min_score;
            const progress = quest.target_progress > 0 ? Math.min(100, (quest.progress / quest.target_progress) * 100) : 0;

            return (
              <div key={quest.id} className="card">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <p className="font-display text-lg font-semibold">{quest.name}</p>
                    <p className="mt-1 text-sm text-text-secondary">{quest.description}</p>
                    <p className="mt-2 text-xs text-text-muted">
                      Reward: <span className="text-brand-400 num">{formatCents(quest.reward_amount)}</span>
                      {quest.min_score > 0 && <> | Min score: <span className="num">{quest.min_score}</span></>}
                    </p>
                  </div>
                  {quest.status === 'claimed' ? (
                    <span className="rounded-full bg-brand-400/15 px-3 py-1 text-xs font-semibold text-brand-400 shrink-0">Claimed</span>
                  ) : (
                    <button
                      onClick={() => handleClaim(quest.id)}
                      disabled={!canClaim || claiming === quest.id}
                      className="btn-primary text-sm shrink-0"
                    >
                      {claiming === quest.id ? 'Claiming...' : 'Claim'}
                    </button>
                  )}
                </div>

                {quest.status !== 'claimed' && (
                  <div className="mt-4">
                    <div className="h-2 rounded-full bg-surface-300 overflow-hidden">
                      <div className="h-2 rounded-full bg-brand-400 transition-all" style={{ width: `${progress}%` }} />
                    </div>
                    <p className="mt-1 text-xs text-text-muted num">{quest.progress} / {quest.target_progress}</p>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
