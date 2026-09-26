<!--
  文件职责：渲染首页软件运行状况、业务统计和演示图表。
  说明：首页只展示本机运行面，不承载更新检查或业务入口；四张服务卡由启动阶段 API 结果聚合。
-->

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, type Component } from 'vue'
import {
  BarChart3,
  CircleCheck,
  Clock,
  Database,
  Globe,
  Monitor,
  Server,
  TriangleAlert,
} from '@lucide/vue'
import GlassPanel from '@/features/shared/GlassPanel.vue'
import { useAppStore } from '@/stores/app'
import type { IconTone } from '@/shared/views'
import type { StartupApiKey, StartupApiStatus } from '@/app/state'

const appStore = useAppStore()

type ServiceStatus = 'ok' | 'warning' | 'error'

// ServiceCard 对应设计稿 .service-card：状态徽标 + 说明句 + 底部键值行。
type ServiceCard = {
  badge: string
  desc: string
  footerKey: string
  footerValue: string
  icon: Component
  status: ServiceStatus
  title: string
  tone: IconTone
}

// DemoStat 对应 .stat-card：标签 + 图标 + 数值 + 变化幅度徽标。
type DemoStat = {
  delta: string
  deltaTone: 'negative' | 'neutral' | 'positive'
  icon: Component
  label: string
  tone: IconTone
  value: string
}

const demoStats: DemoStat[] = [
  { delta: '+12.4%', deltaTone: 'positive', icon: BarChart3, label: '今日处理量', tone: 'indigo', value: '128' },
  { delta: '+1.8%', deltaTone: 'positive', icon: CircleCheck, label: '成功率', tone: 'green', value: '98.7%' },
  { delta: '-6', deltaTone: 'neutral', icon: Clock, label: '待处理', tone: 'purple', value: '23' },
  { delta: '+1', deltaTone: 'negative', icon: TriangleAlert, label: '异常记录', tone: 'orange', value: '3' },
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
  { color: 'var(--chart-1)', label: '自动处理', value: 58 },
  { color: 'var(--chart-2)', label: '人工复核', value: 27 },
  { color: 'var(--chart-5)', label: '异常挂起', value: 15 },
]

// 服务卡的图标、标题和语义色固定，只有状态文案随启动结果变化。
const serviceIdentities = {
  application: { icon: Server, title: '应用服务', tone: 'orange' },
  database: { icon: Database, title: 'SQLite 数据库', tone: 'purple' },
  network: { icon: Globe, title: '网络', tone: 'green' },
  webview: { icon: Monitor, title: 'WebView', tone: 'indigo' },
} as const satisfies Record<string, { icon: Component; title: string; tone: IconTone }>

// networkOnline 只读取浏览器在线状态；它不代表 GitHub Release 或本地更新源一定可达。
const networkOnline = ref<boolean | null>(typeof navigator === 'undefined' ? null : navigator.onLine)

const serviceCards = computed<ServiceCard[]>(() => [
  webviewCard(),
  applicationCard(),
  databaseCard(),
  networkCard(),
])

// 汇总徽标：任一异常优先，其次任一检测中，最后才是运行正常。
const softwareSummary = computed(() => {
  if (serviceCards.value.some((item) => item.status === 'error')) {
    return { badge: '存在异常', tone: 'error' }
  }
  if (serviceCards.value.some((item) => item.status === 'warning')) {
    return { badge: '检测中', tone: 'warn' }
  }
  return { badge: '运行正常', tone: 'ok' }
})

// serviceCard 把固定身份和随状态变化的文案合成一张卡。
function serviceCard(
  key: keyof typeof serviceIdentities,
  state: { badge: string; desc: string; footerValue: string; status: ServiceStatus; footerKey?: string },
): ServiceCard {
  return { ...serviceIdentities[key], footerKey: state.footerKey ?? '状态描述', ...state }
}

function startupStatus(key: StartupApiKey): StartupApiStatus {
  return appStore.startupApiStatuses[key] ?? { state: 'idle', message: '', updatedAt: '' }
}

