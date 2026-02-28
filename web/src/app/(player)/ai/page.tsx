'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/format';

interface Conversation { id: string; title: string; created_at: string; }

export default function AIChatListPage() {
  const token = useAuthStore((s) => s.token)!;
  const router = useRouter();
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    api<Conversation[]>('/ai/conversations', { token }).then((c) => setConversations(c || [])).finally(() => setLoading(false));
  }, [token]);

  async function handleNew() {
    setCreating(true);
    try {
      const conv = await api<Conversation>('/ai/conversations', { method: 'POST', token, body: {} });
      router.push(`/ai/${conv.id}`);
    } catch {
      setCreating(false);
    }
  }

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-3xl space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl font-bold">AI Chat</h1>
        <button onClick={handleNew} disabled={creating} className="btn-primary text-sm">
          {creating ? 'Creating...' : 'New Conversation'}
        </button>
      </div>

      {conversations.length === 0 ? (
        <div className="card-glass text-center py-12">
          <p className="text-text-muted">No conversations yet. Start one!</p>
        </div>
      ) : (
        <div className="space-y-2">
          {conversations.map((conv) => (
            <button
              key={conv.id}
              onClick={() => router.push(`/ai/${conv.id}`)}
              className="card w-full text-left hover:border-brand-400/20 transition-colors"
            >
              <p className="font-medium">{conv.title || 'Untitled'}</p>
              <p className="text-xs text-text-muted mt-1">{formatDate(conv.created_at)}</p>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
