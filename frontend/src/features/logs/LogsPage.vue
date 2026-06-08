<!--
  文件职责：渲染日志筛选、日志列表、清理入口和状态提示。
  说明：注释覆盖组件脚本状态、方法、生命周期和模板结构；不改变渲染逻辑。
-->

<script setup lang="ts">
import { computed, defineComponent, h, onUnmounted, ref, watch } from 'vue'
import { FileText, Maximize2, RefreshCw, Search, SlidersHorizontal, TimerReset, Trash2 } from '@lucide/vue'
import { useAppStore } from '@/stores/app'
import { cn } from '@/lib/utils'
import { toMessage } from '@/app/state'
import { formatDateTime } from '@/shared/format'
import { displayMessage } from '@/shared/labels'

// knownLogScopes 覆盖运行时内置来源；动态来源会继续从当前日志结果合并。
const knownLogScopes = ['all', 'app', 'process', 'window', 'startup', 'shortcut', 'update', 'settings', 'storage', 'log-file', 'crash', 'panic', 'single-instance']
// logPageSize 固定页面批量大小，避免大日志量一次性撑爆前端渲染。
const logPageSize = 50

// appStore 保存 Pinia store 实例，集中访问应用共享状态和动作。
const appStore = useAppStore()
// selectedLogFileName 保存当前文件选择；为空时后端默认读取当前每日文件。
const selectedLogFileName = ref('')
// keyword/scope/severity 是日志查询条件；变化后立即刷新第一页。
const keyword = ref('')
// scope 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const scope = ref('all')
// severity 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const severity = ref('all')
// filtersOpen 默认关闭，让日志流成为首屏主体内容。
const filtersOpen = ref(false)
// fullscreen 表示日志专注模式，只放大当前日志视图，不改变业务数据。
const fullscreen = ref(false)
// autoRefresh 控制 5 秒轮询，适合跟踪安装器或运行时异常。
const autoRefresh = ref(false)
// clearDialogOpen 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const clearDialogOpen = ref(false)
// timer 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
let timer: number | undefined

// logScopes 保存由响应式状态推导出的只读结果，模板直接消费该值。
const logScopes = computed(() => {
  // scopes 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
  const scopes = new Set([...knownLogScopes, ...appStore.logs.map((log) => log.scope).filter(Boolean)])
  return Array.from(scopes)
})

// activeFilterCount 保存由响应式状态推导出的只读结果，模板直接消费该值。
const activeFilterCount = computed(() => {
  // count 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
  let count = 0
  if (scope.value !== 'all') count += 1
  if (severity.value !== 'all') count += 1
  if (keyword.value.trim() !== '') count += 1
  return count
})

// filterSummary 保存由响应式状态推导出的只读结果，模板直接消费该值。
const filterSummary = computed(() => {
  // parts 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
  const parts = [logScopeLabel(scope.value), logLevelLabel(severity.value)]
  // value 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
  const value = keyword.value.trim()
  if (value) parts.push(`关键词：${value}`)
  return parts.join(' / ')
})

// Stat 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
const Stat = defineComponent({
  props: {
    label: { type: String, required: true },
    tone: { type: String, default: '' },
    value: { type: Number, required: true },
  },
  setup(props) {
    return () => h('div', { class: cn('log-stat-item', props.tone === 'danger' && 'is-danger') }, [
      h('strong', String(props.value)),
      h('span', props.label),
    ])
  },
})

// buildQuery 处理 渲染日志筛选、日志列表、清理入口和状态提示 中的用户动作、生命周期动作或数据转换。
function buildQuery(page: number) {
  return {
    fileName: selectedLogFileName.value,
    scope: scope.value,
    severity: severity.value,
    keyword: keyword.value.trim(),
    page,
    pageSize: logPageSize,
  }
}

function showError(error: unknown, fallback: string) {
  const message = toMessage(error)
  appStore.applyAction({ type: 'errorSet', payload: message || fallback })
}

// refreshLogs 按当前筛选条件请求指定页日志，是手动刷新、轮询和筛选变化的统一入口。
async function refreshLogs(page = 1) {
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

// clearFilters 处理 渲染日志筛选、日志列表、清理入口和状态提示 中的用户动作、生命周期动作或数据转换。
function clearFilters() {
  keyword.value = ''
  scope.value = 'all'
  severity.value = 'all'
}

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch([keyword, scope, severity], () => {
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
  void refreshLogs(1)
})

// watch 监听关键响应式状态变化，并把变更同步到派生状态或远端服务。
watch(autoRefresh, (enabled) => {
  if (timer) {
    window.clearInterval(timer)
    timer = undefined
  }
  if (enabled) {
    timer = window.setInterval(() => {
      void refreshLogs(appStore.logPage)
    }, 5000)
  }
})

// watch 监听专注模式状态，把全局页面壳隐藏交给根 class 控制。
watch(fullscreen, (enabled) => {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('is-log-fullscreen', enabled)
}, { immediate: true })

// onUnmounted 在组件卸载前释放订阅和运行时资源，避免重复监听。
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
  if (typeof document !== 'undefined') document.documentElement.classList.remove('is-log-fullscreen')
})

