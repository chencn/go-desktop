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
  type DisplayPreferences,
  type LicenseStatus,
  type LogQuery,
  type Settings,
  type UpdateStatus,
} from '../api/wails'
import { exportDisplayPreferences, hydrateDisplayPreferences } from '../app/display'
import { appReducer, defaultLogQuery, defaultStartupApiStatuses, initialAppState, toMessage, type AppAction, type AppState, type StartupApiKey } from '../app/state'

// runtimeUnsubscribers 保存 Wails 运行时事件取消订阅函数。
let runtimeUnsubscribers: Array<() => void> = []

function updateStatusFromEventData(data: UpdateStatus | UpdateStatus[]) {
  return Array.isArray(data) ? data[0] : data
}

function isUpdateTerminalStatus(status?: string) {
  return !['downloading', 'verifying', 'installing'].includes(String(status ?? ''))
}

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

function normaliseLicenseKey(value: string) {
  return value.split(/\s+/).join('')
}

// useAppStore 保存 Pinia store 实例，集中访问应用共享状态和动作。
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
        return
      }

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
      }
    },

    subscribeRuntimeUpdates() {
      if (runtimeUnsubscribers.length > 0) return
      try {
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
        // Browser preview mode has no Wails event bridge.
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
