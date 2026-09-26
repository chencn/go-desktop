<!--
  文件职责：渲染设置表单并把用户输入提交给应用状态 store。
  业务设置保存到后端 Settings；显示偏好（亮暗模式、全局尺寸）保存到独立 DisplayPreferences。
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Window } from '@wailsio/runtime'
import {
  Calendar,
  Clock,
  DollarSign,
  Download,
  EyeOff,
  Globe,
  Maximize2,
  Monitor,
  Palette,
  Pin,
  Power,
  Rocket,
  SquareArrowDown,
  Star,
  Sun,
  Terminal,
  Wrench,
} from '@lucide/vue'
import { useDisplayPreferences, type DisplaySize, type ThemeMode } from '@/app/display'
import { glassIntensityMax, glassIntensityMin, useGlassPreferences, type GlassStyle } from '@/app/glass'
import { useAppStore } from '@/stores/app'
import { defaultRuntimeSettings, type LogLevel, type Settings, type UpdateSource } from '@/api/wails'
import AlertDialog from '../shared/AlertDialog.vue'

const appStore = useAppStore()
// display 是全局显示偏好的响应式 facade，实际持久化仍通过 appStore.persistDisplayPreferences。
const display = useDisplayPreferences()
// glass 是液态玻璃材质偏好（极光背景 / 折射风格 / 光强），与主题、尺寸共用后端 display.preferences.v3。
const glass = useGlassPreferences()
const settingsReady = computed(() => Boolean(appStore.settings))
const displayReady = computed(() => Boolean(appStore.displayPreferences))
// draft 是业务设置表单草稿；保存成功后以后端返回值为准重新覆盖。
const draft = ref<Settings>({ ...defaultRuntimeSettings })
// settingsSaveDelayMs 控制设置保存的短延迟，合并连续点击产生的多次后端写入。
const settingsSaveDelayMs = 180
// saveRevision 标记最新业务设置保存请求，旧 revision 进入队列后会被跳过。
let saveRevision = 0
// saveQueue 串行化 SaveSettings，避免快速点击导致后端写入乱序。
let saveQueue = Promise.resolve()
// saveTimer 保存业务设置防抖计时器；浏览器 window.setTimeout 返回 number。
let saveTimer: number | undefined
// displaySaveRevision 标记最新显示偏好保存请求，配合队列丢弃过期写入。
let displaySaveRevision = 0
// displaySaveQueue 串行化 SaveDisplayPreferences，保证落库顺序和 UI 最终状态一致。
let displaySaveQueue = Promise.resolve()
// displaySaveTimer 保存显示偏好防抖计时器；浏览器 window.setTimeout 返回 number。
let displaySaveTimer: number | undefined

// updateIntervalOptions 必须和后端允许的检查间隔保持一致；非法值会回退默认值。
const updateIntervalOptions = [1, 3, 6, 12]
const updateSourceOptions: Array<[UpdateSource, string]> = [
  ['github', 'GitHub Release (官方公网发布)'],
  ['local', '本地静态服务 (企业自建内网)']
]
const logRetentionOptions: Array<[number, string]> = [
  [7, '7 天'],
  [30, '30 天'],
  [90, '90 天'],
  [365, '365 天'],
  [-1, '永不清理']
]
const logLevelOptions: Array<[LogLevel, string]> = [
  ['debug', 'debug (调试记录)'],
  ['info', 'info (默认信息)'],
  ['warning', 'warning (警告级别)'],
  ['error', 'error (仅记录错误)']
]
// themeOptions/sizeOptions 是外观偏好当前仅有的两项；主题色方案后续再开放。
const themeOptions: Array<[ThemeMode, string]> = [['light', '亮色'], ['dark', '暗色']]
const sizeOptions: Array<[DisplaySize, string]> = [['large', '大'], ['default', '中'], ['small', '小']]
// 液态玻璃折射风格三档，顺序与设计稿 #lgStyleSegmentGroup 一致。
const glassStyleOptions: Array<[GlassStyle, string]> = [
  ['fresnel', '自然高透'],
  ['frosted', '晶莹磨砂'],
  ['sheen', '双层反射'],
]

// 后端设置变化时重建草稿；归一化保证旧配置或异常值不会进入控件。
watch(() => appStore.settings, (settings) => {
  if (settings) {
    draft.value = normaliseSettingsDraft({ ...defaultRuntimeSettings, ...settings })
  }
}, { immediate: true })

