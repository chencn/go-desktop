// 前端状态层的纯函数模块。
// 只放 state shape、reducer、默认查询和状态映射，确保可以脱离 Vue DOM 单独测试。

import {
  defaultDisplayPreferences,
  defaultRuntimeSettings,
  type AppInfo,
  type DisplayPreferences,
  type EnvironmentInfo,
  type LicenseStatus,
  type LogEntry,
  type LogFileInfo,
  type LogQuery,
  type LogResponse,
  type LogStats,
  type Settings,
  type UpdateCheckResult,
  type UpdateStatus,
} from '../api/wails'

/** 启动诊断面板追踪的启动期 API 清单，key 与 readStartupApi 调用一一对应。 */
export type StartupApiKey = 'settings' | 'displayPreferences' | 'appInfo' | 'environmentInfo' | 'licenseStatus' | 'updateStatus' | 'logFiles' | 'logs'
/** 启动 API 调用状态：idle 未调用、loading 进行中、ok 成功、error 失败。 */
export type StartupApiCallState = 'idle' | 'loading' | 'ok' | 'error'

// 单个启动 API 的读状态，用于启动诊断面板区分“未调用、调用中、成功、失败”。
export type StartupApiStatus = {
  state: StartupApiCallState
  message: string
  updatedAt: string
}

/** 按 StartupApiKey 建索引的启动 API 状态表。 */
export type StartupApiStatusMap = Record<StartupApiKey, StartupApiStatus>

// 前端纯状态模型；更新链只保存最近一次检查结果和当前生命周期快照，不维护历史列表。
export type AppState = {
  // 启动一次性读取的运行时快照。
  appInfo?: AppInfo
  environmentInfo?: EnvironmentInfo
  licenseStatus?: LicenseStatus
  // 授权页独立 loading/error，避免阻塞主页面初始化状态。
  licenseLoading: boolean
  licenseError: string
  settings?: Settings
  displayPreferences?: DisplayPreferences
  // latestUpdateCheck 来自 CheckUpdate；updateStatus 来自后端状态机或事件流。
  latestUpdateCheck?: UpdateCheckResult
  updateStatus?: UpdateStatus
  // 日志列表状态由 QueryLogs 的分页响应展开，currentLogQuery 保存最近一次筛选条件。
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
  // 页面级操作状态：loading 覆盖启动流程，checking/downloading 覆盖更新按钮状态。
  loading: boolean
  checking: boolean
  downloading: boolean
  errorMessage: string
  // 每个启动 API 的细粒度状态，不参与业务判断，只用于展示和排障。
  startupApiStatuses: StartupApiStatusMap
}

// InitialPayload 定义初始化一次性载入的数据。
export type InitialPayload = {
  appInfo: AppInfo
  environmentInfo: EnvironmentInfo
  licenseStatus: LicenseStatus
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
  | { type: 'licenseLoadingSet'; payload: boolean }
  | { type: 'licenseErrorSet'; payload: string }
  | { type: 'licenseStatusApplied'; payload: LicenseStatus }
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

// 默认状态只使用前端常量；真实设置和显示偏好会在 initialise 阶段覆盖。
export const initialAppState: AppState = {
  appInfo: undefined,
  environmentInfo: undefined,
  licenseStatus: undefined,
  licenseLoading: false,
  licenseError: '',
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

/** 构造每个启动 API 的 idle 初始状态表。 */
export function defaultStartupApiStatuses(): StartupApiStatusMap {
  return {
    settings: defaultStartupApiStatus(),
    displayPreferences: defaultStartupApiStatus(),
    appInfo: defaultStartupApiStatus(),
    environmentInfo: defaultStartupApiStatus(),
    licenseStatus: defaultStartupApiStatus(),
    updateStatus: defaultStartupApiStatus(),
    logFiles: defaultStartupApiStatus(),
    logs: defaultStartupApiStatus(),
  }
}

/** 判断显示偏好是否已读取完毕（成功或失败均可），决定外壳是否可以进入主题渲染。 */
export function isDisplayPreferencesReadyForShell(status: StartupApiStatus) {
  return status.state === 'ok' || status.state === 'error'
}

/** 构造单个启动 API 的 idle 初始状态。 */
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

// 将 CheckUpdate 的一次性结果转换为 UI 可消费的更新状态；真实下载进度仍以后端状态机为准。
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
    source: result.source,
    errorReason,
    updatedAt: result.checkedAt,
  }
}

// 统一把 Wails/Go/JSON 包装过的异常转成 UI 可展示消息。
export function toMessage(error: unknown) {
  if (error instanceof Error) {
    return normalizeErrorMessage(error.message)
  }
  if (typeof error === 'string') {
    return normalizeErrorMessage(error)
  }
  const objectMessage = messageFromObject(error)
  if (objectMessage) {
    return objectMessage
  }
  return '操作失败，请查看日志。'
}

/** 去除首尾空白后返回；空白消息统一替换为可展示的默认失败文案。 */
function normalizeErrorMessage(message: string) {
  const trimmed = message.trim()
  if (!trimmed) return '操作失败，请查看日志。'
  const parsedMessage = messageFromJsonString(trimmed)
  return parsedMessage || trimmed
}

/** 尝试把形如 JSON 对象的字符串解析出 message 字段；不是 JSON 或没有 message 返回空串。 */
function messageFromJsonString(value: string) {
  if (!value.startsWith('{') || !value.endsWith('}')) return ''
  try {
    return messageFromObject(JSON.parse(value))
  } catch {
    return ''
  }
}

/** 从对象中提取字符串 message 字段；非对象或空值返回空串。 */
function messageFromObject(value: unknown) {
  if (!value || typeof value !== 'object') return ''
  const message = (value as { message?: unknown }).message
  return typeof message === 'string' && message.trim() ? message.trim() : ''
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
  if (action.type === 'licenseLoadingSet') {
    return { ...state, licenseLoading: action.payload }
  }
  if (action.type === 'licenseErrorSet') {
    return { ...state, licenseError: action.payload }
  }
  if (action.type === 'licenseStatusApplied') {
    return { ...state, licenseStatus: action.payload, licenseError: '' }
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
    const { appInfo, environmentInfo, licenseStatus, settings, displayPreferences, updateStatus, logFiles, logResponse } = action.payload
    return {
      ...state,
      appInfo,
      environmentInfo,
      licenseStatus,
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
