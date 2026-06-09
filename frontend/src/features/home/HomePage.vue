<!--
  文件职责：渲染首页软件运行状态、业务统计和演示图表。
  说明：首页只展示本机运行面，不承载更新检查或业务入口。
-->

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, type Component } from 'vue'
import {
  Activity,
  AlertTriangle,
  BarChart3,
  Database,
  Globe2,
  MonitorCheck,
  Server,
  TrendingUp,
} from '@lucide/vue'
import { useAppStore } from '@/stores/app'
import type { StartupApiKey, StartupApiStatus } from '@/app/state'

const appStore = useAppStore()

type ServiceStatus = 'ok' | 'warning' | 'error'
type ServiceTone = 'icon-tone-indigo' | 'icon-tone-green' | 'icon-tone-orange' | 'icon-tone-purple'

type ServiceCard = {
  title: string
  label: string
  meta: string
  status: ServiceStatus
  tone: ServiceTone
  icon: Component
}

// networkOnline 只读取浏览器在线状态；它不代表 GitHub Release 或本地更新源一定可达。
const networkOnline = ref<boolean | null>(typeof navigator === 'undefined' ? null : navigator.onLine)

const demoStats = [
  { label: '今日处理量', value: '128', delta: '+12.4%', tone: 'positive' },
  { label: '成功率', value: '98.7%', delta: '+1.8%', tone: 'positive' },
  { label: '待处理', value: '23', delta: '-6', tone: 'neutral' },
  { label: '异常记录', value: '3', delta: '+1', tone: 'negative' },
]

const trendPoints = [
  { label: '周一', value: 44 },
  { label: '周二', value: 58 },
  { label: '周三', value: 49 },
  { label: '周四', value: 76 },
  { label: '周五', value: 68 },
  { label: '周六', value: 84 },
  { label: '周日', value: 72 },
]

const distributionSegments = [
  { label: '自动处理', value: 58, className: 'is-chart-1' },
  { label: '人工复核', value: 27, className: 'is-chart-2' },
  { label: '异常挂起', value: 15, className: 'is-chart-5' },
]

// serviceCards 聚合启动阶段 API 状态，用于首页健康卡；不主动重新请求后端。
const serviceCards = computed<ServiceCard[]>(() => [
  webviewStatus(),
  applicationStatus(),
  databaseStatus(),
  networkStatus(),
])

const softwareSummary = computed(() => {
  if (serviceCards.value.some((item) => item.status === 'error')) {
    return { label: '存在异常', status: 'error' as ServiceStatus }
  }
  if (serviceCards.value.some((item) => item.status === 'warning')) {
    return { label: '检测中', status: 'warning' as ServiceStatus }
  }
  return { label: '运行正常', status: 'ok' as ServiceStatus }
})

function syncNetworkStatus() {
  networkOnline.value = typeof navigator === 'undefined' ? null : navigator.onLine
}

onMounted(() => {
  syncNetworkStatus()
  window.addEventListener('online', syncNetworkStatus)
  window.addEventListener('offline', syncNetworkStatus)
})

onUnmounted(() => {
  window.removeEventListener('online', syncNetworkStatus)
  window.removeEventListener('offline', syncNetworkStatus)
})

function startupStatus(key: StartupApiKey): StartupApiStatus {
  return appStore.startupApiStatuses[key] ?? { state: 'idle', message: '', updatedAt: '' }
}

// pendingStatus 把 idle/loading 都视为检测中，避免启动 API 尚未开始时被误判为异常。
function pendingStatus(status: StartupApiStatus) {
  return status.state === 'idle' || status.state === 'loading'
}

// webviewStatus 只证明前端已渲染，不代表 Go 后端 API 可用。
function webviewStatus(): ServiceCard {
  return {
    title: 'WebView',
    label: '正常',
    meta: '已渲染',
    status: 'ok',
    tone: 'icon-tone-indigo',
    icon: MonitorCheck,
  }
}

