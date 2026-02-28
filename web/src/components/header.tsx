'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents } from '@/lib/format';

const NAV_ITEMS = [
  { href: '/lobby', label: 'Lobby' },
  { href: '/quests', label: 'Quests' },
  { href: '/wallet', label: 'Wallet' },
  { href: '/slots', label: 'Slots' },
  { href: '/sportsbook', label: 'Sports' },
  { href: '/predictions', label: 'Predict' },
  { href: '/ai', label: 'AI' },
  { href: '/history', label: 'History' },
  { href: '/profile', label: 'Profile' },
];

interface Balance {
  balance: number;
  bonus_balance: number;
  currency: string;
}

export function Header() {
  const pathname = usePathname();
  const token = useAuthStore((s) => s.token);
  const logout = useAuthStore((s) => s.logout);
  const [menuOpen, setMenuOpen] = useState(false);
  const [bal, setBal] = useState<Balance | null>(null);

  useEffect(() => {
    if (!token) return;
    api<Balance>('/wallet/balance', { token }).then(setBal).catch(() => {});
  }, [token]);

  return (
    <header className="sticky top-0 z-50 border-b border-white/[0.06]" style={{ background: 'rgba(18,18,18,0.7)', backdropFilter: 'blur(20px)', WebkitBackdropFilter: 'blur(20px)' }}>
      <div className="flex items-center justify-between px-6 py-3">
        {/* Logo */}
        <Link href="/dashboard" className="font-display text-xl font-bold text-brand-400 shrink-0">
          ATTABOY
        </Link>

        {/* Desktop Nav */}
        <nav className="hidden lg:flex items-center gap-1 mx-6">
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${
                pathname.startsWith(item.href)
                  ? 'bg-brand-400/15 text-brand-400'
                  : 'text-text-secondary hover:text-text-primary hover:bg-surface-50/50'
              }`}
            >
              {item.label}
            </Link>
          ))}
        </nav>

        {/* Right side */}
        <div className="flex items-center gap-3 shrink-0">
          {/* Balance Pill */}
          {bal && (
            <div className="hidden sm:flex items-center gap-2 rounded-full bg-surface-200 border border-surface-50 px-4 py-1.5 text-xs">
              <span className="text-brand-400 font-semibold num">${formatCents(bal.balance)}</span>
              <span className="text-text-muted">|</span>
              <span className="text-electric-cyan num">${formatCents(bal.bonus_balance)}</span>
            </div>
          )}

          {/* Logout */}
          <button
            onClick={() => { logout(); window.location.href = '/login'; }}
            className="hidden lg:block text-xs text-text-muted hover:text-text-secondary transition-colors"
          >
            Log out
          </button>

          {/* Mobile hamburger */}
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="lg:hidden flex flex-col gap-1 p-2"
            aria-label="Menu"
          >
            <span className={`block h-0.5 w-5 bg-text-secondary transition-transform ${menuOpen ? 'rotate-45 translate-y-1.5' : ''}`} />
            <span className={`block h-0.5 w-5 bg-text-secondary transition-opacity ${menuOpen ? 'opacity-0' : ''}`} />
            <span className={`block h-0.5 w-5 bg-text-secondary transition-transform ${menuOpen ? '-rotate-45 -translate-y-1.5' : ''}`} />
          </button>
        </div>
      </div>

      {/* Mobile menu */}
      {menuOpen && (
        <nav className="lg:hidden border-t border-surface-50 px-4 py-3 space-y-1">
          {bal && (
            <div className="flex items-center gap-3 px-3 py-2 text-sm mb-2">
              <span className="text-brand-400 font-semibold num">${formatCents(bal.balance)}</span>
              <span className="text-text-muted">|</span>
              <span className="text-electric-cyan num">${formatCents(bal.bonus_balance)}</span>
            </div>
          )}
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              onClick={() => setMenuOpen(false)}
              className={`block rounded-lg px-3 py-2 text-sm font-medium ${
                pathname.startsWith(item.href)
                  ? 'bg-brand-400/15 text-brand-400'
                  : 'text-text-secondary hover:text-text-primary'
              }`}
            >
              {item.label}
            </Link>
          ))}
          <button
            onClick={() => { logout(); window.location.href = '/login'; }}
            className="block w-full text-left rounded-lg px-3 py-2 text-sm text-text-muted hover:text-text-secondary"
          >
            Log out
          </button>
        </nav>
      )}
    </header>
  );
}
