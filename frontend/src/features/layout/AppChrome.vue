<!--
  文件职责：渲染 macOS 风格应用外壳——顶栏、红绿灯窗口按钮、侧栏分组导航和手机端 TabBar。
  外壳只负责导航、主题/极光切换、窗口命令和更新入口；页面数据与更新生命周期由 store 维护。
  样式全部在 styles/layout.css（跨页外壳原子）与 styles/liquid-glass.css（舞台材质），本组件 CSS 只留一条胶水规则。
-->

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Maximize, Minus, Moon, PanelLeft, RefreshCw, Sparkles, Sun, X } from '@lucide/vue'
import { ElMessage } from 'element-plus'
import { Window } from '@wailsio/runtime'
import { useDisplayPreferences } from '@/app/display'
import { useGlassPreferences } from '@/app/glass'
import { toMessage } from '@/app/state'
import { useAppStore } from '@/stores/app'
import { cn } from '@/lib/utils'
import { projectMetadata } from '@/shared/project'
import { navigation, navigationGroups, pageSubtitle, pageTitle, type ViewKey } from '@/shared/views'
import UpdateStatusDialog from '@/features/update/UpdateStatusDialog.vue'

const props = defineProps<{
  // activeView 由 App.vue 持有，保证侧栏和手机端 TabBar 共享同一个页面状态。
  activeView: ViewKey
}>()

const emit = defineEmits<{
  // navigate 只通知父级切换前端页面，不触发后端路由或浏览器历史变更。
  navigate: [view: ViewKey]
}>()

const appStore = useAppStore()
// display 控制全局主题 facade；保存失败时要回滚该 facade，而不是只报错。
const display = useDisplayPreferences()
// glass 控制极光舞台与折射光学，纯本地偏好，不进后端 DisplayPreferences。
const glass = useGlassPreferences()
// updateOpen 是弹窗开关，关闭弹窗不会取消 store 中的更新任务。
const updateOpen = ref(false)
// sidebarCollapsed 只控制桌面侧栏展示密度，不进入持久化显示偏好。
const sidebarCollapsed = ref(false)
// mobileSidebarOpen 是手机端抽屉状态，与桌面折叠状态互不干扰。
const mobileSidebarOpen = ref(false)
// tabletSidebarOpen 是平板 768–1023 的浮起展开态：平时是 70px 图标轨，点开才铺满 232px 并压上遮罩。
const tabletSidebarOpen = ref(false)
// isMobile / isTablet 决定切换按钮操作抽屉、浮层还是折叠；断点必须与 layout.css 的 767px 与 1023px 一致。
const isMobile = ref(false)
const isTablet = ref(false)
// isWindowMaximised 只服务红绿灯最大化按钮的辅助文案，不作为业务状态持久化。
const isWindowMaximised = ref(false)

const activeTitle = computed(() => pageTitle(props.activeView))
const activeSubtitle = computed(() => pageSubtitle(props.activeView))
// 置顶态直接读后端设置：设计稿用它点亮顶栏徽标与窗口高程。
const isAlwaysOnTop = computed(() => appStore.settings?.alwaysOnTop === true)
// updateTone 把后端更新状态压缩成图标状态类，避免模板散落状态机判断。
const updateTone = computed(() => updateIconTone(appStore.updateStatus?.status))
const updateHint = computed(() => {
  const version = appStore.updateStatus?.version
  if (updateTone.value === 'is-ready' && version) {
    return `查看更新状态 (v${version} 可用)`
  }
  return '查看更新状态'
})
// 侧栏页脚状态灯：真实后端没有 PID，这里按数据库就绪度和错误汇总运行健康度。
const runtimeHealth = computed(() => {
  if (appStore.errorMessage) {
    return { dot: 'is-error', text: '运行异常' }
  }
  const environment = appStore.environmentInfo
  if (environment && !environment.databaseReady) {
    return { dot: 'is-warn', text: '配置存储未就绪' }
  }
  return { dot: '', text: '服务正常运行' }
})
// 页脚右侧展示真实运行环境，替代设计稿里的进程号占位。
const runtimeDetail = computed(() => {
  const environment = appStore.environmentInfo
  if (!environment) {
    return projectMetadata.appName
  }
  return `${environment.os}/${environment.arch}`
})
// H5/浏览器预览没有原生窗口运行时，窗口控制按钮只保留占位展示，不允许触发 Wails 调用。
const isWindowControlsDisabled = computed(() => {
  if (typeof window === 'undefined') return true
  return !hasNativeWindowRuntime(window as WailsHostWindow)
})

