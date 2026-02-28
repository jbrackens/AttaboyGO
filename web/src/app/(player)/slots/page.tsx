'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

interface Game {
  id: string;
  name: string;
  provider: string;
  rtp: number;
}

export default function SlotsLobbyPage() {
  const token = useAuthStore((s) => s.token)!;
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [providerFilter, setProviderFilter] = useState('all');

  useEffect(() => {
    api<Game[]>('/slots/games', { token }).then((g) => setGames(g || [])).finally(() => setLoading(false));
  }, [token]);

  const providers = ['all', ...Array.from(new Set(games.map((g) => g.provider)))];
  const filtered = games
    .filter((g) => providerFilter === 'all' || g.provider === providerFilter)
    .filter((g) => !search || g.name.toLowerCase().includes(search.toLowerCase()));

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-6xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Slots</h1>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <input
          type="text"
          placeholder="Search games..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="input-field sm:max-w-xs"
        />
        <div className="flex gap-2 overflow-x-auto">
          {providers.map((p) => (
            <button
              key={p}
              onClick={() => setProviderFilter(p)}
              className={`whitespace-nowrap rounded-full px-3 py-1 text-xs font-medium transition-all ${
                providerFilter === p
                  ? 'bg-brand-400 text-surface-900'
                  : 'bg-surface-200 text-text-secondary border border-surface-50'
              }`}
            >
              {p === 'all' ? 'All Providers' : p}
            </button>
          ))}
        </div>
      </div>

      {/* Game grid */}
      {filtered.length === 0 ? (
        <p className="text-sm text-text-muted text-center py-12">No games found</p>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {filtered.map((game) => (
            <Link
              key={game.id}
              href={`/slots/play?game=${game.id}`}
              className="card group hover:border-brand-400/20 transition-all"
            >
              <div className="aspect-video rounded-lg bg-surface-300 mb-3 flex items-center justify-center">
                <span className="font-display text-2xl text-text-muted">SLOTS</span>
              </div>
              <p className="font-medium text-sm truncate">{game.name}</p>
              <p className="text-xs text-text-muted">{game.provider}</p>
              <div className="mt-2 flex items-center justify-between">
                <span className="text-xs text-text-muted">RTP</span>
                <span className="text-xs font-semibold text-brand-400 num">{(game.rtp * 100).toFixed(1)}%</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
