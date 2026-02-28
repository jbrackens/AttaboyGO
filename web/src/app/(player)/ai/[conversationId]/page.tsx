'use client';

import { useEffect, useRef, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';

interface Message { id: string; role: 'user' | 'assistant'; content: string; created_at: string; }

export default function AIConversationPage() {
  const { conversationId } = useParams<{ conversationId: string }>();
  const router = useRouter();
  const token = useAuthStore((s) => s.token)!;

  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    api<Message[]>(`/ai/conversations/${conversationId}/messages`, { token })
      .then(setMessages)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [conversationId, token]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  async function handleSend(e: React.FormEvent) {
    e.preventDefault();
    if (!input.trim()) return;
    const text = input;
    setInput('');
    setSending(true);

    const userMsg: Message = { id: `temp-${Date.now()}`, role: 'user', content: text, created_at: new Date().toISOString() };
    setMessages((prev) => [...prev, userMsg]);

    try {
      const reply = await api<Message>(`/conversations/${conversationId}/messages`, { method: 'POST', token, body: { content: text } });
      setMessages((prev) => [...prev.filter((m) => m.id !== userMsg.id), userMsg, reply]);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to send');
    } finally {
      setSending(false);
    }
  }

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-3xl flex flex-col h-[calc(100vh-200px)] animate-fade-in">
      <button onClick={() => router.push('/ai')} className="text-sm text-brand-400 hover:underline mb-4">&larr; All conversations</button>

      {error && <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta mb-3">{error}</div>}

      {/* Messages */}
      <div className="flex-1 space-y-3 overflow-y-auto rounded-2xl bg-surface-200 border border-surface-50 p-4">
        {messages.length === 0 && <p className="text-center text-sm text-text-muted py-8">Start the conversation</p>}
        {messages.map((msg) => (
          <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[75%] rounded-2xl px-4 py-2.5 text-sm whitespace-pre-wrap ${
              msg.role === 'user'
                ? 'bg-brand-400 text-surface-900'
                : 'bg-surface-300 text-text-primary'
            }`}>
              {msg.content}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <form onSubmit={handleSend} className="mt-3 flex gap-2">
        <input
          type="text"
          placeholder="Type a message..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          className="input-field flex-1"
        />
        <button type="submit" disabled={sending || !input.trim()} className="btn-primary">
          {sending ? '...' : 'Send'}
        </button>
      </form>
    </div>
  );
}