type WailsHostWindow = typeof globalThis & {
  chrome?: { webview?: { postMessage?: unknown } }
  webkit?: { messageHandlers?: { external?: { postMessage?: unknown } } }
  wails?: { invoke?: unknown }
}

let mobileQuery: MediaQueryList | undefined
let tabletQuery: MediaQueryList | undefined

function hasNativeWindowRuntime(win: WailsHostWindow) {
  return typeof win.chrome?.webview?.postMessage === 'function'
    || typeof win.webkit?.messageHandlers?.external?.postMessage === 'function'
    || typeof win.wails?.invoke === 'function'
}

// toggleTheme 先乐观切换 DOM 主题，再调用 SaveDisplayPreferences；失败后恢复原主题。
async function toggleTheme() {
  const previous = display.themeMode.value
  const next = previous === 'dark' ? 'light' : 'dark'
  display.setThemeMode(next)
  try {
    await appStore.persistDisplayPreferences()
  } catch (error) {
    display.setThemeMode(previous)
    appStore.applyAction({ type: 'errorSet', payload: toMessage(error) || '主题保存失败' })
  }
}

// toggleBackdrop 与主题同一口径：先乐观切换材质档位，再持久化显示偏好；保存失败回滚，避免界面与后端不一致。
async function toggleBackdrop() {
  const previous = glass.backdrop.value
  const next = !previous
  glass.setBackdrop(next)
  try {
    await appStore.persistDisplayPreferences()
    ElMessage.success(next ? '已开启极光流光折射背景' : '已恢复清爽极简桌面（避免视觉疲劳）')
  } catch (error) {
    glass.setBackdrop(previous)
    appStore.applyAction({ type: 'errorSet', payload: toMessage(error) || '极光背景保存失败' })
  }
}

// updateIconTone 只关心用户可感知阶段：错误、忙碌、已准备，其余保持默认。
function updateIconTone(status?: string) {
  const value = String(status ?? 'idle')
  if (value === 'error') return 'is-danger'
  if (['downloading', 'verifying', 'installing'].includes(value)) return 'is-busy'
  if (['update_available', 'verified', 'pending_install'].includes(value)) return 'is-ready'
  return ''
}

async function refreshWindowMaximised() {
  if (isWindowControlsDisabled.value) {
    isWindowMaximised.value = false
    return
  }
  try {
    isWindowMaximised.value = await Window.IsMaximised()
  } catch {
    isWindowMaximised.value = false
  }
}

async function runWindowCommand(command: () => Promise<void>) {
  if (isWindowControlsDisabled.value) return
  try {
    await command()
    await refreshWindowMaximised()
  } catch (error) {
    appStore.applyAction({ type: 'errorSet', payload: toMessage(error) || '窗口操作失败' })
  }
}

function minimiseWindow() {
  void runWindowCommand(() => Window.Minimise())
}

function toggleWindowMaximise() {
  void runWindowCommand(() => Window.ToggleMaximise())
}

function closeWindow() {
  void runWindowCommand(() => Window.Close())
}

