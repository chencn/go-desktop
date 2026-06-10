/**
 * ============================================================================
 * 文件: api/wails.ts
 * 描述: Wails Go 后端服务绑定层
 *
 * 功能概述:
 * - 定义所有后端 API 的 TypeScript 类型和接口
 * - 封装 Wails 绑定调用，提供类型安全的异步函数
 * - 支持显式前端预览模式（VITE_PREVIEW=true 时提供 fallback 数据）
 * - 处理外部链接打开（优先使用 Wails Browser API，降级到 window.open）
 *
 * 架构说明:
 * - 通过 Wails 的代码生成绑定（bindings/ 目录）调用 Go 后端方法
 * - 读取类 API 可在显式预览模式降级；保存类调用必须抛错，避免假保存成功
 * ============================================================================
 */

import { Browser } from '@wailsio/runtime'
import * as AppBinding from '../../bindings/github.com/chencn/go-desktop/app'
import { defaultSettings, projectMetadata } from '../shared/project'

// ============================================================================
// 应用信息类型
// ============================================================================

/** 应用基本信息 */
export type AppInfo = {
  /** 应用名称 */
  name: string
  /** 应用版本号（semver 格式） */
  version: string
  /** 应用描述 */
  description: string
  /** 仓库地址 */
  repository: string
  /** 应用启动时间（ISO 8601 格式） */
  startedAt: string
}

/** 运行环境信息 */
export type EnvironmentInfo = {
  /** 操作系统（如 "windows", "darwin", "linux"） */
  os: string
  /** CPU 架构（如 "amd64", "arm64"） */
  arch: string
  /** Go 运行时版本 */
  goVersion: string
  /** Wails 框架版本 */
  wailsVersion: string
  /** SQLite 数据库路径 */
  databasePath: string
  /** SQLite 配置库是否已打开并完成默认配置初始化 */
  databaseReady: boolean
  /** SQLite 配置库状态 */
  databaseStatus: string
  /** SQLite 配置库状态说明 */
  databaseMessage: string
  /** 文件日志路径 */
  logFilePath: string
  /** 缓存目录路径 */
  cachePath: string
}

// ============================================================================
// 授权状态类型
// ============================================================================

/** 当前客户端授权状态 */
export type LicenseStatus = {
  /** 授权功能是否被构建配置启用 */
  enabled: boolean
  /** 当前发行版是否要求授权 */
  required: boolean
  /** 当前设备是否已通过授权码验签 */
  authorized: boolean
  /** 当前设备短码，用于生成授权码 */
  deviceCode: string
  /** 可展示给用户的授权状态说明 */
  message: string
  /** 授权码过期时间，空字符串或缺失表示永久 */
  expiresAt?: string
  /** 最近一次授权失败原因 */
  lastError?: string
}

/** 前端预览模式默认不启用授权，避免浏览器预览被授权页拦住。 */
export const defaultLicenseStatus: LicenseStatus = {
  enabled: false,
  required: false,
  authorized: true,
  deviceCode: '',
  message: '授权未启用',
}

// ============================================================================
// 更新检查类型
// ============================================================================

/** 更新检查结果状态 */
export type UpdateCheckStatus = 'no_update' | 'update_available' | 'ignored' | 'error'
export type UpdateSource = 'github' | 'local'

/** 单次更新检查结果 */
export type UpdateCheckResult = {
  /** 更新源 */
  source?: UpdateSource
  /** 检查状态 */
  status: UpdateCheckStatus
  /** 当前版本 */
  currentVersion: string
  /** 请求的 GitHub API URL */
  requestUrl?: string
  /** HTTP 状态码 */
  httpStatus?: number
  /** 最新版本号 */
  latestVersion?: string
  /** Git 标签名 */
  tagName?: string
  /** 发布页面 URL */
  releaseUrl?: string
  /** 发布说明（Markdown） */
  releaseNotes?: string
  /** 安装包文件名 */
  assetName?: string
  /** 安装包大小（字节） */
  assetSizeBytes?: number
  /** 安装包下载链接 */
  assetDownloadUrl?: string
  /** SHA256 校验值 */
  sha256?: string
  /** SHA256 校验值来源 */
  sha256Source?: 'github_digest' | 'sha256_asset'
  /** 跳过原因 */
  skipReason?: 'offline' | 'disabled' | 'rate_limited'
  /** 错误原因（内部错误码） */
  errorReason?: string
  /** 检查时间（ISO 8601 格式） */
  checkedAt: string
  /** 可读的结果消息 */
  message: string
}

