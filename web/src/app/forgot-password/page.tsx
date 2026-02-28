'use client';

import { useState } from 'react';
import Link from 'next/link';
import { api, ApiError } from '@/lib/api';

export default function ForgotPasswordPage() {
  const [step, setStep] = useState<'request' | 'confirm'>('request');
  const [email, setEmail] = useState('');
  const [token, setToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleRequest(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api('/auth/password-reset/request', { method: 'POST', body: { email } });
      setSuccess('Reset token sent. Check your email.');
      setStep('confirm');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Request failed');
    } finally {
      setLoading(false);
    }
  }

  async function handleConfirm(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    if (newPassword.length < 8) { setError('Password must be at least 8 characters'); return; }
    setLoading(true);
    try {
      await api('/auth/password-reset/confirm', { method: 'POST', body: { email, token, new_password: newPassword } });
      setSuccess('Password reset successfully.');
      setStep('request');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Reset failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="card-glass w-full max-w-md space-y-6">
        <div className="text-center">
          <span className="font-display text-2xl font-bold text-brand-400">ATTABOY</span>
          <h1 className="mt-4 font-display text-3xl font-bold">RESET PASSWORD</h1>
        </div>

        {error && (
          <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">
            {error}
          </div>
        )}
        {success && (
          <div className="rounded-lg bg-brand-400/10 border border-brand-400/30 px-4 py-3 text-sm text-brand-400">
            {success}
          </div>
        )}

        {step === 'request' ? (
          <form onSubmit={handleRequest} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">Email</label>
              <input type="email" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="player@example.com" className="input-field" />
            </div>
            <button type="submit" disabled={loading} className="btn-primary w-full">
              {loading ? 'Sending...' : 'Send reset token'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleConfirm} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">Reset token</label>
              <input type="text" required value={token} onChange={(e) => setToken(e.target.value)} className="input-field" />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1">New password</label>
              <input type="password" required minLength={8} value={newPassword} onChange={(e) => setNewPassword(e.target.value)} placeholder="••••••••" className="input-field" />
            </div>
            <button type="submit" disabled={loading} className="btn-primary w-full">
              {loading ? 'Resetting...' : 'Reset password'}
            </button>
          </form>
        )}

        <div className="divider-glow" />

        <p className="text-center text-sm text-text-muted">
          <Link href="/login" className="text-brand-400 hover:underline">Back to sign in</Link>
        </p>
      </div>
    </div>
  );
}
