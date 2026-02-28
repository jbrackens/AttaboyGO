'use client';

export default function BonusesPage() {
  return (
    <div className="mx-auto max-w-4xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Bonuses</h1>
      <div className="card-glass text-center py-16">
        <div className="text-4xl mb-4">🎁</div>
        <h2 className="font-display text-xl font-semibold mb-2">No active bonuses</h2>
        <p className="text-sm text-text-muted">Check back later for new promotions and bonus offers.</p>
      </div>
    </div>
  );
}
