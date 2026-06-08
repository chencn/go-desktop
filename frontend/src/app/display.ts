// 文件职责：定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

import { readonly, ref, watch } from 'vue'

// ThemeMode 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type ThemeMode = 'light' | 'dark'
// DisplayScheme 定义当前显示偏好方案，shadcn 保留全量自定义，antd 使用 Ant Design 托管项。
export type DisplayScheme = 'shadcn' | 'antd'
// TextSize 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type TextSize = 'small' | 'normal' | 'medium' | 'large'
// UIStyle 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type UIStyle = 'reka' | 'vega' | 'nova' | 'maia' | 'lyra' | 'mira' | 'luma' | 'sera'
// BaseColor 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type BaseColor = 'neutral' | 'stone' | 'zinc' | 'mauve' | 'olive' | 'mist' | 'taupe'
// AccentColor 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type AccentColor = 'neutral' | 'stone' | 'zinc' | 'mauve' | 'olive' | 'mist' | 'taupe' | 'amber' | 'blue' | 'cyan' | 'emerald' | 'fuchsia' | 'green' | 'indigo' | 'lime' | 'orange' | 'pink' | 'purple' | 'red' | 'rose' | 'sky' | 'teal' | 'violet' | 'yellow'
// ThemeColor 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type ThemeColor = AccentColor
// ChartColor 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type ChartColor = AccentColor
// IconTone 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type IconTone = 'default' | 'colorful'
// Menu 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type Menu = 'default' | 'inverted' | 'default-translucent' | 'inverted-translucent'
// MenuAccent 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type MenuAccent = 'subtle' | 'bold'
// Radius 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type Radius = 'default' | 'none' | 'small' | 'medium' | 'large'
// Density 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type Density = 'compact' | 'comfortable'
// CardBorder 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type CardBorder = 'visible' | 'soft' | 'hidden'

export type DisplayProfile = {
  accentColor: AccentColor
  baseColor: BaseColor
  cardBorder: CardBorder
  chartColor: ChartColor
  density: Density
  iconTone: IconTone
  menu: Menu
  menuAccent: MenuAccent
  radius: Radius
  textSize: TextSize
  themeColor: ThemeColor
  uiStyle: UIStyle
}

export type DisplayProfiles = Record<DisplayScheme, DisplayProfile>

// DisplayPreferences 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type DisplayPreferences = {
  accentColor: AccentColor
  baseColor: BaseColor
  cardBorder: CardBorder
  chartColor: ChartColor
  density: Density
  displayScheme: DisplayScheme
  iconTone: IconTone
  menu: Menu
  menuAccent: MenuAccent
  radius: Radius
  textSize: TextSize
  themeColor: ThemeColor
  themeMode: ThemeMode
  uiStyle: UIStyle
  profiles: DisplayProfiles
}

type IncomingDisplayProfile = Partial<Record<keyof DisplayProfile, string | undefined>>
type IncomingDisplayPreferences = Partial<Record<Exclude<keyof DisplayPreferences, 'profiles'>, string | undefined>> & {
  profiles?: Partial<Record<DisplayScheme, IncomingDisplayProfile>>
}

const shadcnProfileDefaults = {
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
} satisfies DisplayProfile

const antDesignProfileDefaults = {
  accentColor: 'blue',
  baseColor: 'neutral',
  cardBorder: 'visible',
  chartColor: 'blue',
  density: 'comfortable',
  iconTone: 'default',
  menu: 'default',
  menuAccent: 'subtle',
  radius: 'medium',
  textSize: 'normal',
  themeColor: 'blue',
  uiStyle: 'vega',
} satisfies DisplayProfile

// displayPreferenceDefaults 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
export const displayPreferenceDefaults = {
  ...shadcnProfileDefaults,
  displayScheme: 'shadcn',
  themeMode: 'light',
  profiles: {
    shadcn: { ...shadcnProfileDefaults },
    antd: { ...antDesignProfileDefaults },
  },
} satisfies DisplayPreferences

const displayProfiles: Record<DisplayScheme, DisplayProfile> = {
  shadcn: { ...shadcnProfileDefaults },
  antd: { ...antDesignProfileDefaults },
}