// ============================================================================
// 设置类型
// ============================================================================

export type LogLevel = 'debug' | 'info' | 'warning' | 'error'

/** 应用设置 */
export type Settings = {
  /** 更新源 */
  updateSource: UpdateSource
  /** GitHub 仓库所有者 */
  githubOwner: string
  /** GitHub 仓库名称 */
  githubRepo: string
  /** GitHub API 代理地址（空字符串表示不使用代理） */
  githubProxyBase: string
  /** 自动检查更新间隔（小时） */
  updateCheckIntervalHours: number
  /** 是否关闭到系统托盘 */
  minimizeToTray: boolean
  /** 日志保留天数（-1 表示永久保留） */
  logRetentionDays: number
  /** 最小记录日志级别 */
  logLevel: LogLevel
  /** 是否开机自启 */
  autoLaunch: boolean
  /** 是否创建桌面快捷图标 */
  createDesktopShortcut: boolean
  /** 开机自启时是否隐藏到托盘 */
  launchHiddenToTray: boolean
}


export const defaultRuntimeSettings: Settings = {
  ...defaultSettings,
  logLevel: 'info',
}

export type DisplayProfile = {
  /** 组件风格 */
  uiStyle: string
  /** 基础色盘 */
  baseColor: string
  /** 主题色 */
  themeColor: string
  /** 强调色 */
  accentColor: string
  /** 图表色 */
  chartColor: string
  /** 图标颜色模式 */
  iconTone: string
  /** 菜单样式 */
  menu: string
  /** 菜单强调样式 */
  menuAccent: string
  /** 圆角 */
  radius: string
  /** 密度 */
  density: string
  /** 字体大小 */
  textSize: string
  /** 卡片边框强度 */
  cardBorder: string
}

export type DisplayProfiles = {
  shadcn: DisplayProfile
  artistic: DisplayProfile
}

/** 显示偏好 */
export type DisplayPreferences = DisplayProfile & {
  /** 显示方案 */
  displayScheme: string
  /** 主题模式 */
  themeMode: string
  /** 所有平级显示方案的独立偏好 */
  profiles: DisplayProfiles
}

