'use client';

import { useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate, cn } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Balance {
  balance: number;
  bonus_balance: number;
  reserved_balance: number;
  currency: string;
}

interface Transaction {
  id: string;
  type: string;
  amount: number;
  created_at: string;
}

const TYPE_COLORS: Record<string, string> = {
  deposit: 'bg-green-50 text-green-700',
  withdrawal: 'bg-red-50 text-red-700',
  bet: 'bg-amber-50 text-amber-700',
  win: 'bg-green-50 text-green-700',
  bonus: 'bg-purple-50 text-purple-700',
};

export default function WalletPage() {
  const token = useAuthStore((s) => s.token)!;
  const [balance, setBalance] = useState<Balance | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [depositAmount, setDepositAmount] = useState('');
  const [withdrawAmount, setWithdrawAmount] = useState('');
  const [actionError, setActionError] = useState('');
  const [actionLoading, setActionLoading] = useState(false);

  async function loadBalance() {
    const b = await api<Balance>('/wallet/balance', { token });
    setBalance(b);
  }

  async function loadTransactions(cur?: string) {
    const url = cur ? `/wallet/transactions?limit=20&cursor=${cur}` : '/wallet/transactions?limit=20';
    const txs = await api<Transaction[]>(url, { token });
    if (cur) {
      setTransactions((prev) => [...prev, ...txs]);
    } else {
      setTransactions(txs);
    }
    setHasMore(txs.length === 20);
    if (txs.length > 0) setCursor(txs[txs.length - 1].id);
  }

  useEffect(() => {
    Promise.all([loadBalance(), loadTransactions()])
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  async function handleDeposit(e: React.FormEvent) {
    e.preventDefault();
    setActionError('');
    setActionLoading(true);
    try {
      await api('/payments/deposit', {
        method: 'POST',
        token,
        body: { amount: Math.round(parseFloat(depositAmount) * 100) },
      });
      setDepositAmount('');
      await Promise.all([loadBalance(), loadTransactions()]);
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Deposit failed');
    } finally {
      setActionLoading(false);
    }
  }

  async function handleWithdraw(e: React.FormEvent) {
    e.preventDefault();
    setActionError('');
    setActionLoading(true);
    try {
      await api('/payments/withdraw', {
        method: 'POST',
        token,
        body: { amount: Math.round(parseFloat(withdrawAmount) * 100) },
      });
      setWithdrawAmount('');
      await Promise.all([loadBalance(), loadTransactions()]);
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Withdrawal failed');
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      {balance && (
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg bg-white p-5 shadow-sm">
            <p className="text-sm text-gray-500">Real Balance</p>
            <p className="mt-1 text-2xl font-bold text-gray-900">${formatCents(balance.balance)}</p>
          </div>
          <div className="rounded-lg bg-white p-5 shadow-sm">
            <p className="text-sm text-gray-500">Bonus Balance</p>
            <p className="mt-1 text-2xl font-bold text-amber-600">${formatCents(balance.bonus_balance)}</p>
          </div>
          <div className="rounded-lg bg-white p-5 shadow-sm">
            <p className="text-sm text-gray-500">Reserved</p>
            <p className="mt-1 text-2xl font-bold text-gray-400">${formatCents(balance.reserved_balance)}</p>
          </div>
        </div>
      )}

      <ErrorMessage message={actionError} />

      <div className="grid grid-cols-2 gap-4">
        <form onSubmit={handleDeposit} className="rounded-lg bg-white p-5 shadow-sm">
          <h3 className="mb-3 text-sm font-medium text-gray-700">Deposit</h3>
          <div className="flex gap-2">
            <input
              type="number"
              step="0.01"
              min="0.01"
              required
              placeholder="Amount ($)"
              value={depositAmount}
              onChange={(e) => setDepositAmount(e.target.value)}
              className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
            <button
              type="submit"
              disabled={actionLoading}
              className="rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
            >
              Deposit
            </button>
          </div>
        </form>

        <form onSubmit={handleWithdraw} className="rounded-lg bg-white p-5 shadow-sm">
          <h3 className="mb-3 text-sm font-medium text-gray-700">Withdraw</h3>
          <div className="flex gap-2">
            <input
              type="number"
              step="0.01"
              min="0.01"
              required
              placeholder="Amount ($)"
              value={withdrawAmount}
              onChange={(e) => setWithdrawAmount(e.target.value)}
              className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
            <button
              type="submit"
              disabled={actionLoading}
              className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
            >
              Withdraw
            </button>
          </div>
        </form>
      </div>

      <div className="rounded-lg bg-white p-5 shadow-sm">
        <h3 className="mb-4 text-sm font-medium text-gray-700">Transaction History</h3>
        {transactions.length === 0 ? (
          <p className="text-sm text-gray-400">No transactions yet</p>
        ) : (
          <div className="space-y-2">
            {transactions.map((tx) => (
              <div key={tx.id} className="flex items-center justify-between rounded-md border border-gray-100 px-4 py-2">
                <div className="flex items-center gap-3">
                  <span
                    className={cn(
                      'rounded-full px-2 py-0.5 text-xs font-medium',
                      TYPE_COLORS[tx.type] || 'bg-gray-50 text-gray-700',
                    )}
                  >
                    {tx.type}
                  </span>
                  <span className="text-xs text-gray-400">{formatDate(tx.created_at)}</span>
                </div>
                <span
                  className={`text-sm font-semibold ${tx.amount >= 0 ? 'text-green-600' : 'text-red-600'}`}
                >
                  {tx.amount >= 0 ? '+' : ''}${formatCents(tx.amount)}
                </span>
              </div>
            ))}
            {hasMore && (
              <button
                onClick={() => loadTransactions(cursor!)}
                className="w-full rounded-md border border-gray-200 py-2 text-sm text-gray-500 hover:bg-gray-50"
              >
                Load more
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
