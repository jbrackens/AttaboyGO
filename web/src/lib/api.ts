const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3100';

export class ApiError extends Error {
  code: number;
  constructor(code: number, message: string) {
    super(message);
    this.code = code;
  }
}

export async function api<T>(
  path: string,
  opts: {
    method?: string;
    body?: unknown;
    token?: string;
  } = {},
): Promise<T> {
  const { method = 'GET', body, token } = opts;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let msg = res.statusText;
    try {
      const err = await res.json();
      msg = err.message || msg;
    } catch {
      // use statusText
    }
    throw new ApiError(res.status, msg);
  }

  if (res.status === 204) return undefined as T;

  return res.json();
}
