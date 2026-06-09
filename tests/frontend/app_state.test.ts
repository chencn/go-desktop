// 前端状态层的纯函数测试。
// 放在 tests/frontend 下，避免把测试文件混进 frontend/src 生产源码目录。
import { describe, expect, it, vi } from 'vitest'

vi.mock('../../frontend/src/api/wails', () => {
  const defaultRuntimeSettings = {
    updateSource: 'local',
    githubOwner: 'chencn',
    githubRepo: 'go-desktop',
    githubProxyBase: '',
    updateCheckIntervalHours: 6,
    minimizeToTray: true,
    logRetentionDays: 30,
    logLevel: 'info',
    autoLaunch: false,
    createDesktopShortcut: false,
    launchHiddenToTray: false,
  }
  const defaultShadcnDisplayProfile = {
    accentColor: 'neutral',
    baseColor: 'neutral',
    cardBorder: 'visible',
    chartColor: 'neutral',
    density: 'comfortable',
    iconTone: 'default',
    menu: 'default',
    menuAccent: 'subtle',
    radius: 'default',
    textSize: 'normal',
    themeColor: 'neutral',
    uiStyle: 'vega',
  }
  const defaultAntDesignDisplayProfile = {
    ...defaultShadcnDisplayProfile,
    accentColor: 'blue',
    chartColor: 'blue',
    radius: 'medium',
    themeColor: 'blue',
  }

  return {
    defaultRuntimeSettings,
    defaultLicenseStatus: {
      enabled: false,
      required: false,
      authorized: true,
      deviceCode: '',
      message: '授权未启用',
    },
    defaultDisplayPreferences: {
      ...defaultShadcnDisplayProfile,
      displayScheme: 'shadcn',
      themeMode: 'light',
      profiles: {
        shadcn: defaultShadcnDisplayProfile,
        antd: defaultAntDesignDisplayProfile,
      },
    },
  }
})

import {
  type AppAction,
  appReducer,
  initialAppState,
  statusFromCheckResult,
  toMessage,
} from '../../frontend/src/app/state'
import type {
  LogResponse,
  UpdateCheckResult,
} from '../../frontend/src/api/wails'

// checkResult 是最小更新检查样例；缺少 sha256 时必须进入受保护错误态。
const checkResult: UpdateCheckResult = {
  source: 'local',
  status: 'update_available',
  currentVersion: '1.0.0',
  latestVersion: '0.0.2',
  tagName: 'v0.0.2',
  assetName: 'go-desktop.exe',
  checkedAt: '2026-06-04T00:00:00Z',
  message: '发现新版本',
}

describe('app state reducer', () => {
  it('normalizes Wails runtime error payloads to readable messages', () => {
    expect(toMessage('{"message":"授权码格式无效","cause":{},"kind":"RuntimeError"}')).toBe('授权码格式无效')
    expect(toMessage(new Error('{"message":"授权码格式无效","cause":{},"kind":"RuntimeError"}'))).toBe('授权码格式无效')
    expect(toMessage({ message: '授权码签名无效', cause: {}, kind: 'RuntimeError' })).toBe('授权码签名无效')
  })

  it('maps update checks without sha256 to a guarded error state', () => {
    expect(statusFromCheckResult(checkResult)).toMatchObject({
      status: 'error',
      version: '0.0.2',
      errorReason: 'sha256_missing',
      source: 'local',
      verified: false,
    })
  })

  it('stores latest update check and updates lifecycle status', () => {
    const action: AppAction = { type: 'updateCheckApplied', payload: { ...checkResult, sha256: 'abc123' } }
    const next = appReducer(initialAppState, action)

    expect(next.latestUpdateCheck?.latestVersion).toBe('0.0.2')
    expect(next.updateStatus?.status).toBe('update_available')
    expect(next.updateStatus?.source).toBe('local')
  })

  it('stores license status and activation errors independently', () => {
    const next = appReducer(initialAppState, {
      type: 'licenseStatusApplied',
      payload: {
        enabled: true,
        required: true,
        authorized: false,
        deviceCode: 'GD-7K3F-9P2X-MQ8C',
        message: '需要授权',
      },
    })

    expect(next.licenseStatus?.required).toBe(true)
    expect(next.licenseStatus?.authorized).toBe(false)
    expect(next.licenseError).toBe('')

    const failed = appReducer(next, { type: 'licenseErrorSet', payload: '授权码签名无效' })
    expect(failed.licenseError).toBe('授权码签名无效')
  })

  it('applies paged log responses as one immutable state update', () => {
    const response: LogResponse = {
      logs: [{ time: '2026-06-04T00:00:00Z', scope: 'app', severity: 'info', message: 'ok' }],
      source: 'file',
      fileName: 'go-desktop-2026-06-04.log',
      filePath: 'D:/app/go/go-desktop/bin/data/logs/go-desktop-2026-06-04.log',
      total: 1,
      page: 2,
      pageSize: 50,
      hasMore: false,
      stats: { total: 1, debug: 0, info: 1, warning: 0, error: 0 },
    }

    const next = appReducer(initialAppState, { type: 'logsApplied', payload: response })

    expect(next.logs).toEqual(response.logs)
    expect(next.selectedLogFileName).toBe('go-desktop-2026-06-04.log')
    expect(next.logSource).toBe('file')
    expect(next.logPage).toBe(2)
    expect(next.logStats.info).toBe(1)
    expect(initialAppState.logs).toHaveLength(0)
  })

  it('stores the current log query independently from log responses', () => {
    const query = {
      fileName: 'go-desktop-2026-06-04.log',
      scope: 'update',
      severity: 'error',
      keyword: '失败',
      page: 3,
      pageSize: 25,
    }

    const next = appReducer(initialAppState, { type: 'logQueryApplied', payload: query })

    expect(next.currentLogQuery).toEqual(query)
    expect(initialAppState.currentLogQuery.scope).toBe('all')
  })
})
