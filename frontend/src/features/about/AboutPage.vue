<!--
  文件职责：渲染关于页的应用身份、运行状态、Release 来源、本地路径和技术栈。
  说明：关于页只承载只读信息；可编辑的界面偏好留在设置页。
-->

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, ref } from 'vue'
import {
  Activity,
  Box,
  CalendarClock,
  Code2,
  Cpu,
  ExternalLink,
  FolderGit2,
  Gauge,
  HardDrive,
  Package,
  Rocket,
  ServerCog,
} from '@lucide/vue'
import { openExternalURL } from '@/api/wails'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/shared/format'
import { projectMetadata } from '@/shared/project'

type DetailItem = {
  label: string
  value: string
}

type TechGroup = {
  title: string
  description: string
  items: string[]
}

const appStore = useAppStore()
const now = ref(Date.now())

let uptimeTimer: number | undefined

const appName = computed(() => appStore.appInfo?.name ?? projectMetadata.appName)
const appDescription = computed(() => appStore.appInfo?.description ?? projectMetadata.description)
const currentVersion = computed(() => appStore.appInfo?.version ?? projectMetadata.defaultVersion)
const startedAtLabel = computed(() => formatDateTime(appStore.appInfo?.startedAt))
const uptimeLabel = computed(() => formatUptime(appStore.appInfo?.startedAt, now.value))
const platformLabel = computed(() => `${appStore.environmentInfo?.os ?? '未获取'} / ${appStore.environmentInfo?.arch ?? '未获取'}`)
const releaseSourceLabel = computed(() => `${appStore.settings?.githubOwner ?? projectMetadata.github.owner}/${appStore.settings?.githubRepo ?? projectMetadata.github.repo}`)

const appDetails = computed<DetailItem[]>(() => [
  { label: '应用名称', value: appName.value },
  { label: '当前版本', value: currentVersion.value },
  { label: '模块路径', value: projectMetadata.modulePath },
  { label: '公开仓库', value: appStore.appInfo?.repository ?? projectMetadata.repositoryUrl },
])

const runtimeDetails = computed<DetailItem[]>(() => [
  { label: '启动时间', value: startedAtLabel.value },
  { label: '运行时长', value: uptimeLabel.value },
  { label: '运行平台', value: platformLabel.value },
  { label: 'Go 版本', value: appStore.environmentInfo?.goVersion ?? '未获取' },
  { label: 'Wails 版本', value: appStore.environmentInfo?.wailsVersion ?? '未获取' },
])

const releaseDetails = computed<DetailItem[]>(() => [
  { label: 'Release 来源', value: releaseSourceLabel.value },
  { label: 'GitHub Owner', value: appStore.settings?.githubOwner ?? projectMetadata.github.owner },
  { label: 'GitHub Repo', value: appStore.settings?.githubRepo ?? projectMetadata.github.repo },
  { label: 'API 代理', value: appStore.settings?.githubProxyBase || '未启用' },
  { label: 'User-Agent', value: projectMetadata.github.userAgent },
])

const localDataDetails = computed<DetailItem[]>(() => [
  { label: '配置数据库', value: appStore.environmentInfo?.databasePath ?? '未配置' },
  { label: '文件日志', value: appStore.environmentInfo?.logFilePath ?? '未配置' },
  { label: '更新缓存', value: appStore.environmentInfo?.cachePath ?? '未配置' },
])

const techGroups: TechGroup[] = [
  {
    title: '桌面运行层',
    description: 'Go 提供本地能力，Wails3 负责窗口和前后端桥接。',
    items: ['Go', 'Wails3', 'WebView2'],
  },
  {
    title: '前端界面',
    description: 'Vue 负责界面状态，shadcn-vue 提供基础组件体系。',
    items: ['Vue 3', 'TypeScript', 'Pinia', 'Tailwind CSS v4', 'shadcn-vue', 'Lucide'],
  },
  {
    title: '数据与发布',
    description: '本地数据落 SQLite 和文件日志，版本分发走 GitHub Release。',
    items: ['SQLite', '文件日志', 'GitHub Release', 'NSIS'],
  },
]

onMounted(() => {
  uptimeTimer = window.setInterval(() => {
    now.value = Date.now()
  }, 30000)
})

onUnmounted(() => {
  if (uptimeTimer !== undefined) {
    window.clearInterval(uptimeTimer)
  }
})

function formatUptime(value: string | undefined, currentTime: number) {
  if (!value) return '未记录'
  const startedAt = new Date(value).getTime()
  if (!Number.isFinite(startedAt)) return '时间无效'
  const totalSeconds = Math.max(0, Math.floor((currentTime - startedAt) / 1000))
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  if (days > 0) return `${days}天 ${hours}小时`
  if (hours > 0) return `${hours}小时 ${minutes}分钟`
  return `${minutes}分钟`
}

