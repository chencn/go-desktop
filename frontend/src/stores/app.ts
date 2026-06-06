// 文件职责：集中管理前端应用信息、设置、日志、更新状态和运行时事件。
// 说明：更新只保存当前检查结果和当前状态，不维护历史列表。

import { defineStore } from 'pinia'
import { Events } from '@wailsio/runtime'
import {
  checkUpdate as checkUpdateApi,
  clearLogs,
  downloadUpdate,
  getAppInfo,
  defaultDisplayPreferences,
  defaultRuntimeSettings,
  getDisplayPreferences,
  getEnvironmentInfo,
  getSettings,
  getUpdateStatus,
  installDownloadedUpdate as installDownloadedUpdateApi,
  listLogFiles,
  queryLogs,
  scheduleDownloadedUpdateOnStartup as scheduleDownloadedUpdateOnStartupApi,
  saveDisplayPreferences,
  saveSettings,
  type DisplayPreferences,
  type LogQuery,
  type Settings,
  type UpdateStatus,
} from '../api/wails'
import { exportDisplayPreferences, hydrateDisplayPreferences } from '../app/display'
import { appReducer, defaultLogQuery, defaultStartupApiStatuses, initialAppState, toMessage, type AppAction, type AppState, type StartupApiKey } from '../app/state'

// runtimeUnsubscribers 保存 Wails 运行时事件取消订阅函数。
let runtimeUnsubscribers: Array<() => void> = []

function traceSettingsStore(message: string, payload?: unknown) {
  if (payload === undefined) {
    console.info(`[settings-trace] ${message}`)
    return
  }
  console.info(`[settings-trace] ${message}`, payload)
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
      traceSettingsStore('store.initialise：开始', {
        hasSettings: Boolean(this.settings),
        hasDisplayPreferences: Boolean(this.displayPreferences),
      })
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
        traceSettingsStore(`store.initialise：准备读取 ${label}`)
        try {
          const value = await reader()
          applyValue(value)
          this.applyAction({
            type: 'startupApiStatusSet',
            payload: { key, state: 'ok', message: `${label}已返回，耗时 ${Date.now() - startedAt}ms` },
          })
          traceSettingsStore(`store.initialise：读取 ${label} 完成`, value)
        } catch (error) {
          const message = toMessage(error)
          applyFallback?.()
          this.applyAction({ type: 'startupApiStatusSet', payload: { key, state: 'error', message } })
          if (surfaceError) {
            errors.push(`${label}失败：${message}`)
          }
          traceSettingsStore(`store.initialise：读取 ${label} 异常`, error)
        }
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
        traceSettingsStore('store.initialise：结束', {
          hasSettings: Boolean(this.settings),
          hasDisplayPreferences: Boolean(this.displayPreferences),
          hasAppInfo: Boolean(this.appInfo),
          hasEnvironmentInfo: Boolean(this.environmentInfo),
          loading: this.loading,
          errorMessage: this.errorMessage,
          startupApiStatuses: this.startupApiStatuses,
        })
      }
    },

    subscribeRuntimeUpdates() {
      if (runtimeUnsubscribers.length > 0) return
      try {
        runtimeUnsubscribers.push(Events.On('update:status:changed', async (event: { data: UpdateStatus }) => {
          this.applyAction({ type: 'updateStatusApplied', payload: event.data })
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
