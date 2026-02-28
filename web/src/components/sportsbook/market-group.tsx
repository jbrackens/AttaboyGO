'use client';

import { useState } from 'react';
import { BetButton } from './bet-button';

interface Selection {
  id: string;
  name: string;
  odds: number;
}

interface MarketGroupProps {
  name: string;
  status: string;
  selections: Selection[];
  activeSelectionId: string | null;
  onSelectSelection: (sel: Selection) => void;
}

export function MarketGroup({ name, status, selections, activeSelectionId, onSelectSelection }: MarketGroupProps) {
  const [expanded, setExpanded] = useState(true);

  return (
    <div className="card">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center justify-between w-full text-left"
      >
        <h3 className="font-display text-sm font-semibold">{name}</h3>
        <span className="text-xs text-text-muted">{expanded ? '▲' : '▼'}</span>
      </button>
      {expanded && (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 mt-3">
          {selections.map((sel) => (
            <BetButton
              key={sel.id}
              name={sel.name}
              odds={sel.odds}
              active={activeSelectionId === sel.id}
              onClick={() => onSelectSelection(sel)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
