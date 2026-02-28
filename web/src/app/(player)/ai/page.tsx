'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Conversation {
  id: string;
  title: string;
  created_at: string;
}

export default function AIChatListPage() {
  const token = useAuthStore((s) => s.token)!;
  const router = useRouter();
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    api<Conversation[]>('/ai/conversations', { token })
      .then(setConversations)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  async function handleNew() {
    setCreating(true);
    try {
      const conv = await api<Conversation>('/ai/conversations', {
        method: 'POST',
        token,
        body: {},
      });
      router.push(`/ai/${conv.id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to create conversation');
      setCreating(false);
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">AI Chat</h2>
        <button
          onClick={handleNew}
          disabled={creating}
          className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {creating ? 'Creating...' : 'New Conversation'}
        </button>
      </div>

      <div className="space-y-2">
        {conversations.length === 0 ? (
          <p className="text-sm text-gray-400">No conversations yet</p>
        ) : (
          conversations.map((conv) => (
            <button
              key={conv.id}
              onClick={() => router.push(`/ai/${conv.id}`)}
              className="block w-full rounded-lg bg-white p-4 text-left shadow-sm hover:shadow-md transition-shadow"
            >
              <p className="font-medium text-gray-900">{conv.title || 'Untitled'}</p>
              <p className="text-xs text-gray-400">{formatDate(conv.created_at)}</p>
            </button>
          ))
        )}
      </div>
    </div>
  );
}
