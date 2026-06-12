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
      updateSource: 'github',
      githubProxyBase: 'https://gh-proxy.com',
    })
  })

  it('falls back in local Vite browser dev when Wails is unavailable', async () => {
    vi.stubEnv('DEV', true)
    vi.stubGlobal('window', {
      ...window,
      location: {
        hostname: '127.0.0.1',
        port: '9245',
        protocol: 'http:',
      },
      localStorage: window.localStorage,
    })
    const { getLicenseStatus } = await import('../../frontend/src/api/wails')

    await expect(getLicenseStatus()).resolves.toMatchObject({
      required: false,
      authorized: true,
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
      displayScheme: 'artistic',
      themeMode: 'dark',
      profiles: {
        ...defaultDisplayPreferences.profiles,
        artistic: {
          ...defaultDisplayPreferences.profiles.artistic,
          menu: 'inverted',
        },
      },
    })

    await expect(getDisplayPreferences()).resolves.toMatchObject({
      displayScheme: 'artistic',
      themeMode: 'dark',
      profiles: {
        artistic: {
          menu: 'inverted',
        },
      },
    })
  })

  it('normalizes preview display profiles without legacy scheme aliases', async () => {
    vi.stubEnv('VITE_PREVIEW', 'true')
    window.localStorage.setItem('go-desktop.preview.displayPreferences', JSON.stringify({
      displayScheme: 'artistic',
      themeMode: 'dark',
      profiles: {
        shadcn: {
          themeColor: 'rose',
        },
        artistic: {
          menu: 'inverted',
        },
      },
    }))
    const { getDisplayPreferences } = await import('../../frontend/src/api/wails')

    await expect(getDisplayPreferences()).resolves.toMatchObject({
      displayScheme: 'artistic',
      themeMode: 'dark',
      profiles: {
        shadcn: {
          themeColor: 'rose',
        },
        artistic: {
          themeColor: 'apple-blue',
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
