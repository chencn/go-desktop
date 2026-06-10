<!--
  文件职责：渲染设置表单并把用户输入提交给应用状态 store。
  业务设置保存到后端 Settings；显示偏好保存到独立 DisplayPreferences。
  界面重构：采用温暖灿烂的暖日玻璃质感设计，将参数平铺与可视化选择融合。
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Archive, CalendarClock, CloudDownload, EyeOff, ListFilter, MonitorUp, Palette, PanelBottomClose, Rocket, RotateCcw, Sun, Moon, Wrench } from '@lucide/vue'
import { exportDisplayPreferences, useDisplayPreferences, type AccentColor, type BaseColor, type CardBorder, type ChartColor, type Density, type DisplayScheme, type IconTone, type Menu as MenuPreference, type Radius, type TextSize, type ThemeColor, type ThemeMode, type UIStyle } from '@/app/display'
import { useAppStore } from '@/stores/app'
import { defaultRuntimeSettings, type LogLevel, type Settings, type UpdateSource } from '@/api/wails'

const appStore = useAppStore()
// display 是全局显示偏好的响应式 facade，实际持久化仍通过 appStore.persistDisplayPreferences。
const display = useDisplayPreferences()
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
// saveTimer 保存业务设置防抖计时器。
let saveTimer: ReturnType<typeof window.setTimeout> | undefined
// displaySaveRevision 标记最新显示偏好保存请求，配合队列丢弃过期写入。
let displaySaveRevision = 0
// displaySaveQueue 串行化 SaveDisplayPreferences，保证落库顺序和 UI 最终状态一致。
let displaySaveQueue = Promise.resolve()
// displaySaveTimer 保存显示偏好防抖计时器。
let displaySaveTimer: ReturnType<typeof window.setTimeout> | undefined
// resetDisplayDialogOpen 控制恢复当前显示方案默认值的二次确认弹窗。
const resetDisplayDialogOpen = ref(false)

// styleOptions 对应 shadcn-vue create 的 style，artistic 方案通过主题样式扩展同一组 token。
const styleOptions: Array<[UIStyle, string]> = [
  ['reka', '标准 Reka'],
  ['vega', '极简 Vega'],
  ['nova', '开阔 Nova'],
  ['maia', '紧凑 Maia'],
  ['lyra', '雅致 Lyra'],
  ['mira', '柔和 Mira'],
  ['luma', '大气 Luma'],
  ['sera', '轻盈 Sera']
]
// themeOptions 控制全局亮暗模式，和具体显示方案的 profile 分开保存。
const themeOptions: Array<[ThemeMode, string]> = [['light', '亮色'], ['dark', '暗色']]
type DisplayColorKind = 'neutral' | 'base' | 'brand'
type DisplayColorOption<T extends AccentColor = AccentColor> = {
  value: T
  label: string
  kind: DisplayColorKind
  baseLabel?: string
}
// displayColorOptions 是设置页 18 色唯一 token 真源；base/theme/chart 选项都从这里派生。
const displayColorOptions: DisplayColorOption[] = [
  { value: 'neutral', label: 'Neutral (灰阶)', kind: 'neutral' },
  { value: 'stone', label: '石灰色 (Stone)', kind: 'base', baseLabel: 'Stone (石灰)' },
  { value: 'zinc', label: '质感锌 (Zinc Gray)', kind: 'base', baseLabel: 'Zinc (锌灰)' },
  { value: 'mauve', label: '淡紫灰 (Mauve)', kind: 'base', baseLabel: 'Mauve (淡紫灰)' },
  { value: 'olive', label: '橄榄灰 (Olive)', kind: 'base', baseLabel: 'Olive (橄榄绿)' },
  { value: 'mist', label: '雾蓝灰 (Mist)', kind: 'base', baseLabel: 'Mist (雾蓝)' },
  { value: 'taupe', label: '褐灰色 (Taupe)', kind: 'base', baseLabel: 'Taupe (褐灰)' },
  { value: 'orange', label: '落日橘 (Sunset Orange)', kind: 'brand' },
  { value: 'rose', label: '玫瑰红 (Coral Rose)', kind: 'brand' },
  { value: 'pink', label: '樱花粉 (Sakura Pink)', kind: 'brand' },
  { value: 'amber', label: '琥珀黄 (Amber Gold)', kind: 'brand' },
  { value: 'emerald', label: '薄荷绿 (Emerald)', kind: 'brand' },
  { value: 'teal', label: '松石绿 (Teal Forest)', kind: 'brand' },
  { value: 'cyan', label: '晴空蓝 (Sky Cyan)', kind: 'brand' },
  { value: 'apple-blue', label: 'Apple 蓝 (Apple Blue)', kind: 'brand' },
  { value: 'blue', label: 'AntD 蓝 (Ant Design Blue)', kind: 'brand' },
  { value: 'indigo', label: '靛蓝色 (Indigo Night)', kind: 'brand' },
  { value: 'sky', label: '天际蓝 (Sky)', kind: 'brand' },
]
function toColorOption<T extends AccentColor>(option: DisplayColorOption<T>): [T, string] {
  return [option.value, option.label]
}