// persistSettingsPatch 合并单项变更并防抖调用 SaveSettings；失败时回滚到 store 中最后成功的设置。
function persistSettingsPatch(patch: Partial<Settings>) {
  const base = appStore.settings
  if (!base) {
    appStore.applyAction({ type: 'errorSet', payload: '设置尚未加载，暂不能保存。' })
    return saveQueue
  }
  const revision = ++saveRevision
  const next = normaliseSettingsDraft({ ...base, ...draft.value, ...patch })
  draft.value = next

  if (saveTimer) {
    window.clearTimeout(saveTimer)
  }
  saveTimer = window.setTimeout(() => {
    saveQueue = saveQueue
      .catch(() => undefined)
      .then(async () => {
        if (revision !== saveRevision) {
          return
        }
        try {
          const saved = await appStore.persistSettings(next)
          if (revision === saveRevision) {
            draft.value = { ...saved }
          }
        } catch (error) {
          if (revision === saveRevision) {
            if (appStore.settings) {
              draft.value = normaliseSettingsDraft({ ...defaultRuntimeSettings, ...appStore.settings })
            }
            appStore.applyAction({ type: 'errorSet', payload: error instanceof Error ? error.message : '设置保存失败' })
          }
        }
      })
  }, settingsSaveDelayMs)

  return saveQueue
}

// normaliseSettingsDraft 是前端提交前的最后一道约束；后端仍会再次校验并返回最终值。
function normaliseSettingsDraft(settings: Settings): Settings {
  return {
    updateSource: normaliseUpdateSource(settings.updateSource),
    githubProxyBase: settings.githubProxyBase.trim(),
    updateCheckIntervalHours: normaliseUpdateCheckIntervalHours(settings.updateCheckIntervalHours),
    minimizeToTray: Boolean(settings.minimizeToTray),
    alwaysOnTop: Boolean(settings.alwaysOnTop),
    logRetentionDays: Number(settings.logRetentionDays) === -1 ? -1 : Math.max(1, Number(settings.logRetentionDays) || defaultRuntimeSettings.logRetentionDays),
    logLevel: normaliseLogLevel(settings.logLevel),
    autoLaunch: Boolean(settings.autoLaunch),
    createDesktopShortcut: Boolean(settings.createDesktopShortcut),
    launchHiddenToTray: Boolean(settings.launchHiddenToTray),
  }
}

function normaliseUpdateSource(value: string): UpdateSource {
  return updateSourceOptions.some(([source]) => source === value) ? value as UpdateSource : defaultRuntimeSettings.updateSource
}

// normaliseUpdateCheckIntervalHours 只接受 UI 暴露的枚举值，防止手工改 DOM 写入异常间隔。
function normaliseUpdateCheckIntervalHours(value: number) {
  const interval = Number(value)
  return updateIntervalOptions.includes(interval) ? interval : defaultRuntimeSettings.updateCheckIntervalHours
}

function normaliseLogLevel(value: string): LogLevel {
  return logLevelOptions.some(([level]) => level === value) ? value as LogLevel : defaultRuntimeSettings.logLevel
}

function ensureDisplayReady() {
  if (displayReady.value) {
    return true
  }
  appStore.applyAction({ type: 'errorSet', payload: '显示偏好尚未加载，暂不能保存。' })
  return false
}

// asThemeMode 切换全局亮暗模式；写 display facade 后防抖保存完整偏好快照。
function asThemeMode(value: string) {
  if (!ensureDisplayReady()) return
  display.setThemeMode(value as ThemeMode)
  persistDisplayPreferences()
}

// asSize 切换 Element Plus 全局组件尺寸（el-config-provider size）。
function asSize(value: string) {
  if (!ensureDisplayReady()) return
  display.setSize(value as DisplaySize)
  persistDisplayPreferences()
}

// asGlassStyle 切换折射风格；app/glass.ts 的 watch 负责写 <html data-lg-style>，偏好与主题同一链路落后端。
function asGlassStyle(value: string) {
  if (!ensureDisplayReady()) return
  glass.setStyle(value as GlassStyle)
  persistDisplayPreferences()
}

