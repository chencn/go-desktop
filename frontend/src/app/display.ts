// 文件职责：维护显示偏好 token、方案 profile、DOM dataset 同步和持久化快照转换。

import { readonly, ref, watch } from 'vue'

// 显示偏好由后端持久化，前端用字面量联合约束可写入的 token 值。
export type ThemeMode = 'light' | 'dark'
// DisplayScheme 定义当前显示偏好方案，shadcn 保留经典中性外观，artistic 使用主题样式扩展。
export type DisplayScheme = 'shadcn' | 'artistic'
export type TextSize = 'small' | 'normal' | 'medium' | 'large'
export type UIStyle = 'reka' | 'vega' | 'nova' | 'maia' | 'lyra' | 'mira' | 'luma' | 'sera'
export type BaseColor = 'neutral' | 'stone' | 'zinc' | 'mauve' | 'olive' | 'mist' | 'taupe'
export type AccentColor = 'neutral' | 'stone' | 'zinc' | 'mauve' | 'olive' | 'mist' | 'taupe' | 'amber' | 'blue' | 'cyan' | 'emerald' | 'fuchsia' | 'green' | 'indigo' | 'lime' | 'orange' | 'pink' | 'purple' | 'red' | 'rose' | 'sky' | 'teal' | 'violet' | 'yellow'
export type ThemeColor = AccentColor
export type ChartColor = AccentColor
export type IconTone = 'default' | 'colorful'
export type Menu = 'default' | 'inverted' | 'default-translucent' | 'inverted-translucent'
export type MenuAccent = 'subtle' | 'bold'
export type Radius = 'default' | 'none' | 'small' | 'medium' | 'large'
export type Density = 'compact' | 'comfortable'
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

// DisplayPreferences 同时保存当前方案的平铺字段和各方案独立 profile。
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

const artisticProfileDefaults = {
  accentColor: 'orange',
  baseColor: 'stone',
  cardBorder: 'soft',
  chartColor: 'emerald',
  density: 'comfortable',
  iconTone: 'colorful',
  menu: 'default',
  menuAccent: 'bold',
  radius: 'large',
  textSize: 'normal',
  themeColor: 'orange',
  uiStyle: 'vega',
} satisfies DisplayProfile

// 前端默认值用于冷启动和 preview fallback；真实运行时会被后端持久化值覆盖。
export const displayPreferenceDefaults = {
  ...artisticProfileDefaults,
  displayScheme: 'artistic',
  themeMode: 'light',
  profiles: {
    shadcn: { ...shadcnProfileDefaults },
    artistic: { ...artisticProfileDefaults },
  },
} satisfies DisplayPreferences

const displayProfiles: Record<DisplayScheme, DisplayProfile> = {
  shadcn: { ...shadcnProfileDefaults },
  artistic: { ...artisticProfileDefaults },
}

// 模块级响应式状态让 AppChrome、SettingsPage 和 CSS dataset 使用同一份显示偏好。
const themeMode = ref<ThemeMode>(displayPreferenceDefaults.themeMode)
// displayScheme 保存当前显示偏好方案。
const displayScheme = ref<DisplayScheme>(displayPreferenceDefaults.displayScheme)
const uiStyle = ref<UIStyle>(displayPreferenceDefaults.uiStyle)
const textSize = ref<TextSize>(displayPreferenceDefaults.textSize)
const baseColor = ref<BaseColor>(displayPreferenceDefaults.baseColor)
const themeColor = ref<ThemeColor>(displayPreferenceDefaults.themeColor)
const accentColor = ref<AccentColor>(displayPreferenceDefaults.accentColor)
const chartColor = ref<ChartColor>(displayPreferenceDefaults.chartColor)
const iconTone = ref<IconTone>(displayPreferenceDefaults.iconTone)
const menu = ref<Menu>(displayPreferenceDefaults.menu)
const menuAccent = ref<MenuAccent>(displayPreferenceDefaults.menuAccent)
const radius = ref<Radius>(displayPreferenceDefaults.radius)
const density = ref<Density>(displayPreferenceDefaults.density)
const cardBorder = ref<CardBorder>(displayPreferenceDefaults.cardBorder)

// 对组件暴露只读 ref 和显式 setter，避免页面直接改模块级状态。
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

export function setUiStyle(value: UIStyle) {
  uiStyle.value = value
}

export function setTextSize(value: TextSize) {
  textSize.value = value
}

export function setBaseColor(value: BaseColor) {
  baseColor.value = value
}

export function setThemeColor(value: ThemeColor) {
  themeColor.value = value
  if (displayScheme.value === 'artistic') {
    accentColor.value = value
  }
}

export function setAccentColor(value: AccentColor) {
  accentColor.value = displayScheme.value === 'artistic' ? themeColor.value : value
}

export function setChartColor(value: ChartColor) {
  chartColor.value = value
}