function applicationStatus(): ServiceCard {
  const status = startupStatus('appInfo')
  if (status.state === 'error') {
    return {
      title: '应用服务',
      label: '异常',
      meta: '接口异常',
      status: 'error',
      tone: 'icon-tone-orange',
      icon: Server,
    }
  }
  if (status.state === 'ok') {
    return {
      title: '应用服务',
      label: '正常',
      meta: appStore.appInfo?.version ? `v${appStore.appInfo.version}` : 'GetAppInfo',
      status: 'ok',
      tone: 'icon-tone-orange',
      icon: Server,
    }
  }
  if (!appStore.appInfo || pendingStatus(status)) {
    return {
      title: '应用服务',
      label: '检测中',
      meta: '读取中',
      status: 'warning',
      tone: 'icon-tone-orange',
      icon: Server,
    }
  }
  return {
    title: '应用服务',
    label: '正常',
    meta: `v${appStore.appInfo.version}`,
    status: 'ok',
    tone: 'icon-tone-orange',
    icon: Server,
  }
}

function databaseStatus(): ServiceCard {
  // 数据库健康同时参考 EnvironmentInfo 和依赖 SQLite 的设置/显示偏好读取结果。
  const environmentStatus = startupStatus('environmentInfo')
  const settingsStatus = startupStatus('settings')
  const displayStatus = startupStatus('displayPreferences')
  const failedStatus = [environmentStatus, settingsStatus, displayStatus].find((item) => item.state === 'error')

  if (failedStatus) {
    return {
      title: 'SQLite 数据库',
      label: '异常',
      meta: '接口异常',
      status: 'error',
      tone: 'icon-tone-purple',
      icon: Database,
    }
  }
  if (appStore.environmentInfo?.databaseReady || appStore.environmentInfo?.databaseStatus === 'ok') {
    return {
      title: 'SQLite 数据库',
      label: '正常',
      meta: '配置库就绪',
      status: 'ok',
      tone: 'icon-tone-purple',
      icon: Database,
    }
  }
  if (environmentStatus.state === 'ok' && settingsStatus.state === 'ok' && displayStatus.state === 'ok') {
    return {
      title: 'SQLite 数据库',
      label: '正常',
      meta: '配置库就绪',
      status: 'ok',
      tone: 'icon-tone-purple',
      icon: Database,
    }
  }
  if (!appStore.environmentInfo || [environmentStatus, settingsStatus, displayStatus].some(pendingStatus)) {
    return {
      title: 'SQLite 数据库',
      label: '检测中',
      meta: '读取中',
      status: 'warning',
      tone: 'icon-tone-purple',
      icon: Database,
    }
  }
  return {
    title: 'SQLite 数据库',
    label: appStore.environmentInfo.databaseStatus === 'disabled' ? '检测中' : '异常',
    meta: appStore.environmentInfo.databaseStatus === 'disabled' ? '未启用' : '未就绪',
    status: appStore.environmentInfo.databaseStatus === 'disabled' ? 'warning' : 'error',
    tone: 'icon-tone-purple',
    icon: Database,
  }
}

function networkStatus(): ServiceCard {
  if (networkOnline.value === null) {
    return {
      title: '网络',
      label: '检测中',
      meta: '读取中',
      status: 'warning',
      tone: 'icon-tone-green',
      icon: Globe2,
    }
  }
  if (!networkOnline.value) {
    return {
      title: '网络',
      label: '异常',
      meta: '离线',
      status: 'error',
      tone: 'icon-tone-green',
      icon: Globe2,
    }
  }
  return {
    title: '网络',
    label: '正常',
    meta: '在线',
    status: 'ok',
    tone: 'icon-tone-green',
    icon: Globe2,
  }
}

</script>

