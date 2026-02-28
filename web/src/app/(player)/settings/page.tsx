'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Player {
  id: string;
  email: string;
  currency: string;
  created_at: string;
}

export default function SettingsPage() {
  const token = useAuthStore((s) => s.token)!;
  const logout = useAuthStore((s) => s.logout);
  const [player, setPlayer] = useState<Player | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api<Player>('/players/me', { token })
      .then(setPlayer)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;
  if (!player) return null;

  return (
    <div className="space-y-6">
      <div className="rounded-lg bg-white p-6 shadow-sm">
        <h2 className="mb-4 text-lg font-semibold text-gray-900">Profile</h2>
        <dl className="space-y-3">
          <div>
            <dt className="text-sm text-gray-500">Player ID</dt>
            <dd className="text-sm font-medium text-gray-900 font-mono">{player.id}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500">Email</dt>
            <dd className="text-sm font-medium text-gray-900">{player.email}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500">Currency</dt>
            <dd className="text-sm font-medium text-gray-900">{player.currency}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500">Member since</dt>
            <dd className="text-sm font-medium text-gray-900">{formatDate(player.created_at)}</dd>
          </div>
        </dl>
      </div>

      <button
        onClick={() => {
          logout();
          window.location.href = '/login';
        }}
        className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
      >
        Log out
      </button>
    </div>
  );
}
