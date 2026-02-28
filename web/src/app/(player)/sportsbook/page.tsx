'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate } from '@/lib/format';
import { SportsTabs } from '@/components/sportsbook/sports-tabs';
import { EventCard } from '@/components/sportsbook/event-card';
import { Betslip } from '@/components/sportsbook/betslip';

interface Sport { id: string; name: string; }
interface Event { id: string; name: string; starts_at: string; status: string; }
interface Bet { id: string; event_id: string; selection_name: string; stake: number; odds: number; status: string; created_at: string; }

export default function SportsbookPage() {
  const token = useAuthStore((s) => s.token)!;
  const [sports, setSports] = useState<Sport[]>([]);
  const [activeSport, setActiveSport] = useState<string | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [bets, setBets] = useState<Bet[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'events' | 'bets'>('events');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  useEffect(() => {
    Promise.all([
      api<Sport[]>('/sportsbook/sports', { token }),
      api<Bet[]>('/sportsbook/bets/me', { token }).catch(() => []),
    ])
      .then(([s, b]) => { setSports(s); setBets(b || []); if (s.length > 0) setActiveSport(s[0].id); })
      .finally(() => setLoading(false));
  }, [token]);

  useEffect(() => {
    if (!activeSport) return;
    api<Event[]>(`/sports/${activeSport}/events`, { token }).then(setEvents).catch(() => setEvents([]));
  }, [activeSport, token]);

  const filteredEvents = statusFilter === 'all' ? events : events.filter((e) => e.status === statusFilter);

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-6xl animate-fade-in">
      <h1 className="font-display text-2xl font-bold mb-6">Sportsbook</h1>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        <button onClick={() => setTab('events')} className={`rounded-lg px-4 py-2 text-sm font-medium transition-all ${tab === 'events' ? 'bg-brand-400 text-surface-900' : 'bg-surface-200 text-text-secondary border border-surface-50'}`}>
          Events
        </button>
        <button onClick={() => setTab('bets')} className={`rounded-lg px-4 py-2 text-sm font-medium transition-all ${tab === 'bets' ? 'bg-brand-400 text-surface-900' : 'bg-surface-200 text-text-secondary border border-surface-50'}`}>
          My Bets
        </button>
      </div>

      {tab === 'events' && (
        <div className="flex gap-6">
          {/* Main content */}
          <div className="flex-1 space-y-4">
            <SportsTabs sports={sports} activeSport={activeSport} onSelect={setActiveSport} />

            {/* Status filter */}
            <div className="flex gap-2">
              {['all', 'upcoming', 'live', 'settled'].map((s) => (
                <button key={s} onClick={() => setStatusFilter(s)} className={`text-xs rounded-full px-3 py-1 transition-colors ${statusFilter === s ? 'bg-surface-50 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>
                  {s.charAt(0).toUpperCase() + s.slice(1)}
                </button>
              ))}
            </div>

            {filteredEvents.length === 0 ? (
              <p className="text-sm text-text-muted py-8 text-center">No events found</p>
            ) : (
              <div className="space-y-3">
                {filteredEvents.map((event) => (
                  <EventCard key={event.id} {...event} />
                ))}
              </div>
            )}
          </div>

          {/* Betslip sidebar */}
          <div className="hidden lg:block w-80 shrink-0">
            <div className="sticky top-20">
              <Betslip />
            </div>
          </div>
        </div>
      )}

      {tab === 'bets' && (
        <div className="space-y-3">
          {bets.length === 0 ? (
            <div className="card-glass text-center py-12">
              <p className="text-text-muted">No bets placed yet</p>
            </div>
          ) : (
            bets.map((bet) => (
              <div key={bet.id} className="card">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">{bet.selection_name}</p>
                    <p className="text-xs text-text-muted">Odds: <span className="num">{bet.odds}</span> | {formatDate(bet.created_at)}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold num">${formatCents(bet.stake)}</p>
                    <span className={`text-xs font-medium ${bet.status === 'won' ? 'text-brand-400' : bet.status === 'lost' ? 'text-electric-magenta' : 'text-electric-cyan'}`}>
                      {bet.status}
                    </span>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {/* Mobile betslip */}
      <div className="lg:hidden fixed bottom-0 left-0 right-0 p-4 z-40">
        <Betslip />
      </div>
    </div>
  );
}
