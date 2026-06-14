import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ApiError } from '@/lib/api/client'

// We test the apiFetch behavior by mocking globalThis.fetch.
// The actual module is imported dynamically below to avoid top-level fetch calls.

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch)
})

afterEach(() => {
  vi.restoreAllMocks()
})

function makeResponse(status: number, body: unknown, contentLength?: string) {
  const bodyText = JSON.stringify(body)
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get: (name: string) => {
        if (name === 'Content-Length') return contentLength ?? null
        return null
      },
    },
    json: async () => body,
    text: async () => bodyText,
  }
}

describe('ApiError', () => {
  it('carries status and body', () => {
    const err = new ApiError(404, { error: 'not found' }, 'GET /foo → 404')
    expect(err.status).toBe(404)
    expect(err.body).toEqual({ error: 'not found' })
    expect(err.message).toBe('GET /foo → 404')
    expect(err.name).toBe('ApiError')
    expect(err instanceof Error).toBe(true)
  })
})

describe('api.get', () => {
  it('returns parsed JSON on 200', async () => {
    mockFetch.mockResolvedValueOnce(makeResponse(200, { items: [] }))
    const { api } = await import('@/lib/api/client')
    const result = await api.get('/v1/admin/apps')
    expect(result).toEqual({ items: [] })
    expect(mockFetch).toHaveBeenCalledWith('/v1/admin/apps', expect.objectContaining({
      method: 'GET',
      credentials: 'include',
    }))
  })

  it('throws ApiError on 4xx', async () => {
    mockFetch.mockResolvedValueOnce(makeResponse(401, { error: 'unauthorized' }))
    const { api } = await import('@/lib/api/client')
    await expect(api.get('/v1/account')).rejects.toBeInstanceOf(ApiError)
  })

  it('returns undefined on 204', async () => {
    const noContentResponse = {
      ok: true,
      status: 204,
      headers: { get: () => null },
      json: async () => { throw new Error('no body') },
      text: async () => '',
    }
    mockFetch.mockResolvedValueOnce(noContentResponse)
    const { api } = await import('@/lib/api/client')
    const result = await api.delete('/v1/account/apps/some-id')
    expect(result).toBeUndefined()
  })

  it('returns undefined when Content-Length is 0', async () => {
    const emptyResponse = {
      ok: true,
      status: 200,
      headers: { get: (name: string) => name === 'Content-Length' ? '0' : null },
      json: async () => { throw new Error('no body') },
      text: async () => '',
    }
    mockFetch.mockResolvedValueOnce(emptyResponse)
    const { api } = await import('@/lib/api/client')
    const result = await api.get('/v1/empty')
    expect(result).toBeUndefined()
  })
})

describe('api.post', () => {
  it('sends JSON body', async () => {
    mockFetch.mockResolvedValueOnce(makeResponse(201, { id: 'new-id' }))
    const { api } = await import('@/lib/api/client')
    const result = await api.post('/v1/admin/apps', { name: 'test' })
    expect(result).toEqual({ id: 'new-id' })
    expect(mockFetch).toHaveBeenCalledWith('/v1/admin/apps', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'test' }),
    }))
  })

  it('sends without body when body is null', async () => {
    mockFetch.mockResolvedValueOnce(makeResponse(200, {}))
    const { api } = await import('@/lib/api/client')
    await api.post('/v1/endpoint', null)
    const call = mockFetch.mock.calls[0]
    expect(call[1].body).toBeUndefined()
  })
})
