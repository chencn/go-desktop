<!--
  文件职责：渲染日志筛选、日志列表、清理入口和状态提示。
  页面所有查询最终走 appStore.refreshLogs，对应后端 QueryLogs 分页过滤接口。
-->

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { FileText, Maximize2, RefreshCw, Search, SlidersHorizontal, TimerReset, Trash2 } from '@lucide/vue'
import { useAppStore } from '@/stores/app'
import { cn } from '@/lib/utils'
import { toMessage } from '@/app/state'
import { formatDateTime } from '@/shared/format'
import { displayMessage } from '@/shared/labels'

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
// clearDialogOpen 只控制二次确认；真正清理由 confirmClearLogs 触发后端 ClearLogs。
const clearDialogOpen = ref(false)
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
const canGoNext = computed(() => totalPages.value > 0 && appStore.logPage < totalPages.value)
const displayedLogs = computed(() => logLayoutReady.value ? appStore.logs : [])

function calculateLogPageSize(tableElement?: HTMLElement | null, paginationElement?: HTMLElement | null) {
  if (typeof window === 'undefined' || !paginationElement) return 0
  const isDesktop = window.matchMedia('(min-width: 761px)').matches
  if (!isDesktop) return 12
  if (!tableElement) return 0
  const maxRows = isDesktop ? 40 : 30
  const headerHeight = tableElement.querySelector('thead')?.getBoundingClientRect().height ?? 38
  const rowElement = tableElement.querySelector('tbody tr:not(.log-empty-row)')
  const rowHeight = rowElement?.getBoundingClientRect().height || (isDesktop ? 48 : 56)

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

// confirmClearLogs 确认清理当前作用域日志，并用第一页查询条件刷新列表。
async function confirmClearLogs() {
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

// watch 监听专注模式状态，把全局页面壳隐藏交给根 class 控制。
watch(fullscreen, (enabled) => {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('is-log-fullscreen', enabled)
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
  if (typeof document !== 'undefined') document.documentElement.classList.remove('is-log-fullscreen')
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
  <Teleport to="body" :disabled="!fullscreen">
    <div :class="cn('page-stack log-page', fullscreen && 'log-fullscreen', filtersOpen && 'has-open-filters')">
      <section class="log-command-card" aria-label="日志筛选工具条">
        <div class="log-command-toolbar">
          <div class="log-command-tabs" aria-label="日志级别快捷查询">
            <button type="button" class="log-command-tab" :class="{ 'is-active': severity === 'all' }" @click="applySeverityFilter('all')">
              <span>全部</span>
              <strong>{{ appStore.logStats.total }}</strong>
            </button>
            <button type="button" class="log-command-tab is-debug" :class="{ 'is-active': severity === 'debug' }" @click="applySeverityFilter('debug')">
              <span>debug</span>
              <strong>{{ appStore.logStats.debug }}</strong>
            </button>
            <button type="button" class="log-command-tab is-info" :class="{ 'is-active': severity === 'info' }" @click="applySeverityFilter('info')">
              <span>info</span>
              <strong>{{ appStore.logStats.info }}</strong>
            </button>
            <button type="button" class="log-command-tab is-warning" :class="{ 'is-active': severity === 'warning' }" @click="applySeverityFilter('warning')">
              <span>warning</span>
              <strong>{{ appStore.logStats.warning }}</strong>
            </button>
            <button type="button" class="log-command-tab is-error" :class="{ 'is-active': severity === 'error' }" @click="applySeverityFilter('error')">
              <span>error</span>
              <strong>{{ appStore.logStats.error }}</strong>
            </button>
          </div>

          <label class="log-command-search">
            <Search class="icon-tone-gray" :size="18" />
            <UiInput v-model="keyword" placeholder="错误、阶段、文件名..." aria-label="搜索日志关键词" />
          </label>

          <UiButton variant="secondary" @click="refreshLogs(appStore.logPage)">
            <RefreshCw class="icon-tone-green" :size="18" />
            刷新
          </UiButton>
          <UiButton :class="cn(autoRefresh && 'is-active')" variant="secondary" @click="autoRefresh = !autoRefresh">
            <TimerReset class="icon-tone-indigo" :size="18" />
            {{ autoRefresh ? '停止自动' : '自动刷新' }}
          </UiButton>
          <UiButton :aria-expanded="filtersOpen" variant="secondary" @click="filtersOpen = !filtersOpen">
            <SlidersHorizontal class="icon-tone-indigo" :size="18" />
            {{ filtersOpen ? '收起' : '筛选' }}
            <UiBadge v-if="activeFilterCount > 0" variant="outline">{{ activeFilterCount }}</UiBadge>
          </UiButton>
          <UiButton :aria-pressed="fullscreen" variant="secondary" @click="fullscreen = !fullscreen">
            <Maximize2 class="icon-tone-gray" :size="18" />
            {{ fullscreen ? '退出专注' : '专注模式' }}
          </UiButton>
        </div>
      </section>

      <section v-if="filtersOpen" class="log-filter-panel log-collapsed-filter" aria-label="更多日志筛选">
        <div class="log-toolbar log-collapsed-toolbar">
          <UiField class="log-file-field">
            <UiLabel>日期/日志文件</UiLabel>
            <span class="input-with-icon">
              <FileText class="icon-tone-gray" :size="17" />
              <UiSelect :model-value="selectedLogFileName" :disabled="appStore.logFiles.length === 0" @update:model-value="selectedLogFileName = String($event)">
                <UiSelectTrigger class="settings-control-select" aria-label="日志文件">
                  <UiSelectValue placeholder="日志文件" />
                </UiSelectTrigger>
                <UiSelectContent>
                  <UiSelectItem v-if="appStore.logFiles.length === 0" value="">内存临时日志</UiSelectItem>
                  <UiSelectItem v-for="file in appStore.logFiles" :key="file.fileName" :value="file.fileName">
                    {{ formatLogFileOption(file) }}
                  </UiSelectItem>
                </UiSelectContent>
              </UiSelect>
            </span>
          </UiField>
          <UiField>
            <UiLabel>作用域</UiLabel>
            <UiSelect :model-value="scope" @update:model-value="scope = String($event)">
              <UiSelectTrigger class="settings-control-select" aria-label="作用域">
                <UiSelectValue placeholder="全部作用域" />
              </UiSelectTrigger>
              <UiSelectContent>
                <UiSelectItem v-for="item in logScopes" :key="item" :value="item">{{ logScopeLabel(item) }}</UiSelectItem>
              </UiSelectContent>
            </UiSelect>
          </UiField>
          <div class="log-filter-actions">
            <UiButton :disabled="activeFilterCount === 0" variant="secondary" @click="clearFilters">重置筛选</UiButton>
            <UiButton :disabled="appStore.logTotal === 0" variant="destructive" @click="clearDialogOpen = true">
              <Trash2 :size="18" />
              清空当前视图
            </UiButton>
          </div>
        </div>
      </section>

      <section class="log-stream-panel" aria-label="日志流">
        <div ref="logListRef" class="log-mobile-list" aria-label="应用日志移动列表">
          <article v-if="displayedLogs.length === 0" class="log-mobile-card log-mobile-empty">
            {{ logLayoutReady ? '暂无匹配日志' : '' }}
          </article>
          <article v-for="log in displayedLogs" :key="`${log.time}-${log.scope}-${log.message}`" class="log-mobile-card">
            <div class="log-mobile-card__top">
              <UiBadge class="log-level-badge" :class="logLevelClass(log.severity)" variant="outline">{{ logLevelLabel(log.severity) }}</UiBadge>
              <span>{{ formatDateTime(log.time) }}</span>
            </div>
            <p class="log-mobile-card__message">{{ displayMessage(log.message) }}</p>
            <dl class="log-mobile-card__meta">
              <div>
                <dt>来源</dt>
                <dd>{{ logScopeLabel(log.scope) }}</dd>
              </div>
              <div>
                <dt>级别</dt>
                <dd>{{ logLevelLabel(log.severity) }}</dd>
              </div>
            </dl>
          </article>
        </div>

        <div ref="logTableRef" class="log-table-shell">
          <UiTable class="log-table" aria-label="应用日志">
            <colgroup>
              <col class="log-col-time">
              <col class="log-col-scope">
              <col class="log-col-level">
              <col class="log-col-message">
            </colgroup>
            <UiTableHeader>
              <UiTableRow>
                <UiTableHead>时间</UiTableHead>
                <UiTableHead>来源</UiTableHead>
                <UiTableHead>级别</UiTableHead>
                <UiTableHead>内容</UiTableHead>
              </UiTableRow>
            </UiTableHeader>
            <UiTableBody>
              <UiTableRow v-if="displayedLogs.length === 0" class="log-empty-row">
                <UiTableCell colspan="4" class="log-empty-cell">{{ logLayoutReady ? '暂无匹配日志' : '' }}</UiTableCell>
              </UiTableRow>
              <UiTableRow
                v-for="log in displayedLogs"
                :key="`${log.time}-${log.scope}-${log.message}`"
              >
                <UiTableCell class="log-time-cell">{{ formatDateTime(log.time) }}</UiTableCell>
                <UiTableCell class="log-scope-cell">{{ logScopeLabel(log.scope) }}</UiTableCell>
                <UiTableCell class="log-level-cell">
                  <UiBadge class="log-level-badge" :class="logLevelClass(log.severity)" variant="outline">{{ logLevelLabel(log.severity) }}</UiBadge>
                </UiTableCell>
                <UiTableCell class="log-message-cell" :title="displayMessage(log.message)">{{ displayMessage(log.message) }}</UiTableCell>
              </UiTableRow>
            </UiTableBody>
          </UiTable>
        </div>

        <footer ref="logPaginationRef" class="log-footer log-pagination-card">
          <span class="log-pagination-summary">{{ logLayoutReady ? `共 ${appStore.logTotal} 条，每页 ${displayedPageSize} 条，当前第 ${displayedLogPage} / ${totalPages} 页` : '' }}</span>
          <div class="button-row">
            <UiButton :disabled="appStore.logPage <= 1" variant="secondary" @click="refreshLogs(appStore.logPage - 1)">上一页</UiButton>
            <UiButton :disabled="!canGoNext" variant="secondary" @click="refreshLogs(appStore.logPage + 1)">下一页</UiButton>
          </div>
        </footer>
      </section>

      <UiAlertDialog
        :open="clearDialogOpen"
        title="清空当前视图"
        description="只隐藏当前视图中的匹配日志，每日文件日志不会被删除。"
        confirm-text="清空"
        @close="clearDialogOpen = false"
        @confirm="confirmClearLogs"
      />
    </div>
  </Teleport>
</template>

<style scoped src="./LogsPage.css"></style>