// pendingStatus 把 idle/loading 都视为检测中，避免启动 API 尚未开始时被误判为异常。
function pendingStatus(status: StartupApiStatus) {
  return status.state === 'idle' || status.state === 'loading'
}

function appVersion() {
  return appStore.appInfo?.version ? `v${appStore.appInfo.version}` : '未获取'
}

// webviewCard 只证明前端已渲染，不代表 Go 后端 API 可用。
function webviewCard(): ServiceCard {
  return serviceCard('webview', {
    badge: '正常',
    desc: '现代 Web 引擎就绪，DOM 渲染管线就绪。',
    footerValue: '已渲染',
    status: 'ok',
  })
}

function applicationCard(): ServiceCard {
  const status = startupStatus('appInfo')
  if (status.state === 'error') {
    return serviceCard('application', {
      badge: '异常',
      desc: 'IPC 调用失败，请查看应用日志定位原因。',
      footerValue: '接口异常',
      status: 'error',
    })
  }
  if (pendingStatus(status)) {
    return serviceCard('application', {
      badge: '检测中',
      desc: '正在读取 GetAppInfo 运行时快照。',
      footerValue: '读取中',
      status: 'warning',
    })
  }
  return serviceCard('application', {
    badge: '正常',
    desc: 'IPC 通信畅通，GetAppInfo 接口响应健全。',
    footerKey: '当前版本',
    footerValue: appVersion(),
    status: 'ok',
  })
}

function databaseCard(): ServiceCard {
  // 数据库健康同时参考 EnvironmentInfo 和依赖 SQLite 的设置/显示偏好读取结果。
  const environmentStatus = startupStatus('environmentInfo')
  const settingsStatus = startupStatus('settings')
  const displayStatus = startupStatus('displayPreferences')
  const statuses = [environmentStatus, settingsStatus, displayStatus]

  if (statuses.some((item) => item.state === 'error')) {
    return serviceCard('database', {
      badge: '异常',
      desc: '配置库读写失败，设置可能无法持久化。',
      footerValue: '接口异常',
      status: 'error',
    })
  }
  const environment = appStore.environmentInfo
  if (environment?.databaseReady || environment?.databaseStatus === 'ok') {
    return serviceCard('database', {
      badge: '正常',
      desc: '本地事务存储已开启，WAL 模式持久化正常。',
      footerValue: '配置库就绪',
      status: 'ok',
    })
  }
  if (statuses.every((item) => item.state === 'ok')) {
    return serviceCard('database', {
      badge: '正常',
      desc: '本地事务存储已开启，WAL 模式持久化正常。',
      footerValue: '配置库就绪',
      status: 'ok',
    })
  }
  if (!environment || statuses.some(pendingStatus)) {
    return serviceCard('database', {
      badge: '检测中',
      desc: '正在读取运行环境信息。',
      footerValue: '读取中',
      status: 'warning',
    })
  }
  const disabled = environment.databaseStatus === 'disabled'
  return serviceCard('database', {
    badge: disabled ? '检测中' : '异常',
    desc: disabled ? '配置库未启用，运行期使用默认值。' : '配置库未就绪，设置可能无法保存。',
    footerValue: disabled ? '未启用' : '未就绪',
    status: disabled ? 'warning' : 'error',
  })
}

function networkCard(): ServiceCard {
  if (networkOnline.value === null) {
    return serviceCard('network', {
      badge: '检测中',
      desc: '正在探测本地网络状态。',
      footerValue: '读取中',
      status: 'warning',
    })
  }
  if (!networkOnline.value) {
    return serviceCard('network', {
      badge: '异常',
      desc: '当前离线，更新检查会走本地源或跳过。',
      footerValue: '离线',
      status: 'error',
    })
  }
  return serviceCard('network', {
    badge: '正常',
    desc: '本地网络畅通，支持更新拉取与外部调用。',
    footerValue: '在线',
    status: 'ok',
  })
}

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
</script>

