export class ApiError extends Error {
  readonly status: number
  readonly body: unknown

  constructor(status: number, body: unknown, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })

  if (!res.ok) {
    let body: unknown
    try {
      body = await res.json()
    } catch {
      body = await res.text()
    }
    throw new ApiError(res.status, body, `${init?.method ?? 'GET'} ${path} → ${res.status}`)
  }

  if (res.status === 204 || res.headers.get('Content-Length') === '0') {
    return undefined as T
  }

  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string, init?: Omit<RequestInit, 'method'>) =>
    apiFetch<T>(path, { ...init, method: 'GET' }),

  post: <T>(path: string, body?: unknown, init?: Omit<RequestInit, 'method' | 'body'>) =>
    apiFetch<T>(path, { ...init, method: 'POST', body: body != null ? JSON.stringify(body) : undefined }),

  patch: <T>(path: string, body?: unknown, init?: Omit<RequestInit, 'method' | 'body'>) =>
    apiFetch<T>(path, { ...init, method: 'PATCH', body: body != null ? JSON.stringify(body) : undefined }),

  put: <T>(path: string, body?: unknown, init?: Omit<RequestInit, 'method' | 'body'>) =>
    apiFetch<T>(path, { ...init, method: 'PUT', body: body != null ? JSON.stringify(body) : undefined }),

  delete: <T>(path: string, init?: Omit<RequestInit, 'method'>) =>
    apiFetch<T>(path, { ...init, method: 'DELETE' }),
}
