'use client';

import { useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';
import { useAuthStore } from '@/lib/auth-store';
import { formatDate } from '@/lib/utils';
import { LoadingSpinner } from '@/components/loading-spinner';
import { ErrorMessage } from '@/components/error-message';

interface Post {
  id: string;
  player_id: string;
  content: string;
  created_at: string;
}

export default function SocialPage() {
  const token = useAuthStore((s) => s.token)!;
  const playerId = useAuthStore((s) => s.playerId);
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [content, setContent] = useState('');
  const [posting, setPosting] = useState(false);
  const [postError, setPostError] = useState('');

  async function loadPosts() {
    const p = await api<Post[]>('/social/posts', { token });
    setPosts(p);
  }

  useEffect(() => {
    loadPosts()
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  async function handlePost(e: React.FormEvent) {
    e.preventDefault();
    if (!content.trim()) return;
    setPostError('');
    setPosting(true);
    try {
      await api('/social/posts', {
        method: 'POST',
        token,
        body: { content },
      });
      setContent('');
      await loadPosts();
    } catch (err) {
      setPostError(err instanceof ApiError ? err.message : 'Failed to post');
    } finally {
      setPosting(false);
    }
  }

  async function handleDelete(postId: string) {
    try {
      await api(`/social/posts/${postId}`, { method: 'DELETE', token });
      setPosts((prev) => prev.filter((p) => p.id !== postId));
    } catch {
      // silent
    }
  }

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div className="space-y-6">
      <form onSubmit={handlePost} className="rounded-lg bg-white p-5 shadow-sm">
        <ErrorMessage message={postError} />
        <textarea
          rows={3}
          placeholder="What's on your mind?"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
        />
        <div className="mt-2 flex justify-end">
          <button
            type="submit"
            disabled={posting || !content.trim()}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {posting ? 'Posting...' : 'Post'}
          </button>
        </div>
      </form>

      <div className="space-y-3">
        {posts.length === 0 ? (
          <p className="text-sm text-gray-400">No posts yet. Be the first!</p>
        ) : (
          posts.map((post) => (
            <div key={post.id} className="rounded-lg bg-white p-5 shadow-sm">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <p className="text-sm text-gray-900 whitespace-pre-wrap">{post.content}</p>
                  <p className="mt-2 text-xs text-gray-400">{formatDate(post.created_at)}</p>
                </div>
                {post.player_id === playerId && (
                  <button
                    onClick={() => handleDelete(post.id)}
                    className="ml-3 text-xs text-red-500 hover:text-red-700"
                  >
                    Delete
                  </button>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