// logScopeLabel 处理 渲染日志筛选、日志列表、清理入口和状态提示 中的用户动作、生命周期动作或数据转换。
function logScopeLabel(scope: string) {
  // labels 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
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

// logLevelLabel 处理 渲染日志筛选、日志列表、清理入口和状态提示 中的用户动作、生命周期动作或数据转换。
function logLevelLabel(level: string) {
  // labels 保存 渲染日志筛选、日志列表、清理入口和状态提示 使用的配置、引用或中间结果。
  const labels: Record<string, string> = {
    all: '全部级别',
    debug: 'debug',
    info: 'info',
    warning: 'warning',
    error: 'error',
  }
  return labels[level] ?? level
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
  <!-- 模板结构：声明当前组件对外呈现的布局、插槽和交互入口。 -->
  <Teleport to="body" :disabled="!fullscreen">
    <div :class="cn('page-stack log-page', fullscreen && 'log-fullscreen', filtersOpen && 'has-open-filters')">
      <UiCard>
        <UiCardHeader class="split-header">
          <div>
            <UiCardTitle>日志</UiCardTitle>
          </div>
          <div class="button-row">
            <UiButton :aria-expanded="filtersOpen" variant="secondary" @click="filtersOpen = !filtersOpen">
              <SlidersHorizontal class="icon-tone-indigo" :size="18" />
              {{ filtersOpen ? '收起筛选' : '筛选' }}
              <UiBadge v-if="activeFilterCount > 0" variant="outline">{{ activeFilterCount }}</UiBadge>
            </UiButton>
            <UiButton :class="cn(autoRefresh && 'is-active')" variant="secondary" @click="autoRefresh = !autoRefresh">
              <TimerReset class="icon-tone-indigo" :size="18" />
              {{ autoRefresh ? '停止自动' : '自动刷新' }}
            </UiButton>
            <UiButton variant="secondary" @click="refreshLogs(appStore.logPage)">
              <RefreshCw class="icon-tone-green" :size="18" />
              刷新
            </UiButton>
            <UiButton :aria-pressed="fullscreen" variant="secondary" @click="fullscreen = !fullscreen">
              <Maximize2 class="icon-tone-gray" :size="18" />
              {{ fullscreen ? '退出专注' : '专注模式' }}
            </UiButton>
          </div>
        </UiCardHeader>

        <UiCardContent class="log-page-main">
          <section v-if="filtersOpen" class="log-filter-panel" aria-label="日志筛选">
            <div class="log-toolbar">
              <UiField class="log-file-field">
                <UiLabel>日期/日志文件</UiLabel>
                <span class="input-with-icon">
                  <FileText class="icon-tone-gray" :size="17" />
                  <UiNativeSelect v-model="selectedLogFileName" :disabled="appStore.logFiles.length === 0">
                    <option v-if="appStore.logFiles.length === 0" value="">内存临时日志</option>
                    <option v-for="file in appStore.logFiles" :key="file.fileName" :value="file.fileName">
                      {{ formatLogFileOption(file) }}
                    </option>
                  </UiNativeSelect>
                </span>
              </UiField>
              <UiField>
                <UiLabel>作用域</UiLabel>
                <UiNativeSelect v-model="scope">
                  <option v-for="item in logScopes" :key="item" :value="item">{{ logScopeLabel(item) }}</option>
                </UiNativeSelect>
              </UiField>
              <UiField>
                <UiLabel>级别</UiLabel>
                <UiNativeSelect v-model="severity">
                  <option value="all">全部级别</option>
                  <option value="debug">debug</option>
                  <option value="info">info</option>
                  <option value="warning">warning</option>
                  <option value="error">error</option>
                </UiNativeSelect>
              </UiField>
              <UiField class="log-search">
                <UiLabel>关键词</UiLabel>
                <span class="input-with-icon">
                  <Search class="icon-tone-gray" :size="17" />
                  <UiInput v-model="keyword" placeholder="错误、阶段、文件名..." />
                </span>
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

          <section class="log-stats-strip" aria-label="日志统计">
            <Stat label="全部" :value="appStore.logStats.total" />
            <Stat label="debug" :value="appStore.logStats.debug" />
            <Stat label="info" :value="appStore.logStats.info" />
            <Stat label="warning" :value="appStore.logStats.warning" />
            <Stat label="error" :value="appStore.logStats.error" tone="danger" />
          </section>

          <section class="log-stream-panel" aria-label="日志流">
            <div class="log-stream-header">
              <div>
                <strong>日志流</strong>
                <span>{{ filterSummary }}</span>
              </div>
            </div>

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
                <UiTableRow
                  v-for="log in appStore.logs"
                  :key="`${log.time}-${log.scope}-${log.message}`"
                  :class="cn(log.severity === 'error' && 'is-error', log.severity === 'warning' && 'is-warning')"
                >
                  <UiTableCell>{{ formatDateTime(log.time) }}</UiTableCell>
                  <UiTableCell>{{ logScopeLabel(log.scope) }}</UiTableCell>
                  <UiTableCell>{{ logLevelLabel(log.severity) }}</UiTableCell>
                  <UiTableCell class="log-message-cell" :title="displayMessage(log.message)">{{ displayMessage(log.message) }}</UiTableCell>
                </UiTableRow>
              </UiTableBody>
            </UiTable>
            <div v-if="appStore.logs.length === 0" class="empty-state">暂无匹配日志</div>

            <footer class="log-footer">
              <span>第 {{ appStore.logPage }} 页 · 每页 {{ appStore.logPageSize }} 条</span>
              <div class="button-row">
                <UiButton :disabled="appStore.logPage <= 1" variant="secondary" @click="refreshLogs(appStore.logPage - 1)">上一页</UiButton>
                <UiButton :disabled="!appStore.logHasMore" variant="secondary" @click="refreshLogs(appStore.logPage + 1)">下一页</UiButton>
              </div>
            </footer>
          </section>
        </UiCardContent>
      </UiCard>

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