export function setIconTone(value: IconTone) {
  iconTone.value = value
}

export function setMenu(value: Menu) {
  menu.value = value
}

export function setMenuAccent(value: MenuAccent) {
  menuAccent.value = value
}

export function setRadius(value: Radius) {
  radius.value = value
}

export function setDensity(value: Density) {
  density.value = value
}

export function setCardBorder(value: CardBorder) {
  cardBorder.value = value
}

// 恢复全局默认：回到当前默认方案，同时重置亮暗模式。
export function resetDisplayPreferences() {
  displayProfiles.shadcn = { ...shadcnProfileDefaults }
  displayProfiles.artistic = { ...artisticProfileDefaults }
  displayScheme.value = displayPreferenceDefaults.displayScheme
  applyProfile(profileFallback(displayPreferenceDefaults.displayScheme))
  setThemeMode(displayPreferenceDefaults.themeMode)
  rememberCurrentProfile(displayScheme.value)
}

// resetDisplayPreferencesForCurrentScheme 按当前显示方案恢复默认值，保留全局亮暗模式。
export function resetDisplayPreferencesForCurrentScheme() {
  const mode = themeMode.value
  if (displayScheme.value === 'artistic') {
    applyProfile(artisticProfileDefaults)
    setThemeMode(mode)
    rememberCurrentProfile('artistic')
    return
  }
  applyProfile(shadcnProfileDefaults)
  setThemeMode(mode)
  rememberCurrentProfile('shadcn')
}

// 从后端/preview store 注入显示偏好；非法值按当前方案回退到默认 profile。
export function hydrateDisplayPreferences(preferences?: IncomingDisplayPreferences) {
  if (!preferences) return
  const nextScheme = normaliseValue(preferences.displayScheme, isDisplayScheme, displayPreferenceDefaults.displayScheme)
  const nextProfiles = normaliseProfiles(preferences, nextScheme)
  displayProfiles.shadcn = nextProfiles.shadcn
  displayProfiles.artistic = nextProfiles.artistic
  displayScheme.value = nextScheme
  applyProfile(displayProfiles[nextScheme])
  setThemeMode(normaliseValue(preferences.themeMode, isThemeMode, displayPreferenceDefaults.themeMode))
}

// 导出当前可持久化快照；切换方案前先记住当前方案 profile。
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
      artistic: cloneProfile(displayProfiles.artistic),
    },
  }
}

// 以下 watch 只同步 DOM dataset / CSS 变量；持久化由 SettingsPage/store 显式触发。
watch(themeMode, (value) => {
  const root = documentRoot()
  if (!root) return
  root.classList.toggle('dark', value === 'dark')
  root.dataset.theme = value === 'dark' ? 'night' : 'day'
}, { immediate: true })

watch(displayScheme, (value) => {
  const root = documentRoot()
  if (root) root.dataset.displayScheme = value
}, { immediate: true })

watch(uiStyle, (value) => {
  const root = documentRoot()
  if (root) root.dataset.style = value
}, { immediate: true })

watch(textSize, (value) => {
  const root = documentRoot()
  if (root) root.dataset.textSize = value
}, { immediate: true })

watch(baseColor, (value) => {
  const root = documentRoot()
  if (root) root.dataset.baseColor = value
}, { immediate: true })

watch(themeColor, (value) => {
  const root = documentRoot()
  if (root) root.dataset.themeColor = value
}, { immediate: true })

watch(accentColor, (value) => {
  const root = documentRoot()
  if (root) root.dataset.accentColor = value
}, { immediate: true })

watch(chartColor, (value) => {
  const root = documentRoot()
  if (root) root.dataset.chartColor = value
}, { immediate: true })

watch(iconTone, (value) => {
  const root = documentRoot()
  if (root) root.dataset.iconTone = value
}, { immediate: true })

watch(menu, (value) => {
  const root = documentRoot()
  if (root) root.dataset.menu = value
}, { immediate: true })

watch(menuAccent, (value) => {
  const root = documentRoot()
  if (root) root.dataset.menuAccent = value
}, { immediate: true })

watch(radius, (value) => {
  const root = documentRoot()
  if (root) {
    root.dataset.radius = value
    root.style.setProperty('--radius', radiusValue(value))
  }
}, { immediate: true })

watch(density, (value) => {
  const root = documentRoot()
  if (root) root.dataset.density = value
}, { immediate: true })

watch(cardBorder, (value) => {
  const root = documentRoot()
  if (root) root.dataset.cardBorder = value
}, { immediate: true })