const defaultShadcnDisplayProfile: DisplayProfile = {
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



const defaultArtisticDisplayProfile: DisplayProfile = {
  accentColor: 'apple-blue',
  baseColor: 'stone',
  cardBorder: 'soft',
  chartColor: 'emerald',
  density: 'comfortable',
  iconTone: 'colorful',
  menu: 'default',
  menuAccent: 'bold',
  radius: 'large',
  textSize: 'normal',
  themeColor: 'apple-blue',
  uiStyle: 'vega',
}

/** 前端预览模式使用的显示偏好默认值，真实运行时以后端 SQLite KV 为准。 */
export const defaultDisplayPreferences: DisplayPreferences = {
  ...defaultArtisticDisplayProfile,
  displayScheme: 'artistic',
  themeMode: 'light',
  profiles: {
    shadcn: { ...defaultShadcnDisplayProfile },
    artistic: { ...defaultArtisticDisplayProfile },
  },
}

const previewDisplayPreferencesStorageKey = 'go-desktop.preview.displayPreferences'

// ============================================================================
// 日志类型
// ============================================================================

/** 单条日志记录 */
export type LogEntry = {
  /** 日志时间（ISO 8601 格式） */
  time: string
  /** 日志来源（如 "app", "window", "update"） */
  scope: string
  /** 日志内容 */
  message: string
  /** 日志级别（"info", "warning", "error"） */
  severity: string
}

/** 可选择的每日日志文件 */
export type LogFileInfo = {
  /** 日期（YYYY-MM-DD） */
  date: string
  /** 文件名 */
  fileName: string
  /** 完整路径 */
  filePath: string
  /** 文件大小 */
  sizeBytes: number
  /** 修改时间 */
  modifiedAt: string
  /** 是否为当前写入文件 */
  current: boolean
}

/** 日志查询参数 */
export type LogQuery = {
  /** 指定日志文件名，空字符串表示当前每日文件 */
  fileName: string
  /** 按来源过滤（"all" 表示全部） */
  scope: string
  /** 按级别过滤（"all" 表示全部） */
  severity: string
  /** 关键词搜索 */
  keyword: string
  /** 页码（从 1 开始） */
  page: number
  /** 每页条数 */
  pageSize: number
}

/** 日志统计信息 */
export type LogStats = {
  /** 总条数 */
  total: number
  /** 调试级别条数 */
  debug: number
  /** 信息级别条数 */
  info: number
  /** 警告级别条数 */
  warning: number
  /** 错误级别条数 */
  error: number
}

/** 日志查询响应 */
export type LogResponse = {
  /** 当前页的日志列表 */
  logs: LogEntry[]
  /** 查询来源：file 或 memory */
  source: string
  /** 当前查询文件名 */
  fileName: string
  /** 当前查询文件路径 */
  filePath: string
  /** 匹配的总条数 */
  total: number
  /** 当前页码 */
  page: number
  /** 每页条数 */
  pageSize: number
  /** 是否有下一页 */
  hasMore: boolean
  /** 统计信息 */
  stats: LogStats
}

// ============================================================================
// 更新生命周期类型
// ============================================================================

/** 更新生命周期状态（对应 Go 后端状态机） */
export type UpdateLifecycleStatus =
  | 'idle'              // 空闲，未检查
  | 'update_available'  // 发现可更新版本
  | 'downloading'       // 正在下载安装包
  | 'verifying'         // 正在校验 SHA256
  | 'verified'          // 校验通过
  | 'pending_install'   // 等待自动安装
  | 'installing'        // 正在安装
  | 'install_started'   // 安装器已启动
  | 'no_update'         // 当前已是最新版本
  | 'skipped'           // 已跳过
  | 'error'             // 出错

/** 更新状态快照 */
export type UpdateStatus = {
  /** 当前状态 */
  status: UpdateLifecycleStatus | string
  /** 状态消息 */
  message: string
  /** 目标版本号 */
  version?: string
  /** 安装包文件名 */
  assetName?: string
  /** 本地缓存路径 */
  filePath?: string
  /** 已下载字节数 */
  downloadedBytes?: number
  /** 总字节数 */
  totalBytes?: number
  /** 下载进度百分比（0-100） */
  progressPercent?: number
  /** SHA256 校验值 */
  sha256?: string
  /** 是否已通过校验 */
  verified: boolean
  /** 错误原因（内部错误码） */
  errorReason?: string
  /** 更新源 */
  source?: UpdateSource
  /** 状态更新时间 */
  updatedAt: string
}

// ============================================================================
// Wails 服务绑定类型
// ============================================================================

/**
 * Go 后端 API 绑定类型
 * 对应 app/api.go 中注册的方法
 */
export type ServiceBinding = {
  /** 获取应用信息 */
  GetAppInfo?: () => Promise<AppInfo>
  /** 获取运行环境信息 */
  GetEnvironmentInfo?: () => Promise<EnvironmentInfo>
  /** 获取当前授权状态 */
  GetLicenseStatus?: () => Promise<LicenseStatus>
  /** 激活授权码 */
  ActivateLicense?: (licenseKey: string) => Promise<LicenseStatus>
  /** 获取当前设置 */
  GetSettings?: () => Promise<Settings>
  /** 保存设置 */
  SaveSettings?: (settings: Settings) => Promise<Settings>
  /** 获取显示偏好 */
  GetDisplayPreferences?: () => Promise<DisplayPreferences>
  /** 保存显示偏好 */
  SaveDisplayPreferences?: (preferences: DisplayPreferences) => Promise<DisplayPreferences>
  /** 检查更新 */
  CheckUpdate?: () => Promise<UpdateCheckResult>
  /** 获取当前更新状态 */
  GetUpdateStatus?: () => Promise<UpdateStatus>
  /** 下载最新更新 */
  DownloadUpdate?: () => Promise<UpdateStatus>
  /** 安装已下载的更新 */
  InstallDownloadedUpdate?: () => Promise<UpdateStatus>
  /** 将已校验更新安排到下次启动安装 */
  ScheduleDownloadedUpdateOnStartup?: () => Promise<UpdateStatus>
  /** 列出所有日志 */
  ListLogs?: () => Promise<LogEntry[]>
  /** 列出每日日志文件 */
  ListLogFiles?: () => Promise<LogFileInfo[]>
  /** 查询日志（支持分页和过滤） */
  QueryLogs?: (query: LogQuery) => Promise<LogResponse>
  /** 清空指定作用域的日志 */
  ClearLogs?: (scope: string) => Promise<boolean>
  /** 显示主窗口 */
  ShowMainWindow?: () => Promise<void>
  /** 退出应用 */
  QuitApp?: () => Promise<void>
}

// ============================================================================
// 内部工具函数
// ============================================================================

/**
 * 获取 Wails 绑定服务实例
 * 只使用 app facade 生成的绑定，避免直接依赖 internal runtime binding 路径。
 */
function service(): ServiceBinding {
  return AppBinding.API as unknown as ServiceBinding
}

// 显式 preview 下主动制造绑定缺失错误，让读接口走受控 fallback。
class WailsBindingUnavailableError extends Error {
  constructor(method: string) {
    super(`Wails 绑定不可用：${method}`)
    this.name = 'WailsBindingUnavailableError'
  }
}

function isExplicitPreview() {
  return import.meta.env.VITE_PREVIEW === 'true'
}

function isSettingsTraceEnabled() {
  return import.meta.env.VITE_SETTINGS_TRACE === 'true'
}

// 所有 Wails API 调用先经过这里；真实桌面模式下绑定缺失必须暴露为错误。
function binding<K extends keyof ServiceBinding>(method: K): NonNullable<ServiceBinding[K]> {
  if (isExplicitPreview()) {
    throw new WailsBindingUnavailableError(String(method))
  }
  const api = service()
  const value = api[method]
  if (typeof value !== 'function') {
    throw new WailsBindingUnavailableError(String(method))
  }
  return value as NonNullable<ServiceBinding[K]>
}

function shouldUsePreviewFallback(error: unknown) {
  return isExplicitPreview() && error instanceof WailsBindingUnavailableError
}

// 显示偏好在 dev 浏览器预览中允许落到 localStorage，便于不连接 Wails 时调试主题。
function shouldUseDisplayPreferencesPreviewStore(error: unknown) {
  if (shouldUsePreviewFallback(error)) return true
  if (!import.meta.env.DEV || !(error instanceof Error)) return false
  return /Wails|runtime|runtimeCallWithID|desktop|browser|service|backend|not available|unavailable/i.test(`${error.message}\n${error.stack ?? ''}`)
}

function traceFrontend(message: string, payload?: unknown) {
  if (!isSettingsTraceEnabled()) return
  if (payload === undefined) {
    console.info(`[settings-trace] ${message}`)
    return
  }
  console.info(`[settings-trace] ${message}`, payload)
}

/**
 * 预览模式降级处理
 * 只在 VITE_PREVIEW=true 且绑定不可用时返回 fallback；真实后端错误必须继续向上抛。
 *
 * @param factory - 生成 fallback 数据的工厂函数
 * @param error - 原始错误
 * @returns fallback 数据或抛出错误
 */
async function previewFallback<T>(factory: () => T, error: unknown): Promise<T> {
  if (!shouldUsePreviewFallback(error)) {
    throw error instanceof Error ? error : new Error('Wails 服务不可用。')
  }
  return factory()
}

function throwSaveError(label: string, error: unknown): never {
  if (isExplicitPreview() && error instanceof WailsBindingUnavailableError) {
    throw new Error(`${label}失败：Wails 服务不可用。`)
  }
  throw error instanceof Error ? error : new Error(`${label}失败。`)
}

// 显示偏好包含嵌套 profiles，preview store 读写前复制一层，避免共享默认对象引用。
function cloneDisplayPreferences(value: DisplayPreferences): DisplayPreferences {
  return {
    ...value,
    profiles: {
      shadcn: { ...(value.profiles?.shadcn ?? defaultDisplayPreferences.profiles.shadcn) },
      artistic: { ...(value.profiles?.artistic ?? defaultDisplayPreferences.profiles.artistic) },
    },
  }
}

function normalisePreviewDisplayPreferences(value: unknown): DisplayPreferences {
  const parsed = typeof value === 'object' && value !== null ? value as Partial<DisplayPreferences> : {}
  const profiles = typeof parsed.profiles === 'object' && parsed.profiles !== null ? parsed.profiles as Partial<DisplayProfiles> : {}
  const displayScheme = parsed.displayScheme === 'shadcn' ? 'shadcn' : defaultDisplayPreferences.displayScheme
  return {
    ...cloneDisplayPreferences(defaultDisplayPreferences),
    ...parsed,
    displayScheme,
    profiles: {
      shadcn: {
        ...defaultDisplayPreferences.profiles.shadcn,
        ...profiles.shadcn,
      },
      artistic: {
        ...defaultDisplayPreferences.profiles.artistic,
        ...profiles.artistic,
      },
    },
  }
}

// 仅供 preview/dev fallback 使用；真实运行时的来源是后端 SQLite KV。
function readPreviewDisplayPreferences(): DisplayPreferences {
  if (typeof window === 'undefined') return cloneDisplayPreferences(defaultDisplayPreferences)
  const raw = window.localStorage.getItem(previewDisplayPreferencesStorageKey)
  if (!raw) return cloneDisplayPreferences(defaultDisplayPreferences)
  try {
    return normalisePreviewDisplayPreferences(JSON.parse(raw))
  } catch {
    return cloneDisplayPreferences(defaultDisplayPreferences)
  }
}

// preview/dev 下模拟保存显示偏好，保证设置页刷新后仍能看到本地改动。
function savePreviewDisplayPreferences(preferences: DisplayPreferences): DisplayPreferences {
  const saved = cloneDisplayPreferences(preferences)
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(previewDisplayPreferencesStorageKey, JSON.stringify(saved))
  }
  return saved
}