// themeMode 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const themeMode = ref<ThemeMode>(displayPreferenceDefaults.themeMode)
// displayScheme 保存当前显示偏好方案。
const displayScheme = ref<DisplayScheme>(displayPreferenceDefaults.displayScheme)
// uiStyle 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const uiStyle = ref<UIStyle>(displayPreferenceDefaults.uiStyle)
// textSize 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const textSize = ref<TextSize>(displayPreferenceDefaults.textSize)
// baseColor 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const baseColor = ref<BaseColor>(displayPreferenceDefaults.baseColor)
// themeColor 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const themeColor = ref<ThemeColor>(displayPreferenceDefaults.themeColor)
// accentColor 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const accentColor = ref<AccentColor>(displayPreferenceDefaults.accentColor)
// chartColor 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const chartColor = ref<ChartColor>(displayPreferenceDefaults.chartColor)
// iconTone 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const iconTone = ref<IconTone>(displayPreferenceDefaults.iconTone)
// menu 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const menu = ref<Menu>(displayPreferenceDefaults.menu)
// menuAccent 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const menuAccent = ref<MenuAccent>(displayPreferenceDefaults.menuAccent)
// radius 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const radius = ref<Radius>(displayPreferenceDefaults.radius)
// density 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const density = ref<Density>(displayPreferenceDefaults.density)
// cardBorder 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const cardBorder = ref<CardBorder>(displayPreferenceDefaults.cardBorder)

// useDisplayPreferences 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function useDisplayPreferences() {
  return {
    accentColor: readonly(accentColor),
    baseColor: readonly(baseColor),
    cardBorder: readonly(cardBorder),
    chartColor: readonly(chartColor),
    density: readonly(density),
    displayScheme: readonly(displayScheme),
    iconTone: readonly(iconTone),
    menu: readonly(menu),
    menuAccent: readonly(menuAccent),
    radius: readonly(radius),
    resetDisplayPreferences,
    resetDisplayPreferencesForCurrentScheme,
    setAccentColor,
    setBaseColor,
    setCardBorder,
    setChartColor,
    setDensity,
    setDisplayScheme,
    setIconTone,
    setMenu,
    setMenuAccent,
    setRadius,
    setTextSize,
    setThemeColor,
    setThemeMode,
    setUiStyle,
    textSize: readonly(textSize),
    themeColor: readonly(themeColor),
    themeMode: readonly(themeMode),
    uiStyle: readonly(uiStyle),
  }
}

// setThemeMode 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setThemeMode(value: ThemeMode) {
  themeMode.value = value
}

// setDisplayScheme 切换当前显示偏好方案。
export function setDisplayScheme(value: DisplayScheme) {
  if (displayScheme.value === value) return
  rememberCurrentProfile(displayScheme.value)
  displayScheme.value = value
  applyProfile(displayProfiles[value])
}

// setUiStyle 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setUiStyle(value: UIStyle) {
  uiStyle.value = value
}

// setTextSize 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setTextSize(value: TextSize) {
  textSize.value = value
}

// setBaseColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setBaseColor(value: BaseColor) {
  baseColor.value = value
}

// setThemeColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setThemeColor(value: ThemeColor) {
  themeColor.value = value
}

// setAccentColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setAccentColor(value: AccentColor) {
  accentColor.value = value
}

// setChartColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setChartColor(value: ChartColor) {
  chartColor.value = value
}

// setIconTone 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setIconTone(value: IconTone) {
  iconTone.value = value
}

// setMenu 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setMenu(value: Menu) {
  menu.value = value
}

// setMenuAccent 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setMenuAccent(value: MenuAccent) {
  menuAccent.value = value
}

// setRadius 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setRadius(value: Radius) {
  radius.value = value
}

// setDensity 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setDensity(value: Density) {
  density.value = value
}

// setCardBorder 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function setCardBorder(value: CardBorder) {
  cardBorder.value = value
}

// resetDisplayPreferences 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function resetDisplayPreferences() {
  displayScheme.value = displayPreferenceDefaults.displayScheme
  applyProfile(shadcnProfileDefaults)
  setThemeMode(displayPreferenceDefaults.themeMode)
  rememberCurrentProfile(displayScheme.value)
}

// resetDisplayPreferencesForCurrentScheme 按当前显示方案恢复默认值，保留全局亮暗模式。
export function resetDisplayPreferencesForCurrentScheme() {
  const mode = themeMode.value
  if (displayScheme.value === 'antd') {
    applyProfile(antDesignProfileDefaults)
    setThemeMode(mode)
    rememberCurrentProfile('antd')
    return
  }
  applyProfile(shadcnProfileDefaults)
  setThemeMode(mode)
  rememberCurrentProfile('shadcn')
}

// hydrateDisplayPreferences 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function hydrateDisplayPreferences(preferences?: IncomingDisplayPreferences) {
  if (!preferences) return
  const nextScheme = normaliseValue(preferences.displayScheme, isDisplayScheme, displayPreferenceDefaults.displayScheme)
  const nextProfiles = normaliseProfiles(preferences, nextScheme)
  displayProfiles.shadcn = nextProfiles.shadcn
  displayProfiles.antd = nextProfiles.antd
  displayScheme.value = nextScheme
  applyProfile(displayProfiles[nextScheme])
  setThemeMode(normaliseValue(preferences.themeMode, isThemeMode, displayPreferenceDefaults.themeMode))
}