function isBaseColorOption(option: DisplayColorOption): option is DisplayColorOption<BaseColor> {
  return option.kind === 'neutral' || option.kind === 'base'
}

function isBrandColorOption(option: DisplayColorOption): option is DisplayColorOption<ThemeColor> {
  return option.kind === 'neutral' || option.kind === 'brand'
}

function toBaseColorOption(option: DisplayColorOption<BaseColor>): [BaseColor, string] {
  return [option.value, option.baseLabel ?? option.label]
}

const colorOptions: Array<[AccentColor, string]> = displayColorOptions.map(toColorOption)
const baseOptions: Array<[BaseColor, string]> = displayColorOptions
  .filter(isBaseColorOption)
  .map(toBaseColorOption)
const brandColorOptions: Array<[ThemeColor, string]> = displayColorOptions.filter(isBrandColorOption).map(toColorOption)
const themeColorOptions: Array<[ThemeColor, string]> = brandColorOptions
const chartOptions: Array<[ChartColor, string]> = colorOptions
// iconToneOptions 只影响语义图标是否彩色，不改变 lucide 图标本身。
const iconToneOptions: Array<[IconTone, string]> = [['default', '默认颜色'], ['colorful', '彩色图标']]
// menuOptions 包含当前主题层支持的侧边栏变体。
const menuOptions: Array<[MenuPreference, string]> = [
  ['default', '默认 (Default)'],
  ['inverted', '反色 (Inverted)']
]
// textOptions 写入 DOM dataset，由 CSS token 层统一响应。
const textOptions: Array<[TextSize, string]> = [['small', '小'], ['normal', '正常'], ['medium', '中'], ['large', '大']]
// radiusOptions 写入 --radius，具体边界由主题样式解释。
const radiusOptions: Array<[Radius, string]> = [['default', '默认'], ['none', '无'], ['small', '小'], ['medium', '中'], ['large', '大']]
// densityOptions 控制页面和控件密度 token。
const densityOptions: Array<[Density, string]> = [['compact', '紧凑'], ['comfortable', '舒展']]
// cardBorderOptions 控制容器边框强度，不影响业务设置字段。
const cardBorderOptions: Array<[CardBorder, string]> = [['visible', '清晰'], ['soft', '柔和'], ['hidden', '隐藏']]
// updateIntervalOptions 必须和后端允许的检查间隔保持一致；非法值会回退默认值。
const updateIntervalOptions = [1, 3, 6, 12]
const updateSourceOptions: Array<[UpdateSource, string]> = [['github', 'GitHub Release'], ['local', '本地静态服务']]
const logRetentionOptions: Array<[number, string]> = [
  [7, '7 天'],
  [30, '30 天'],
  [60, '60 天'],
  [90, '90 天'],
  [180, '180 天'],
  [365, '365 天'],
  [-1, '永不清理']
]
const logLevelOptions: Array<[LogLevel, string]> = [['debug', 'debug'], ['info', 'info'], ['warning', 'warning'], ['error', 'error']]

// 显示方案卡片信息定义，附带特色色彩发光类
const schemeCardOptions: Array<[DisplayScheme, string, string, string]> = [
  ['shadcn', 'shadcn', '经典灵活的自由配置', 'glow-shadcn'],
  ['artistic', 'Artistic', '清爽柔和的品牌主题', 'glow-artistic']
]

