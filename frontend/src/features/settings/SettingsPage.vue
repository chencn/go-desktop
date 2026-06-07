<!--
  文件职责：渲染设置表单并把用户输入提交给应用状态 store。
  说明：注释覆盖组件脚本状态、方法、生命周期和模板结构；不改变渲染逻辑。
-->

<script setup lang="ts">
import { computed, defineComponent, h, ref, watch, type Component, type PropType } from 'vue'
import { AppWindow, BarChart3, Menu as MenuIcon, Paintbrush, Power, RefreshCw, RotateCcw, Settings2, SwatchBook, Terminal, Type } from '@lucide/vue'
import { exportDisplayPreferences, useDisplayPreferences, type AccentColor, type BaseColor, type CardBorder, type ChartColor, type Density, type IconTone, type Menu as MenuPreference, type MenuAccent, type Radius, type TextSize, type ThemeColor, type ThemeMode, type UIStyle } from '@/app/display'
import { useAppStore } from '@/stores/app'
import { defaultRuntimeSettings, type LogLevel, type Settings, type UpdateSource } from '@/api/wails'
import SettingsColorSelect from './SettingsColorSelect.vue'

// appStore 保存 Pinia store 实例，集中访问应用共享状态和动作。
const appStore = useAppStore()
// display 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const display = useDisplayPreferences()
const settingsReady = computed(() => Boolean(appStore.settings))
const displayReady = computed(() => Boolean(appStore.displayPreferences))
// draft 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const draft = ref<Settings>({ ...defaultRuntimeSettings })
// settingsSaveDelayMs 控制设置保存的短延迟，合并连续点击产生的多次后端写入。
const settingsSaveDelayMs = 180
// saveRevision 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
let saveRevision = 0
// saveQueue 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
let saveQueue = Promise.resolve()
// saveTimer 保存业务设置防抖计时器。
let saveTimer: ReturnType<typeof window.setTimeout> | undefined
// displaySaveRevision 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
let displaySaveRevision = 0
// displaySaveQueue 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
let displaySaveQueue = Promise.resolve()
// displaySaveTimer 保存显示偏好防抖计时器。
let displaySaveTimer: ReturnType<typeof window.setTimeout> | undefined

watch([settingsReady, displayReady], ([nextSettingsReady, nextDisplayReady]) => {
  console.info('[settings-trace] SettingsPage：disabled 来源变化', {
    settingsReady: nextSettingsReady,
    displayReady: nextDisplayReady,
    settingsDisabled: !nextSettingsReady,
    displayDisabled: !nextDisplayReady,
    hasStoreSettings: Boolean(appStore.settings),
    hasStoreDisplayPreferences: Boolean(appStore.displayPreferences),
    settings: appStore.settings,
    displayPreferences: appStore.displayPreferences,
  })
}, { immediate: true })

// styleOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const styleOptions: Array<[UIStyle, string]> = [['reka', 'Reka'], ['vega', 'Vega'], ['nova', 'Nova'], ['maia', 'Maia'], ['lyra', 'Lyra'], ['mira', 'Mira'], ['luma', 'Luma'], ['sera', 'Sera']]
// themeOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const themeOptions: Array<[ThemeMode, string]> = [['light', '亮色'], ['dark', '暗色']]
// baseOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const baseOptions: Array<[BaseColor, string]> = [['neutral', 'Neutral'], ['stone', 'Stone'], ['zinc', 'Zinc'], ['mauve', 'Mauve'], ['olive', 'Olive'], ['mist', 'Mist'], ['taupe', 'Taupe']]
// colorOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const colorOptions: Array<[AccentColor, string]> = [
  ['neutral', 'Neutral'],
  ['stone', 'Stone'],
  ['zinc', 'Zinc'],
  ['mauve', 'Mauve'],
  ['olive', 'Olive'],
  ['mist', 'Mist'],
  ['taupe', 'Taupe'],
  ['amber', 'Amber'],
  ['blue', 'Blue'],
  ['cyan', 'Cyan'],
  ['emerald', 'Emerald'],
  ['fuchsia', 'Fuchsia'],
  ['green', 'Green'],
  ['indigo', 'Indigo'],
  ['lime', 'Lime'],
  ['orange', 'Orange'],
  ['pink', 'Pink'],
  ['purple', 'Purple'],
  ['red', 'Red'],
  ['rose', 'Rose'],
  ['sky', 'Sky'],
  ['teal', 'Teal'],
  ['violet', 'Violet'],
  ['yellow', 'Yellow'],
]
// themeColorOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const themeColorOptions: Array<[ThemeColor, string]> = colorOptions
// accentOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const accentOptions: Array<[AccentColor, string]> = colorOptions
// chartOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const chartOptions: Array<[ChartColor, string]> = colorOptions
// iconToneOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const iconToneOptions: Array<[IconTone, string]> = [['default', '默认颜色'], ['colorful', '彩色图标']]
// menuOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const menuOptions: Array<[MenuPreference, string]> = [['default', 'Default'], ['inverted', 'Inverted'], ['default-translucent', 'Default Translucent'], ['inverted-translucent', 'Inverted Translucent']]
// menuAccentOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const menuAccentOptions: Array<[MenuAccent, string]> = [['subtle', 'Subtle'], ['bold', 'Bold']]
// textOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const textOptions: Array<[TextSize, string]> = [['small', '小'], ['normal', '正常'], ['medium', '中'], ['large', '大']]
// radiusOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const radiusOptions: Array<[Radius, string]> = [['default', '默认'], ['none', '无'], ['small', '小'], ['medium', '中'], ['large', '大']]
// densityOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const densityOptions: Array<[Density, string]> = [['compact', '紧凑'], ['comfortable', '舒展']]
// cardBorderOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const cardBorderOptions: Array<[CardBorder, string]> = [['visible', '清晰'], ['soft', '柔和'], ['hidden', '隐藏']]
// updateIntervalOptions 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const updateIntervalOptions = [1, 3, 6, 12]
const updateSourceOptions: Array<[UpdateSource, string]> = [['github', 'GitHub Release'], ['local', '本地静态服务']]
const logLevelOptions: Array<[LogLevel, string]> = [['debug', 'debug'], ['info', 'info'], ['warning', 'warning'], ['error', 'error']]

// PreferenceRow 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
const PreferenceRow = defineComponent({
  props: {
    description: { type: String, required: true },
    icon: { type: [Object, Function] as PropType<Component>, required: true },
    title: { type: String, required: true },
    tone: { type: String, required: true },
  },
  setup(props, { slots }) {
    return () => h('div', { class: 'preference-row' }, [
      h('span', { class: ['data-icon', props.tone], 'aria-hidden': 'true' }, [h(props.icon, { size: 17 })]),
      h('span', { class: 'preference-copy' }, [h('strong', props.title), h('small', props.description)]),
      h('span', { class: 'preference-control' }, slots.default?.()),
    ])
  },
})

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(() => appStore.settings, (settings) => {
  if (settings) {
    draft.value = normaliseSettingsDraft({ ...defaultRuntimeSettings, ...settings })
  }
}, { immediate: true })

// persistSettingsPatch 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function persistSettingsPatch(patch: Partial<Settings>) {
  const base = appStore.settings
  if (!base) {
    appStore.applyAction({ type: 'errorSet', payload: '设置尚未加载，暂不能保存。' })
    return saveQueue
  }
  // revision 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
  const revision = ++saveRevision
  // next 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
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
          // saved 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
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

// normaliseSettingsDraft 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
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

// normaliseUpdateCheckIntervalHours 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function normaliseUpdateCheckIntervalHours(value: number) {
  // interval 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
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

// asStyle 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asStyle(value: string) {
  if (!ensureDisplayReady()) return
  display.setUiStyle(value as UIStyle)
  persistDisplayPreferences()
}

// asThemeMode 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asThemeMode(value: string) {
  if (!ensureDisplayReady()) return
  display.setThemeMode(value as ThemeMode)
  persistDisplayPreferences()
}

// asBaseColor 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asBaseColor(value: string) {
  if (!ensureDisplayReady()) return
  display.setBaseColor(value as BaseColor)
  persistDisplayPreferences()
}

// asThemeColor 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asThemeColor(value: string) {
  if (!ensureDisplayReady()) return
  display.setThemeColor(value as ThemeColor)
  persistDisplayPreferences()
}

// asAccentColor 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asAccentColor(value: string) {
  if (!ensureDisplayReady()) return
  display.setAccentColor(value as AccentColor)
  persistDisplayPreferences()
}

