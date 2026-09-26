// 文件职责：维护液态玻璃光学偏好（极光舞台 / 折射风格 / 光强）、DOM 同步与指针随动光学。
// 这三项与亮暗模式、控件尺寸同属显示偏好，统一由 app/display.ts 走后端 `display.preferences.v3` 持久化，
// 本模块不再写 localStorage 之类的浏览器本地副本，避免出现「一份偏好两个存储」的分裂状态。

import { onBeforeUnmount, onMounted, readonly, ref, watch } from 'vue'

export type GlassStyle = 'fresnel' | 'frosted' | 'sheen'

export type GlassPreferences = {
  backdrop: boolean
  intensity: number
  style: GlassStyle
}

// 光强口径与设计稿滑块一致：30–100，默认 75；后端 display.GlassIntensityMin/Max 使用同一区间。
export const glassIntensityMin = 30
export const glassIntensityMax = 100

export const glassPreferenceDefaults: GlassPreferences = {
  backdrop: false,
  intensity: 75,
  style: 'fresnel',
}

// 指针静止位：没有指针输入时高光停在左上，与设计稿 rAF 引擎的 rest position 相同。
const pointerRest = { x: 28, y: 12 }
// 阻尼混合步长（ms）：值越大跟随越慢，取自 D:\liquid-glass 的 pointerOptics。
const pointerMixTau = 62

const backdrop = ref(glassPreferenceDefaults.backdrop)
const intensity = ref(glassPreferenceDefaults.intensity)
const style = ref<GlassStyle>(glassPreferenceDefaults.style)

export function useGlassPreferences() {
  return {
    backdrop: readonly(backdrop),
    intensity: readonly(intensity),
    resetGlassPreferences,
    setBackdrop,
    setIntensity,
    setStyle,
    style: readonly(style),
  }
}

export function setBackdrop(value: boolean) {
  backdrop.value = value
}

export function setStyle(value: GlassStyle) {
  style.value = value
}

// setIntensity 把光强夹到合法区间，避免外部传入越界值写进 CSS 变量。
export function setIntensity(value: number) {
  if (!Number.isFinite(value)) return
  intensity.value = Math.min(glassIntensityMax, Math.max(glassIntensityMin, Math.round(value)))
}

// 恢复设计稿 resetAesthetic 的光学默认值（不动主题与控件尺寸）。
export function resetGlassPreferences() {
  backdrop.value = glassPreferenceDefaults.backdrop
  intensity.value = glassPreferenceDefaults.intensity
  style.value = glassPreferenceDefaults.style
}

// IncomingGlassPreferences 是后端快照里的光学三项；类型放宽以便同一份归一化逻辑服务 preview 兜底。
export type IncomingGlassPreferences = {
  backdrop?: boolean
  lgIntensity?: number
  lgStyle?: string
}

// hydrateGlassPreferences 从后端/preview 快照注入光学偏好，非法值逐项回退默认。
// 后端是权威口径（Go 侧 display.NormalizeV3 同样逐项校验），这里再校验一次是为了兼容旧 SQLite 数据缺字段。
export function hydrateGlassPreferences(preferences?: IncomingGlassPreferences) {
  if (!preferences) return
  if (typeof preferences.backdrop === 'boolean') backdrop.value = preferences.backdrop
  setStyle(normaliseGlassStyle(preferences.lgStyle))
  setIntensity(normaliseGlassIntensity(preferences.lgIntensity))
}

// exportGlassPreferences 导出当前可持久化快照，字段名与后端 JSON 一致。
export function exportGlassPreferences(): { backdrop: boolean; lgIntensity: number; lgStyle: GlassStyle } {
  return {
    backdrop: backdrop.value,
    lgIntensity: intensity.value,
    lgStyle: style.value,
  }
}

function normaliseGlassStyle(value?: string): GlassStyle {
  return isGlassStyle(value) ? value : glassPreferenceDefaults.style
}

