// 前端状态层的纯函数模块。
// 只放 state shape、reducer、默认查询和状态映射，确保可以脱离 Vue DOM 单独测试。

import {
  defaultDisplayPreferences,
  defaultRuntimeSettings,
  type AppInfo,
  type DisplayPreferences,
  type EnvironmentInfo,
  type LogEntry,
  type LogFileInfo,
  type LogQuery,
  type LogResponse,
  type LogStats,
  type Settings,
  type UpdateCheckResult,
  type UpdateStatus,
} from '../api/wails'

export type StartupApiKey = 'settings' | 'displayPreferences' | 'appInfo' | 'environmentInfo' | 'updateStatus' | 'logFiles' | 'logs'
export type StartupApiCallState = 'idle' | 'loading' | 'ok' | 'error'

export type StartupApiStatus = {
  state: StartupApiCallState
  message: string
  updatedAt: string
}

export type StartupApiStatusMap = Record<StartupApiKey, StartupApiStatus>

// AppState 定义前端纯状态模型，更新只保存当前检查结果和当前状态。
export type AppState = {
  appInfo?: AppInfo
  environmentInfo?: EnvironmentInfo
  settings?: Settings
  displayPreferences?: DisplayPreferences
  latestUpdateCheck?: UpdateCheckResult
  updateStatus?: UpdateStatus
  logFiles: LogFileInfo[]
  selectedLogFileName: string
  logSource: string
  logFilePath: string
  logs: LogEntry[]
  logStats: LogStats
  logTotal: number
  logPage: number
  logPageSize: number
  logHasMore: boolean
  currentLogQuery: LogQuery
  loading: boolean
  checking: boolean
  downloading: boolean
  errorMessage: string
  startupApiStatuses: StartupApiStatusMap
}

// InitialPayload 定义初始化一次性载入的数据。
export type InitialPayload = {
  appInfo: AppInfo
  environmentInfo: EnvironmentInfo
  settings: Settings
  displayPreferences: DisplayPreferences
  updateStatus: UpdateStatus
  logFiles: LogFileInfo[]
  logResponse: LogResponse
}

// AppAction 定义状态层允许的同步变更。
export type AppAction =
  | { type: 'loadingSet'; payload: boolean }
  | { type: 'checkingSet'; payload: boolean }
  | { type: 'downloadingSet'; payload: boolean }
  | { type: 'errorSet'; payload: string }
  | { type: 'startupApiStatusSet'; payload: { key: StartupApiKey; state: StartupApiCallState; message?: string } }
  | { type: 'initialised'; payload: InitialPayload }
  | { type: 'appInfoApplied'; payload: AppInfo }
  | { type: 'environmentInfoApplied'; payload: EnvironmentInfo }
  | { type: 'updateCheckApplied'; payload: UpdateCheckResult }
  | { type: 'updateStatusApplied'; payload: UpdateStatus }
  | { type: 'settingsApplied'; payload: Settings }
  | { type: 'displayPreferencesApplied'; payload: DisplayPreferences }
  | { type: 'logFilesApplied'; payload: LogFileInfo[] }
  | { type: 'logQueryApplied'; payload: LogQuery }
  | { type: 'logsApplied'; payload: LogResponse }

// initialAppState 保存前端默认状态。
export const initialAppState: AppState = {
  appInfo: undefined,
  environmentInfo: undefined,
  settings: { ...defaultRuntimeSettings },
  displayPreferences: { ...defaultDisplayPreferences },
  latestUpdateCheck: undefined,
  updateStatus: undefined,
  logFiles: [],
  selectedLogFileName: '',
  logSource: 'file',
  logFilePath: '',
  logs: [],
  logStats: { total: 0, debug: 0, info: 0, warning: 0, error: 0 },
  logTotal: 0,
  logPage: 1,
  logPageSize: 50,
  logHasMore: false,
  currentLogQuery: defaultLogQuery(),
  loading: false,
  checking: false,
  downloading: false,
  errorMessage: '',
  startupApiStatuses: defaultStartupApiStatuses(),
}

export function defaultStartupApiStatuses(): StartupApiStatusMap {
  return {
    settings: defaultStartupApiStatus(),
    displayPreferences: defaultStartupApiStatus(),
    appInfo: defaultStartupApiStatus(),
    environmentInfo: defaultStartupApiStatus(),
    updateStatus: defaultStartupApiStatus(),
    logFiles: defaultStartupApiStatus(),
    logs: defaultStartupApiStatus(),
  }
}

