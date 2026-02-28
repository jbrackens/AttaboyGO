'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore, useHasMounted } from '@/lib/auth-store';
import Link from 'next/link';

export default function HomePage() {
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const mounted = useHasMounted();

  useEffect(() => {
    if (mounted && token) router.replace('/dashboard');
  }, [mounted, token, router]);

  if (mounted && token) return null;

  return (
    <div className="min-h-screen flex flex-col">
      {/* Nav */}
      <nav className="flex items-center justify-between px-8 py-5">
        <span className="font-display text-2xl font-bold text-brand-400">ATTABOY</span>
        <div className="flex items-center gap-4">
          <Link href="/login" className="btn-outline text-sm">Sign in</Link>
          <Link href="/register" className="btn-primary text-sm">Get started</Link>
        </div>
      </nav>

      {/* Hero */}
      <section className="flex-1 flex flex-col items-center justify-center text-center px-6 pb-20">
        <div className="relative mb-8">
          <div className="absolute -inset-20 rounded-full bg-brand-400/5 blur-3xl" />
          <h1 className="relative font-display text-6xl sm:text-7xl lg:text-8xl font-bold tracking-tight">
            <span className="text-text-primary">PLAY.</span>{' '}
            <span className="text-brand-400 text-glow">WIN.</span>{' '}
            <span className="text-text-primary">CELEBRATE.</span>
          </h1>
        </div>
        <p className="max-w-xl text-lg text-text-secondary mb-10">
          Sports betting, slots, prediction markets, quests, and AI-powered insights — all on one platform built for the next generation.
        </p>
        <div className="flex items-center gap-4">
          <Link href="/register" className="btn-primary text-base px-8 py-3">Create account</Link>
          <Link href="/login" className="btn-secondary text-base px-8 py-3">Sign in</Link>
        </div>
      </section>

      {/* Features Grid */}
      <section className="px-8 pb-20">
        <div className="mx-auto max-w-5xl grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {[
            { title: 'Sportsbook', desc: 'Live odds, singles & parlays across every major sport.', color: 'text-brand-400' },
            { title: 'Slot Games', desc: 'Premium slots with RTP transparency and instant payouts.', color: 'text-electric-cyan' },
            { title: 'Predictions', desc: 'Binary outcome markets powered by real-world data.', color: 'text-electric-purple' },
            { title: 'Quests', desc: 'Complete challenges, earn rewards, climb the leaderboard.', color: 'text-electric-magenta' },
            { title: 'AI Insights', desc: 'Chat with AI for personalized stats and recommendations.', color: 'text-electric-blue' },
            { title: 'Social', desc: 'Share wins, follow friends, build your community.', color: 'text-brand-400' },
          ].map((f) => (
            <div key={f.title} className="card-glass group hover:border-brand-400/20 transition-colors">
              <h3 className={`font-display text-lg font-semibold ${f.color} mb-2`}>{f.title}</h3>
              <p className="text-sm text-text-secondary">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* CTA */}
      <section className="border-t border-surface-50 py-16 text-center">
        <h2 className="font-display text-3xl font-bold mb-4">Ready to play?</h2>
        <p className="text-text-secondary mb-8">Join thousands of players on Attaboy.</p>
        <Link href="/register" className="btn-primary text-base px-10 py-3">Get started free</Link>
      </section>

      {/* Footer */}
      <footer className="border-t border-surface-50 px-8 py-6 text-center text-xs text-text-muted">
        &copy; {new Date().getFullYear()} Attaboy. Play responsibly. 18+
      </footer>
    </div>
  );
}