// ============================================================================
// 应用信息 API
// ============================================================================

/** 获取应用信息（名称、版本、描述等） */
export async function getAppInfo(): Promise<AppInfo> {
  try {
    return await binding('GetAppInfo')()
  } catch (error) {
    return previewFallback(() => ({
      name: projectMetadata.appName,
      version: projectMetadata.defaultVersion,
      description: projectMetadata.description,
      repository: projectMetadata.repositoryUrl,
      startedAt: new Date().toISOString(),
    }), error)
  }
}

/** 获取运行环境信息（操作系统、架构、Go 版本等） */
export async function getEnvironmentInfo(): Promise<EnvironmentInfo> {
  try {
    return await binding('GetEnvironmentInfo')()
  } catch (error) {
    return previewFallback(() => ({
      os: navigator.platform || 'browser',
      arch: 'preview',
      goVersion: '未连接 Go 运行时',
      wailsVersion: '未连接 Wails 运行时',
      databasePath: '前端预览模式',
      databaseReady: false,
      databaseStatus: 'disabled',
      databaseMessage: '前端预览模式未连接 SQLite 配置库。',
      logFilePath: '前端预览模式',
      cachePath: '前端预览模式',
    }), error)
  }
}

/** 获取当前授权状态 */
export async function getLicenseStatus(): Promise<LicenseStatus> {
  try {
    return await binding('GetLicenseStatus')()
  } catch (error) {
    return previewFallback(() => ({ ...defaultLicenseStatus }), error)
  }
}