// asChartColor 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asChartColor(value: string) {
  if (!ensureDisplayReady()) return
  display.setChartColor(value as ChartColor)
  persistDisplayPreferences()
}

// asIconTone 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asIconTone(value: string) {
  if (!ensureDisplayReady()) return
  display.setIconTone(value as IconTone)
  persistDisplayPreferences()
}

// asMenu 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asMenu(value: string) {
  if (!ensureDisplayReady()) return
  display.setMenu(value as MenuPreference)
  persistDisplayPreferences()
}

// asMenuAccent 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asMenuAccent(value: string) {
  if (!ensureDisplayReady()) return
  display.setMenuAccent(value as MenuAccent)
  persistDisplayPreferences()
}

// asRadius 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asRadius(value: string) {
  if (!ensureDisplayReady()) return
  display.setRadius(value as Radius)
  persistDisplayPreferences()
}

// asDensity 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asDensity(value: string) {
  if (!ensureDisplayReady()) return
  display.setDensity(value as Density)
  persistDisplayPreferences()
}

// asTextSize 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asTextSize(value: string) {
  if (!ensureDisplayReady()) return
  display.setTextSize(value as TextSize)
  persistDisplayPreferences()
}

// asCardBorder 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function asCardBorder(value: string) {
  if (!ensureDisplayReady()) return
  display.setCardBorder(value as CardBorder)
  persistDisplayPreferences()
}

// resetDisplayPreferencesAndPersist 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function resetDisplayPreferencesAndPersist() {
  if (!ensureDisplayReady()) return
  display.resetDisplayPreferences()
  persistDisplayPreferences()
}

// persistDisplayPreferences 处理 渲染设置表单并把用户输入提交给应用状态 store 中的用户动作、生命周期动作或数据转换。
function persistDisplayPreferences() {
  if (!ensureDisplayReady()) {
    return displaySaveQueue
  }
  // revision 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
  const revision = ++displaySaveRevision
  // next 保存 渲染设置表单并把用户输入提交给应用状态 store 使用的配置、引用或中间结果。
  const next = exportDisplayPreferences()

  if (displaySaveTimer) {
    window.clearTimeout(displaySaveTimer)
  }
  displaySaveTimer = window.setTimeout(() => {
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
  }, settingsSaveDelayMs)

  return displaySaveQueue
}
</script>

