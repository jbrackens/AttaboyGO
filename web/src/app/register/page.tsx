'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

export default function RegisterPage() {
  const router = useRouter();
  const setAuth = useAuthStore((s) => s.setAuth);
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [currency, setCurrency] = useState('USD');
  const [dob, setDob] = useState('');
  const [country, setCountry] = useState('');
  const [ageConfirm, setAgeConfirm] = useState(false);
  const [termsAccept, setTermsAccept] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    if (password.length < 8) { setError('Password must be at least 8 characters'); return; }
    if (password !== confirmPassword) { setError('Passwords do not match'); return; }
    if (!ageConfirm) { setError('You must confirm you are 18+'); return; }
    if (!termsAccept) { setError('You must accept the terms'); return; }
    setLoading(true);
    try {
      const res = await api<{ token: string; player_id: string }>('/auth/register', {
        method: 'POST',
        body: { email, password, currency, first_name: firstName, last_name: lastName, date_of_birth: dob, country },
      });
      setAuth(res.token, res.player_id, email);
      router.push('/dashboard');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Registration failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="card-glass w-full max-w-lg space-y-6">
        <div className="text-center">
          <span className="font-display text-2xl font-bold text-brand-400">ATTABOY</span>
          <h1 className="mt-4 font-display text-3xl font-bold">CREATE ACCOUNT</h1>
          <p className="mt-1 text-sm text-text-muted">Join the next-gen gaming platform</p>
        </div>

        {error && (
          <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Name row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">First name</label>
              <input type="text" value={firstName} onChange={(e) => setFirstName(e.target.value)} placeholder="John" className="input-field" />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">Last name</label>
              <input type="text" value={lastName} onChange={(e) => setLastName(e.target.value)} placeholder="Doe" className="input-field" />
            </div>
          </div>

          {/* Email */}
          <div>
            <label className="block text-sm font-medium text-text-secondary mb-1">Email</label>
            <input type="email" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="player@example.com" className="input-field" />
          </div>

          {/* Password row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">Password</label>
              <input type="password" required minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} placeholder="••••••••" className="input-field" />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">Confirm</label>
              <input type="password" required value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} placeholder="••••••••" className="input-field" />
            </div>
          </div>

          {/* Currency & DOB row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">Currency</label>
              <select value={currency} onChange={(e) => setCurrency(e.target.value)} className="input-field">
                <option value="USD">USD</option>
                <option value="EUR">EUR</option>
                <option value="GBP">GBP</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">Date of birth</label>
              <input type="date" value={dob} onChange={(e) => setDob(e.target.value)} className="input-field" />
            </div>
          </div>

          {/* Country */}
          <div>
            <label className="block text-sm font-medium text-text-secondary mb-1">Country</label>
            <input type="text" value={country} onChange={(e) => setCountry(e.target.value)} placeholder="United States" className="input-field" />
          </div>

          {/* Checkboxes */}
          <div className="space-y-3">
            <label className="flex items-start gap-3 text-sm text-text-secondary cursor-pointer">
              <input type="checkbox" checked={ageConfirm} onChange={(e) => setAgeConfirm(e.target.checked)} className="mt-0.5 accent-brand-400" />
              I confirm I am at least 18 years old
            </label>
            <label className="flex items-start gap-3 text-sm text-text-secondary cursor-pointer">
              <input type="checkbox" checked={termsAccept} onChange={(e) => setTermsAccept(e.target.checked)} className="mt-0.5 accent-brand-400" />
              I accept the terms of service and privacy policy
            </label>
          </div>

          <button type="submit" disabled={loading} className="btn-primary w-full">
            {loading ? 'Creating account...' : 'Create account'}
          </button>
        </form>

        <div className="divider-glow" />

        <p className="text-center text-sm text-text-muted">
          Already have an account?{' '}
          <Link href="/login" className="text-brand-400 hover:underline">Sign in</Link>
        </p>
      </div>
    </div>
  );
}