<template>
  <div class="page-stack">
    <section aria-label="软件运行状况">
      <div class="split-header monitor-heading">
        <div class="section-title-row">
          <span class="data-icon icon-tone-indigo" aria-hidden="true"><Activity :size="17" /></span>
          <div>
            <div class="section-title-with-status">
              <h3>软件运行状况</h3>
              <span :class="`status-inline is-${softwareSummary.status}`">
                <span class="status-inline-dot" aria-hidden="true" />
                {{ softwareSummary.label }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div class="software-status-grid">
        <UiCard v-for="item in serviceCards" :key="item.title" :class="`software-status-card is-${item.status}`">
          <UiCardHeader class="software-status-card-header">
            <div class="software-status-main">
              <span :class="`software-status-icon ${item.tone}`" aria-hidden="true">
                <component :is="item.icon" :size="19" />
              </span>
              <div>
                <div class="software-status-title-row">
                  <UiCardTitle>{{ item.title }}</UiCardTitle>
                  <span :class="`status-inline is-${item.status}`">
                    <span class="status-inline-dot" aria-hidden="true" />
                    {{ item.label }}
                  </span>
                </div>
                <UiCardDescription class="software-status-meta">{{ item.meta }}</UiCardDescription>
              </div>
            </div>
          </UiCardHeader>
        </UiCard>
      </div>
    </section>

    <section aria-label="业务统计">
      <div class="split-header monitor-heading">
        <div class="section-title-row">
          <span class="data-icon icon-tone-green" aria-hidden="true"><BarChart3 :size="17" /></span>
          <div>
            <h3>业务统计</h3>
            <p>当前暂无真实业务数据，以下为 Demo 展示。</p>
          </div>
        </div>
      </div>

      <div class="business-stats-grid">
        <UiCard v-for="stat in demoStats" :key="stat.label" :class="`business-stat-card is-${stat.tone}`">
          <UiCardHeader class="business-stat-header">
            <UiCardDescription>{{ stat.label }}</UiCardDescription>
            <UiCardAction>
              <UiBadge :class="`metric-delta-badge is-${stat.tone}`" variant="outline">{{ stat.delta }}</UiBadge>
            </UiCardAction>
          </UiCardHeader>
          <UiCardContent class="business-stat-content">
            <UiCardTitle>{{ stat.value }}</UiCardTitle>
          </UiCardContent>
        </UiCard>
      </div>
    </section>

    <section class="dashboard-grid" aria-label="演示图表">
      <UiCard class="chart-panel chart-panel-wide">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-green" aria-hidden="true"><TrendingUp :size="17" /></span>
            <div>
              <UiCardTitle>业务趋势</UiCardTitle>
              <UiCardDescription>样例业务量趋势。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent>
          <div class="demo-bar-chart" aria-label="演示趋势图">
            <div v-for="point in trendPoints" :key="point.label" class="demo-bar-column">
              <span class="demo-bar-track">
                <span class="demo-bar-fill" :style="{ height: `${point.value}%` }" />
              </span>
              <small>{{ point.label }}</small>
            </div>
          </div>
        </UiCardContent>
      </UiCard>

      <UiCard class="chart-panel">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-indigo" aria-hidden="true"><BarChart3 :size="17" /></span>
            <div>
              <UiCardTitle>处理分布</UiCardTitle>
              <UiCardDescription>样例处理类型占比。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent>
          <div class="distribution-list">
            <div v-for="segment in distributionSegments" :key="segment.label" class="distribution-row">
              <div>
                <span>{{ segment.label }}</span>
                <strong>{{ segment.value }}%</strong>
              </div>
              <span class="distribution-track">
                <span :class="`distribution-fill ${segment.className}`" :style="{ width: `${segment.value}%` }" />
              </span>
            </div>
          </div>
        </UiCardContent>
      </UiCard>

      <UiCard class="chart-panel">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-orange" aria-hidden="true"><AlertTriangle :size="17" /></span>
            <div>
              <UiCardTitle>日志摘要</UiCardTitle>
              <UiCardDescription>当前运行日志统计。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent>
          <div class="log-summary-grid">
            <div><strong>{{ appStore.logStats.info }}</strong><span>信息</span></div>
            <div><strong>{{ appStore.logStats.warning }}</strong><span>警告</span></div>
            <div><strong>{{ appStore.logStats.error }}</strong><span>错误</span></div>
          </div>
        </UiCardContent>
      </UiCard>
    </section>
  </div>
</template>

<style scoped src="./HomePage.css"></style>
