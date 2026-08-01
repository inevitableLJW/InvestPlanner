import { afterEach, describe, expect, it, vi } from 'vitest'
import { APIError, api, onUnauthorized } from './api'

afterEach(() => vi.restoreAllMocks())

describe('api client', () => {
  it('sends credentials and parses successful responses', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 200 }))
    await expect(api<{ ok: boolean }>('/example')).resolves.toEqual({ ok: true })
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/example', expect.objectContaining({ credentials: 'include' }))
  })

  it('normalizes conflicts and retryable failures', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ error: { code: 'version_conflict', message: '数据已更新' } }), { status: 409 }))
    const error = await api('/example').catch(reason => reason) as APIError
    expect(error).toBeInstanceOf(APIError)
    expect(error.status).toBe(409)
    expect(error.retryable).toBe(false)
  })

  it('notifies the app when a session expires', async () => {
    const expired = vi.fn(); onUnauthorized(expired)
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ code: 'unauthorized', message: '请登录' }), { status: 401 }))
    await expect(api('/private')).rejects.toBeInstanceOf(APIError)
    expect(expired).toHaveBeenCalledOnce()
  })
})
