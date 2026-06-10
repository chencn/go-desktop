// 文件职责：集中管理前端应用信息、设置、日志、更新状态和运行时事件。
// 说明：更新只保存当前检查结果和当前状态，不维护历史列表。

import { defineStore } from 'pinia'
import { Events } from '@wailsio/runtime'
import {
  checkUpdate as checkUpdateApi,
  clearLogs,
  downloadUpdate,
  activateLicense,
  getAppInfo,
  defaultDisplayPreferences,
  defaultRuntimeSettings,
  getDisplayPreferences,
  getEnvironmentInfo,
  getLicenseStatus,
  getSettings,
  getUpdateStatus,
  installDownloadedUpdate as installDownloadedUpdateApi,
  listLogFiles,
  queryLogs,
  scheduleDownloadedUpdateOnStartup as scheduleDownloadedUpdateOnStartupApi,
  saveDisplayPreferences,
  saveSettings,
  showMainWindow,
  type DisplayPreferences,
  type LicenseStatus,
  type LogQuery,
  type Settings,
  type UpdateStatus,
} from '../api/wails'
import { exportDisplayPreferences, hydrateDisplayPreferences } from '../app/display'
import { appReducer, defaultLogQuery, defaultStartupApiStatuses, initialAppState, toMessage, type AppAction, type AppState, type StartupApiKey } from '../app/state'

// Wails 事件订阅跨组件生命周期保存，避免 App.vue 重挂载时重复监听更新状态。
let runtimeUnsubscribers: Array<() => void> = []

// Wails Events.On 的 data 在不同运行时包装下可能是值或单元素数组，这里统一成状态快照。
function updateStatusFromEventData(data: UpdateStatus | UpdateStatus[]) {
  return Array.isArray(data) ? data[0] : data
}

// 这些状态不再需要维持检查/下载按钮的 busy 态。
function isUpdateTerminalStatus(status?: string) {
  return !['downloading', 'verifying', 'installing'].includes(String(status ?? ''))
}

// 授权状态读取失败时按“需要授权且未授权”处理，防止绕过授权页进入主界面。
function failedLicenseStatus(message = '授权状态读取失败'): LicenseStatus {
  return {
    enabled: true,
    required: true,
    authorized: false,
    deviceCode: '',
    message,
    lastError: message,
  }
}

// 授权码允许用户复制时带空格或换行，提交给后端前统一压缩。
function normaliseLicenseKey(value: string) {
  return value.split(/\s+/).join('')
}