// 断点两侧语义不同：手机拉开抽屉，平板浮起展开图标轨，桌面折叠成图标轨。
function toggleSidebar() {
  if (isMobile.value) {
    mobileSidebarOpen.value = !mobileSidebarOpen.value
    return
  }
  if (isTablet.value) {
    tabletSidebarOpen.value = !tabletSidebarOpen.value
    return
  }
  sidebarCollapsed.value = !sidebarCollapsed.value
}

// 手机端与平板端切页后自动收起浮层，否则遮罩会挡住新页面。
function navigate(view: ViewKey) {
  closeSidebarOverlays()
  emit('navigate', view)
}

// closeSidebarOverlays 由遮罩点击与断点切换共用，保证两种浮层不会同时残留。
function closeSidebarOverlays() {
  mobileSidebarOpen.value = false
  tabletSidebarOpen.value = false
}

// 跨断点时必须清掉该断点专属的浮层态：否则平板展开着拖到桌面，会留下打不开的遮罩。
function onViewportChange() {
  isMobile.value = mobileQuery?.matches ?? false
  isTablet.value = tabletQuery?.matches ?? false
  if (!isMobile.value) mobileSidebarOpen.value = false
  if (!isTablet.value) tabletSidebarOpen.value = false
}

onMounted(() => {
  void refreshWindowMaximised()
  if (typeof window === 'undefined' || !window.matchMedia) return
  mobileQuery = window.matchMedia('(max-width: 767px)')
  tabletQuery = window.matchMedia('(min-width: 768px) and (max-width: 1023px)')
  isMobile.value = mobileQuery.matches
  isTablet.value = tabletQuery.matches
  mobileQuery.addEventListener('change', onViewportChange)
  tabletQuery.addEventListener('change', onViewportChange)
})

onBeforeUnmount(() => {
  mobileQuery?.removeEventListener('change', onViewportChange)
  tabletQuery?.removeEventListener('change', onViewportChange)
})

// 桌面窗口尺寸变化时最大化态可能已失效，只在按钮可用时重新读取。
watch(() => isWindowControlsDisabled.value, (disabled) => {
  if (!disabled) void refreshWindowMaximised()
})
</script>