/** 激活授权码；预览模式下必须失败，避免误以为已写入授权。 */
export async function activateLicense(licenseKey: string): Promise<LicenseStatus> {
  try {
    return await binding('ActivateLicense')(licenseKey)
  } catch (error) {
    throwSaveError('激活授权', error)
  }
}

// ============================================================================
// 设置 API
// ============================================================================

/** 获取当前设置 */
export async function getSettings(): Promise<Settings> {
  traceFrontend('getSettings：前端开始调用后端绑定')
  try {
    const settings = await binding('GetSettings')()
    traceFrontend('getSettings：前端收到后端返回', settings)
    return settings
  } catch (error) {
    traceFrontend('getSettings：前端读取异常', error)
    return previewFallback(() => ({ ...defaultRuntimeSettings }), error)
  }
}

/** 保存设置 */
export async function saveSettings(settings: Settings): Promise<Settings> {
  traceFrontend('saveSettings：前端开始调用后端绑定', settings)
  try {
    const saved = await binding('SaveSettings')(settings)
    traceFrontend('saveSettings：前端收到后端返回', saved)
    return saved
  } catch (error) {
    traceFrontend('saveSettings：前端保存异常', error)
    throwSaveError('保存设置', error)
  }
}

/** 获取当前显示偏好 */
export async function getDisplayPreferences(): Promise<DisplayPreferences> {
  traceFrontend('getDisplayPreferences：前端开始调用后端绑定')
  try {
    const preferences = await binding('GetDisplayPreferences')()
    traceFrontend('getDisplayPreferences：前端收到后端返回', preferences)
    return preferences
  } catch (error) {
    traceFrontend('getDisplayPreferences：前端读取异常', error)
    if (shouldUseDisplayPreferencesPreviewStore(error)) {
      return readPreviewDisplayPreferences()
    }
    return previewFallback(readPreviewDisplayPreferences, error)
  }
}

