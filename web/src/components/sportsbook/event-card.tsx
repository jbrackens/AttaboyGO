'use client';

import Link from 'next/link';
import { formatDate } from '@/lib/format';

interface EventCardProps {
  id: string;
  name: string;
  starts_at: string;
  status: string;
}

const STATUS_BADGE: Record<string, string> = {
  live: 'badge-live',
  upcoming: 'badge-upcoming',
  settled: 'badge-settled',
};

export function EventCard({ id, name, starts_at, status }: EventCardProps) {
  return (
    <Link
      href={`/sportsbook/${id}`}
      className="card group hover:border-brand-400/20 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <p className="font-medium truncate">{name}</p>
          <p className="text-xs text-text-muted mt-1">{formatDate(starts_at)}</p>
        </div>
        <span className={STATUS_BADGE[status] || 'badge-upcoming'}>{status}</span>
      </div>
    </Link>
  );
}
