import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@wailsio/runtime', () => ({
  Browser: {},
  Call: {
    ByID: () => Promise.reject(new Error('unexpected Wails call in browser fallback test')),
  },
  CancellablePromise: Promise,
  Create: {
    Any: (value: unknown) => value,
    Array: (factory: (value: unknown) => unknown) => (values: unknown) => Array.isArray(values) ? values.map(factory) : [],
  },
}))

describe('wails api fallback boundaries', () => {
  beforeEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('does not fallback in production when reading settings fails', async () => {
    const { getSettings } = await import('../../frontend/src/api/wails')

    await expect(getSettings()).rejects.toThrow('unexpected Wails call in browser fallback test')
  })

  it('falls back only in explicit preview mode when reading settings fails', async () => {
    vi.stubEnv('VITE_PREVIEW', 'true')
    const { getSettings } = await import('../../frontend/src/api/wails')

    await expect(getSettings()).resolves.toMatchObject({
      githubOwner: 'chencn',
      githubRepo: 'go-desktop',
    })
  })

  it('fails when saving settings in explicit preview mode', async () => {
    vi.stubEnv('VITE_PREVIEW', 'true')
    const { defaultRuntimeSettings, saveSettings } = await import('../../frontend/src/api/wails')

    await expect(saveSettings(defaultRuntimeSettings)).rejects.toThrow('保存设置失败：Wails 服务不可用。')
  })
})
