// 文件职责：验证 app_state.test.ts 覆盖的生产行为、结构约束或构建脚本约束。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

// 前端状态层的纯函数测试。
// 放在 tests/frontend 下，避免把测试文件混进 frontend/src 生产源码目录。
import {describe, expect, it} from 'vitest'
import {type AppAction, appReducer, initialAppState, statusFromCheckResult,} from '../../frontend/src/app/state'
import type {LogResponse, UpdateCheckResult} from '../../frontend/src/api/wails'

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
    // action 保存 验证 app_state.test.ts 覆盖的生产行为、结构约束或构建脚本约束 使用的配置、引用或中间结果。
    const action: AppAction = { type: 'updateCheckApplied', payload: { ...checkResult, sha256: 'abc123' } }
    // next 保存 验证 app_state.test.ts 覆盖的生产行为、结构约束或构建脚本约束 使用的配置、引用或中间结果。
    const next = appReducer(initialAppState, action)

    expect(next.latestUpdateCheck?.latestVersion).toBe('0.0.2')
    expect(next.updateStatus?.status).toBe('update_available')
    expect(next.updateStatus?.source).toBe('local')
  })

  it('applies paged log responses as one immutable state update', () => {
    // response 保存 验证 app_state.test.ts 覆盖的生产行为、结构约束或构建脚本约束 使用的配置、引用或中间结果。
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

    // next 保存 验证 app_state.test.ts 覆盖的生产行为、结构约束或构建脚本约束 使用的配置、引用或中间结果。
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
