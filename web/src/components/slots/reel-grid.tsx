'use client';

const SYMBOL_COLORS: Record<string, string> = {
  '7': 'text-electric-magenta',
  'BAR': 'text-brand-400',
  'CHERRY': 'text-electric-magenta',
  'LEMON': 'text-brand-300',
  'ORANGE': 'text-orange-400',
  'PLUM': 'text-electric-purple',
  'BELL': 'text-brand-400',
  'DIAMOND': 'text-electric-cyan',
  'STAR': 'text-brand-400',
  'WILD': 'text-brand-400',
  'SCATTER': 'text-electric-purple',
  'BONUS': 'text-electric-cyan',
};

interface ReelGridProps {
  symbols: string[];
  spinning: boolean;
  win: boolean;
  cols?: number;
  rows?: number;
}

export function ReelGrid({ symbols, spinning, win, cols = 5, rows = 3 }: ReelGridProps) {
  const grid: string[][] = [];
  for (let r = 0; r < rows; r++) {
    const row: string[] = [];
    for (let c = 0; c < cols; c++) {
      const idx = r * cols + c;
      row.push(symbols[idx] || '?');
    }
    grid.push(row);
  }

  return (
    <div className={`panel-inset ${win ? 'animate-pulse-glow' : ''}`}>
      <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
        {grid.flat().map((sym, i) => (
          <div
            key={i}
            className={`flex items-center justify-center rounded-lg bg-surface-200 border border-surface-50 aspect-square text-2xl font-bold ${
              spinning ? 'animate-spin-reel' : ''
            } ${SYMBOL_COLORS[sym.toUpperCase()] || 'text-text-primary'}`}
          >
            {sym}
          </div>
        ))}
      </div>
    </div>
  );
}
