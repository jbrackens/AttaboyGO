'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatCents, formatDate, txTypeLabel } from '@/lib/format';

interface Transaction { id: string; type: string; amount: number; created_at: string; }

export default function HistoryPage() {
  const token = useAuthStore((s) => s.token)!;
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const limit = 20;

  useEffect(() => {
    setLoading(true);
    api<{ transactions: Transaction[] }>(`/wallet/transactions?limit=${limit}&offset=${page * limit}`, { token })
      .then((res) => setTransactions(res?.transactions || []))
      .catch(() => setTransactions([]))
      .finally(() => setLoading(false));
  }, [token, page]);

  return (
    <div className="mx-auto max-w-4xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Transaction History</h1>

      <div className="card">
        {loading ? (
          <div className="flex items-center justify-center py-12"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>
        ) : transactions.length === 0 ? (
          <p className="text-sm text-text-muted py-8 text-center">No transactions found</p>
        ) : (
          <>
            {/* Table header */}
            <div className="hidden sm:grid grid-cols-4 gap-4 px-4 py-2 text-xs font-semibold uppercase tracking-wider text-text-muted border-b border-surface-50 mb-2">
              <span>Type</span>
              <span>Date</span>
              <span className="text-right">Amount</span>
              <span className="text-right">ID</span>
            </div>

            <div className="space-y-1">
              {transactions.map((tx) => (
                <div key={tx.id} className="grid grid-cols-1 sm:grid-cols-4 gap-2 sm:gap-4 items-center rounded-lg bg-surface-300 px-4 py-3">
                  <span className="text-sm font-medium">{txTypeLabel(tx.type)}</span>
                  <span className="text-xs text-text-muted">{formatDate(tx.created_at)}</span>
                  <span className={`text-sm font-semibold num text-right ${tx.amount >= 0 ? 'text-brand-400' : 'text-electric-magenta'}`}>
                    {tx.amount >= 0 ? '+' : ''}${formatCents(tx.amount)}
                  </span>
                  <span className="text-xs text-text-muted font-mono text-right truncate">{tx.id.slice(0, 8)}</span>
                </div>
              ))}
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between pt-4 mt-4 border-t border-surface-50">
              <button onClick={() => setPage((p) => Math.max(0, p - 1))} disabled={page === 0} className="btn-secondary text-xs disabled:opacity-30">
                Previous
              </button>
              <span className="text-xs text-text-muted">Page {page + 1}</span>
              <button onClick={() => setPage((p) => p + 1)} disabled={transactions.length < limit} className="btn-secondary text-xs disabled:opacity-30">
                Next
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
