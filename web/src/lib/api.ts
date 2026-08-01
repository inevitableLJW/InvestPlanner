const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class APIError extends Error {
  constructor(public status: number, public code: string, message: string, public details?: unknown) { super(message) }
  get retryable() { return this.status >= 500 }
}

let unauthorizedHandler: (() => void) | undefined
export const onUnauthorized = (handler: () => void) => { unauthorizedHandler = handler }

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init.headers },
  })
  if (response.status === 401) unauthorizedHandler?.()
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: { code: 'request_failed', message: '请求失败，请稍后重试' } }))
    const error = payload.error ?? payload
    throw new APIError(response.status, error.code ?? 'request_failed', error.message ?? '请求失败，请稍后重试', error.details)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const get = <T>(path: string) => api<T>(path)
export const post = <T>(path: string, body?: unknown) => api<T>(path, { method: 'POST', body: body === undefined ? undefined : JSON.stringify(body) })
export const put = <T>(path: string, body: unknown) => api<T>(path, { method: 'PUT', body: JSON.stringify(body) })
export const del = (path: string) => api<void>(path, { method: 'DELETE' })
