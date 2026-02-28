'use client';

import { useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

export default function VideoPage() {
  const token = useAuthStore((s) => s.token)!;
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function createSession() {
    setError('');
    setLoading(true);
    try {
      const res = await api<{ session_id: string }>('/video/sessions', { method: 'POST', token, body: {} });
      setSessionId(res.session_id);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to create session');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Video</h1>

      {error && <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">{error}</div>}

      {!sessionId ? (
        <div className="card-glass text-center py-16">
          <div className="text-4xl mb-4">📹</div>
          <h2 className="font-display text-xl font-semibold mb-2">Start a Video Session</h2>
          <p className="text-sm text-text-muted mb-6">Connect with other players in real-time</p>
          <button onClick={createSession} disabled={loading} className="btn-primary">
            {loading ? 'Creating...' : 'Start Session'}
          </button>
        </div>
      ) : (
        <div className="card space-y-4">
          <div className="aspect-video rounded-xl bg-surface-300 flex items-center justify-center">
            <p className="text-text-muted">Video session: {sessionId}</p>
          </div>
          <button onClick={() => setSessionId(null)} className="btn-secondary text-sm">End Session</button>
        </div>
      )}
    </div>
  );
}