// asGlassIntensity 读取原生 range 的数值。空值时 valueAsNumber 为 NaN，由 setIntensity 直接丢弃。
function asGlassIntensity(event: Event) {
  if (!ensureDisplayReady()) return
  glass.setIntensity((event.target as HTMLInputElement).valueAsNumber)
  persistDisplayPreferences()
}

// asGlassBackdrop 切换极光折射流光背景。
function asGlassBackdrop(value: boolean) {
  if (!ensureDisplayReady()) return
  glass.setBackdrop(value)
  persistDisplayPreferences()
}

// resetDialogOpen 控制「恢复默认外观」二次确认弹窗（设计稿 .apple-alert-dialog）。
const resetDialogOpen = ref(false)

// confirmResetDisplayPreferences 打开确认弹窗，实际恢复交给 runResetDisplayPreferences。
function confirmResetDisplayPreferences() {
  if (!ensureDisplayReady()) return
  resetDialogOpen.value = true
}

// runResetDisplayPreferences 恢复亮暗模式、全局尺寸与液态玻璃光学的默认值并立即落盘（display facade 一次覆盖五轴）。
function runResetDisplayPreferences() {
  resetDialogOpen.value = false
  if (!ensureDisplayReady()) return
  display.resetDisplayPreferences()
  persistDisplayPreferences({ immediate: true })
}

// persistDisplayPreferences 保存显示偏好快照；快照必须在真正写库那一刻才导出，
// 否则防抖等待期间被顶栏极光开关改掉的轴会被这份旧快照覆盖回去（整键覆写会一次丢一整轴）。
function persistDisplayPreferences(options: { immediate?: boolean } = {}) {
  if (!ensureDisplayReady()) {
    return displaySaveQueue
  }
  const revision = ++displaySaveRevision

  if (displaySaveTimer) {
    window.clearTimeout(displaySaveTimer)
    displaySaveTimer = undefined
  }
  const runSave = () => {
    displaySaveQueue = displaySaveQueue
      .catch(() => undefined)
      .then(async () => {
        if (revision !== displaySaveRevision) {
          return
        }
        try {
          await appStore.persistDisplayPreferences()
        } catch (error) {
          if (revision === displaySaveRevision) {
            appStore.applyAction({ type: 'errorSet', payload: error instanceof Error ? error.message : '显示偏好保存失败' })
          }
        }
      })
  }

  if (options.immediate) {
    runSave()
    return displaySaveQueue
  }

  displaySaveTimer = window.setTimeout(runSave, settingsSaveDelayMs)

  return displaySaveQueue
}
// applyAlwaysOnTopToWindow 把「窗口置顶」设置真正落到 Wails 窗口；浏览器预览没有窗口运行时会静默跳过。
async function applyAlwaysOnTopToWindow(enabled: boolean) {
  try {
    await Window.SetAlwaysOnTop(enabled)
  } catch {
    // 预览模式或平台不支持时，设置仍然会持久化，只丢失即时效果。
  }
}

watch(() => draft.value.alwaysOnTop, (enabled) => {
  void applyAlwaysOnTopToWindow(enabled)
})
</script>

