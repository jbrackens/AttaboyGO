export function ErrorMessage({ message }: { message: string }) {
  if (!message) return null;
  return (
    <div className="rounded-lg bg-electric-magenta/10 border border-electric-magenta/30 px-4 py-3 text-sm text-electric-magenta">
      {message}
    </div>
  );
}
