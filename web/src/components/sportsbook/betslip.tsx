'use client';

import { useState } from 'react';
import { useBetslipStore, getCombinedOdds, getParlayReturn, type BetSelection } from '@/stores/betslip-store';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

export function Betslip() {
  const token = useAuthStore((s) => s.token)!;
  const { selections, mode, parlayStake, removeSelection, setStake, setParlayStake, setMode, clear } = useBetslipStore();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  if (selections.length === 0) return null;

  async function placeSingleBet(sel: BetSelection) {
    await api('/sportsbook/bets', {
      method: 'POST',
      token,
      body: {
        event_id: sel.eventId,
        market_id: sel.marketId,
        selection_id: sel.selectionId,
        stake: Math.round(sel.stake * 100),
      },
    });
  }

  async function handlePlaceBets() {
    setError('');
    setSuccess('');
    setLoading(true);
    try {
      if (mode === 'single') {
        const valid = selections.filter((s) => s.stake > 0);
        if (valid.length === 0) { setError('Enter a stake'); setLoading(false); return; }
        await Promise.all(valid.map(placeSingleBet));
        setSuccess(`${valid.length} bet(s) placed`);
      } else {
        if (parlayStake <= 0) { setError('Enter parlay stake'); setLoading(false); return; }
        await api('/sportsbook/bets', {
          method: 'POST',
          token,
          body: {
            type: 'parlay',
            selections: selections.map((s) => s.selectionId),
            stake: Math.round(parlayStake * 100),
          },
        });
        setSuccess('Parlay placed');
      }
      clear();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to place bet');
    } finally {
      setLoading(false);
    }
  }

  const combinedOdds = getCombinedOdds(selections);

  return (
    <div className="card-glass space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-display text-lg font-semibold">Bet Slip</h3>
        <span className="rounded-full bg-brand-400/15 px-2.5 py-0.5 text-xs font-semibold text-brand-400">
          {selections.length}
        </span>
      </div>

      {/* Mode tabs */}
      <div className="flex gap-2">
        <button onClick={() => setMode('single')} className={`rounded-lg px-3 py-1 text-xs font-medium transition-colors ${mode === 'single' ? 'bg-brand-400 text-surface-900' : 'bg-surface-200 text-text-secondary'}`}>
          Singles
        </button>
        <button onClick={() => setMode('parlay')} className={`rounded-lg px-3 py-1 text-xs font-medium transition-colors ${mode === 'parlay' ? 'bg-brand-400 text-surface-900' : 'bg-surface-200 text-text-secondary'}`}>
          Parlay
        </button>
      </div>

      {/* Selections */}
      <div className="space-y-2 max-h-64 overflow-y-auto">
        {selections.map((sel) => (
          <div key={sel.selectionId} className="rounded-lg bg-surface-200 p-3">
            <div className="flex items-start justify-between">
              <div className="min-w-0 flex-1">
                <p className="text-xs text-text-muted truncate">{sel.eventName}</p>
                <p className="text-sm font-medium truncate">{sel.selectionName}</p>
                <p className="text-xs text-brand-400 num">{sel.odds.toFixed(2)}</p>
              </div>
              <button onClick={() => removeSelection(sel.selectionId)} className="text-text-muted hover:text-electric-magenta text-xs ml-2">✕</button>
            </div>
            {mode === 'single' && (
              <input
                type="number"
                step="0.01"
                min="0"
                placeholder="Stake ($)"
                value={sel.stake || ''}
                onChange={(e) => setStake(sel.selectionId, parseFloat(e.target.value) || 0)}
                className="input-field mt-2 text-xs"
              />
            )}
          </div>
        ))}
      </div>

      {/* Parlay stake */}
      {mode === 'parlay' && (
        <div className="space-y-2">
          <div className="flex items-center justify-between text-xs text-text-muted">
            <span>Combined odds</span>
            <span className="num text-brand-400">{combinedOdds.toFixed(2)}</span>
          </div>
          <input
            type="number"
            step="0.01"
            min="0"
            placeholder="Parlay stake ($)"
            value={parlayStake || ''}
            onChange={(e) => setParlayStake(parseFloat(e.target.value) || 0)}
            className="input-field text-xs"
          />
          {parlayStake > 0 && (
            <div className="flex items-center justify-between text-xs">
              <span className="text-text-muted">Potential return</span>
              <span className="text-brand-400 font-semibold num">${getParlayReturn(selections, parlayStake).toFixed(2)}</span>
            </div>
          )}
        </div>
      )}

      {error && <p className="text-xs text-electric-magenta">{error}</p>}
      {success && <p className="text-xs text-brand-400">{success}</p>}

      <div className="flex gap-2">
        <button onClick={handlePlaceBets} disabled={loading} className="btn-primary flex-1 text-sm">
          {loading ? 'Placing...' : 'Place Bet'}
        </button>
        <button onClick={clear} className="btn-secondary text-sm">Clear</button>
      </div>
    </div>
  );
}
