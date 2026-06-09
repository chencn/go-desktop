import { beforeEach, describe, expect, it, vi } from 'vitest'

const runtimeMock = {
  Browser: {},
  Call: {
    ByID: () => Promise.reject(new Error('unexpected Wails call in browser fallback test')),
  },
  CancellablePromise: Promise,
  Create: {
    Any: (value: unknown) => value,
    Array: (factory: (value: unknown) => unknown) => (values: unknown) => Array.isArray(values) ? values.map(factory) : [],
  },
}

function mockWailsRuntime() {
  vi.doMock('@wailsio/runtime', () => runtimeMock)
  vi.doMock('@wailsio/runtime/dist/index.js', () => runtimeMock)
  vi.doMock('./frontend/node_modules/@wailsio/runtime/dist/index.js', () => runtimeMock)
  vi.doMock('/frontend/node_modules/@wailsio/runtime/dist/index.js', () => runtimeMock)
}

function installLocalStorage() {
  const values = new Map<string, string>()
  vi.stubGlobal('window', {
    location: { origin: 'http://127.0.0.1' },
    localStorage: {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
      clear: () => values.clear(),
    },
  })
  vi.stubGlobal('localStorage', window.localStorage)
}

describe('wails api fallback boundaries', () => {
  beforeEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
    mockWailsRuntime()
    installLocalStorage()
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

  it('persists display preferences in explicit preview mode only', async () => {
    vi.stubEnv('VITE_PREVIEW', 'true')
    const { defaultDisplayPreferences, getDisplayPreferences, saveDisplayPreferences } = await import('../../frontend/src/api/wails')

    await saveDisplayPreferences({
      ...defaultDisplayPreferences,
      displayScheme: 'antd',
      themeMode: 'dark',
      profiles: {
        ...defaultDisplayPreferences.profiles,
        antd: {
          ...defaultDisplayPreferences.profiles.antd,
          menu: 'inverted',
        },
      },
    })

    await expect(getDisplayPreferences()).resolves.toMatchObject({
      displayScheme: 'antd',
      themeMode: 'dark',
      profiles: {
        antd: {
          menu: 'inverted',
        },
      },
    })
  })

  it('预览模式下授权状态默认关闭并允许进入应用', async () => {
    vi.stubEnv('VITE_PREVIEW', 'true')
    const { getLicenseStatus } = await import('../../frontend/src/api/wails')

    await expect(getLicenseStatus()).resolves.toMatchObject({
      required: false,
      authorized: true,
    })
  })

  it('预览模式下激活授权不能假成功', async () => {
    vi.stubEnv('VITE_PREVIEW', 'true')
    const { activateLicense } = await import('../../frontend/src/api/wails')

    await expect(activateLicense('GD1-test')).rejects.toThrow('激活授权失败：Wails 服务不可用。')
  })
})
