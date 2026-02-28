'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, txTypeLabel, formatDate } from '@/lib/format';

interface Balance { balance: number; bonus_balance: number; reserved_balance: number; currency: string; }
interface Transaction { id: string; type: string; amount: number; created_at: string; }

const QUICK_AMOUNTS = [10, 25, 50, 100, 250];

export default function WalletPage() {
  const token = useAuthStore((s) => s.token)!;
  const [balance, setBalance] = useState<Balance | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);

  const [depositAmount, setDepositAmount] = useState('');
  const [withdrawAmount, setWithdrawAmount] = useState('');
  const [actionError, setActionError] = useState('');
  const [actionLoading, setActionLoading] = useState(false);

  async function loadBalance() { setBalance(await api<Balance>('/wallet/balance', { token })); }
  async function loadTransactions() { setTransactions(await api<Transaction[]>('/wallet/transactions?limit=10', { token })); }

  useEffect(() => {
    Promise.all([loadBalance(), loadTransactions()]).finally(() => setLoading(false));
  }, [token]);

  async function handleDeposit(e: React.FormEvent) {
    e.preventDefault();
    setActionError('');
    setActionLoading(true);
    try {
      await api('/payments/deposit', { method: 'POST', token, body: { amount: Math.round(parseFloat(depositAmount) * 100) } });
      setDepositAmount('');
      await Promise.all([loadBalance(), loadTransactions()]);
    } catch (err) { setActionError(err instanceof ApiError ? err.message : 'Deposit failed'); }
    finally { setActionLoading(false); }
  }

  async function handleWithdraw(e: React.FormEvent) {
    e.preventDefault();
    setActionError('');
    setActionLoading(true);
    try {
      await api('/payments/withdraw', { method: 'POST', token, body: { amount: Math.round(parseFloat(withdrawAmount) * 100) } });
      setWithdrawAmount('');
      await Promise.all([loadBalance(), loadTransactions()]);
    } catch (err) { setActionError(err instanceof ApiError ? err.message : 'Withdrawal failed'); }
    finally { setActionLoading(false); }
  }

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-4xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Wallet</h1>

      {/* Balance Cards */}
      {balance && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="panel-inset">
            <p className="text-xs text-text-muted uppercase tracking-wider">Real Balance</p>
            <p className="mt-2 font-display text-3xl font-bold text-brand-400 text-glow num">${formatCents(balance.balance)}</p>
          </div>
          <div className="panel-inset">
            <p className="text-xs text-text-muted uppercase tracking-wider">Bonus Balance</p>
            <p className="mt-2 font-display text-3xl font-bold text-electric-cyan num">${formatCents(balance.bonus_balance)}</p>
          </div>
          <div className="panel-inset">
            <p className="text-xs text-text-muted uppercase tracking-wider">Reserved</p>
            <p className="mt-2 font-display text-3xl font-bold text-text-muted num">${formatCents(balance.reserved_balance)}</p>
          </div>
        </div>
      )}

      {actionError && (
        <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">{actionError}</div>
      )}

      {/* Deposit / Withdraw */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <form onSubmit={handleDeposit} className="card space-y-4">
          <h3 className="font-display text-lg font-semibold">Deposit</h3>
          <div className="flex flex-wrap gap-2">
            {QUICK_AMOUNTS.map((a) => (
              <button key={a} type="button" onClick={() => setDepositAmount(String(a))} className="btn-secondary text-xs px-3 py-1.5">${a}</button>
            ))}
          </div>
          <div className="flex gap-2">
            <input type="number" step="0.01" min="0.01" required placeholder="Amount ($)" value={depositAmount} onChange={(e) => setDepositAmount(e.target.value)} className="input-field flex-1" />
            <button type="submit" disabled={actionLoading} className="btn-primary">Deposit</button>
          </div>
        </form>

        <form onSubmit={handleWithdraw} className="card space-y-4">
          <h3 className="font-display text-lg font-semibold">Withdraw</h3>
          <div className="flex gap-2">
            <input type="number" step="0.01" min="0.01" required placeholder="Amount ($)" value={withdrawAmount} onChange={(e) => setWithdrawAmount(e.target.value)} className="input-field flex-1" />
            <button type="submit" disabled={actionLoading} className="btn-outline">Withdraw</button>
          </div>
        </form>
      </div>

      {/* Recent Transactions */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-text-muted">Recent Transactions</h3>
          <Link href="/history" className="text-xs text-brand-400 hover:underline">View all</Link>
        </div>
        {transactions.length === 0 ? (
          <p className="text-sm text-text-muted">No transactions yet</p>
        ) : (
          <div className="space-y-2">
            {transactions.map((tx) => (
              <div key={tx.id} className="flex items-center justify-between rounded-lg bg-surface-300 px-4 py-2.5">
                <div>
                  <span className="text-sm font-medium">{txTypeLabel(tx.type)}</span>
                  <p className="text-xs text-text-muted">{formatDate(tx.created_at)}</p>
                </div>
                <span className={`text-sm font-semibold num ${tx.amount >= 0 ? 'text-brand-400' : 'text-electric-magenta'}`}>
                  {tx.amount >= 0 ? '+' : ''}${formatCents(tx.amount)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
