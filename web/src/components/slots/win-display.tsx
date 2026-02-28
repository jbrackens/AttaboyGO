'use client';

import { useEffect, useState } from 'react';
import { formatCents } from '@/lib/format';

interface WinDisplayProps {
  payout: number;
  onDismiss: () => void;
}

export function WinDisplay({ payout, onDismiss }: WinDisplayProps) {
  const [displayAmount, setDisplayAmount] = useState(0);

  useEffect(() => {
    const duration = 1000;
    const steps = 30;
    const increment = payout / steps;
    let current = 0;
    let step = 0;

    const timer = setInterval(() => {
      step++;
      current += increment;
      if (step >= steps) {
        setDisplayAmount(payout);
        clearInterval(timer);
      } else {
        setDisplayAmount(Math.round(current));
      }
    }, duration / steps);

    const dismiss = setTimeout(onDismiss, 4000);

    return () => { clearInterval(timer); clearTimeout(dismiss); };
  }, [payout, onDismiss]);

  return (
    <div className="animate-count-up text-center py-6">
      <div className="relative inline-block">
        <div className="absolute -inset-8 rounded-full bg-brand-400/10 blur-2xl" />
        <p className="relative font-display text-5xl font-bold text-brand-400 text-glow num">
          +${formatCents(displayAmount)}
        </p>
      </div>
      <p className="mt-2 text-sm text-text-secondary">You won!</p>
    </div>
  );
}