/** 保存显示偏好 */
export async function saveDisplayPreferences(preferences: DisplayPreferences): Promise<DisplayPreferences> {
  traceFrontend('saveDisplayPreferences：前端开始调用后端绑定', preferences)
  try {
    const saved = await binding('SaveDisplayPreferences')(preferences)
    traceFrontend('saveDisplayPreferences：前端收到后端返回', saved)
    return saved
  } catch (error) {
    traceFrontend('saveDisplayPreferences：前端保存异常', error)
    if (shouldUseDisplayPreferencesPreviewStore(error)) {
      return savePreviewDisplayPreferences(preferences)
    }
    throwSaveError('保存显示偏好', error)
  }
}

// ============================================================================
// 更新管理 API
// ============================================================================

/** 检查更新 */
export async function checkUpdate(): Promise<UpdateCheckResult> {
  try {
    return await binding('CheckUpdate')()
  } catch (error) {
    return previewFallback(() => ({
      source: defaultRuntimeSettings.updateSource,
      status: 'ignored',
      currentVersion: projectMetadata.defaultVersion,
      requestUrl: `${projectMetadata.github.apiBase}/repos/${projectMetadata.github.owner}/${projectMetadata.github.repo}/releases?per_page=30`,
      skipReason: 'offline',
      errorReason: 'offline',
      checkedAt: new Date().toISOString(),
      message: '当前处于前端预览模式，已跳过真实更新检查。',
    }), error)
  }
}