// 辅助函数：快速获取色名对应标签
function getThemeColorLabel(val: string) {
  return colorOptions.find(([v]) => v === val)?.[1] ?? val
}

function getBaseColorLabel(val: string) {
  return baseOptions.find(([v]) => v === val)?.[1] ?? val
}

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
    githubOwner: settings.githubOwner.trim() || defaultRuntimeSettings.githubOwner,
    githubRepo: settings.githubRepo.trim() || defaultRuntimeSettings.githubRepo,
    githubProxyBase: settings.githubProxyBase.trim(),
    updateCheckIntervalHours: normaliseUpdateCheckIntervalHours(settings.updateCheckIntervalHours),
    minimizeToTray: Boolean(settings.minimizeToTray),
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

function asDisplayScheme(value: string) {
  if (!ensureDisplayReady()) return
  const scheme = value as DisplayScheme
  display.setDisplayScheme(scheme)
  persistDisplayPreferences({ immediate: true })
}

// asStyle 只更新当前显示方案的 uiStyle，随后防抖保存完整 DisplayPreferences。
function asStyle(value: string) {
  if (!ensureDisplayReady()) return
  display.setUiStyle(value as UIStyle)
  persistDisplayPreferences()
}

// asThemeMode 切换全局亮暗模式；该值不随 shadcn/artistic profile 分离。
function asThemeMode(value: string) {
  if (!ensureDisplayReady()) return
  display.setThemeMode(value as ThemeMode)
  persistDisplayPreferences()
}

// 下面这些 as* 方法都是控件边界：先写 display facade，再持久化完整偏好快照。
function asBaseColor(value: string) {
  if (!ensureDisplayReady()) return
  display.setBaseColor(value as BaseColor)
  persistDisplayPreferences()
}

function asThemeColor(value: string) {
  if (!ensureDisplayReady()) return
  display.setThemeColor(value as ThemeColor)
  persistDisplayPreferences()
}

function asChartColor(value: string) {
  if (!ensureDisplayReady()) return
  display.setChartColor(value as ChartColor)
  persistDisplayPreferences()
}

function asIconTone(value: string) {
  if (!ensureDisplayReady()) return
  display.setIconTone(value as IconTone)
  persistDisplayPreferences()
}

function asMenu(value: string) {
  if (!ensureDisplayReady()) return
  display.setMenu(value as MenuPreference)
  persistDisplayPreferences()
}

function asRadius(value: string) {
  if (!ensureDisplayReady()) return
  display.setRadius(value as Radius)
  persistDisplayPreferences()
}

function asDensity(value: string) {
  if (!ensureDisplayReady()) return
  display.setDensity(value as Density)
  persistDisplayPreferences()
}

function asTextSize(value: string) {
  if (!ensureDisplayReady()) return
  display.setTextSize(value as TextSize)
  persistDisplayPreferences()
}

function asCardBorder(value: string) {
  if (!ensureDisplayReady()) return
  display.setCardBorder(value as CardBorder)
  persistDisplayPreferences()
}

// resetDisplayPreferencesAndPersist 只恢复当前显示方案默认值，保留另一套方案和亮暗模式。
function resetDisplayPreferencesAndPersist() {
  if (!ensureDisplayReady()) return
  display.resetDisplayPreferencesForCurrentScheme()
  persistDisplayPreferences()
}

function confirmResetDisplayPreferences() {
  resetDisplayDialogOpen.value = false
  resetDisplayPreferencesAndPersist()
}

