<!--
  文件职责：渲染日志筛选工具条、日志流表格、移动端卡片流和分页条。
  说明：查询统一走 appStore.refreshLogs（后端 QueryLogs 分页过滤）；页大小由可视区高度实测得出，
  不采用设计稿的固定 12 条，也不写 calc(100vh - N) 这类魔法数。
-->

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { Maximize2, RefreshCw, Search, SlidersHorizontal, TimerReset, Trash2 } from '@lucide/vue'
import { useAppStore } from '@/stores/app'
import { useDisplayPreferences } from '@/app/display'
import { toMessage } from '@/app/state'
import { formatDateTime } from '@/shared/format'
import { displayMessage } from '@/shared/labels'
import AlertDialog from '../shared/AlertDialog.vue'

// knownLogScopes 覆盖运行时内置来源；动态来源会继续从当前日志结果合并。
const knownLogScopes = ['all', 'app', 'process', 'window', 'startup', 'shortcut', 'update', 'settings', 'storage', 'log-file', 'crash', 'panic', 'single-instance']

const appStore = useAppStore()
// logTableRef/logListRef/logPaginationRef 用于按可见空间动态计算页大小，参考列表页分页密度。
const logTableRef = ref<HTMLElement | null>(null)
const logListRef = ref<HTMLElement | null>(null)
const logPaginationRef = ref<HTMLElement | null>(null)
// logPageSize 是前端请求页大小，后端仍会做最终归一化。
const logPageSize = ref(0)
// logLayoutReady 避免日志页首次渲染直接使用启动预加载的默认 50 条日志。
const logLayoutReady = ref(false)
// selectedLogFileName 保存当前文件选择；为空时后端默认读取当前每日文件。
const selectedLogFileName = ref('')
// keyword/scope/severity 是日志查询条件；变化后立即刷新第一页。
const keyword = ref('')
// scope 为后端日志作用域过滤值；all 表示不按来源过滤。
const scope = ref('all')
// severity 为后端日志级别过滤值；all 表示不按级别过滤。
const severity = ref('all')
// filtersOpen 默认关闭，让日志流成为首屏主体内容。
const filtersOpen = ref(false)
// fullscreen 表示日志专注模式，只放大当前日志视图，不改变业务数据。
const fullscreen = ref(false)
// autoRefresh 控制 5 秒轮询，适合跟踪安装器或运行时异常。
const autoRefresh = ref(false)
// timer 保存自动刷新 interval id，组件卸载或关闭自动刷新时必须清理。
let timer: number | undefined
// pageSizeObserver 跟随表格区域、分页条和窗口尺寸更新 pageSize。
let pageSizeObserver: ResizeObserver | undefined
let suppressLogPageSizeWatch = false

// logScopes 合并内置来源和当前查询结果中的动态来源，避免新后端 scope 无法筛选。
const logScopes = computed(() => {
  const scopes = new Set([...knownLogScopes, ...appStore.logs.map((log) => log.scope).filter(Boolean)])
  return Array.from(scopes)
})

// activeFilterCount 只统计非默认筛选，供筛选按钮 badge 和重置按钮使用。
const activeFilterCount = computed(() => {
  let count = 0
  if (scope.value !== 'all') count += 1
  if (severity.value !== 'all') count += 1
  if (keyword.value.trim() !== '') count += 1
  return count
})

const effectiveLogPageSize = computed(() => logLayoutReady.value && logPageSize.value > 0 ? logPageSize.value : appStore.logPageSize)

const totalPages = computed(() => {
  const pageSize = effectiveLogPageSize.value
  if (appStore.logTotal <= 0 || pageSize <= 0) return 0
  return Math.ceil(appStore.logTotal / pageSize)
})

const displayedLogPage = computed(() => {
  if (totalPages.value === 0) return 0
  return Math.min(appStore.logPage, totalPages.value)
})

const displayedPageSize = computed(() => effectiveLogPageSize.value)
const displayedLogs = computed(() => logLayoutReady.value ? appStore.logs : [])

