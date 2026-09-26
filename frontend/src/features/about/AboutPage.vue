<!--
  文件职责：渲染关于页的应用身份、运行元数据、Release 来源、本地路径和授权卡片。
  说明：关于页只承载只读信息，唯一的可写入口是内嵌的授权激活卡片；界面偏好仍归设置页。
-->

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Database, Monitor, SquareCheck } from '@lucide/vue'
import LicenseCard from '../license/LicenseCard.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/shared/format'
import { projectMetadata } from '@/shared/project'

type DetailItem = {
  label: string
  value: string
}

const appStore = useAppStore()
// now 只用于驱动运行时长刷新；关于页其余信息保持只读，不在这里重新拉取后端数据。
const now = ref(Date.now())

// uptimeTimer 每 30 秒刷新一次运行时长，避免页面常驻时显示停在首次渲染。
let uptimeTimer: number | undefined

// 应用身份优先使用后端 GetAppInfo，预览或未返回时回退到生成的项目 metadata。
const appName = computed(() => appStore.appInfo?.name ?? projectMetadata.appName)
const appDescription = computed(() => appStore.appInfo?.description ?? projectMetadata.description)
const currentVersion = computed(() => appStore.appInfo?.version ?? projectMetadata.defaultVersion)
const startedAtLabel = computed(() => formatDateTime(appStore.appInfo?.startedAt))
const uptimeLabel = computed(() => formatUptime(appStore.appInfo?.startedAt, now.value))
const platformLabel = computed(() => `${appStore.environmentInfo?.os ?? '未获取'} / ${appStore.environmentInfo?.arch ?? '未获取'}`)
const releaseSourceLabel = computed(() => `${projectMetadata.github.owner}/${projectMetadata.github.repo}`)

const runtimeDetails = computed<DetailItem[]>(() => [
  { label: '启动时间', value: startedAtLabel.value },
  { label: '运行时长', value: uptimeLabel.value },
  { label: '操作系统', value: platformLabel.value },
  { label: 'Go 核心', value: appStore.environmentInfo?.goVersion ?? '未获取' },
  { label: 'Wails 框架', value: appStore.environmentInfo?.wailsVersion ?? '未获取' },
])

const releaseDetails = computed<DetailItem[]>(() => [
  { label: 'Release 来源', value: releaseSourceLabel.value },
  { label: '公开仓库', value: appStore.appInfo?.repository ?? projectMetadata.repositoryUrl },
  { label: 'API 代理', value: appStore.settings?.githubProxyBase || '未启用' },
  { label: 'User-Agent', value: projectMetadata.github.userAgent },
])

const localDataDetails = computed<DetailItem[]>(() => [
  { label: '配置数据库', value: appStore.environmentInfo?.databasePath ?? '未配置' },
  { label: '文件日志', value: appStore.environmentInfo?.logFilePath ?? '未配置' },
  { label: '更新缓存', value: appStore.environmentInfo?.cachePath ?? '未配置' },
])

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
  if (days > 0) return `${days} 天 ${hours} 小时`
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`
  return `${minutes} 分钟`
}
</script>

<template>
  <div class="about-container">
    <section class="about-hero-card" aria-labelledby="about-title">
      <div class="about-app-icon" aria-hidden="true">
        <SquareCheck :size="36" />
      </div>
      <div class="about-hero-content">
        <h2 id="about-title">{{ appName }}</h2>
        <p>{{ appDescription }}</p>
        <div class="hero-badge-row">
          <span class="ver-badge">v{{ currentVersion }}</span>
          <span class="ver-badge">{{ projectMetadata.copyright }}</span>
        </div>
      </div>
    </section>

    <section class="meta-grid-2" aria-label="运行与发布信息">
      <div class="details-card">
        <h3 class="panel-title">
          <Monitor :size="16" aria-hidden="true" />
          运行时与平台
        </h3>
        <div v-for="item in runtimeDetails" :key="item.label" class="detail-row">
          <span class="detail-key">{{ item.label }}</span>
          <span class="detail-val">{{ item.value }}</span>
        </div>
      </div>

      <div class="details-card">
        <h3 class="panel-title">
          <!-- lucide 已移除品牌图标，此处内联设计稿同款 GitHub 标记 -->
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            aria-hidden="true"
          >
            <path
              d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4"
            />
          </svg>
          Release 与分发
        </h3>
        <div v-for="item in releaseDetails" :key="item.label" class="detail-row">
          <span class="detail-key">{{ item.label }}</span>
          <span class="detail-val">{{ item.value }}</span>
        </div>
      </div>
    </section>

    <section class="details-card">
      <h3 class="panel-title">
        <Database :size="16" aria-hidden="true" />
        本地数据与路径
      </h3>
      <div v-for="item in localDataDetails" :key="item.label" class="detail-row">
        <span class="detail-key">{{ item.label }}</span>
        <span class="detail-val">{{ item.value }}</span>
      </div>
    </section>

    <LicenseCard />
  </div>
</template>

<style scoped src="./AboutPage.css"></style>