// exportDisplayPreferences 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
export function exportDisplayPreferences(): DisplayPreferences {
  rememberCurrentProfile(displayScheme.value)
  return {
    accentColor: accentColor.value,
    baseColor: baseColor.value,
    cardBorder: cardBorder.value,
    chartColor: chartColor.value,
    density: density.value,
    displayScheme: displayScheme.value,
    iconTone: iconTone.value,
    menu: menu.value,
    menuAccent: menuAccent.value,
    radius: radius.value,
    textSize: textSize.value,
    themeColor: themeColor.value,
    themeMode: themeMode.value,
    uiStyle: uiStyle.value,
    profiles: {
      shadcn: cloneProfile(displayProfiles.shadcn),
      antd: cloneProfile(displayProfiles.antd),
    },
  }
}

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(themeMode, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (!root) return
  root.classList.toggle('dark', value === 'dark')
  root.dataset.theme = value === 'dark' ? 'night' : 'day'
}, { immediate: true })

// watch 监听显示方案，并写入 DOM dataset 供 CSS token 层使用。
watch(displayScheme, (value) => {
  const root = documentRoot()
  if (root) root.dataset.displayScheme = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(uiStyle, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.style = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(textSize, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.textSize = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(baseColor, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.baseColor = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(themeColor, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.themeColor = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(accentColor, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.accentColor = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(chartColor, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.chartColor = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(iconTone, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.iconTone = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(menu, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.menu = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(menuAccent, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.menuAccent = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(radius, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) {
    root.dataset.radius = value
    root.style.setProperty('--radius', radiusValue(value))
  }
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(density, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.density = value
}, { immediate: true })

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(cardBorder, (value) => {
  // root 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const root = documentRoot()
  if (root) root.dataset.cardBorder = value
}, { immediate: true })

// normaliseValue 统一校验持久化读取到的显示偏好值，非法值会回退到类型安全的默认值。
function normaliseValue<T extends string>(value: string | undefined, guard: (value: string | null) => value is T, fallback: T) {
  // candidate 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const candidate = typeof value === 'string' ? value : null
  return guard(candidate) ? candidate : fallback
}

function toProfile(preferences: DisplayPreferences): DisplayProfile {
  return {
    accentColor: preferences.accentColor,
    baseColor: preferences.baseColor,
    cardBorder: preferences.cardBorder,
    chartColor: preferences.chartColor,
    density: preferences.density,
    iconTone: preferences.iconTone,
    menu: preferences.menu,
    menuAccent: preferences.menuAccent,
    radius: preferences.radius,
    textSize: preferences.textSize,
    themeColor: preferences.themeColor,
    uiStyle: preferences.uiStyle,
  }
}

function currentProfile(): DisplayProfile {
  return {
    accentColor: accentColor.value,
    baseColor: baseColor.value,
    cardBorder: cardBorder.value,
    chartColor: chartColor.value,
    density: density.value,
    iconTone: iconTone.value,
    menu: menu.value,
    menuAccent: menuAccent.value,
    radius: radius.value,
    textSize: textSize.value,
    themeColor: themeColor.value,
    uiStyle: uiStyle.value,
  }
}

function profileFallback(scheme: DisplayScheme) {
  return scheme === 'antd' ? antDesignProfileDefaults : shadcnProfileDefaults
}

function rememberCurrentProfile(scheme: DisplayScheme) {
  displayProfiles[scheme] = currentProfile()
}

function applyProfile(profile: DisplayProfile) {
  setAccentColor(profile.accentColor)
  setBaseColor(profile.baseColor)
  setCardBorder(profile.cardBorder)
  setChartColor(profile.chartColor)
  setDensity(profile.density)
  setIconTone(profile.iconTone)
  setMenu(profile.menu)
  setMenuAccent(profile.menuAccent)
  setRadius(profile.radius)
  setTextSize(profile.textSize)
  setThemeColor(profile.themeColor)
  setUiStyle(profile.uiStyle)
}

function normaliseProfiles(preferences: IncomingDisplayPreferences, scheme: DisplayScheme): DisplayProfiles {
  const explicitShadcn = preferences.profiles?.shadcn
  const explicitAntD = preferences.profiles?.antd
  const flatProfile = normaliseProfile(preferences, scheme)
  return {
    shadcn: normaliseProfile(explicitShadcn ?? (scheme === 'shadcn' ? flatProfile : displayProfiles.shadcn), 'shadcn'),
    antd: normaliseProfile(explicitAntD ?? (scheme === 'antd' ? flatProfile : displayProfiles.antd), 'antd'),
  }
}

function normaliseProfile(profile: IncomingDisplayProfile, scheme: DisplayScheme): DisplayProfile {
  const fallback = profileFallback(scheme)
  const menuGuard = scheme === 'antd' ? isAntDesignMenu : isMenu
  const radiusGuard = scheme === 'antd' ? isAntDesignRadius : isRadius
  return {
    accentColor: normaliseValue(profile.accentColor, isAccentColor, fallback.accentColor),
    baseColor: normaliseValue(profile.baseColor, isBaseColor, fallback.baseColor),
    cardBorder: normaliseValue(profile.cardBorder, isCardBorder, fallback.cardBorder),
    chartColor: normaliseValue(profile.chartColor, isChartColor, fallback.chartColor),
    density: normaliseValue(profile.density, isDensity, fallback.density),
    iconTone: normaliseValue(profile.iconTone, isIconTone, fallback.iconTone),
    menu: normaliseValue(profile.menu, menuGuard, fallback.menu),
    menuAccent: normaliseValue(profile.menuAccent, isMenuAccent, fallback.menuAccent),
    radius: normaliseValue(profile.radius, radiusGuard, fallback.radius),
    textSize: normaliseValue(profile.textSize, isTextSize, fallback.textSize),
    themeColor: normaliseValue(profile.themeColor, isThemeColor, fallback.themeColor),
    uiStyle: normaliseValue(profile.uiStyle, isUIStyle, fallback.uiStyle),
  }
}

function cloneProfile(profile: DisplayProfile): DisplayProfile {
  return { ...profile }
}

// documentRoot 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function documentRoot() {
  if (typeof document === 'undefined') return undefined
  return document.documentElement
}

// radiusValue 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function radiusValue(value: Radius) {
  // values 保存 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 使用的配置、引用或中间结果。
  const values: Record<Radius, string> = {
    default: '0.625rem',
    none: '0rem',
    small: '0.375rem',
    medium: '0.625rem',
    large: '0.875rem',
  }
  return values[value]
}

// isTextSize 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isTextSize(value: string | null): value is TextSize {
  return value === 'small' || value === 'normal' || value === 'medium' || value === 'large'
}

// isThemeMode 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isThemeMode(value: string | null): value is ThemeMode {
  return value === 'light' || value === 'dark'
}

// isDisplayScheme 校验显示方案。
function isDisplayScheme(value: string | null): value is DisplayScheme {
  return value === 'shadcn' || value === 'antd'
}

// isUIStyle 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isUIStyle(value: string | null): value is UIStyle {
  return value === 'reka' || value === 'vega' || value === 'nova' || value === 'maia' || value === 'lyra' || value === 'mira' || value === 'luma' || value === 'sera'
}

// isBaseColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isBaseColor(value: string | null): value is BaseColor {
  return value === 'neutral' || value === 'stone' || value === 'zinc' || value === 'mauve' || value === 'olive' || value === 'mist' || value === 'taupe'
}

// isAccentColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isAccentColor(value: string | null): value is AccentColor {
  return value === 'neutral' || value === 'stone' || value === 'zinc' || value === 'mauve' || value === 'olive' || value === 'mist' || value === 'taupe' || value === 'amber' || value === 'blue' || value === 'cyan' || value === 'emerald' || value === 'fuchsia' || value === 'green' || value === 'indigo' || value === 'lime' || value === 'orange' || value === 'pink' || value === 'purple' || value === 'red' || value === 'rose' || value === 'sky' || value === 'teal' || value === 'violet' || value === 'yellow'
}

// isThemeColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isThemeColor(value: string | null): value is ThemeColor {
  return isAccentColor(value)
}

// isChartColor 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isChartColor(value: string | null): value is ChartColor {
  return isAccentColor(value)
}

// isIconTone 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isIconTone(value: string | null): value is IconTone {
  return value === 'default' || value === 'colorful'
}

// isMenu 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isMenu(value: string | null): value is Menu {
  return value === 'default' || value === 'inverted' || value === 'default-translucent' || value === 'inverted-translucent'
}

function isAntDesignMenu(value: string | null): value is Menu {
  return value === 'default' || value === 'inverted'
}

// isMenuAccent 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isMenuAccent(value: string | null): value is MenuAccent {
  return value === 'subtle' || value === 'bold'
}

// isRadius 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isRadius(value: string | null): value is Radius {
  return value === 'default' || value === 'none' || value === 'small' || value === 'medium' || value === 'large'
}

function isAntDesignRadius(value: string | null): value is Radius {
  return value === 'default' || value === 'small' || value === 'medium' || value === 'large'
}

// isDensity 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isDensity(value: string | null): value is Density {
  return value === 'compact' || value === 'comfortable'
}

// isCardBorder 处理 定义前端页面展示所需的统计卡、表单项、状态文案和颜色映射 中的用户动作、生命周期动作或数据转换。
function isCardBorder(value: string | null): value is CardBorder {
  return value === 'visible' || value === 'soft' || value === 'hidden'
}