// paginationSummary 对应设计稿分页条左侧摘要；布局未就绪时留白，避免先显示默认 50 条页大小。
// 结尾附带后端最低门禁级别，与设置页「控制台日志级别」联动，方便确认筛选结果为空是被门禁挡住。
const paginationSummary = computed(() => {
  if (!logLayoutReady.value) return ''
  const gate = String(appStore.settings?.logLevel ?? 'info').toUpperCase()
  return `每页 ${displayedPageSize.value} 条，当前第 ${displayedLogPage.value} / ${totalPages.value} 页 (门禁过滤后共 ${appStore.logTotal} 条记录，系统最低门禁: ≥ ${gate})`
})

// emptyLogsText 与设计稿一致：空结果时提示当前生效的最低门禁级别。
const emptyLogsText = computed(() => {
  const gate = String(appStore.settings?.logLevel ?? 'info').toUpperCase()
  return `暂无匹配日志 (门禁级别: ≥ ${gate})`
})

// 三档字号会改变单元格文本宽度，但 el-table 的列宽是按像素写死的：大档下设计稿的 196px 会把时间戳折成两行。
// 列宽因此跟随写在 <html> 上的 --ui-scale 等比放大，标准档仍与设计稿逐像素相等。
const displayPreferences = useDisplayPreferences()
const uiScale = ref(1)

// scaledColumnWidth 把设计稿列宽换算成当前档的像素宽度。
function scaledColumnWidth(base: number) {
  return Math.round(base * uiScale.value)
}

// syncUiScale 从样式表读回当前缩放档；读不到（SSR、样式未就绪）一律按 1 处理。
function syncUiScale() {
  if (typeof document === 'undefined') return
  const value = Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ui-scale'))
  uiScale.value = Number.isFinite(value) && value > 0 ? value : 1
}

watch(displayPreferences.size, async () => {
  await nextTick()
  syncUiScale()
  // 行高同样随档位变化，不重算的话分页会按旧行高塞进行不下的行数。
  updateLogPageSize()
})

function calculateLogPageSize(tableElement?: HTMLElement | null, paginationElement?: HTMLElement | null) {
  if (typeof window === 'undefined' || !paginationElement) return 0
  // 767px 是设计稿的手机/桌面分界，与本页 @media 断点保持一致。
  const isDesktop = window.matchMedia('(min-width: 768px)').matches
  if (!isDesktop) return 12
  if (!tableElement) return 0
  const maxRows = isDesktop ? 40 : 30
  // el-table 的表头和表体分别在 header-wrapper/body-wrapper 内，按内部 DOM 实测高度。
  const headerHeight = tableElement.querySelector('.el-table__header-wrapper thead')?.getBoundingClientRect().height ?? 38
  const rowElement = tableElement.querySelector('.el-table__body-wrapper tbody tr')
  const rowHeight = rowElement?.getBoundingClientRect().height || (isDesktop ? 41 : 56)

  // 两种模式现在都使用受 Grid 约束的稳定容器高度来进行精确计算
  const available = Math.max(0, tableElement.getBoundingClientRect().height - headerHeight - 6)

  return Math.max(1, Math.min(maxRows, Math.floor(available / rowHeight)))
}

function updateLogPageSize() {
  const next = calculateLogPageSize(logTableRef.value, logPaginationRef.value)
  if (next > 0 && logPageSize.value !== next) {
    logPageSize.value = next
    return true
  }
  return false
}

// buildQuery 把页面筛选项转换成后端 LogQuery；page 由刷新入口显式传入。
function buildQuery(page: number) {
  return {
    fileName: selectedLogFileName.value,
    scope: scope.value,
    severity: severity.value,
    keyword: keyword.value.trim(),
    page,
    pageSize: logPageSize.value,
  }
}

function showError(error: unknown, fallback: string) {
  const message = toMessage(error)
  appStore.applyAction({ type: 'errorSet', payload: message || fallback })
}

// refreshLogs 按当前筛选条件请求指定页日志，是手动刷新、轮询和筛选变化的统一入口。
async function refreshLogs(page = 1) {
  if (!logLayoutReady.value || logPageSize.value <= 0) return
  try {
    await appStore.refreshLogFiles()
    await appStore.refreshLogs(buildQuery(page))
  } catch (error) {
    showError(error, '日志刷新失败')
  }
}

// clearDialogOpen 控制「清空当前视图」二次确认弹窗（设计稿 .apple-alert-dialog）。
const clearDialogOpen = ref(false)

