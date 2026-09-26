// 文件职责：维护显示偏好（亮暗模式、全局尺寸、液态玻璃光学）、DOM 同步和持久化快照转换。
// 持久化只有一条链路：后端 `display.preferences.v3` JSON KV；液态玻璃三项同样走这里，不再另设本地存储。

import { readonly, ref, watch } from 'vue'

import { type GlassStyle, exportGlassPreferences, hydrateGlassPreferences, resetGlassPreferences } from './glass'

// 显示偏好由后端持久化，前端用字面量联合约束可写入的 token 值。
export type ThemeMode = 'light' | 'dark'
// DisplaySize 对应 Element Plus 全局组件尺寸（el-config-provider size）。
export type DisplaySize = 'large' | 'default' | 'small'

export type DisplayPreferences = {
  backdrop: boolean
  lgIntensity: number
  lgStyle: GlassStyle
  size: DisplaySize
  themeMode: ThemeMode
}

// IncomingDisplayPreferences 是后端/preview 快照的宽松形态：字段可能缺失（老数据没有玻璃三项）。
export type IncomingDisplayPreferences = {
  backdrop?: boolean
  lgIntensity?: number
  lgStyle?: string
  size?: string
  themeMode?: string
}

// 前端默认值用于冷启动和 preview fallback；真实运行时会被后端持久化值覆盖。
export const displayPreferenceDefaults: DisplayPreferences = {
  themeMode: 'light',
  size: 'default',
  backdrop: false,
  lgStyle: 'fresnel',
  lgIntensity: 75,
}

// 模块级响应式状态让 AppChrome、SettingsPage、App.vue 使用同一份显示偏好。
const themeMode = ref<ThemeMode>(displayPreferenceDefaults.themeMode)
const size = ref<DisplaySize>(displayPreferenceDefaults.size)

// 对组件暴露只读 ref 和显式 setter，避免页面直接改模块级状态。
export function useDisplayPreferences() {
  return {
    resetDisplayPreferences,
    setSize,
    setThemeMode,
    size: readonly(size),
    themeMode: readonly(themeMode),
  }
}

export function setThemeMode(value: ThemeMode) {
  themeMode.value = value
}

export function setSize(value: DisplaySize) {
  size.value = value
}

// 恢复全局默认显示偏好：主题、尺寸与液态玻璃光学一起回到设计稿默认档。
export function resetDisplayPreferences() {
  themeMode.value = displayPreferenceDefaults.themeMode
  size.value = displayPreferenceDefaults.size
  resetGlassPreferences()
}

// 从后端/preview store 注入显示偏好；非法值回退到默认值。
export function hydrateDisplayPreferences(preferences?: IncomingDisplayPreferences) {
  if (!preferences) return
  themeMode.value = normaliseValue(preferences.themeMode, isThemeMode, displayPreferenceDefaults.themeMode)
  size.value = normaliseValue(preferences.size, isDisplaySize, displayPreferenceDefaults.size)
  hydrateGlassPreferences(preferences)
}

// 导出当前可持久化快照（五项轴，字段名与 Go 侧 PreferencesV3 的 JSON tag 一致）。
export function exportDisplayPreferences(): DisplayPreferences {
  return {
    themeMode: themeMode.value,
    size: size.value,
    ...exportGlassPreferences(),
  }
}

// 以下 watch 只同步 DOM；持久化由 SettingsPage/store 显式触发。
watch(themeMode, (value) => {
  // html.dark 同时驱动 Element Plus 暗色变量和 Tailwind dark: 变体。
  documentRoot()?.classList.toggle('dark', value === 'dark')
}, { immediate: true })

watch(size, (value) => {
  // 字体、布局边距、卡片内边距的缩放由 styles.css 的 [data-display-size] 档位解释；
  // EP 控件尺寸另经 el-config-provider 同步下发。
  const root = documentRoot()
  if (root) root.dataset.displaySize = value
}, { immediate: true })

// normaliseValue 统一校验持久化读取到的显示偏好值，非法值会回退到类型安全的默认值。
function normaliseValue<T extends string>(value: string | undefined, guard: (value: string | null) => value is T, fallback: T) {
  const candidate = typeof value === 'string' ? value : null
  return guard(candidate) ? candidate : fallback
}

// SSR/测试环境可能没有 document，DOM token 同步在这种情况下静默跳过。
function documentRoot() {
  if (typeof document === 'undefined') return undefined
  return document.documentElement
}

function isThemeMode(value: string | null): value is ThemeMode {
  return value === 'light' || value === 'dark'
}

function isDisplaySize(value: string | null): value is DisplaySize {
  return value === 'large' || value === 'default' || value === 'small'
}