// persistDisplayPreferences 保存 exportDisplayPreferences 的完整快照；切换显示方案需要 immediate 避免旧方案覆盖。
function persistDisplayPreferences(options: { immediate?: boolean } = {}) {
  if (!ensureDisplayReady()) {
    return displaySaveQueue
  }
  const revision = ++displaySaveRevision
  const next = exportDisplayPreferences()

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
          await appStore.persistDisplayPreferences(next)
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
</script>

<template>
  <div class="page-stack">
    <section class="settings-grid-layout">

      <!-- 左栏：应用基础与业务设置 -->
      <UiCard class="settings-main-card">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="nav-icon icon-tone-orange" aria-hidden="true"><Wrench :size="19" /></span>
            <div>
              <UiCardTitle>应用与业务设置</UiCardTitle>
              <UiCardDescription>控制窗口托盘行为、开机自启策略、自动更新周期及每日日志的清理策略。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent class="settings-control-list">

          <div class="settings-row-item">
            <span class="data-icon icon-tone-cyan" aria-hidden="true"><PanelBottomClose :size="17" /></span>
            <div class="row-copy">
              <strong>关闭到系统托盘</strong>
              <small>点击关闭按钮时隐藏窗口至后台，点击最小化仍进入任务栏。</small>
            </div>
            <UiSwitch class="settings-control-switch" :checked="draft.minimizeToTray" :disabled="!settingsReady" aria-label="关闭到系统托盘" @update:checked="persistSettingsPatch({ minimizeToTray: $event })" />
          </div>

          <div class="settings-row-item">
            <span class="data-icon icon-tone-green" aria-hidden="true"><Rocket :size="17" /></span>
            <div class="row-copy">
              <strong>开机自启</strong>
              <small>系统完成引导并登录 Windows 后自动运行该应用。</small>
            </div>
            <UiSwitch class="settings-control-switch" :checked="draft.autoLaunch" :disabled="!settingsReady" aria-label="开机自启" @update:checked="persistSettingsPatch({ autoLaunch: $event })" />
          </div>

          <div class="settings-row-item">
            <span class="data-icon icon-tone-purple" aria-hidden="true"><EyeOff :size="17" /></span>
            <div class="row-copy">
              <strong>自启时隐藏到系统托盘</strong>
              <small>仅自启生效，前台不展示应用窗口。手动双击启动仍正常显示。</small>
            </div>
            <UiSwitch class="settings-control-switch" :checked="draft.launchHiddenToTray" :disabled="!draft.autoLaunch" aria-label="开机自启时隐藏到托盘" @update:checked="persistSettingsPatch({ launchHiddenToTray: $event })" />
          </div>

          <div class="settings-row-item">
            <span class="data-icon icon-tone-blue" aria-hidden="true"><MonitorUp :size="17" /></span>
            <div class="row-copy">
              <strong>创建桌面快捷图标</strong>
              <small>在当前登录用户的桌面生成指向本程序的快捷启动图标。</small>
            </div>
            <UiSwitch class="settings-control-switch" :checked="draft.createDesktopShortcut" :disabled="!settingsReady" aria-label="创建桌面快捷图标" @update:checked="persistSettingsPatch({ createDesktopShortcut: $event })" />
          </div>

          <div class="settings-row-item is-select-row">
            <span class="data-icon icon-tone-blue" aria-hidden="true"><CloudDownload :size="17" /></span>
            <div class="row-copy">
              <strong>系统更新源</strong>
              <small>选择获取客户端新版本 manifest 的检查来源。</small>
            </div>
            <UiSelect :model-value="draft.updateSource" :disabled="!settingsReady" @update:model-value="persistSettingsPatch({ updateSource: normaliseUpdateSource(String($event)) })">
              <UiSelectTrigger class="settings-control-select" aria-label="更新源">
                <UiSelectValue placeholder="更新源" />
              </UiSelectTrigger>
              <UiSelectContent>
                <UiSelectItem v-for="[value, label] in updateSourceOptions" :key="value" :value="value">{{ label }}</UiSelectItem>
              </UiSelectContent>
            </UiSelect>
          </div>

          <div class="settings-row-item is-select-row">
            <span class="data-icon icon-tone-amber" aria-hidden="true"><CalendarClock :size="17" /></span>
            <div class="row-copy">
              <strong>自动更新检查间隔</strong>
              <small>后台自动轮询线上新版本发布的时间跨度。</small>
            </div>
            <UiSelect :model-value="draft.updateCheckIntervalHours" :disabled="!settingsReady" @update:model-value="persistSettingsPatch({ updateCheckIntervalHours: Number($event) })">
              <UiSelectTrigger class="settings-control-select" aria-label="检查间隔">
                <UiSelectValue placeholder="检查间隔" />
              </UiSelectTrigger>
              <UiSelectContent>
                <UiSelectItem v-for="hours in updateIntervalOptions" :key="hours" :value="hours">{{ hours }} 小时</UiSelectItem>
              </UiSelectContent>
            </UiSelect>
          </div>

          <div class="settings-row-item is-select-row">
            <span class="data-icon icon-tone-orange" aria-hidden="true"><Archive :size="17" /></span>
            <div class="row-copy">
              <strong>每日日志保留周期</strong>
              <small>日志文件夹中 JSONL 文本文件保留的最大时长。</small>
            </div>
            <UiSelect :model-value="draft.logRetentionDays" :disabled="!settingsReady" @update:model-value="persistSettingsPatch({ logRetentionDays: Number($event) })">
              <UiSelectTrigger class="settings-control-select" aria-label="保留周期">
                <UiSelectValue placeholder="保留周期" />
              </UiSelectTrigger>
              <UiSelectContent>
                <UiSelectItem v-for="[value, label] in logRetentionOptions" :key="value" :value="value">{{ label }}</UiSelectItem>
              </UiSelectContent>
            </UiSelect>
          </div>

          <div class="settings-row-item is-select-row">
            <span class="data-icon icon-tone-red" aria-hidden="true"><ListFilter :size="17" /></span>
            <div class="row-copy">
              <strong>控制台日志级别</strong>
              <small>调低日志级别可以帮您记录更详尽的信息用于异常定位。</small>
            </div>
            <UiSelect :model-value="draft.logLevel" :disabled="!settingsReady" @update:model-value="persistSettingsPatch({ logLevel: normaliseLogLevel(String($event)) })">
              <UiSelectTrigger class="settings-control-select" aria-label="日志级别">
                <UiSelectValue placeholder="日志级别" />
              </UiSelectTrigger>
              <UiSelectContent>
                <UiSelectItem v-for="[value, label] in logLevelOptions" :key="value" :value="value">{{ label }}</UiSelectItem>
              </UiSelectContent>
            </UiSelect>
          </div>

        </UiCardContent>
      </UiCard>

      <!-- 右栏：外观与个性化艺术主题设置 -->
      <UiCard class="settings-main-card">
        <UiCardHeader class="aesthetic-card-header">
          <div class="section-title-row">
            <span class="nav-icon icon-tone-purple" aria-hidden="true"><Palette :size="19" /></span>
            <div>
              <UiCardTitle>外观与个性化</UiCardTitle>
              <UiCardDescription>选择显示方案、调配主题与圆角，配置仅即时反馈至您的桌面偏好中。</UiCardDescription>
            </div>
          </div>
          <UiButton class="aesthetic-reset-btn" variant="outline" size="sm" :disabled="!displayReady" @click="resetDisplayDialogOpen = true">
            <RotateCcw :size="14" /> 恢复默认预设
          </UiButton>
        </UiCardHeader>
        <UiCardContent class="aesthetic-content-stack">

          <!-- 显示方案选择卡片组 -->
          <div class="aesthetic-field-col scheme-field-container">
            <label class="aesthetic-field-title">主题显示方案</label>
            <div class="scheme-cards-row">
              <button
                v-for="[value, label, desc, glowClass] in schemeCardOptions"
                :key="value"
                type="button"
                class="scheme-card-box"
                :class="[{ 'is-active': display.displayScheme.value === value }, glowClass]"
                :disabled="!displayReady"
                @click="asDisplayScheme(value)"
              >
                <strong class="scheme-title-text">{{ label }}</strong>
                <span class="scheme-desc-text">{{ desc }}</span>
              </button>
            </div>
          </div>

          <!-- 全局主色彩模式 -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">主色彩模式</label>
            <div class="visual-segmented-control select-mode-control">
              <button
                v-for="[value, label] in themeOptions"
                :key="value"
                type="button"
                class="visual-segment-btn"
                :class="{ 'is-active': display.themeMode.value === value }"
                :disabled="!displayReady"
                @click="asThemeMode(value)"
              >
                <Sun v-if="value === 'light'" :size="15" />
                <Moon v-else :size="15" />
                <span>{{ label }}</span>
              </button>
            </div>
          </div>

          <!-- 中性色盘色调 -->
          <div class="aesthetic-field-col">
            <div class="flex justify-between items-center mb-1">
              <label class="aesthetic-field-title">中性灰阶色调 (Base Color)</label>
              <span class="active-badge">{{ getBaseColorLabel(display.baseColor.value) }}</span>
            </div>
            <p class="field-desc-para">影响亮色和暗色模式下的全局灰色基底与背景偏色。</p>
            <div class="color-dot-palette base-palette" :class="{ 'is-disabled-grid': !displayReady }">
              <button
                v-for="[value, label] in baseOptions"
                :key="value"
                type="button"
                class="color-dot-btn"
                :class="{ 'is-selected': display.baseColor.value === value }"
                :disabled="!displayReady"
                :title="label"
                @click="asBaseColor(value)"
              >
                <span class="color-palette-circle" :data-accent="value" />
              </button>
            </div>
          </div>

          <!-- 品牌主题色 - 平铺圆形色块 -->
          <div class="aesthetic-field-col">
            <div class="flex justify-between items-center mb-1">
              <label class="aesthetic-field-title">品牌主题色 (Theme Color)</label>
              <span class="active-badge">{{ getThemeColorLabel(display.themeColor.value) }}</span>
            </div>
            <p class="field-desc-para">控制主操作按钮、关键激活项、输入焦点环和进度指示条的配色。</p>
            <div class="color-dot-palette" :class="{ 'is-disabled-grid': !displayReady }">
              <button
                v-for="[value, label] in themeColorOptions"
                :key="value"
                type="button"
                class="color-dot-btn"
                :class="{ 'is-selected': display.themeColor.value === value }"
                :disabled="!displayReady"
                :title="label"
                @click="asThemeColor(value)"
              >
                <span class="color-palette-circle" :data-accent="value" />
              </button>
            </div>
          </div>

          <!-- 品牌辅助色由主题托管，展示同一品牌色的浅一号状态，不提供独立编辑入口。 -->
          <div class="aesthetic-field-col is-managed-field" aria-disabled="true">
            <div class="flex justify-between items-center mb-1">
              <label class="aesthetic-field-title">品牌辅助色 (Accent Color)</label>
              <span class="active-badge managed-badge">{{ getThemeColorLabel(display.themeColor.value) }}</span>
            </div>
            <p class="field-desc-para">跟随品牌主题色，用于下拉选中项、辅助强调和轻量交互状态。</p>
            <div class="color-dot-palette is-managed-palette" :class="{ 'is-disabled-grid': !displayReady }">
              <button
                v-for="[value, label] in themeColorOptions"
                :key="value"
                type="button"
                class="color-dot-btn"
                :class="{ 'is-selected': display.themeColor.value === value, 'is-managed-selected': display.themeColor.value === value }"
                disabled
                :title="label"
              >
                <span class="color-palette-circle" :data-accent="value" />
              </button>
            </div>
          </div>

          <!-- 图标色彩风格 (Icon Color Style) -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">图标色彩风格 (Icon Color Style)</label>
            <p class="field-desc-para">设定侧边栏及卡片内部的图标色彩风格（单色默认 vs 彩色视觉）。</p>
            <div class="visual-segmented-control">
              <button
                v-for="[value, label] in iconToneOptions"
                :key="value"
                type="button"
                class="visual-segment-btn"
                :class="{ 'is-active': display.iconTone.value === value }"
                :disabled="!displayReady"
                @click="asIconTone(value)"
              >
                <span>{{ label }}</span>
              </button>
            </div>
          </div>

          <!-- 图表配色 -->
          <div class="aesthetic-field-col">
            <div class="flex justify-between items-center mb-1">
              <label class="aesthetic-field-title">图表配色体系 (Chart Color)</label>
              <span class="active-badge">{{ getThemeColorLabel(display.chartColor.value) }}</span>
            </div>
            <p class="field-desc-para">专属用于日志分析图表、可视化数据和状态看板。</p>
            <div class="color-dot-palette" :class="{ 'is-disabled-grid': !displayReady }">
              <button
                v-for="[value, label] in chartOptions"
                :key="value"
                type="button"
                class="color-dot-btn"
                :class="{ 'is-selected': display.chartColor.value === value }"
                :disabled="!displayReady"
                :title="label"
                @click="asChartColor(value)"
              >
                <span class="color-palette-circle" :data-accent="value" />
              </button>
            </div>
          </div>

          <!-- 视觉圆角选项 -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">圆角大小 (Border Radius)</label>
            <p class="field-desc-para">设定按钮、输入框、对话框以及卡片的边角弧度。</p>
            <div class="visual-segmented-control">
              <button
                v-for="[value, label] in radiusOptions"
                :key="value"
                type="button"
                class="visual-segment-btn"
                :class="{ 'is-active': display.radius.value === value }"
                :disabled="!displayReady"
                @click="asRadius(value)"
              >
                <span>{{ label }}</span>
              </button>
            </div>
          </div>

          <!-- 界面字号大小 -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">字体字号 (Font Size)</label>
            <p class="field-desc-para">调整系统各层级文字的字号大小，适配高分屏显示。</p>
            <div class="visual-segmented-control">
              <button
                v-for="[value, label] in textOptions"
                :key="value"
                type="button"
                class="visual-segment-btn"
                :class="{ 'is-active': display.textSize.value === value }"
                :disabled="!displayReady"
                @click="asTextSize(value)"
              >
                <span>{{ label }}</span>
              </button>
            </div>
          </div>

          <!-- 界面元素密度 -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">界面布局密度 (Density)</label>
            <p class="field-desc-para">调整界面组件的紧凑程度，优化边距与列表行高。</p>
            <div class="visual-segmented-control">
              <button
                v-for="[value, label] in densityOptions"
                :key="value"
                type="button"
                class="visual-segment-btn"
                :class="{ 'is-active': display.density.value === value }"
                :disabled="!displayReady"
                @click="asDensity(value)"
              >
                <span>{{ label }}</span>
              </button>
            </div>
          </div>

          <!-- 容器边框强度 -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">容器与卡片边框强度</label>
            <p class="field-desc-para">设定面板与卡片的边框可见度，增强页面空间层次。</p>
            <div class="visual-segmented-control">
              <button
                v-for="[value, label] in cardBorderOptions"
                :key="value"
                type="button"
                class="visual-segment-btn"
                :class="{ 'is-active': display.cardBorder.value === value }"
                :disabled="!displayReady"
                @click="asCardBorder(value)"
              >
                <span>{{ label }}</span>
              </button>
            </div>
          </div>

          <!-- 侧边菜单风格 -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">侧边导航风格 (Sidebar Style)</label>
            <p class="field-desc-para">选择左侧主导航栏的底色模式与半透明模糊度。</p>
            <UiSelect :model-value="display.menu.value" :disabled="!displayReady" @update:model-value="asMenu">
              <UiSelectTrigger class="settings-control-select" aria-label="菜单风格">
                <UiSelectValue placeholder="菜单风格" />
              </UiSelectTrigger>
              <UiSelectContent>
                <UiSelectItem v-for="[value, label] in menuOptions" :key="value" :value="value">{{ label }}</UiSelectItem>
              </UiSelectContent>
            </UiSelect>
          </div>

          <!-- 组件风格基底 -->
          <div class="aesthetic-field-col">
            <label class="aesthetic-field-title">组件高矮风格 (UI Height Base)</label>
            <p class="field-desc-para">调整主要按钮和表单输入框的基础高度与比例。</p>
            <UiSelect :model-value="display.uiStyle.value" :disabled="!displayReady" @update:model-value="asStyle">
              <UiSelectTrigger class="settings-control-select" aria-label="组件风格">
                <UiSelectValue placeholder="组件风格" />
              </UiSelectTrigger>
              <UiSelectContent>
                <UiSelectItem v-for="[value, label] in styleOptions" :key="value" :value="value">{{ label }}</UiSelectItem>
              </UiSelectContent>
            </UiSelect>
          </div>



        </UiCardContent>
      </UiCard>

    </section>

    <!-- 恢复默认值的确认模态框 -->
    <UiAlertDialog
      :open="resetDisplayDialogOpen"
      title="恢复当前方案默认预设"
      description="此操作将会重置您在当前方案下定义的所有细项偏好。是否确认继续？"
      confirm-text="恢复默认"
      @close="resetDisplayDialogOpen = false"
      @confirm="confirmResetDisplayPreferences"
    />
  </div>
</template>

<style scoped src="./SettingsPage.css"></style>