// normaliseValue 统一校验持久化读取到的显示偏好值，非法值会回退到类型安全的默认值。
function normaliseValue<T extends string>(value: string | undefined, guard: (value: string | null) => value is T, fallback: T) {
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
  if (scheme === 'artistic') return artisticProfileDefaults
  return shadcnProfileDefaults
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
  const explicitArtistic = preferences.profiles?.artistic
  const flatProfile = normaliseProfile(preferences, scheme)
  return {
    shadcn: normaliseProfile(explicitShadcn ?? (scheme === 'shadcn' ? flatProfile : displayProfiles.shadcn), 'shadcn'),
    artistic: normaliseProfile(explicitArtistic ?? (scheme === 'artistic' ? flatProfile : displayProfiles.artistic), 'artistic'),
  }
}

function normaliseProfile(profile: IncomingDisplayProfile, scheme: DisplayScheme): DisplayProfile {
  const fallback = profileFallback(scheme)
  const themeColorValue = normaliseValue(profile.themeColor, isThemeColor, fallback.themeColor)
  return {
    accentColor: scheme === 'artistic' ? themeColorValue : normaliseValue(profile.accentColor, isAccentColor, fallback.accentColor),
    baseColor: normaliseValue(profile.baseColor, isBaseColor, fallback.baseColor),
    cardBorder: normaliseValue(profile.cardBorder, isCardBorder, fallback.cardBorder),
    chartColor: normaliseValue(profile.chartColor, isChartColor, fallback.chartColor),
    density: normaliseValue(profile.density, isDensity, fallback.density),
    iconTone: normaliseValue(profile.iconTone, isIconTone, fallback.iconTone),
    menu: normaliseValue(profile.menu, isMenu, fallback.menu),
    menuAccent: normaliseValue(profile.menuAccent, isMenuAccent, fallback.menuAccent),
    radius: normaliseValue(profile.radius, isRadius, fallback.radius),
    textSize: normaliseValue(profile.textSize, isTextSize, fallback.textSize),
    themeColor: themeColorValue,
    uiStyle: normaliseValue(profile.uiStyle, isUIStyle, fallback.uiStyle),
  }
}

function cloneProfile(profile: DisplayProfile): DisplayProfile {
  return { ...profile }
}

// SSR/测试环境可能没有 document，DOM token 同步在这种情况下静默跳过。
function documentRoot() {
  if (typeof document === 'undefined') return undefined
  return document.documentElement
}

// Radius 字面量映射到全局 --radius，组件再由 CSS 变量派生具体圆角。
function radiusValue(value: Radius) {
  const values: Record<Radius, string> = {
    default: '0.625rem',
    none: '0rem',
    small: '0.375rem',
    medium: '0.625rem',
    large: '0.875rem',
  }
  return values[value]
}

function isTextSize(value: string | null): value is TextSize {
  return value === 'small' || value === 'normal' || value === 'medium' || value === 'large'
}

function isThemeMode(value: string | null): value is ThemeMode {
  return value === 'light' || value === 'dark'
}

// isDisplayScheme 校验显示方案。
function isDisplayScheme(value: string | null): value is DisplayScheme {
  return value === 'shadcn' || value === 'artistic'
}

function isUIStyle(value: string | null): value is UIStyle {
  return value === 'reka' || value === 'vega' || value === 'nova' || value === 'maia' || value === 'lyra' || value === 'mira' || value === 'luma' || value === 'sera'
}

function isBaseColor(value: string | null): value is BaseColor {
  return value === 'neutral' || value === 'stone' || value === 'zinc' || value === 'mauve' || value === 'olive' || value === 'mist' || value === 'taupe'
}

function isAccentColor(value: string | null): value is AccentColor {
  return value === 'neutral' || value === 'stone' || value === 'zinc' || value === 'mauve' || value === 'olive' || value === 'mist' || value === 'taupe' || value === 'amber' || value === 'blue' || value === 'cyan' || value === 'emerald' || value === 'fuchsia' || value === 'green' || value === 'indigo' || value === 'lime' || value === 'orange' || value === 'pink' || value === 'purple' || value === 'red' || value === 'rose' || value === 'sky' || value === 'teal' || value === 'violet' || value === 'yellow'
}

function isThemeColor(value: string | null): value is ThemeColor {
  return isAccentColor(value)
}

function isChartColor(value: string | null): value is ChartColor {
  return isAccentColor(value)
}

function isIconTone(value: string | null): value is IconTone {
  return value === 'default' || value === 'colorful'
}

function isMenu(value: string | null): value is Menu {
  return value === 'default' || value === 'inverted' || value === 'default-translucent' || value === 'inverted-translucent'
}

function isMenuAccent(value: string | null): value is MenuAccent {
  return value === 'subtle' || value === 'bold'
}

function isRadius(value: string | null): value is Radius {
  return value === 'default' || value === 'none' || value === 'small' || value === 'medium' || value === 'large'
}

function isDensity(value: string | null): value is Density {
  return value === 'compact' || value === 'comfortable'
}

function isCardBorder(value: string | null): value is CardBorder {
  return value === 'visible' || value === 'soft' || value === 'hidden'
}
