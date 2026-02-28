'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { useBetslipStore } from '@/stores/betslip-store';
import { MarketGroup } from '@/components/sportsbook/market-group';
import { Betslip } from '@/components/sportsbook/betslip';

interface Market { id: string; name: string; status: string; }
interface Selection { id: string; name: string; odds: number; }

export default function EventDetailPage() {
  const { eventId } = useParams<{ eventId: string }>();
  const router = useRouter();
  const token = useAuthStore((s) => s.token)!;
  const { toggleSelection, hasSelection } = useBetslipStore();

  const [markets, setMarkets] = useState<Market[]>([]);
  const [selections, setSelections] = useState<Record<string, Selection[]>>({});
  const [loading, setLoading] = useState(true);
  const [activeSelId, setActiveSelId] = useState<string | null>(null);

  useEffect(() => {
    api<Market[]>(`/events/${eventId}/markets`, { token })
      .then(async (mkts) => {
        setMarkets(mkts);
        const selMap: Record<string, Selection[]> = {};
        await Promise.all(
          mkts.map(async (m) => {
            const sels = await api<Selection[]>(`/markets/${m.id}/selections`, { token }).catch(() => []);
            selMap[m.id] = sels || [];
          }),
        );
        setSelections(selMap);
      })
      .finally(() => setLoading(false));
  }, [eventId, token]);

  function handleSelectSelection(market: Market, sel: Selection) {
    toggleSelection({
      selectionId: sel.id,
      eventId: eventId,
      eventName: `Event ${eventId}`,
      marketName: market.name,
      selectionName: sel.name,
      odds: sel.odds,
    });
    setActiveSelId(hasSelection(sel.id) ? null : sel.id);
  }

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-6xl animate-fade-in">
      <button onClick={() => router.back()} className="text-sm text-brand-400 hover:underline mb-6 inline-block">
        &larr; Back to sportsbook
      </button>

      <div className="flex gap-6">
        {/* Markets */}
        <div className="flex-1 space-y-4">
          {markets.length === 0 ? (
            <div className="card-glass text-center py-12">
              <p className="text-text-muted">No markets available for this event</p>
            </div>
          ) : (
            markets.map((market) => (
              <MarketGroup
                key={market.id}
                name={market.name}
                status={market.status}
                selections={selections[market.id] || []}
                activeSelectionId={activeSelId}
                onSelectSelection={(sel) => handleSelectSelection(market, sel)}
              />
            ))
          )}
        </div>

        {/* Betslip sidebar */}
        <div className="hidden lg:block w-80 shrink-0">
          <div className="sticky top-20">
            <Betslip />
          </div>
        </div>
      </div>

      {/* Mobile betslip */}
      <div className="lg:hidden fixed bottom-0 left-0 right-0 p-4 z-40">
        <Betslip />
      </div>
    </div>
  );
}
