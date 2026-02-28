'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

interface Game { id: string; name: string; provider: string; category?: string; rtp?: number; }

export default function LobbyPage() {
  const token = useAuthStore((s) => s.token)!;
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');

  useEffect(() => {
    api<Game[]>('/slots/games', { token }).then(setGames).catch(() => {}).finally(() => setLoading(false));
  }, [token]);

  const categories = ['all', ...Array.from(new Set(games.map((g) => g.category || g.provider)))];
  const filtered = games
    .filter((g) => categoryFilter === 'all' || (g.category || g.provider) === categoryFilter)
    .filter((g) => !search || g.name.toLowerCase().includes(search.toLowerCase()));

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-6xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Game Lobby</h1>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <input type="text" placeholder="Search games..." value={search} onChange={(e) => setSearch(e.target.value)} className="input-field sm:max-w-xs" />
        <div className="flex gap-2 overflow-x-auto">
          {categories.map((c) => (
            <button key={c} onClick={() => setCategoryFilter(c)} className={`whitespace-nowrap rounded-full px-3 py-1 text-xs font-medium transition-all ${categoryFilter === c ? 'bg-brand-400 text-surface-900' : 'bg-surface-200 text-text-secondary border border-surface-50'}`}>
              {c === 'all' ? 'All' : c}
            </button>
          ))}
        </div>
      </div>

      {/* Game Grid */}
      {filtered.length === 0 ? (
        <p className="text-sm text-text-muted text-center py-12">No games found</p>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
          {filtered.map((game) => (
            <Link key={game.id} href={`/game/${game.id}`} className="card group hover:border-brand-400/20 transition-all p-4">
              <div className="aspect-square rounded-lg bg-surface-300 mb-3 flex items-center justify-center">
                <span className="font-display text-xl text-text-muted">GAME</span>
              </div>
              <p className="font-medium text-sm truncate">{game.name}</p>
              <p className="text-xs text-text-muted">{game.provider}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
