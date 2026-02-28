'use client';

import { useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/format';

interface Post { id: string; player_id: string; content: string; created_at: string; }

export default function SocialPage() {
  const token = useAuthStore((s) => s.token)!;
  const playerId = useAuthStore((s) => s.playerId);
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);
  const [content, setContent] = useState('');
  const [posting, setPosting] = useState(false);
  const [postError, setPostError] = useState('');

  async function loadPosts() { const p = await api<Post[]>('/social/posts', { token }); setPosts(p || []); }

  useEffect(() => { loadPosts().finally(() => setLoading(false)); }, [token]);

  async function handlePost(e: React.FormEvent) {
    e.preventDefault();
    if (!content.trim()) return;
    setPostError('');
    setPosting(true);
    try {
      await api('/social/posts', { method: 'POST', token, body: { content } });
      setContent('');
      await loadPosts();
    } catch (err) {
      setPostError(err instanceof ApiError ? err.message : 'Failed to post');
    } finally {
      setPosting(false);
    }
  }

  async function handleDelete(postId: string) {
    await api(`/social/posts/${postId}`, { method: 'DELETE', token }).catch(() => {});
    setPosts((prev) => prev.filter((p) => p.id !== postId));
  }

  if (loading) return <div className="flex items-center justify-center py-20"><div className="h-8 w-8 animate-spin rounded-full border-4 border-surface-50 border-t-brand-400" /></div>;

  return (
    <div className="mx-auto max-w-2xl space-y-6 animate-fade-in">
      <h1 className="font-display text-2xl font-bold">Social</h1>

      {/* Compose */}
      <form onSubmit={handlePost} className="card space-y-3">
        {postError && <p className="text-xs text-electric-magenta">{postError}</p>}
        <textarea
          rows={3}
          placeholder="Share a win, a tip, or just say hey..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
          className="input-field resize-none"
        />
        <div className="flex justify-end">
          <button type="submit" disabled={posting || !content.trim()} className="btn-primary text-sm">
            {posting ? 'Posting...' : 'Post'}
          </button>
        </div>
      </form>

      {/* Feed */}
      {posts.length === 0 ? (
        <div className="card-glass text-center py-12"><p className="text-text-muted">No posts yet. Be the first!</p></div>
      ) : (
        <div className="space-y-3">
          {posts.map((post) => (
            <div key={post.id} className="card">
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <p className="text-sm whitespace-pre-wrap">{post.content}</p>
                  <p className="mt-2 text-xs text-text-muted">{formatDate(post.created_at)}</p>
                </div>
                {post.player_id === playerId && (
                  <button onClick={() => handleDelete(post.id)} className="ml-3 text-xs text-electric-magenta hover:underline shrink-0">Delete</button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
