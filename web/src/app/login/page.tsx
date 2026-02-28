'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

export default function LoginPage() {
  const router = useRouter();
  const setAuth = useAuthStore((s) => s.setAuth);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await api<{ token: string; player_id: string }>('/auth/login', {
        method: 'POST',
        body: { email, password },
      });
      setAuth(res.token, res.player_id, email);
      router.push('/dashboard');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="card-glass w-full max-w-md space-y-6">
        <div className="text-center">
          <span className="font-display text-2xl font-bold text-brand-400">ATTABOY</span>
          <h1 className="mt-4 font-display text-3xl font-bold">WELCOME BACK</h1>
          <p className="mt-1 text-sm text-text-muted">Sign in to continue playing</p>
        </div>

        {error && (
          <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-text-secondary mb-1">Email</label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="player@example.com"
              className="input-field"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-secondary mb-1">Password</label>
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="input-field"
            />
          </div>
          <button type="submit" disabled={loading} className="btn-primary w-full">
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>

        <div className="divider-glow" />

        <div className="flex justify-between text-sm">
          <Link href="/register" className="text-brand-400 hover:underline">Create account</Link>
          <Link href="/forgot-password" className="text-text-muted hover:text-text-secondary">Forgot password?</Link>
        </div>
      </div>
    </div>
  );
}
