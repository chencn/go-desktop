import { beforeEach, describe, expect, it, vi } from 'vitest'

import { LocalApiRejectedError, LocalApiUnavailableError, callLocalApi, localApiLooksAvailable } from '../../frontend/src/api/http'

function respondWith(body: string, status = 200) {
  vi.stubGlobal('fetch', vi.fn(async () => ({
    status,
    text: async () => body,
  })))
}

describe('本地 /api 门面传输层', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('按位置发送参数并取回 data', async () => {
    respondWith('{"ok":true,"data":{"themeMode":"dark"}}')

    await expect(callLocalApi('GetDisplayPreferences', [])).resolves.toEqual({ themeMode: 'dark' })
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/GetDisplayPreferences')
    expect(init.method).toBe('POST')
    expect(init.body).toBe('[]')
  })

  it('门面返回后端错误时抛 LocalApiRejectedError，供上层禁止兜底', async () => {
    respondWith('{"ok":false,"error":"数据库未就绪"}', 500)

    await expect(callLocalApi('SaveSettings', [{}])).rejects.toBeInstanceOf(LocalApiRejectedError)
  })

  it('被 SPA fallback 用 index.html 兜住时判定为门面不可用', async () => {
    respondWith('<!doctype html><html><body></body></html>')

    await expect(callLocalApi('GetSettings', [])).rejects.toBeInstanceOf(LocalApiUnavailableError)
    expect(localApiLooksAvailable()).toBe(false)
  })
})
