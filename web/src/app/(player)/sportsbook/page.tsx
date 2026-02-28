'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Sport {
  id: string;
  name: string;
}

interface Event {
  id: string;
  name: string;
  starts_at: string;
  status: string;
}

interface Bet {
  id: string;
  event_id: string;
  selection_name: string;
  stake: number;
  odds: number;
  status: string;
  created_at: string;
}

export default function SportsbookPage() {
  const token = useAuthStore((s) => s.token)!;
  const [sports, setSports] = useState<Sport[]>([]);
  const [activeSport, setActiveSport] = useState<string | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [bets, setBets] = useState<Bet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'events' | 'bets'>('events');

  useEffect(() => {
    Promise.all([
      api<Sport[]>('/sportsbook/sports', { token }),
      api<Bet[]>('/sportsbook/bets/me', { token }).catch(() => []),
    ])
      .then(([s, b]) => {
        setSports(s);
        setBets(b || []);
        if (s.length > 0) setActiveSport(s[0].id);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  useEffect(() => {
    if (!activeSport) return;
    api<Event[]>(`/sports/${activeSport}/events`, { token })
      .then(setEvents)
      .catch(() => setEvents([]));
  }, [activeSport, token]);

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      <div className="flex gap-2">
        <button
          onClick={() => setTab('events')}
          className={`rounded-md px-4 py-2 text-sm font-medium ${tab === 'events' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 shadow-sm'}`}
        >
          Events
        </button>
        <button
          onClick={() => setTab('bets')}
          className={`rounded-md px-4 py-2 text-sm font-medium ${tab === 'bets' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 shadow-sm'}`}
        >
          My Bets
        </button>
      </div>

      {tab === 'events' && (
        <>
          <div className="flex gap-2 overflow-x-auto">
            {sports.map((sport) => (
              <button
                key={sport.id}
                onClick={() => setActiveSport(sport.id)}
                className={`whitespace-nowrap rounded-full px-4 py-1.5 text-sm font-medium ${
                  activeSport === sport.id
                    ? 'bg-indigo-100 text-indigo-700'
                    : 'bg-white text-gray-600 shadow-sm hover:bg-gray-50'
                }`}
              >
                {sport.name}
              </button>
            ))}
          </div>

          <div className="space-y-2">
            {events.length === 0 ? (
              <p className="text-sm text-gray-400">No events for this sport</p>
            ) : (
              events.map((event) => (
                <Link
                  key={event.id}
                  href={`/sportsbook/${event.id}`}
                  className="block rounded-lg bg-white p-4 shadow-sm hover:shadow-md transition-shadow"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium text-gray-900">{event.name}</p>
                      <p className="text-xs text-gray-400">{formatDate(event.starts_at)}</p>
                    </div>
                    <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
                      {event.status}
                    </span>
                  </div>
                </Link>
              ))
            )}
          </div>
        </>
      )}

      {tab === 'bets' && (
        <div className="space-y-2">
          {bets.length === 0 ? (
            <p className="text-sm text-gray-400">No bets placed yet</p>
          ) : (
            bets.map((bet) => (
              <div key={bet.id} className="rounded-lg bg-white p-4 shadow-sm">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{bet.selection_name}</p>
                    <p className="text-xs text-gray-400">Odds: {bet.odds} | {formatDate(bet.created_at)}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold text-gray-900">${formatCents(bet.stake)}</p>
                    <span
                      className={`text-xs font-medium ${
                        bet.status === 'won' ? 'text-green-600' : bet.status === 'lost' ? 'text-red-600' : 'text-amber-600'
                      }`}
                    >
                      {bet.status}
                    </span>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