const InfoRow = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
  },
  setup(props) {
    return () => h('div', { class: 'about-info-row' }, [
      h('span', props.label),
      h('strong', props.value),
    ])
  },
})
</script>

<template>
  <div class="page-stack about-page">
    <section class="about-overview" aria-labelledby="about-title">
      <UiCard class="about-identity-card">
        <UiCardHeader class="about-identity-header">
          <span class="data-icon icon-tone-purple" aria-hidden="true"><Package :size="18" /></span>
          <div class="about-title-block">
            <UiCardTitle id="about-title">{{ appName }}</UiCardTitle>
            <UiCardDescription>{{ appDescription }}</UiCardDescription>
          </div>
          <UiCardAction>
            <UiButton size="icon" variant="ghost" aria-label="打开项目地址" @click="openExternalURL(projectMetadata.repositoryUrl)">
              <ExternalLink :size="18" />
            </UiButton>
          </UiCardAction>
        </UiCardHeader>
        <UiCardContent>
          <div class="about-status-strip" aria-label="关键运行状态">
            <div class="about-status-item">
              <Rocket :size="16" aria-hidden="true" />
              <span>版本</span>
              <strong>{{ currentVersion }}</strong>
            </div>
            <div class="about-status-item">
              <CalendarClock :size="16" aria-hidden="true" />
              <span>启动时间</span>
              <strong>{{ startedAtLabel }}</strong>
            </div>
            <div class="about-status-item">
              <Gauge :size="16" aria-hidden="true" />
              <span>运行时长</span>
              <strong>{{ uptimeLabel }}</strong>
            </div>
            <div class="about-status-item">
              <Cpu :size="16" aria-hidden="true" />
              <span>平台</span>
              <strong>{{ platformLabel }}</strong>
            </div>
          </div>
        </UiCardContent>
      </UiCard>
    </section>

    <section class="about-section-grid">
      <UiCard class="about-card">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-purple" aria-hidden="true"><Box :size="17" /></span>
            <div>
              <UiCardTitle>应用信息</UiCardTitle>
              <UiCardDescription>应用身份、版本和仓库来源。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent class="about-card-content">
          <div class="about-info-list">
            <InfoRow v-for="item in appDetails" :key="item.label" :label="item.label" :value="item.value" />
          </div>
        </UiCardContent>
      </UiCard>

      <UiCard class="about-card">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-indigo" aria-hidden="true"><Activity :size="17" /></span>
            <div>
              <UiCardTitle>运行状态</UiCardTitle>
              <UiCardDescription>启动时间、运行时长和运行环境。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent class="about-card-content">
          <div class="about-info-list">
            <InfoRow v-for="item in runtimeDetails" :key="item.label" :label="item.label" :value="item.value" />
          </div>
        </UiCardContent>
      </UiCard>

      <UiCard class="about-card">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-green" aria-hidden="true"><FolderGit2 :size="17" /></span>
            <div>
              <UiCardTitle>Release 来源</UiCardTitle>
              <UiCardDescription>更新检查使用的 GitHub Release 配置。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent class="about-card-content">
          <div class="about-info-list">
            <InfoRow v-for="item in releaseDetails" :key="item.label" :label="item.label" :value="item.value" />
          </div>
        </UiCardContent>
      </UiCard>

      <UiCard class="about-card">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-orange" aria-hidden="true"><HardDrive :size="17" /></span>
            <div>
              <UiCardTitle>本地数据</UiCardTitle>
              <UiCardDescription>设置、数据库、日志和更新缓存位置。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent class="about-card-content">
          <div class="about-info-list">
            <InfoRow v-for="item in localDataDetails" :key="item.label" :label="item.label" :value="item.value" />
          </div>
        </UiCardContent>
      </UiCard>

      <UiCard class="about-card about-card-wide">
        <UiCardHeader>
          <div class="section-title-row">
            <span class="data-icon icon-tone-indigo" aria-hidden="true"><Code2 :size="17" /></span>
            <div>
              <UiCardTitle>技术栈</UiCardTitle>
              <UiCardDescription>按运行层级整理的当前实现基线。</UiCardDescription>
            </div>
          </div>
        </UiCardHeader>
        <UiCardContent class="about-card-content">
          <div class="about-tech-grid">
            <div v-for="group in techGroups" :key="group.title" class="about-tech-group">
              <div>
                <ServerCog :size="16" aria-hidden="true" />
                <strong>{{ group.title }}</strong>
              </div>
              <p>{{ group.description }}</p>
              <div class="about-chip-grid">
                <UiBadge v-for="item in group.items" :key="item" variant="secondary">{{ item }}</UiBadge>
              </div>
            </div>
          </div>
        </UiCardContent>
      </UiCard>
    </section>
  </div>
</template>

<style scoped src="./AboutPage.css"></style>
