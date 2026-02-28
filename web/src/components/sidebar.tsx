'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useAuthStore } from '@/lib/auth-store';
import { cn } from '@/lib/utils';

const NAV_ITEMS = [
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/wallet', label: 'Wallet' },
  { href: '/sportsbook', label: 'Sportsbook' },
  { href: '/slots', label: 'Slots' },
  { href: '/predictions', label: 'Predictions' },
  { href: '/quests', label: 'Quests' },
  { href: '/social', label: 'Social' },
  { href: '/ai', label: 'AI Chat' },
  { href: '/settings', label: 'Settings' },
];

export function Sidebar() {
  const pathname = usePathname();
  const logout = useAuthStore((s) => s.logout);

  return (
    <aside className="flex h-screen w-56 flex-col bg-white border-r border-gray-200">
      <div className="px-5 py-6">
        <h1 className="text-xl font-bold text-indigo-600">Attaboy</h1>
      </div>
      <nav className="flex-1 space-y-1 px-3">
        {NAV_ITEMS.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={cn(
              'block rounded-md px-3 py-2 text-sm font-medium',
              pathname.startsWith(item.href)
                ? 'bg-indigo-50 text-indigo-700'
                : 'text-gray-700 hover:bg-gray-50',
            )}
          >
            {item.label}
          </Link>
        ))}
      </nav>
      <div className="border-t border-gray-200 p-3">
        <button
          onClick={() => {
            logout();
            window.location.href = '/login';
          }}
          className="w-full rounded-md px-3 py-2 text-left text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          Log out
        </button>
      </div>
    </aside>
  );
}