<template>
  <div class="settings-stack">
    <!-- 业务设置 -->
    <div class="settings-group-card">
      <div class="settings-group-header">
        <div>
          <h3 class="settings-group-title">
            <span class="sq-icon-badge size-sm blue" aria-hidden="true"><Wrench :size="13" :stroke-width="2.2" /></span>
            应用基础与业务设置
          </h3>
          <p class="settings-group-desc">控制窗口托盘行为、开机自启策略、自动更新周期及每日日志清理策略。</p>
        </div>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md blue" aria-hidden="true"><SquareArrowDown :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">关闭到系统托盘</span>
            <span class="settings-row-helper">点击关闭按钮时隐藏窗口至后台托盘，最小化仍进入任务栏。</span>
          </div>
        </div>
        <el-switch class="apple-switch" :model-value="draft.minimizeToTray" :disabled="!settingsReady" aria-label="关闭到系统托盘" @update:model-value="persistSettingsPatch({ minimizeToTray: Boolean($event) })" />
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md orange" aria-hidden="true"><Pin :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">窗口置顶</span>
            <span class="settings-row-helper">窗口显示时保持在其他应用窗口上方，不影响托盘策略。</span>
          </div>
        </div>
        <el-switch class="apple-switch" :model-value="draft.alwaysOnTop" :disabled="!settingsReady" aria-label="窗口置顶" @update:model-value="persistSettingsPatch({ alwaysOnTop: Boolean($event) })" />
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md green" aria-hidden="true"><Power :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">开机自启</span>
            <span class="settings-row-helper">系统完成引导并登录 Windows 桌面后自动启动该工具。</span>
          </div>
        </div>
        <el-switch class="apple-switch" :model-value="draft.autoLaunch" :disabled="!settingsReady" aria-label="开机自启" @update:model-value="persistSettingsPatch({ autoLaunch: Boolean($event) })" />
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md teal" aria-hidden="true"><EyeOff :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">自启时隐藏到系统托盘</span>
            <span class="settings-row-helper">开机启动时静默最小化到托盘，不弹出前台主窗口。</span>
          </div>
        </div>
        <el-switch class="apple-switch" :model-value="draft.launchHiddenToTray" :disabled="!settingsReady" aria-label="开机自启时隐藏到托盘" @update:model-value="persistSettingsPatch({ launchHiddenToTray: Boolean($event) })" />
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md purple" aria-hidden="true"><Monitor :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">创建桌面快捷图标</span>
            <span class="settings-row-helper">在当前 Windows 桌面生成直接启动本软件的快捷方式。</span>
          </div>
        </div>
        <el-switch class="apple-switch" :model-value="draft.createDesktopShortcut" :disabled="!settingsReady" aria-label="创建桌面快捷图标" @update:model-value="persistSettingsPatch({ createDesktopShortcut: Boolean($event) })" />
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md indigo" aria-hidden="true"><Download :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">系统更新源</span>
            <span class="settings-row-helper">选择新版本发布来源及分发节点。</span>
          </div>
        </div>
        <el-select class="apple-select" popper-class="ios-select-popover" placement="bottom-end" :model-value="draft.updateSource" :disabled="!settingsReady" aria-label="更新源" @update:model-value="persistSettingsPatch({ updateSource: normaliseUpdateSource(String($event)) })">
          <el-option v-for="[value, label] in updateSourceOptions" :key="value" :value="value" :label="label" />
        </el-select>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md blue" aria-hidden="true"><Rocket :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">GitHub 更新代理加速</span>
            <span class="settings-row-helper">国内网络环境下拉取 GitHub 文件的加速中继地址。</span>
          </div>
        </div>
        <el-input
          class="apple-field settings-proxy-field"
          :model-value="draft.githubProxyBase"
          :disabled="!settingsReady"
          aria-label="GitHub 更新代理"
          placeholder="https://gh-proxy.com"
          @update:model-value="persistSettingsPatch({ githubProxyBase: String($event) })"
        />
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md orange" aria-hidden="true"><Clock :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">自动更新检查间隔</span>
            <span class="settings-row-helper">后台自动轮询线上新版本发布的时间周期。</span>
          </div>
        </div>
        <el-select class="apple-select" popper-class="ios-select-popover" placement="bottom-end" :model-value="draft.updateCheckIntervalHours" :disabled="!settingsReady" aria-label="检查间隔" @update:model-value="persistSettingsPatch({ updateCheckIntervalHours: Number($event) })">
          <el-option v-for="hours in updateIntervalOptions" :key="hours" :value="hours" :label="`${hours} 小时`" />
        </el-select>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md gray" aria-hidden="true"><Calendar :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">日志保留周期</span>
            <span class="settings-row-helper">本地每日日志保留的最大时间上限。</span>
          </div>
        </div>
        <el-select class="apple-select" popper-class="ios-select-popover" placement="bottom-end" :model-value="draft.logRetentionDays" :disabled="!settingsReady" aria-label="保留周期" @update:model-value="persistSettingsPatch({ logRetentionDays: Number($event) })">
          <el-option v-for="[value, label] in logRetentionOptions" :key="value" :value="value" :label="label" />
        </el-select>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md pink" aria-hidden="true"><Terminal :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">控制台日志级别</span>
            <span class="settings-row-helper">设置记录写入运行日志的最低严重程度。</span>
          </div>
        </div>
        <el-select class="apple-select" popper-class="ios-select-popover" placement="bottom-end" :model-value="draft.logLevel" :disabled="!settingsReady" aria-label="日志级别" @update:model-value="persistSettingsPatch({ logLevel: normaliseLogLevel(String($event)) })">
          <el-option v-for="[value, label] in logLevelOptions" :key="value" :value="value" :label="label" />
        </el-select>
      </div>
    </div>

    <!-- 外观与个性化 -->
    <div class="settings-group-card">
      <div class="settings-group-header">
        <div>
          <h3 class="settings-group-title">
            <span class="sq-icon-badge size-sm purple" aria-hidden="true"><Palette :size="13" :stroke-width="2.2" /></span>
            外观与个性化
          </h3>
          <p class="settings-group-desc">提供 macOS HIG 亮暗双模切换、UI 控件尺寸设定与原生液态玻璃材质微调。</p>
        </div>
        <el-button class="btn-apple secondary is-compact" :disabled="!displayReady" @click="confirmResetDisplayPreferences">恢复默认</el-button>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md indigo" aria-hidden="true"><Sun :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">主题模式</span>
            <span class="settings-row-helper">切换亮色与暗色模式，立即全局无缝生效。</span>
          </div>
        </div>
        <el-radio-group class="segmented-control" :model-value="display.themeMode.value" :disabled="!displayReady" aria-label="主题模式" @update:model-value="asThemeMode(String($event))">
          <el-radio-button v-for="[value, label] in themeOptions" :key="value" :value="value">{{ label }}</el-radio-button>
        </el-radio-group>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md blue" aria-hidden="true"><Maximize2 :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">控件尺寸</span>
            <span class="settings-row-helper">调整按钮、表单、表格与文字的全局物理间距。</span>
          </div>
        </div>
        <el-radio-group class="segmented-control" :model-value="display.size.value" :disabled="!displayReady" aria-label="控件尺寸" @update:model-value="asSize(String($event))">
          <el-radio-button v-for="[value, label] in sizeOptions" :key="value" :value="value">{{ label }}</el-radio-button>
        </el-radio-group>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md teal" aria-hidden="true"><DollarSign :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">极光折射流光背景</span>
            <span class="settings-row-helper">开启液态极光光谱色带折射舞台。日常办公默认关闭，保持视觉纯净清爽。</span>
          </div>
        </div>
        <el-switch class="apple-switch" :model-value="glass.backdrop.value" aria-label="开启/关闭极光折射流光背景" @update:model-value="asGlassBackdrop(Boolean($event))" />
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md purple" aria-hidden="true"><Star :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">液态玻璃折射风格</span>
            <span class="settings-row-helper">选择卡片与控件表面液态物理光斑的色散与漫反射特性。</span>
          </div>
        </div>
        <el-radio-group class="segmented-control" :model-value="glass.style.value" aria-label="液态玻璃折射风格" @update:model-value="asGlassStyle(String($event))">
          <el-radio-button v-for="[value, label] in glassStyleOptions" :key="value" :value="value">{{ label }}</el-radio-button>
        </el-radio-group>
      </div>

      <div class="settings-row-item">
        <div class="settings-row-leading">
          <span class="sq-icon-badge size-md cyan" aria-hidden="true"><Globe :size="15" :stroke-width="2.2" /></span>
          <div class="settings-row-info">
            <span class="settings-row-label">液态折射光强</span>
            <span class="settings-row-helper">调节随动高光光斑与表面边缘物理折射的通透度。</span>
          </div>
        </div>
        <div class="settings-slider-wrap">
          <input
            class="settings-slider"
            type="range"
            :min="glassIntensityMin"
            :max="glassIntensityMax"
            step="1"
            :value="glass.intensity.value"
            aria-label="液态折射光强"
            @input="asGlassIntensity($event)"
          >
          <span class="settings-slider-value">{{ glass.intensity.value }}%</span>
        </div>
      </div>
    </div>

    <!-- 恢复默认外观的二次确认（设计稿 .apple-alert-dialog）：非破坏性动作，确认键用主色 -->
    <AlertDialog
      :destructive="false"
      :open="resetDialogOpen"
      confirm-text="恢复默认"
      title="确定恢复默认外观设置？"
      @close="resetDialogOpen = false"
      @confirm="runResetDisplayPreferences"
    >
      该操作会把<strong>主题模式</strong>与<strong>控件尺寸</strong>同时恢复为系统默认值。<br>业务设置（日志级别、更新渠道、自动检查）保持不变，不会被写入。
    </AlertDialog>
  </div>
</template>

<style scoped src="./SettingsPage.css"></style>