/** 获取当前更新状态 */
export async function getUpdateStatus(): Promise<UpdateStatus> {
  try {
    return await binding('GetUpdateStatus')()
  } catch (error) {
    return previewFallback(() => ({
      status: 'idle',
      message: '前端预览模式下未连接更新服务。',
      verified: false,
      source: defaultRuntimeSettings.updateSource,
      updatedAt: new Date().toISOString(),
    }), error)
  }
}

/** 下载最新更新安装包 */
export async function downloadUpdate(): Promise<UpdateStatus> {
  try {
    return await binding('DownloadUpdate')()
  } catch (error) {
    return previewFallback(() => ({
      status: 'skipped',
      message: '前端预览模式下已跳过下载和校验。',
      verified: false,
      source: defaultRuntimeSettings.updateSource,
      errorReason: 'preview',
      updatedAt: new Date().toISOString(),
    }), error)
  }
}

/** 安装已下载的更新 */
export async function installDownloadedUpdate(): Promise<UpdateStatus> {
  try {
    return await binding('InstallDownloadedUpdate')()
  } catch (error) {
    return previewFallback(() => ({
      status: 'error',
      message: '前端预览模式下无法启动安装器。',
      verified: false,
      source: defaultRuntimeSettings.updateSource,
      errorReason: 'preview',
      updatedAt: new Date().toISOString(),
    }), error)
  }
}

/** 将已下载并校验通过的更新安排到下次启动安装 */
export async function scheduleDownloadedUpdateOnStartup(): Promise<UpdateStatus> {
  try {
    return await binding('ScheduleDownloadedUpdateOnStartup')()
  } catch (error) {
    return previewFallback(() => ({
      status: 'pending_install',
      message: '前端预览模式下已模拟安排下次启动更新。',
      verified: true,
      source: defaultRuntimeSettings.updateSource,
      updatedAt: new Date().toISOString(),
    }), error)
  }
}

// ============================================================================
// 日志 API
// ============================================================================

/** 获取所有日志 */
export async function listLogs(): Promise<LogEntry[]> {
  try {
    return await binding('ListLogs')()
  } catch (error) {
    return previewFallback(() => [], error)
  }
}

/** 列出每日日志文件 */
export async function listLogFiles(): Promise<LogFileInfo[]> {
  try {
    return await binding('ListLogFiles')()
  } catch (error) {
    return previewFallback(() => [], error)
  }
}

/** 分页查询日志 */
export async function queryLogs(query: LogQuery): Promise<LogResponse> {
  try {
    return await binding('QueryLogs')(query)
  } catch (error) {
    return previewFallback(
      () => ({
        logs: [],
        source: 'memory',
        fileName: query.fileName,
        filePath: '',
        total: 0,
        page: query.page,
        pageSize: query.pageSize,
        hasMore: false,
        stats: { total: 0, debug: 0, info: 0, warning: 0, error: 0 },
      }),
      error,
    )
  }
}

/** 清空指定作用域的日志 */
export async function clearLogs(scope: string): Promise<boolean> {
  try {
    return await binding('ClearLogs')(scope)
  } catch (error) {
    return previewFallback(() => true, error)
  }
}

// ============================================================================
// 应用控制 API
// ============================================================================

/** 退出应用 */
export async function quitApp() {
  try {
    await binding('QuitApp')()
  } catch (error) {
    await previewFallback(() => undefined, error)
  }
}

/**
 * 打开外部链接
 * 优先使用 Wails Browser API，预览模式下降级到 window.open
 */
export async function openExternalURL(url: string) {
  try {
    await Browser.OpenURL(url)
  } catch (error) {
    if (isExplicitPreview()) {
      window.open(url, '_blank', 'noopener,noreferrer')
      return
    }
    throw new Error('打开外部链接失败。')
  }
}