// clearViewLogs 清理当前作用域日志，并用第一页查询条件刷新列表。
async function clearViewLogs() {
  clearDialogOpen.value = false
  try {
    await appStore.clearLogScope(scope.value, buildQuery(1))
  } catch (error) {
    showError(error, '日志清理失败')
  }
}

// clearFilters 恢复本地筛选默认值，watch 会自动触发第一页刷新。
function clearFilters() {
  keyword.value = ''
  scope.value = 'all'
  severity.value = 'all'
}

// 筛选变化即刷新第一页；分页按钮会绕过这里传入目标页。
watch([keyword, scope, severity], () => {
  if (!logLayoutReady.value) return
  void refreshLogs(1)
})

// pageSize 变化时重置到第一页，避免视口变化后页码指向不同记录段。
watch(logPageSize, () => {
  if (!logLayoutReady.value || suppressLogPageSizeWatch) return
  void refreshLogs(1)
})

// watch 监听当前后端返回的文件名，初始化本地选择。
watch(() => appStore.selectedLogFileName, (fileName) => {
  if (fileName && selectedLogFileName.value === '') {
    selectedLogFileName.value = fileName
  }
}, { immediate: true })

// watch 监听日志文件切换，并重新读取第一页。
watch(selectedLogFileName, () => {
  if (!logLayoutReady.value) return
  void refreshLogs(1)
})

// 自动刷新复用当前页码，便于用户在翻页后继续观察同一页。
watch(autoRefresh, (enabled) => {
  if (timer) {
    window.clearInterval(timer)
    timer = undefined
  }
  if (enabled) {
    timer = window.setInterval(() => {
      if (!logLayoutReady.value) return
      void refreshLogs(appStore.logPage)
    }, 5000)
  }
})

// watch 监听专注模式状态，把外壳隐藏交给根 class 控制（样式见 styles/layout.css）。
watch(fullscreen, (enabled) => {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('is-log-focus', enabled)
  void nextTick(updateLogPageSize)
}, { immediate: true })

watch(filtersOpen, () => {
  void nextTick(updateLogPageSize)
})

watch(() => appStore.logs.length, () => {
  if (suppressLogPageSizeWatch) return
  void nextTick(updateLogPageSize)
})

async function initializeLogPageSize() {
  suppressLogPageSizeWatch = true
  let shouldRefreshAgain = false
  try {
    await nextTick()
    updateLogPageSize()
    logLayoutReady.value = true
    await refreshLogs(1)
    await nextTick()
    shouldRefreshAgain = updateLogPageSize()
  } finally {
    suppressLogPageSizeWatch = false
  }
  if (shouldRefreshAgain) {
    await refreshLogs(1)
  }
}

onMounted(() => {
  syncUiScale()
  void initializeLogPageSize()
  if (typeof ResizeObserver !== 'undefined') {
    pageSizeObserver = new ResizeObserver(updateLogPageSize)
    if (logTableRef.value) pageSizeObserver.observe(logTableRef.value)
    if (logListRef.value) pageSizeObserver.observe(logListRef.value)
    if (logPaginationRef.value) pageSizeObserver.observe(logPaginationRef.value)
  }
  window.addEventListener('resize', updateLogPageSize)
})

// onUnmounted 在组件卸载前释放订阅和运行时资源，避免重复监听。
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
  pageSizeObserver?.disconnect()
  if (typeof window !== 'undefined') window.removeEventListener('resize', updateLogPageSize)
  if (typeof document !== 'undefined') document.documentElement.classList.remove('is-log-focus')
})

// logScopeLabel 只本地化已知 scope；未知动态 scope 保留原值，方便定位新后端来源。
function logScopeLabel(scope: string) {
  const labels: Record<string, string> = {
    all: '全部作用域',
    app: '应用',
    window: '窗口',
    update: '更新',
    settings: '设置',
    startup: '启动集成',
    shortcut: '快捷方式',
    storage: '存储',
    process: '进程',
    'log-file': '文件日志',
    crash: '崩溃',
    panic: 'panic',
    'single-instance': '单实例',
  }
  return labels[scope] ?? scope
}