// normaliseGlassIntensity 只接受区间内的整数；区间外（含旧数据缺字段得到的 0 / undefined）回到默认档。
function normaliseGlassIntensity(value?: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return glassPreferenceDefaults.intensity
  if (value < glassIntensityMin || value > glassIntensityMax) return glassPreferenceDefaults.intensity
  return Math.round(value)
}

// 以下 watch 只做 DOM 同步：材质层的属性全部读 CSS 变量，因此改写变量即可换档。
watch(
  [style, intensity, backdrop],
  () => {
    const root = documentRoot()
    if (!root) return
    root.dataset.lgStyle = style.value
    root.style.setProperty('--lg-intensity-factor', String(intensity.value / 100))
    root.classList.toggle('has-backdrop', backdrop.value)
  },
  { immediate: true },
)

function isGlassStyle(value?: string): value is GlassStyle {
  return value === 'fresnel' || value === 'frosted' || value === 'sheen'
}

function documentRoot(): HTMLElement | undefined {
  if (typeof document === 'undefined') return undefined
  return document.documentElement
}

/**
 * useGlassPointerOptics 驱动卡片表面的随动高光：指数衰减阻尼插值写 --lg-pointer-x / --lg-pointer-y。
 * 极光舞台关闭时不追踪，卡片与侧栏菜单一样保持纯净无色散晶体。
 */
export function useGlassPointerOptics(host: () => HTMLElement | undefined) {
  let currentX = pointerRest.x
  let currentY = pointerRest.y
  let targetX = pointerRest.x
  let targetY = pointerRest.y
  let rafId: number | undefined
  let lastTime = 0

  function updateOptics(time: number) {
    if (!lastTime) lastTime = time
    const dt = Math.min(64, time - lastTime)
    lastTime = time
    const mix = 1 - Math.exp(-dt / pointerMixTau)
    currentX += (targetX - currentX) * mix
    currentY += (targetY - currentY) * mix

    const el = host()
    if (!el) {
      rafId = undefined
      return
    }
    el.style.setProperty('--lg-pointer-x', `${currentX.toFixed(2)}%`)
    el.style.setProperty('--lg-pointer-y', `${currentY.toFixed(2)}%`)

    if (Math.abs(targetX - currentX) > 0.05 || Math.abs(targetY - currentY) > 0.05) {
      rafId = requestAnimationFrame(updateOptics)
    } else {
      rafId = undefined
    }
  }

  function scheduleUpdate() {
    if (rafId !== undefined) return
    lastTime = 0
    rafId = requestAnimationFrame(updateOptics)
  }

  function onPointerMove(event: PointerEvent) {
    if (!backdrop.value) return
    const el = host()
    if (!el) return
    const rect = el.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) return
    targetX = Math.min(100, Math.max(0, ((event.clientX - rect.left) / rect.width) * 100))
    targetY = Math.min(100, Math.max(0, ((event.clientY - rect.top) / rect.height) * 100))
    scheduleUpdate()
  }

  function onPointerLeave() {
    targetX = pointerRest.x
    targetY = pointerRest.y
    scheduleUpdate()
  }

  function applyPointerTracking(active: boolean) {
    const el = host()
    // setup 阶段模板 ref 还没赋值，首次挂载靠 onMounted 再补一次；addEventListener 同名同引用可重复调用。
    if (!el) return
    if (active) {
      el.addEventListener('pointermove', onPointerMove)
      el.addEventListener('pointerleave', onPointerLeave)
      return
    }
    el.removeEventListener('pointermove', onPointerMove)
    el.removeEventListener('pointerleave', onPointerLeave)
    // 关闭极光时立刻清掉随动量，高光位置回到静止位，避免残留光斑。
    el.style.removeProperty('--lg-pointer-x')
    el.style.removeProperty('--lg-pointer-y')
  }

  watch(backdrop, applyPointerTracking, { immediate: true })
  onMounted(() => applyPointerTracking(backdrop.value))
  onBeforeUnmount(() => {
    if (rafId !== undefined) cancelAnimationFrame(rafId)
  })
}
