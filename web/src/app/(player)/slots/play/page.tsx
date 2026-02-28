'use client';

import { useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { ReelGrid } from '@/components/slots/reel-grid';
import { SlotControls } from '@/components/slots/slot-controls';
import { WinDisplay } from '@/components/slots/win-display';
import { formatCents } from '@/lib/format';

interface SpinResult {
  symbols: string[];
  payout: number;
  win: boolean;
}

export default function SlotPlayPage() {
  const searchParams = useSearchParams();
  const gameId = searchParams.get('game') || '';
  const token = useAuthStore((s) => s.token)!;

  const [betAmount, setBetAmount] = useState('1.00');
  const [spinning, setSpinning] = useState(false);
  const [result, setResult] = useState<SpinResult | null>(null);
  const [showWin, setShowWin] = useState(false);
  const [error, setError] = useState('');

  async function handleSpin() {
    if (!gameId) return;
    setError('');
    setShowWin(false);
    setSpinning(true);
    try {
      const res = await api<SpinResult>('/slots/spin', {
        method: 'POST',
        token,
        body: { game_id: gameId, bet: Math.round(parseFloat(betAmount) * 100) },
      });
      setResult(res);
      if (res.win) setShowWin(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Spin failed');
    } finally {
      setSpinning(false);
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6 animate-fade-in">
      <Link href="/slots" className="text-sm text-brand-400 hover:underline">
        &larr; Back to lobby
      </Link>

      {/* Reel Display */}
      <ReelGrid
        symbols={result?.symbols || Array(15).fill('?')}
        spinning={spinning}
        win={result?.win || false}
      />

      {/* Win Display */}
      {showWin && result && result.payout > 0 && (
        <WinDisplay payout={result.payout} onDismiss={() => setShowWin(false)} />
      )}

      {/* No win message */}
      {result && !result.win && !spinning && (
        <p className="text-center text-sm text-text-muted">No win this time</p>
      )}

      {error && (
        <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta text-center">{error}</div>
      )}

      {/* Controls */}
      <SlotControls
        betAmount={betAmount}
        onBetChange={setBetAmount}
        onSpin={handleSpin}
        spinning={spinning}
      />
    </div>
  );
}