function defaultStartupApiStatus(): StartupApiStatus {
  return { state: 'idle', message: '', updatedAt: '' }
}

// 默认日志查询参数集中在状态层，页面只覆盖当前筛选条件。
export function defaultLogQuery(): LogQuery {
  return {
    fileName: '',
    scope: 'all',
    severity: 'all',
    keyword: '',
    page: 1,
    pageSize: 50,
  }
}

// statusFromCheckResult 将一次更新检查结果映射为更新生命周期状态。
export function statusFromCheckResult(result: UpdateCheckResult): UpdateStatus {
  let status = 'idle'
  let errorReason = result.errorReason

  if (result.status === 'update_available') {
    status = result.sha256 ? 'update_available' : 'error'
    if (!result.sha256) {
      errorReason = 'sha256_missing'
    }
  } else if (result.status === 'no_update') {
    status = 'no_update'
  } else if (result.status === 'ignored') {
    status = 'skipped'
    errorReason = result.skipReason ?? result.errorReason
  } else if (result.status === 'error') {
    status = 'error'
  }

  return {
    status,
    message: result.message,
    version: result.latestVersion,
    assetName: result.assetName,
    sha256: result.sha256,
    verified: false,
    errorReason,
    updatedAt: result.checkedAt,
  }
}

// toMessage 统一把底层异常转成 UI 可展示消息。
export function toMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === 'string') {
    return error
  }
  return '操作失败，请查看日志。'
}

// appReducer 只做同步状态变换；异步 API 调用放在 stores/app.ts。
export function appReducer(state: AppState, action: AppAction): AppState {
  if (action.type === 'loadingSet') {
    return { ...state, loading: action.payload }
  }
  if (action.type === 'checkingSet') {
    return { ...state, checking: action.payload }
  }
  if (action.type === 'downloadingSet') {
    return { ...state, downloading: action.payload }
  }
  if (action.type === 'errorSet') {
    return { ...state, errorMessage: action.payload }
  }
  if (action.type === 'startupApiStatusSet') {
    return {
      ...state,
      startupApiStatuses: {
        ...state.startupApiStatuses,
        [action.payload.key]: {
          state: action.payload.state,
          message: action.payload.message ?? '',
          updatedAt: new Date().toISOString(),
        },
      },
    }
  }
  if (action.type === 'initialised') {
    const { appInfo, environmentInfo, settings, displayPreferences, updateStatus, logFiles, logResponse } = action.payload
    return {
      ...state,
      appInfo,
      environmentInfo,
      settings,
      displayPreferences,
      updateStatus,
      logFiles,
      ...logState(logResponse),
      currentLogQuery: {
        ...defaultLogQuery(),
        fileName: logResponse.fileName,
        page: logResponse.page,
        pageSize: logResponse.pageSize,
      },
    }
  }
  if (action.type === 'appInfoApplied') {
    return { ...state, appInfo: action.payload }
  }
  if (action.type === 'environmentInfoApplied') {
    return { ...state, environmentInfo: action.payload }
  }
  if (action.type === 'updateCheckApplied') {
    return {
      ...state,
      latestUpdateCheck: action.payload,
      updateStatus: statusFromCheckResult(action.payload),
    }
  }
  if (action.type === 'updateStatusApplied') {
    return { ...state, updateStatus: action.payload }
  }
  if (action.type === 'settingsApplied') {
    return { ...state, settings: action.payload }
  }
  if (action.type === 'displayPreferencesApplied') {
    return { ...state, displayPreferences: action.payload }
  }
  if (action.type === 'logFilesApplied') {
    return { ...state, logFiles: action.payload }
  }
  if (action.type === 'logQueryApplied') {
    return { ...state, currentLogQuery: { ...action.payload } }
  }
  if (action.type === 'logsApplied') {
    return { ...state, ...logState(action.payload) }
  }
  return state
}

// logState 将日志响应转换为状态字段。
function logState(response: LogResponse) {
  return {
    logs: response.logs,
    selectedLogFileName: response.fileName,
    logSource: response.source,
    logFilePath: response.filePath,
    logStats: response.stats,
    logTotal: response.total,
    logPage: response.page,
    logPageSize: response.pageSize,
    logHasMore: response.hasMore,
  }
}