<template>
  <!-- 模板结构：声明当前组件对外呈现的布局、插槽和交互入口。 -->
  <div class="page-stack">
    <section class="content-grid settings-layout">
      <UiCard class="settings-card-wide">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="nav-icon icon-tone-indigo" aria-hidden="true"><Settings2 :size="19" /></span>
            <div>
              <UiCardTitle>业务设置</UiCardTitle>
              <UiCardDescription>关闭窗口行为、检查间隔和日志保留周期。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent class="settings-compact-list">
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-gray" aria-hidden="true"><AppWindow :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>关闭到系统托盘</strong>
              <small>点击关闭按钮时隐藏窗口；点击最小化仍进任务栏。</small>
            </span>
            <UiSwitch class="settings-control-switch" :checked="draft.minimizeToTray" :disabled="!settingsReady" aria-label="关闭到系统托盘" @update:checked="persistSettingsPatch({ minimizeToTray: $event })" />
          </div>
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-indigo" aria-hidden="true"><Power :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>开机自启</strong>
              <small>登录 Windows 后自动启动应用。</small>
            </span>
            <UiSwitch class="settings-control-switch" :checked="draft.autoLaunch" :disabled="!settingsReady" aria-label="开机自启" @update:checked="persistSettingsPatch({ autoLaunch: $event })" />
          </div>
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-purple" aria-hidden="true"><AppWindow :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>开机自启时隐藏到托盘</strong>
              <small>仅对开机自启入口生效；手动启动仍显示界面。</small>
            </span>
            <UiSwitch class="settings-control-switch" :checked="draft.launchHiddenToTray" :disabled="!draft.autoLaunch" aria-label="开机自启时隐藏到托盘" @update:checked="persistSettingsPatch({ launchHiddenToTray: $event })" />
          </div>
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-green" aria-hidden="true"><AppWindow :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>创建桌面快捷图标</strong>
              <small>在当前用户桌面创建应用启动快捷方式。</small>
            </span>
            <UiSwitch class="settings-control-switch" :checked="draft.createDesktopShortcut" :disabled="!settingsReady" aria-label="创建桌面快捷图标" @update:checked="persistSettingsPatch({ createDesktopShortcut: $event })" />
          </div>
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-indigo" aria-hidden="true"><RefreshCw :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>更新源</strong>
              <small>选择唯一的更新检查来源。</small>
            </span>
            <UiNativeSelect class="settings-control-select" :model-value="draft.updateSource" :disabled="!settingsReady" aria-label="更新源" @update:model-value="persistSettingsPatch({ updateSource: normaliseUpdateSource(String($event)) })">
              <option v-for="[source, label] in updateSourceOptions" :key="source" :value="source">{{ label }}</option>
            </UiNativeSelect>
          </div>
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-indigo" aria-hidden="true"><RefreshCw :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>检查间隔</strong>
              <small>自动检查 Release 的时间间隔。</small>
            </span>
            <UiNativeSelect class="settings-control-select" :model-value="draft.updateCheckIntervalHours" :disabled="!settingsReady" aria-label="检查间隔" @update:model-value="persistSettingsPatch({ updateCheckIntervalHours: Number($event) })">
              <option v-for="hours in updateIntervalOptions" :key="hours" :value="hours">{{ hours }} 小时</option>
            </UiNativeSelect>
          </div>
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-orange" aria-hidden="true"><Terminal :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>保留周期</strong>
              <small>每日文件日志自动清理周期。</small>
            </span>
            <UiNativeSelect class="settings-control-select" :model-value="draft.logRetentionDays" :disabled="!settingsReady" aria-label="保留周期" @update:model-value="persistSettingsPatch({ logRetentionDays: Number($event) })">
              <option :value="7">7 天</option>
              <option :value="30">30 天</option>
              <option :value="60">60 天</option>
              <option :value="90">90 天</option>
              <option :value="180">180 天</option>
              <option :value="365">365 天</option>
              <option :value="-1">永不清理</option>
            </UiNativeSelect>
          </div>
          <div class="settings-compact-row">
            <span class="data-icon icon-tone-indigo" aria-hidden="true"><Terminal :size="17" /></span>
            <span class="settings-compact-copy">
              <strong>日志级别</strong>
              <small>调试级别会记录更详细的后端保存和异常定位信息。</small>
            </span>
            <UiNativeSelect class="settings-control-select" :model-value="draft.logLevel" :disabled="!settingsReady" aria-label="日志级别" @update:model-value="persistSettingsPatch({ logLevel: normaliseLogLevel(String($event)) })">
              <option v-for="[level, label] in logLevelOptions" :key="level" :value="level">{{ label }}</option>
            </UiNativeSelect>
          </div>
        </UiCardContent>
      </UiCard>

      <UiCard class="settings-card-wide">
        <UiCardHeader>
          <div class="settings-display-header">
            <div class="section-title-row">
              <span class="nav-icon icon-tone-purple" aria-hidden="true"><Paintbrush :size="19" /></span>
              <div>
                <UiCardTitle>显示偏好</UiCardTitle>
                <UiCardDescription>参考 shadcn-vue create 参数，覆盖 Style、Base Color、Theme、Chart Color、图标、Radius、Menu 和 Menu Accent。</UiCardDescription>
              </div>
            </div>
            <UiButton class="settings-reset-button" type="button" variant="secondary" size="sm" :disabled="!displayReady" @click="resetDisplayPreferencesAndPersist">
              <RotateCcw :size="16" />
              恢复初始值
            </UiButton>
          </div>
        </UiCardHeader>
        <UiCardContent class="preference-stack">
          <PreferenceRow title="组件风格" description="对应 shadcn-vue create 的 style，影响组件密度和默认视觉基线。" :icon="Settings2" tone="icon-tone-indigo">
            <UiNativeSelect class="preference-native-select" :model-value="display.uiStyle.value" :disabled="!displayReady" aria-label="组件风格" @update:model-value="asStyle">
              <option v-for="[value, label] in styleOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="主题模式" description="右上角保留快捷切换；这里提供完整设置入口。" :icon="Paintbrush" tone="icon-tone-purple">
            <UiNativeSelect class="preference-native-select" :model-value="display.themeMode.value" :disabled="!displayReady" aria-label="主题模式" @update:model-value="asThemeMode">
              <option v-for="[value, label] in themeOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="基础色盘" description="对应 shadcn-vue baseColor，并影响亮色和暗色中性色 token。" :icon="SwatchBook" tone="icon-tone-gray">
            <SettingsColorSelect class="preference-control-color" label="基础色盘" :disabled="!displayReady" :model-value="display.baseColor.value" :options="baseOptions" @update:model-value="asBaseColor" />
          </PreferenceRow>
          <PreferenceRow title="主题" description="对应 shadcn-vue create 的 Theme，控制主按钮、焦点环、选中态和高强调 token。" :icon="Paintbrush" tone="icon-tone-purple">
            <SettingsColorSelect class="preference-control-color" label="主题" :disabled="!displayReady" :model-value="display.themeColor.value" :options="themeColorOptions" @update:model-value="asThemeColor" />
          </PreferenceRow>
          <PreferenceRow title="强调色" description="用于主按钮、选中导航、更新图标、进度、开关和焦点环。" :icon="SwatchBook" tone="icon-tone-indigo">
            <SettingsColorSelect class="preference-control-color" label="强调色" :disabled="!displayReady" :model-value="display.accentColor.value" :options="accentOptions" @update:model-value="asAccentColor" />
          </PreferenceRow>
          <PreferenceRow title="图表色" description="对应 chart token，不再偷用强调色；用于统计和可视化色板。" :icon="BarChart3" tone="icon-tone-green">
            <SettingsColorSelect class="preference-control-color" label="图表色" :disabled="!displayReady" :model-value="display.chartColor.value" :options="chartOptions" @update:model-value="asChartColor" />
          </PreferenceRow>
          <PreferenceRow title="圆角" description="通过 --radius 派生到卡片、按钮、输入框、弹窗和列表。" :icon="Settings2" tone="icon-tone-gray">
            <UiNativeSelect class="preference-native-select" :model-value="display.radius.value" :disabled="!displayReady" aria-label="圆角" @update:model-value="asRadius">
              <option v-for="[value, label] in radiusOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="图标颜色" description="默认使用单色语义图标；需要按含义着色时切到彩色图标。" :icon="SwatchBook" tone="icon-tone-purple">
            <UiNativeSelect class="preference-native-select" :model-value="display.iconTone.value" :disabled="!displayReady" aria-label="图标颜色" @update:model-value="asIconTone">
              <option v-for="[value, label] in iconToneOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="菜单" description="对应 create 页 Menu，支持 Default、Inverted 和透明变体。" :icon="MenuIcon" tone="icon-tone-gray">
            <UiNativeSelect class="preference-native-select" :model-value="display.menu.value" :disabled="!displayReady" aria-label="菜单" @update:model-value="asMenu">
              <option v-for="[value, label] in menuOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="菜单强调" description="对应 create 页 Menu Accent，例如 Subtle，控制菜单 hover 和 active 背景强度。" :icon="MenuIcon" tone="icon-tone-indigo">
            <UiNativeSelect class="preference-native-select" :model-value="display.menuAccent.value" :disabled="!displayReady" aria-label="菜单强调" @update:model-value="asMenuAccent">
              <option v-for="[value, label] in menuAccentOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="密度" description="影响页面留白、工具栏高度和控件最小高度。" :icon="Settings2" tone="icon-tone-gray">
            <UiNativeSelect class="preference-native-select" :model-value="display.density.value" :disabled="!displayReady" aria-label="密度" @update:model-value="asDensity">
              <option v-for="[value, label] in densityOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="字体大小" description="影响标题、正文、按钮和日志字号。" :icon="Type" tone="icon-tone-indigo">
            <UiNativeSelect class="preference-native-select" :model-value="display.textSize.value" :disabled="!displayReady" aria-label="字体大小" @update:model-value="asTextSize">
              <option v-for="[value, label] in textOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
          <PreferenceRow title="卡片边框" description="统一控制 Card、列表、日志表和弹窗边框强度。" :icon="Settings2" tone="icon-tone-gray">
            <UiNativeSelect class="preference-native-select" :model-value="display.cardBorder.value" :disabled="!displayReady" aria-label="卡片边框" @update:model-value="asCardBorder">
              <option v-for="[value, label] in cardBorderOptions" :key="value" :value="value">{{ label }}</option>
            </UiNativeSelect>
          </PreferenceRow>
        </UiCardContent>
      </UiCard>
    </section>
  </div>
</template>

<style scoped src="./SettingsPage.css"></style>