<template>
  <div :class="cn('app-window', isAlwaysOnTop && 'is-always-on-top')">
    <!-- 极光光谱与环境背景：默认关闭，由顶栏或设置页「极光折射流光背景」点亮，材质见 styles/liquid-glass.css -->
    <div class="stage-backdrop" aria-hidden="true">
      <div class="stage-line"></div>
      <div class="stage-band stage-band-lead"></div>
      <div class="stage-band"></div>
      <div class="stage-band stage-band-tail"></div>
      <div class="stage-glow-ambient"></div>
    </div>

    <header class="app-topbar">
      <div class="topbar-left">
        <button
          type="button"
          class="sidebar-toggle-btn"
          :title="sidebarCollapsed && !isMobile ? '展开侧边栏' : '折叠/展开侧边栏'"
          aria-label="切换侧栏"
          @click="toggleSidebar"
        >
          <PanelLeft :size="18" />
        </button>
        <div class="topbar-title-group">
          <div class="topbar-title-line">
            <h1 class="topbar-main-title">{{ activeTitle }}</h1>
            <span class="always-on-top-badge" title="窗口当前处于置顶状态">📌 窗口置顶</span>
          </div>
          <p class="topbar-sub-title">{{ activeSubtitle }}</p>
        </div>
      </div>

      <div class="topbar-right">
        <div class="topbar-actions">
          <button
            type="button"
            class="action-icon-btn"
            title="切换显示外观"
            aria-label="切换主题"
            @click="toggleTheme"
          >
            <Sun v-if="display.themeMode.value === 'dark'" :size="16" />
            <Moon v-else :size="16" />
          </button>
          <button
            type="button"
            :class="cn('action-icon-btn', glass.backdrop.value && 'active')"
            title="切换极光流光背景 (默认清爽模式)"
            aria-label="切换极光背景"
            @click="toggleBackdrop"
          >
            <Sparkles :size="16" />
          </button>
          <button
            type="button"
            :class="cn('action-icon-btn', updateTone)"
            :title="updateHint"
            aria-label="更新状态"
            @click="updateOpen = true"
          >
            <RefreshCw :size="16" />
          </button>
        </div>

        <div class="topbar-divider" aria-hidden="true"></div>

        <!-- 苹果三个控制按钮：从左至右 最小化 → 最大化 → 关闭 -->
        <div class="traffic-lights-right" aria-label="窗口控制">
          <button
            type="button"
            class="traffic-btn minimize"
            title="最小化"
            aria-label="最小化"
            :disabled="isWindowControlsDisabled"
            @click="minimiseWindow"
          >
            <Minus :size="8" />
          </button>
          <button
            type="button"
            class="traffic-btn maximize"
            :title="isWindowMaximised ? '最大化 / 还原' : '最大化'"
            :aria-label="isWindowMaximised ? '还原窗口' : '最大化'"
            :disabled="isWindowControlsDisabled"
            @click="toggleWindowMaximise"
          >
            <Maximize :size="8" />
          </button>
          <button
            type="button"
            class="traffic-btn close"
            title="关闭"
            aria-label="关闭"
            :disabled="isWindowControlsDisabled"
            @click="closeWindow"
          >
            <X :size="8" />
          </button>
        </div>
      </div>
    </header>

    <div class="app-body">
      <div
        :class="cn('mobile-sidebar-backdrop', (mobileSidebarOpen || tabletSidebarOpen) && 'active')"
        aria-hidden="true"
        @click="closeSidebarOverlays"
      ></div>

      <aside
        :class="cn('app-sidebar', sidebarCollapsed && !isMobile && !isTablet && 'is-collapsed', mobileSidebarOpen && 'mobile-open', tabletSidebarOpen && 'tablet-open')"
        aria-label="主导航"
      >
        <nav class="sidebar-nav">
          <template v-for="group in navigationGroups" :key="group.title">
            <div class="nav-group-title">{{ group.title }}</div>
            <button
              v-for="item in group.items"
              :key="item.key"
              type="button"
              :class="cn('nav-item', props.activeView === item.key && 'active')"
              :title="item.subtitle"
              :aria-label="item.title"
              @click="navigate(item.key)"
            >
              <span :class="cn('sq-icon-badge', 'size-md', item.tone)" aria-hidden="true">
                <component :is="item.icon" :size="15" :stroke-width="2.2" />
              </span>
              <span class="nav-item-label">{{ item.title }}</span>
            </button>
          </template>
        </nav>

        <div class="sidebar-footer">
          <div class="runtime-indicator">
            <span :class="cn('indicator-dot', runtimeHealth.dot)"></span>
            <span class="runtime-text">{{ runtimeHealth.text }}</span>
          </div>
          <span class="runtime-text runtime-pid">{{ runtimeDetail }}</span>
        </div>
      </aside>

      <main class="app-main-viewport">
        <el-alert
          v-if="appStore.errorMessage"
          class="apple-alert app-error-banner"
          type="error"
          :title="appStore.errorMessage"
          show-icon
          :closable="false"
        />
        <div :key="props.activeView" :class="cn('view-content', props.activeView === 'logs' && 'view-content--fill')">
          <slot />
        </div>
      </main>
    </div>

    <nav class="mobile-tabbar" aria-label="页面导航">
      <div class="mobile-tabbar-inner">
        <button
          v-for="item in navigation"
          :key="item.key"
          type="button"
          :class="cn('tabbar-item', props.activeView === item.key && 'active')"
          :aria-label="item.label"
          @click="emit('navigate', item.key)"
        >
          <component :is="item.icon" />
          <span>{{ item.label }}</span>
        </button>
      </div>
    </nav>

    <UpdateStatusDialog :open="updateOpen" @close="updateOpen = false" />
  </div>
</template>

<style scoped src="./AppChrome.css"></style>
