'use client';

interface Sport {
  id: string;
  name: string;
}

interface SportTabsProps {
  sports: Sport[];
  activeSport: string | null;
  onSelect: (sportId: string) => void;
}

export function SportsTabs({ sports, activeSport, onSelect }: SportTabsProps) {
  return (
    <div className="flex gap-2 overflow-x-auto pb-1 scrollbar-none">
      {sports.map((sport) => (
        <button
          key={sport.id}
          onClick={() => onSelect(sport.id)}
          className={`whitespace-nowrap rounded-full px-4 py-1.5 text-sm font-medium transition-all ${
            activeSport === sport.id
              ? 'bg-brand-400 text-surface-900 shadow-glow-brand'
              : 'bg-surface-200 text-text-secondary border border-surface-50 hover:border-brand-400/30'
          }`}
        >
          {sport.name}
        </button>
      ))}
    </div>
  );
}
