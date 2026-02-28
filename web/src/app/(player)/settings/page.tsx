'use client';

import { useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

export default function SettingsPage() {
  const token = useAuthStore((s) => s.token)!;
  const [depositLimit, setDepositLimit] = useState('');
  const [lossLimit, setLossLimit] = useState('');
  const [exclusionDays, setExclusionDays] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSetLimit(type: string, value: string) {
    setError('');
    setMessage('');
    setLoading(true);
    try {
      await api('/responsible-gaming/limits', {
        method: 'POST',
        token,
        body: { type, amount: Math.round(parseFloat(value) * 100) },
      });
      setMessage(`${type} limit set successfully`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to set limit');
    } finally {
      setLoading(false);
    }
  }

  async function handleSelfExclude() {
    if (!exclusionDays || parseInt(exclusionDays) < 1) { setError('Enter valid number of days'); return; }
    setError('');
    setMessage('');
    setLoading(true);
    try {
      await api('/responsible-gaming/self-exclude', {
        method: 'POST',
        token,
        body: { days: parseInt(exclusionDays) },
      });
      setMessage('Self-exclusion activated');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to self-exclude');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Responsible Gaming</h1>

      {error && <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">{error}</div>}
      {message && <div className="rounded-lg bg-brand-400/10 border border-brand-400/30 px-4 py-3 text-sm text-brand-400">{message}</div>}

      {/* Deposit Limit */}
      <div className="card space-y-4">
        <h3 className="font-display text-lg font-semibold">Deposit Limit</h3>
        <p className="text-sm text-text-muted">Set a daily deposit limit to manage your spending.</p>
        <div className="flex gap-2">
          <input type="number" step="1" min="1" placeholder="Daily limit ($)" value={depositLimit} onChange={(e) => setDepositLimit(e.target.value)} className="input-field flex-1" />
          <button onClick={() => handleSetLimit('deposit', depositLimit)} disabled={loading || !depositLimit} className="btn-primary">Set limit</button>
        </div>
      </div>

      {/* Loss Limit */}
      <div className="card space-y-4">
        <h3 className="font-display text-lg font-semibold">Loss Limit</h3>
        <p className="text-sm text-text-muted">Set a daily loss limit to protect your balance.</p>
        <div className="flex gap-2">
          <input type="number" step="1" min="1" placeholder="Daily limit ($)" value={lossLimit} onChange={(e) => setLossLimit(e.target.value)} className="input-field flex-1" />
          <button onClick={() => handleSetLimit('loss', lossLimit)} disabled={loading || !lossLimit} className="btn-primary">Set limit</button>
        </div>
      </div>

      {/* Self-Exclusion */}
      <div className="card space-y-4 border-electric-magenta/30">
        <h3 className="font-display text-lg font-semibold text-electric-magenta">Self-Exclusion</h3>
        <p className="text-sm text-text-muted">Temporarily exclude yourself from the platform. This action cannot be undone until the period expires.</p>
        <div className="flex gap-2">
          <input type="number" min="1" placeholder="Number of days" value={exclusionDays} onChange={(e) => setExclusionDays(e.target.value)} className="input-field flex-1" />
          <button onClick={handleSelfExclude} disabled={loading || !exclusionDays} className="inline-flex items-center justify-center rounded-lg bg-electric-magenta px-5 py-2.5 text-sm font-semibold text-white transition-all hover:bg-electric-magenta/80 disabled:opacity-40">
            Activate
          </button>
        </div>
      </div>
    </div>
  );
}
