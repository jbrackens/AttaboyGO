'use client';

interface BetButtonProps {
  name: string;
  odds: number;
  active: boolean;
  onClick: () => void;
}

export function BetButton({ name, odds, active, onClick }: BetButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`odds-btn ${active ? 'odds-btn--active' : ''}`}
    >
      <span className="text-xs text-text-secondary truncate w-full text-center">{name}</span>
      <span className="text-sm font-semibold mt-0.5">{odds.toFixed(2)}</span>
    </button>
  );
}
