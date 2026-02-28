'use client';

import Link from 'next/link';
import { formatDate } from '@/lib/format';

interface EventCardProps {
  id: string;
  home_team: string;
  away_team: string;
  league?: string;
  start_time: string;
  status: string;
  score_home?: number;
  score_away?: number;
}

const STATUS_BADGE: Record<string, string> = {
  live: 'badge-live',
  upcoming: 'badge-upcoming',
  settled: 'badge-settled',
};

export function EventCard({ id, home_team, away_team, league, start_time, status, score_home, score_away }: EventCardProps) {
  return (
    <Link
      href={`/sportsbook/${id}`}
      className="card group hover:border-brand-400/20 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <p className="font-medium truncate">{home_team} vs {away_team}</p>
          <div className="flex items-center gap-2 mt-1">
            {league && <span className="text-xs text-text-muted">{league}</span>}
            <span className="text-xs text-text-muted">{formatDate(start_time)}</span>
          </div>
          {status === 'live' && (
            <p className="text-xs font-semibold text-brand-400 mt-1 num">{score_home} - {score_away}</p>
          )}
        </div>
        <span className={STATUS_BADGE[status] || 'badge-upcoming'}>{status}</span>
      </div>
    </Link>
  );
}
