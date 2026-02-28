import Link from 'next/link';

export default function PromotionsPage() {
  return (
    <div className="min-h-screen bg-surface">
      {/* Nav */}
      <nav className="flex items-center justify-between px-8 py-5 border-b border-surface-50">
        <Link href="/" className="font-display text-2xl font-bold text-brand-400">ATTABOY</Link>
        <div className="flex items-center gap-4">
          <Link href="/login" className="btn-outline text-sm">Sign in</Link>
          <Link href="/register" className="btn-primary text-sm">Get started</Link>
        </div>
      </nav>

      <div className="mx-auto max-w-4xl px-6 py-12 space-y-8">
        <h1 className="font-display text-3xl font-bold text-center">Promotions</h1>

        {/* Welcome Bonus */}
        <div className="card-glass relative overflow-hidden">
          <div className="absolute top-0 right-0 w-32 h-32 bg-brand-400/5 rounded-full blur-3xl" />
          <div className="relative">
            <span className="rounded-full bg-brand-400/15 px-3 py-1 text-xs font-semibold text-brand-400">NEW PLAYERS</span>
            <h2 className="font-display text-2xl font-bold mt-3">Welcome Bonus</h2>
            <p className="font-display text-4xl font-bold text-brand-400 text-glow mt-2">100% up to $500</p>
            <p className="text-sm text-text-secondary mt-3">
              Double your first deposit up to $500. Start your Attaboy journey with extra funds to explore sports betting, slots, and prediction markets.
            </p>
            <ul className="mt-4 space-y-2 text-sm text-text-muted">
              <li>Min deposit: $10</li>
              <li>Wagering: 30x bonus amount</li>
              <li>Valid for 30 days</li>
            </ul>
            <Link href="/register" className="btn-primary mt-6 inline-block">Claim Now</Link>
          </div>
        </div>

        {/* Weekly Reload */}
        <div className="card-glass relative overflow-hidden">
          <div className="absolute top-0 right-0 w-32 h-32 bg-electric-cyan/5 rounded-full blur-3xl" />
          <div className="relative">
            <span className="rounded-full bg-electric-cyan/15 px-3 py-1 text-xs font-semibold text-electric-cyan">WEEKLY</span>
            <h2 className="font-display text-2xl font-bold mt-3">Weekly Reload</h2>
            <p className="font-display text-4xl font-bold text-electric-cyan mt-2">50% up to $100</p>
            <p className="text-sm text-text-secondary mt-3">
              Every week, boost your balance with a 50% reload bonus on your first deposit of the week.
            </p>
            <ul className="mt-4 space-y-2 text-sm text-text-muted">
              <li>Min deposit: $20</li>
              <li>Wagering: 20x bonus amount</li>
              <li>Resets every Monday</li>
            </ul>
            <Link href="/register" className="btn-primary mt-6 inline-block">Get Started</Link>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-surface-50 px-8 py-6 text-center text-xs text-text-muted">
        &copy; {new Date().getFullYear()} Attaboy. Play responsibly. 18+
      </footer>
    </div>
  );
}