// logLevelLabel 保留 debug/info 等后端级别原文，避免改写影响排障搜索。
function logLevelLabel(level: string) {
  const labels: Record<string, string> = {
    all: '全部级别',
    debug: 'debug',
    info: 'info',
    warning: 'warning',
    error: 'error',
  }
  return labels[level] ?? level
}

function logLevelClass(level: string) {
  const classes: Record<string, string> = {
    debug: 'is-debug',
    info: 'is-info',
    warning: 'is-warning',
    error: 'is-error',
  }
  return classes[level] ?? 'is-debug'
}

function applySeverityFilter(value: string) {
  severity.value = value
}

// formatLogFileOption 统一日志文件下拉展示。
function formatLogFileOption(file: { date: string; fileName: string; current: boolean }) {
  const legacy = file.fileName === 'go-desktop.log'
  const parts = [file.date]
  if (file.current) parts.push('当前')
  if (legacy) parts.push('旧格式')
  return parts.join(' · ')
}
</script>

<template>
  <div class="log-page">
    <!-- 命令工具条卡片：级别页签、搜索、操作按钮，展开后在卡内追加筛选面板 -->
    <section class="log-command-card" aria-label="日志筛选工具条">
      <div class="log-command-toolbar">
        <el-radio-group
          class="log-command-tabs segmented-control is-tone-tabs"
          :model-value="severity"
          aria-label="日志级别快捷查询"
          @update:model-value="applySeverityFilter(String($event))"
        >
          <el-radio-button value="all"><span>全部</span><strong>{{ appStore.logStats.total }}</strong></el-radio-button>
          <el-radio-button value="debug" class="is-debug"><span>debug</span><strong>{{ appStore.logStats.debug }}</strong></el-radio-button>
          <el-radio-button value="info" class="is-info"><span>info</span><strong>{{ appStore.logStats.info }}</strong></el-radio-button>
          <el-radio-button value="warning" class="is-warning"><span>warning</span><strong>{{ appStore.logStats.warning }}</strong></el-radio-button>
          <el-radio-button value="error" class="is-error"><span>error</span><strong>{{ appStore.logStats.error }}</strong></el-radio-button>
        </el-radio-group>

        <div class="log-command-search">
          <Search class="log-search-icon" :size="15" aria-hidden="true" />
          <el-input
            v-model="keyword"
            class="apple-field has-leading-icon"
            placeholder="错误、阶段、文件名..."
            aria-label="搜索日志关键词"
          />
        </div>

        <el-button class="btn-apple" @click="refreshLogs(appStore.logPage)">
          <RefreshCw class="log-tool-icon is-success" :size="15" aria-hidden="true" />
          刷新
        </el-button>
        <el-button class="btn-apple" :aria-pressed="autoRefresh" @click="autoRefresh = !autoRefresh">
          <TimerReset class="log-tool-icon is-indigo" :size="15" aria-hidden="true" />
          {{ autoRefresh ? '停止自动' : '自动刷新' }}
        </el-button>
        <el-button class="btn-apple" :aria-expanded="filtersOpen" @click="filtersOpen = !filtersOpen">
          <SlidersHorizontal class="log-tool-icon is-indigo" :size="15" aria-hidden="true" />
          筛选
          <span v-if="activeFilterCount > 0" class="nav-item-badge">{{ activeFilterCount }}</span>
        </el-button>
        <el-button class="btn-apple" :aria-pressed="fullscreen" @click="fullscreen = !fullscreen">
          <Maximize2 class="log-tool-icon" :size="15" aria-hidden="true" />
          {{ fullscreen ? '退出专注' : '专注模式' }}
        </el-button>
      </div>

      <div v-if="filtersOpen" class="log-filter-panel">
        <div class="log-collapsed-toolbar">
          <div class="filter-item-label">
            <span>日期/日志文件:</span>
            <el-select
              :model-value="selectedLogFileName"
              class="apple-select"
              :disabled="appStore.logFiles.length === 0"
              :placeholder="appStore.logFiles.length === 0 ? '内存临时日志' : '当前日志文件'"
              aria-label="日志文件"
              @update:model-value="selectedLogFileName = String($event)"
            >
              <el-option v-if="appStore.logFiles.length === 0" value="" label="内存临时日志" />
              <el-option
                v-for="file in appStore.logFiles"
                :key="file.fileName"
                :value="file.fileName"
                :label="formatLogFileOption(file)"
              />
            </el-select>
          </div>
          <div class="filter-item-label">
            <span>作用域:</span>
            <el-select
              :model-value="scope"
              class="apple-select"
              aria-label="作用域"
              @update:model-value="scope = String($event)"
            >
              <el-option v-for="item in logScopes" :key="item" :value="item" :label="logScopeLabel(item)" />
            </el-select>
          </div>
        </div>
        <div class="log-filter-actions">
          <el-button class="btn-apple" :disabled="activeFilterCount === 0" @click="clearFilters">重置筛选</el-button>
          <el-button class="btn-apple" type="danger" :disabled="appStore.logTotal === 0" @click="clearDialogOpen = true">
            <Trash2 :size="14" aria-hidden="true" />
            清空当前视图
          </el-button>
        </div>
      </div>
    </section>

    <!-- 日志流面板：桌面表格 + 移动卡片 + 分页条，三段共用一块面板底 -->
    <section class="log-stream-panel" aria-label="日志流">
      <div ref="logListRef" class="log-mobile-list" aria-label="应用日志移动列表">
        <p v-if="displayedLogs.length === 0 && logLayoutReady" class="log-mobile-empty">{{ emptyLogsText }}</p>
        <article v-for="log in displayedLogs" :key="`${log.time}-${log.scope}-${log.message}`" class="log-mobile-card">
          <div class="log-mobile-card__top">
            <span :class="`log-level-badge ${logLevelClass(log.severity)}`">{{ logLevelLabel(log.severity) }}</span>
            <span>{{ formatDateTime(log.time) }}</span>
          </div>
          <p class="log-mobile-card__message">{{ displayMessage(log.message) }}</p>
          <div class="log-mobile-card__meta">
            <div><span>来源:</span><strong>{{ logScopeLabel(log.scope) }}</strong></div>
            <div><span>级别:</span><strong>{{ logLevelLabel(log.severity) }}</strong></div>
          </div>
        </article>
      </div>

      <div ref="logTableRef" class="log-table-shell">
        <!-- 列宽对应设计稿（196/128/108 + 内容列补足），表头吸顶交给 el-table 的 height。 -->
        <el-table
          v-if="logLayoutReady || displayedLogs.length > 0"
          class="apple-table"
          :data="displayedLogs"
          height="100%"
          stripe
          aria-label="应用日志"
          :empty-text="logLayoutReady ? emptyLogsText : ''"
        >
          <el-table-column label="时间" :width="scaledColumnWidth(196)">
            <template #default="{ row }">
              <span class="log-time-cell">{{ formatDateTime(row.time) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="来源" :width="scaledColumnWidth(128)">
            <template #default="{ row }">
              <span class="log-scope-tag">{{ logScopeLabel(row.scope) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="级别" :width="scaledColumnWidth(108)">
            <template #default="{ row }">
              <span :class="`log-level-badge ${logLevelClass(row.severity)}`">{{ logLevelLabel(row.severity) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="内容" min-width="240" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="log-message-cell">{{ displayMessage(row.message) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <footer ref="logPaginationRef" class="log-pagination-card">
        <span>{{ paginationSummary }}</span>
        <el-pagination
          class="apple-pagination"
          layout="prev, pager, next"
          :total="appStore.logTotal"
          :page-size="displayedPageSize"
          :current-page="displayedLogPage"
          :disabled="totalPages === 0"
          @current-change="(page: number) => refreshLogs(page)"
        />
      </footer>
    </section>

    <!-- 清空当前视图的二次确认（设计稿 .apple-alert-dialog）：只清视图，每日归档文件保留 -->
    <AlertDialog
      :open="clearDialogOpen"
      confirm-text="确认清空"
      title="确定清空当前视图中的日志？"
      @close="clearDialogOpen = false"
      @confirm="clearViewLogs"
    >
      该操作将从当前视图中移除匹配的 <strong>{{ appStore.logTotal }} 条</strong> 展示记录。<br>本地磁盘上的每日日志归档文件（<code>{{
        appStore.environmentInfo?.logFilePath || '未配置'
      }}</code>）将安全保留，不会被删除。
    </AlertDialog>
  </div>
</template>

<style scoped src="./LogsPage.css"></style>
