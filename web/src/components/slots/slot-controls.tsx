'use client';

interface SlotControlsProps {
  betAmount: string;
  onBetChange: (val: string) => void;
  onSpin: () => void;
  spinning: boolean;
  freeSpins?: number;
}

const BET_STEPS = [0.10, 0.25, 0.50, 1.00, 2.00, 5.00];

export function SlotControls({ betAmount, onBetChange, onSpin, spinning, freeSpins }: SlotControlsProps) {
  const currentBet = parseFloat(betAmount) || 0;
  const currentIdx = BET_STEPS.findIndex((s) => s >= currentBet);

  function stepDown() {
    const idx = currentIdx > 0 ? currentIdx - 1 : 0;
    onBetChange(BET_STEPS[idx].toFixed(2));
  }

  function stepUp() {
    const idx = currentIdx < BET_STEPS.length - 1 ? currentIdx + 1 : BET_STEPS.length - 1;
    onBetChange(BET_STEPS[idx].toFixed(2));
  }

  return (
    <div className="space-y-4">
      {freeSpins && freeSpins > 0 && (
        <div className="rounded-lg bg-brand-400/10 border border-brand-400/30 px-4 py-2 text-center text-sm text-brand-400 font-semibold">
          {freeSpins} Free Spins!
        </div>
      )}

      <div className="flex items-center gap-4">
        {/* Bet control */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-text-muted">BET</span>
          <button onClick={stepDown} className="btn-secondary text-xs px-2 py-1">−</button>
          <span className="text-sm font-semibold num w-16 text-center">${betAmount}</span>
          <button onClick={stepUp} className="btn-secondary text-xs px-2 py-1">+</button>
        </div>

        {/* Spin button */}
        <button
          onClick={onSpin}
          disabled={spinning}
          className="btn-primary flex-1 text-lg py-3 font-bold"
        >
          {spinning ? 'SPINNING...' : freeSpins ? 'FREE SPIN' : 'SPIN'}
        </button>
      </div>
    </div>
  );
}
