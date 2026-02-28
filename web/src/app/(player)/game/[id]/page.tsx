'use client';

import { useParams, useRouter } from 'next/navigation';

export default function GameLauncherPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();

  return (
    <div className="mx-auto max-w-5xl space-y-4 animate-fade-in">
      <button onClick={() => router.back()} className="text-sm text-brand-400 hover:underline">&larr; Back to lobby</button>

      <div className="card p-0 overflow-hidden">
        <div className="aspect-video bg-surface-300 flex items-center justify-center">
          <div className="text-center">
            <p className="font-display text-xl text-text-muted mb-2">Game: {id}</p>
            <p className="text-sm text-text-muted">Game iframe will load here</p>
          </div>
        </div>
      </div>
    </div>
  );
}