// 应用唯一 Pinia store：异步 API、运行时事件和纯 reducer 状态都从这里汇合。
export const useAppStore = defineStore('app', {
  state: (): AppState => ({ ...initialAppState, startupApiStatuses: defaultStartupApiStatuses() }),
  getters: {
    isOfflineSkipped: (state) => state.latestUpdateCheck?.status === 'ignored' && state.latestUpdateCheck.skipReason === 'offline',
  },
  actions: {
    applyAction(action: AppAction) {
      this.$patch(appReducer(this.$state as AppState, action))
    },

    async initialise() {
      this.applyAction({ type: 'loadingSet', payload: true })
      this.applyAction({ type: 'errorSet', payload: '' })
      const errors: string[] = []
      let settings: Settings = { ...defaultRuntimeSettings }
      let displayPreferences: DisplayPreferences = { ...defaultDisplayPreferences }

      // 启动读取统一记录耗时和错误；非关键 API 可选择不冒泡到全局 errorMessage。
      const readStartupApi = async <T>(
        key: StartupApiKey,
        label: string,
        reader: () => Promise<T>,
        applyValue: (value: T) => void,
        applyFallback?: () => void,
        surfaceError = true,
      ) => {
        const startedAt = Date.now()
        this.applyAction({ type: 'startupApiStatusSet', payload: { key, state: 'loading', message: `${label}读取中` } })
        try {
          const value = await reader()
          applyValue(value)
          this.applyAction({
            type: 'startupApiStatusSet',
            payload: { key, state: 'ok', message: `${label}已返回，耗时 ${Date.now() - startedAt}ms` },
          })
        } catch (error) {
          const message = toMessage(error)
          applyFallback?.()
          this.applyAction({ type: 'startupApiStatusSet', payload: { key, state: 'error', message } })
          if (surfaceError) {
            errors.push(`${label}失败：${message}`)
          }
        }
      }

      // 授权先于其他启动 API；未授权时只渲染 LicensePage，避免无意义地读取业务数据。
      let initialLicenseStatus: LicenseStatus | undefined
      await readStartupApi('licenseStatus', 'GetLicenseStatus', getLicenseStatus, (licenseStatus) => {
        initialLicenseStatus = licenseStatus
        this.applyAction({ type: 'licenseStatusApplied', payload: licenseStatus })
      }, () => {
        initialLicenseStatus = failedLicenseStatus()
        this.applyAction({ type: 'licenseStatusApplied', payload: initialLicenseStatus })
      })
      const licenseStatus = initialLicenseStatus ?? failedLicenseStatus()
      if (licenseStatus.required && !licenseStatus.authorized) {
        this.applyAction({ type: 'loadingSet', payload: false })
        // 授权页也需要显示主窗口，否则用户无法看到授权页面。
        await showMainWindow()
        return
      }

      // 其余启动 API 可以并发读取；单项失败只更新对应 startupApiStatuses。
      await Promise.allSettled([
        readStartupApi('settings', 'GetSettings', getSettings, (value) => {
          settings = value
          this.applyAction({ type: 'settingsApplied', payload: settings })
        }, () => {
          this.applyAction({ type: 'settingsApplied', payload: settings })
        }),
        readStartupApi('displayPreferences', 'GetDisplayPreferences', getDisplayPreferences, (value) => {
          displayPreferences = value
          hydrateDisplayPreferences(displayPreferences)
          this.applyAction({ type: 'displayPreferencesApplied', payload: displayPreferences })
        }, () => {
          hydrateDisplayPreferences(displayPreferences)
          this.applyAction({ type: 'displayPreferencesApplied', payload: displayPreferences })
        }),
        readStartupApi('appInfo', 'GetAppInfo', getAppInfo, (appInfo) => {
          this.applyAction({ type: 'appInfoApplied', payload: appInfo })
        }),
        readStartupApi('environmentInfo', 'GetEnvironmentInfo', getEnvironmentInfo, (environmentInfo) => {
          this.applyAction({ type: 'environmentInfoApplied', payload: environmentInfo })
        }),
        readStartupApi('updateStatus', 'GetUpdateStatus', getUpdateStatus, (updateStatus) => {
          this.applyAction({ type: 'updateStatusApplied', payload: updateStatus })
        }, undefined, false),
        readStartupApi('logFiles', 'ListLogFiles', listLogFiles, (logFiles) => {
          this.applyAction({ type: 'logFilesApplied', payload: logFiles })
        }, undefined, false),
        readStartupApi('logs', 'QueryLogs', () => queryLogs(defaultLogQuery()), (logResponse) => {
          this.applyAction({ type: 'logsApplied', payload: logResponse })
        }, undefined, false),
      ])

      try {
        if (errors.length > 0) {
          this.applyAction({ type: 'errorSet', payload: errors[0] })
        }
      } finally {
        this.applyAction({ type: 'loadingSet', payload: false })
        // 前端数据加载完成，通知后端显示主窗口并关闭 splash 加载窗口。
        await showMainWindow()
      }
    },

    subscribeRuntimeUpdates() {
      if (runtimeUnsubscribers.length > 0) return
      try {
        // 后端下载、校验、安装阶段会推送 update:status:changed，前端同步刷新状态和日志。
        runtimeUnsubscribers.push(Events.On('update:status:changed', async (event: { data: UpdateStatus }) => {
          const updateStatus = updateStatusFromEventData(event.data)
          this.applyAction({ type: 'updateStatusApplied', payload: updateStatus })
          if (isUpdateTerminalStatus(updateStatus?.status)) {
            this.applyAction({ type: 'checkingSet', payload: false })
            this.applyAction({ type: 'downloadingSet', payload: false })
          }
          try {
            await this.refreshLogs()
          } catch (error) {
            this.applyAction({ type: 'errorSet', payload: toMessage(error) })
          }
        }) as unknown as () => void)
      } catch {
        // 浏览器预览没有 Wails event bridge，订阅失败不影响静态预览。
      }
    },

    unsubscribeRuntimeUpdates() {
      for (const unsubscribe of runtimeUnsubscribers) {
        unsubscribe()
      }
      runtimeUnsubscribers = []
    },

    async checkUpdate() {
      this.applyAction({ type: 'checkingSet', payload: true })
      this.applyAction({ type: 'errorSet', payload: '' })
      try {
        const result = await checkUpdateApi()
        this.applyAction({ type: 'updateCheckApplied', payload: result })
        this.applyAction({ type: 'updateStatusApplied', payload: await getUpdateStatus() })
        await this.refreshLogs()
      } catch (error) {
        this.applyAction({ type: 'errorSet', payload: toMessage(error) })
      } finally {
        this.applyAction({ type: 'checkingSet', payload: false })
      }
    },

    async refreshUpdateStatus() {
      this.applyAction({ type: 'updateStatusApplied', payload: await getUpdateStatus() })
    },

    async loadLicenseStatus() {
      this.applyAction({ type: 'licenseLoadingSet', payload: true })
      this.applyAction({ type: 'licenseErrorSet', payload: '' })
      try {
        const status = await getLicenseStatus()
        this.applyAction({ type: 'licenseStatusApplied', payload: status })
        return status
      } catch (error) {
        const message = toMessage(error)
        this.applyAction({ type: 'licenseErrorSet', payload: message })
        throw error
      } finally {
        this.applyAction({ type: 'licenseLoadingSet', payload: false })
      }
    },

    async activateLicenseKey(licenseKey: string) {
      this.applyAction({ type: 'licenseLoadingSet', payload: true })
      this.applyAction({ type: 'licenseErrorSet', payload: '' })
      try {
        const status = await activateLicense(normaliseLicenseKey(licenseKey))
        this.applyAction({ type: 'licenseStatusApplied', payload: status })
        if (status.authorized) {
          void this.initialise()
        }
        return status
      } catch (error) {
        const message = toMessage(error)
        this.applyAction({ type: 'licenseErrorSet', payload: message })
        throw error
      } finally {
        this.applyAction({ type: 'licenseLoadingSet', payload: false })
      }
    },

    async downloadLatestUpdate() {
      this.applyAction({ type: 'downloadingSet', payload: true })
      this.applyAction({ type: 'errorSet', payload: '' })
      try {
        this.applyAction({ type: 'updateStatusApplied', payload: await downloadUpdate() })
      } catch (error) {
        this.applyAction({ type: 'errorSet', payload: toMessage(error) })
      } finally {
        this.applyAction({ type: 'downloadingSet', payload: false })
      }
    },

    async installDownloadedUpdate() {
      this.applyAction({ type: 'downloadingSet', payload: true })
      this.applyAction({ type: 'errorSet', payload: '' })
      try {
        this.applyAction({ type: 'updateStatusApplied', payload: await installDownloadedUpdateApi() })
      } catch (error) {
        this.applyAction({ type: 'errorSet', payload: toMessage(error) })
      } finally {
        this.applyAction({ type: 'downloadingSet', payload: false })
      }
    },

    async scheduleDownloadedUpdateOnStartup() {
      this.applyAction({ type: 'errorSet', payload: '' })
      try {
        this.applyAction({ type: 'updateStatusApplied', payload: await scheduleDownloadedUpdateOnStartupApi() })
      } catch (error) {
        this.applyAction({ type: 'errorSet', payload: toMessage(error) })
      }
    },

    async persistSettings(settings: Settings) {
      const saved = await saveSettings(settings)
      this.applyAction({ type: 'settingsApplied', payload: saved })
      this.applyAction({ type: 'errorSet', payload: '' })
      return saved
    },

    async persistDisplayPreferences(preferences: DisplayPreferences = exportDisplayPreferences()) {
      const saved = await saveDisplayPreferences(preferences)
      hydrateDisplayPreferences(saved)
      this.applyAction({ type: 'displayPreferencesApplied', payload: saved })
      this.applyAction({ type: 'errorSet', payload: '' })
      return saved
    },

    async refreshLogs(query: Partial<LogQuery> = {}) {
      const nextQuery = {
        ...defaultLogQuery(),
        ...this.currentLogQuery,
        fileName: this.selectedLogFileName || this.currentLogQuery.fileName,
        ...query,
      }
      this.applyAction({ type: 'logQueryApplied', payload: nextQuery })
      const response = await queryLogs(nextQuery)
      this.applyAction({ type: 'logsApplied', payload: response })
    },

    async refreshLogFiles() {
      this.applyAction({ type: 'logFilesApplied', payload: await listLogFiles() })
    },

    async clearLogScope(scope: string, query: Partial<LogQuery> = {}) {
      const cleared = await clearLogs(scope)
      if (cleared) {
        await this.refreshLogs(query)
      }
    },
  },
})
