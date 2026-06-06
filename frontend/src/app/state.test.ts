// 文件职责：定义前端纯状态模型、默认值、归一化和派生选择器。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

// Vue 状态层的纯函数测试。
// 这里不挂 DOM，只验证 reducer、更新检查映射和日志响应合并这些可独立测试的行为。
import { describe, expect, it } from 'vitest'
import {
  appReducer,
  initialAppState,
  statusFromCheckResult,
  // AppAction 定义前端纯状态模型、默认值、归一化和派生选择器 使用的类型契约，限制跨组件或跨模块传递的数据形状。
  type AppAction,
} from './state'
import type { LogResponse, UpdateCheckResult } from '../api/wails'

// checkResult 保存 定义前端纯状态模型、默认值、归一化和派生选择器 使用的配置、引用或中间结果。
const checkResult: UpdateCheckResult = {
  status: 'update_available',
  currentVersion: '0.0.1',
  latestVersion: '0.0.2',
  tagName: 'v0.0.2',
  assetName: 'go-desktop.exe',
  checkedAt: '2026-06-04T00:00:00Z',
  message: '发现新版本',
}

describe('app state reducer', () => {
  it('maps update checks without sha256 to a guarded error state', () => {
    expect(statusFromCheckResult(checkResult)).toMatchObject({
      status: 'error',
      version: '0.0.2',
      errorReason: 'sha256_missing',
      verified: false,
    })
  })

  it('stores latest update check and updates lifecycle status', () => {
    // action 保存 定义前端纯状态模型、默认值、归一化和派生选择器 使用的配置、引用或中间结果。
    const action: AppAction = { type: 'updateCheckApplied', payload: { ...checkResult, sha256: 'abc123' } }
    // next 保存 定义前端纯状态模型、默认值、归一化和派生选择器 使用的配置、引用或中间结果。
    const next = appReducer(initialAppState, action)

    expect(next.latestUpdateCheck?.latestVersion).toBe('0.0.2')
    expect(next.updateStatus?.status).toBe('update_available')
  })

  it('applies paged log responses as one immutable state update', () => {
    // response 保存 定义前端纯状态模型、默认值、归一化和派生选择器 使用的配置、引用或中间结果。
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

    // next 保存 定义前端纯状态模型、默认值、归一化和派生选择器 使用的配置、引用或中间结果。
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
