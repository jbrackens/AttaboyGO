'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/format';

interface Player {
  id: string;
  email: string;
  currency: string;
  created_at: string;
  first_name?: string;
  last_name?: string;
}

export default function ProfilePage() {
  const token = useAuthStore((s) => s.token)!;
  const logout = useAuthStore((s) => s.logout);
  const [player, setPlayer] = useState<Player | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api<Player>('/players/me', { token })
      .then(setPlayer)
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;
  if (!player) return null;

  return (
    <div className="mx-auto max-w-2xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Profile</h1>

      {/* Account Info */}
      <div className="card">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-4">Account Information</h3>
        <dl className="space-y-4">
          {[
            { label: 'Player ID', value: player.id, mono: true },
            { label: 'Email', value: player.email },
            { label: 'Currency', value: player.currency },
            { label: 'Member since', value: formatDate(player.created_at) },
          ].map((item) => (
            <div key={item.label} className="flex items-center justify-between rounded-lg bg-surface-300 px-4 py-3">
              <dt className="text-sm text-text-muted">{item.label}</dt>
              <dd className={`text-sm font-medium ${item.mono ? 'font-mono' : ''}`}>{item.value}</dd>
            </div>
          ))}
        </dl>
      </div>

      {/* Personal Details */}
      {(player.first_name || player.last_name) && (
        <div className="card">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-4">Personal Details</h3>
          <dl className="space-y-4">
            {player.first_name && (
              <div className="flex items-center justify-between rounded-lg bg-surface-300 px-4 py-3">
                <dt className="text-sm text-text-muted">First name</dt>
                <dd className="text-sm font-medium">{player.first_name}</dd>
              </div>
            )}
            {player.last_name && (
              <div className="flex items-center justify-between rounded-lg bg-surface-300 px-4 py-3">
                <dt className="text-sm text-text-muted">Last name</dt>
                <dd className="text-sm font-medium">{player.last_name}</dd>
              </div>
            )}
          </dl>
        </div>
      )}

      <button
        onClick={() => { logout(); window.location.href = '/login'; }}
        className="inline-flex items-center justify-center rounded-lg bg-electric-magenta px-5 py-2.5 text-sm font-semibold text-white transition-all hover:bg-electric-magenta/80"
      >
        Log out
      </button>
    </div>
  );
}