<template>
  <div class="split-header">
    <div class="section-title-row">
      <div class="section-title-with-status">
        <h3>软件运行状况</h3>
        <span :class="`badge-apple ${softwareSummary.tone}`">{{ softwareSummary.badge }}</span>
      </div>
    </div>
  </div>

  <div class="services-grid">
    <GlassPanel v-for="item in serviceCards" :key="item.title" class="service-card">
      <div class="service-card-top">
        <div class="service-name-wrap">
          <span :class="`sq-icon-badge size-md ${item.tone}`" aria-hidden="true">
            <component :is="item.icon" :size="15" :stroke-width="2.2" />
          </span>
          <span class="service-name">{{ item.title }}</span>
        </div>
        <span :class="`badge-apple ${item.status === 'warning' ? 'warn' : item.status === 'error' ? 'error' : 'ok'}`">
          {{ item.badge }}
        </span>
      </div>
      <p class="service-meta-text">{{ item.desc }}</p>
      <div class="service-footer-ver">
        <span>{{ item.footerKey }}</span>
        <span>{{ item.footerValue }}</span>
      </div>
    </GlassPanel>
  </div>

  <div class="split-header">
    <div class="section-title-row">
      <div>
        <h3>业务统计</h3>
        <p>当前暂无真实业务数据，以下为 Demo 展示。</p>
      </div>
    </div>
  </div>

  <div class="stats-grid">
    <GlassPanel v-for="stat in demoStats" :key="stat.label" class="stat-card">
      <div class="stat-head">
        <span class="stat-label">{{ stat.label }}</span>
        <span :class="`stat-icon-wrap ${stat.tone}`" aria-hidden="true">
          <component :is="stat.icon" :size="17" />
        </span>
      </div>
      <div class="stat-value-row">
        <span class="stat-value">{{ stat.value }}</span>
        <span :class="`stat-delta ${stat.deltaTone}`">{{ stat.delta }}</span>
      </div>
    </GlassPanel>
  </div>

  <div class="dashboard-grid">
    <section class="chart-panel">
      <div>
        <h4 class="chart-panel-title">业务趋势</h4>
        <p class="chart-panel-desc">样例业务量趋势。</p>
      </div>
      <div class="demo-bar-chart" aria-label="演示趋势图">
        <div v-for="point in trendPoints" :key="point.label" class="demo-bar-column">
          <span class="demo-bar-track">
            <span class="demo-bar-fill" :style="{ height: `${point.value}%` }" />
          </span>
          <small>{{ point.label }}</small>
        </div>
      </div>
    </section>

    <section class="chart-panel">
      <div>
        <h4 class="chart-panel-title">处理分布</h4>
        <p class="chart-panel-desc">样例处理类型占比。</p>
      </div>
      <div class="distribution-list">
        <div v-for="segment in distributionSegments" :key="segment.label" class="distribution-row">
          <div class="dist-meta">
            <span>{{ segment.label }}</span>
            <strong>{{ segment.value }}%</strong>
          </div>
          <div class="dist-track">
            <div class="dist-bar" :style="{ width: `${segment.value}%`, background: segment.color }"></div>
          </div>
        </div>
      </div>
    </section>

    <section class="chart-panel">
      <div>
        <h4 class="chart-panel-title">日志摘要</h4>
        <p class="chart-panel-desc">当前运行日志统计。</p>
      </div>
      <div class="log-summary-grid">
        <div class="log-stat-box info">
          <span>信息 (info)</span>
          <strong>{{ appStore.logStats.info }}</strong>
        </div>
        <div class="log-stat-box warn">
          <span>警告 (warning)</span>
          <strong>{{ appStore.logStats.warning }}</strong>
        </div>
        <div class="log-stat-box error">
          <span>错误 (error)</span>
          <strong>{{ appStore.logStats.error }}</strong>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped src="./HomePage.css"></style>
