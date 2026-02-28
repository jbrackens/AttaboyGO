import Link from 'next/link';

export function Footer() {
  return (
    <footer className="border-t border-surface-50 bg-surface-300">
      <div className="mx-auto max-w-6xl px-6 py-10 grid grid-cols-1 sm:grid-cols-3 gap-8">
        {/* Brand */}
        <div>
          <span className="font-display text-lg font-bold text-brand-400">ATTABOY</span>
          <p className="mt-2 text-sm text-text-muted">
            Play. Win. Celebrate.<br />
            The next-generation gaming platform.
          </p>
        </div>

        {/* Quick Links */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-3">Quick Links</h4>
          <ul className="space-y-2 text-sm">
            {[
              { href: '/sportsbook', label: 'Sportsbook' },
              { href: '/slots', label: 'Slots' },
              { href: '/predictions', label: 'Predictions' },
              { href: '/quests', label: 'Quests' },
              { href: '/promotions', label: 'Promotions' },
            ].map((l) => (
              <li key={l.href}>
                <Link href={l.href} className="text-text-secondary hover:text-brand-400 transition-colors">
                  {l.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>

        {/* Responsible Gaming */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-3">Responsible Gaming</h4>
          <p className="text-sm text-text-secondary mb-3">
            Gambling should be fun. Set your limits and play responsibly.
          </p>
          <Link href="/settings" className="text-sm text-brand-400 hover:underline">
            Set deposit limits &rarr;
          </Link>
        </div>
      </div>

      {/* Copyright bar */}
      <div className="border-t border-surface-50 px-6 py-4 text-center text-xs text-text-muted">
        &copy; {new Date().getFullYear()} Attaboy. All rights reserved. 18+ | Play responsibly.
      </div>
    </footer>
  );
}
