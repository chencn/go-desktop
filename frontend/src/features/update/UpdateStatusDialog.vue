<!--
  文件职责：渲染更新状态弹窗并驱动检查、下载、安装动作。
  说明：注释覆盖组件脚本状态、方法、生命周期和模板结构；不改变渲染逻辑。
-->

<script setup lang="ts">
import { computed, defineComponent, h, watch } from 'vue'
import { Download, ExternalLink, Loader2, RefreshCw, ShieldAlert, ShieldCheck, X } from '@lucide/vue'
import { openExternalURL, type UpdateStatus } from '@/api/wails'
import { useAppStore } from '@/stores/app'
import { formatBytes, formatDateTime, shortSha } from '@/shared/format'
import { displayMessage, reasonLabel } from '@/shared/labels'
import { projectMetadata } from '@/shared/project'

// props 描述组件从父级接收的入参，保证模板和脚本使用同一份契约。
const props = defineProps<{
  open: boolean
}>()
// emit 描述组件向父级抛出的事件，保证导航和动作回调具备明确边界。
const emit = defineEmits<{
  close: []
}>()

// appStore 保存 Pinia store 实例，集中访问应用共享状态和动作。
const appStore = useAppStore()
// status 保存由响应式状态推导出的只读结果，模板直接消费该值。
const status = computed(() => String(appStore.updateStatus?.status ?? 'idle'))
// progress 保存由响应式状态推导出的只读结果，模板直接消费该值。
const progress = computed(() => Math.round(appStore.updateStatus?.progressPercent ?? 0))
// isBusy 保存由响应式状态推导出的只读结果，模板直接消费该值。
const isBusy = computed(() => appStore.checking || appStore.downloading || ['downloading', 'verifying', 'installing'].includes(status.value))
// canInstall 保存由响应式状态推导出的只读结果，模板直接消费该值。
const canInstall = computed(() => Boolean(appStore.updateStatus?.verified && appStore.updateStatus?.filePath && ['verified', 'pending_install'].includes(status.value)))
// message 保存由响应式状态推导出的只读结果，模板直接消费该值。
const message = computed(() => displayMessage(appStore.updateStatus?.message ?? appStore.latestUpdateCheck?.message ?? '尚未执行更新检查。'))
// safety 保存由响应式状态推导出的只读结果，模板直接消费该值。
const safety = computed(() => updateSafety(appStore.latestUpdateCheck?.sha256 ?? appStore.updateStatus?.sha256, appStore.updateStatus?.verified))
let openRevision = 0

// Detail 保存 渲染更新状态弹窗并驱动检查、下载、安装动作 使用的配置、引用或中间结果。
const Detail = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
  },
  setup(detailProps) {
    return () => h('div', { class: 'data-row' }, [
      h('span', detailProps.label),
      h('strong', detailProps.value),
    ])
  },
})

// checkAndDownload 触发一次 Release 检查，并沿用 store 内部策略自动衔接下载。
async function checkAndDownload() {
  await appStore.checkUpdate()
}

// installNow 请求后端立即启动已校验安装包的安装流程。
async function installNow() {
  await appStore.installDownloadedUpdate()
}

// scheduleOnStartup 把已下载安装包标记为下次启动时安装，避免当前进程立即退出。
async function scheduleOnStartup() {
  await appStore.scheduleDownloadedUpdateOnStartup()
}

// closeDialog 处理 渲染更新状态弹窗并驱动检查、下载、安装动作 中的用户动作、生命周期动作或数据转换。
function closeDialog() {
  emit('close')
}

watch(() => props.open, async (open) => {
  if (!open) return
  const revision = ++openRevision
  try {
    await appStore.refreshUpdateStatus()
    if (revision !== openRevision || !props.open) return
  } catch (error) {
    appStore.applyAction({ type: 'errorSet', payload: error instanceof Error ? error.message : '读取更新状态失败' })
  }
})

// isTransferState 处理 渲染更新状态弹窗并驱动检查、下载、安装动作 中的用户动作、生命周期动作或数据转换。
function isTransferState(status?: string) {
  return status === 'downloading' || status === 'verifying'
}

// progressText 处理 渲染更新状态弹窗并驱动检查、下载、安装动作 中的用户动作、生命周期动作或数据转换。
function progressText(status: UpdateStatus | undefined, progress: number) {
  if (progress > 0) {
    return `${progress}%（${formatBytes(status?.downloadedBytes)} / ${formatBytes(status?.totalBytes)}）`
  }
  if (status?.status === 'downloading') return '正在下载'
  if (status?.status === 'verifying') return '正在校验'
  return '未开始'
}

// updateSafety 处理 渲染更新状态弹窗并驱动检查、下载、安装动作 中的用户动作、生命周期动作或数据转换。
function updateSafety(sha256?: string, verified?: boolean) {
  if (verified) {
    return { variant: 'success', title: '安装包已验证', description: '可以马上更新，也可以关闭弹窗并在下次启动时更新。' }
  }
  if (!sha256) {
    return { variant: 'danger', title: '缺少 SHA256', description: '为了避免安装未验证文件，下载和安装保持禁用。' }
  }
  return { variant: 'neutral', title: '校验信息已就绪', description: '发现可更新版本后会在后台下载并执行 SHA256 校验。' }
}

// updateStatusLabel 处理 渲染更新状态弹窗并驱动检查、下载、安装动作 中的用户动作、生命周期动作或数据转换。
function updateStatusLabel(status?: string) {
  // labels 保存 渲染更新状态弹窗并驱动检查、下载、安装动作 使用的配置、引用或中间结果。
  const labels: Record<string, string> = {
    idle: '未检查',
    update_available: '发现可更新版本',
    downloading: '正在下载',
    verifying: '正在校验',
    verified: '已校验',
    pending_install: '等待安装',
    installing: '正在安装',
    install_started: '安装器已启动',
    no_update: '当前已是最新',
    skipped: '已跳过',
    ignored: '已跳过',
    error: '更新失败',
  }
  return labels[String(status ?? '')] ?? '未检查'
}

function sourceLabel(source?: string) {
  if (source === 'local') return '本地静态服务'
  if (source === 'github') return 'GitHub Release'
  return '未获取'
}
</script>

<template>
  <!-- 模板结构：声明当前组件对外呈现的布局、插槽和交互入口。 -->
  <UiDialog :open="props.open" label="更新状态" placement="top-right" @close="closeDialog">
      <header class="dialog-header">
        <div>
          <UiBadge variant="outline">更新</UiBadge>
          <h2>{{ updateStatusLabel(status) }}</h2>
          <p>{{ message }}</p>
        </div>
        <div class="dialog-header-actions">
          <span :class="`status-pill status-${status}`">{{ updateStatusLabel(status) }}</span>
          <UiButton aria-label="关闭更新弹窗" size="icon-sm" variant="ghost" @click="closeDialog">
            <X :size="17" />
          </UiButton>
        </div>
      </header>

      <div v-if="isTransferState(status)" class="progress-block">
        <div>
          <span>{{ updateStatusLabel(status) }}</span>
          <strong>{{ progressText(appStore.updateStatus, progress) }}</strong>
        </div>
        <UiProgress :value="Math.max(4, progress)" />
      </div>

      <div :class="`safety-card ${safety.variant}`">
        <ShieldAlert v-if="safety.variant === 'danger'" :size="19" />
        <ShieldCheck v-else :size="19" />
        <div>
          <strong>{{ safety.title }}</strong>
          <span>{{ safety.description }}</span>
        </div>
      </div>

      <div class="dialog-actions">
        <UiButton class="antd-primary-action" :disabled="isBusy" @click="checkAndDownload">
          <Loader2 v-if="appStore.checking || appStore.downloading" class="animate-spin" :size="18" />
          <RefreshCw v-else :size="18" />
          检查更新
        </UiButton>
        <UiButton :disabled="!canInstall || isBusy" variant="secondary" @click="installNow">
          <Loader2 v-if="appStore.downloading && status === 'installing'" class="animate-spin" :size="18" />
          <Download v-else :size="18" />
          马上更新
        </UiButton>
        <UiButton :disabled="!canInstall" variant="ghost" @click="scheduleOnStartup">
          下次启动再更新
        </UiButton>
      </div>

      <div class="data-list">
        <Detail label="当前版本" :value="appStore.latestUpdateCheck?.currentVersion ?? appStore.appInfo?.version ?? projectMetadata.defaultVersion" />
        <Detail label="最新版本" :value="appStore.latestUpdateCheck?.latestVersion ?? appStore.updateStatus?.version ?? '未获取'" />
        <Detail label="版本标签" :value="appStore.latestUpdateCheck?.tagName ?? '未获取'" />
        <Detail label="安装包" :value="appStore.updateStatus?.assetName ?? appStore.latestUpdateCheck?.assetName ?? '未匹配'" />
        <Detail label="安装包大小" :value="formatBytes(appStore.latestUpdateCheck?.assetSizeBytes)" />
        <Detail label="SHA256" :value="shortSha(appStore.updateStatus?.sha256 ?? appStore.latestUpdateCheck?.sha256)" />
        <Detail label="更新源" :value="sourceLabel(appStore.updateStatus?.source ?? appStore.latestUpdateCheck?.source)" />
        <Detail label="缓存路径" :value="appStore.updateStatus?.filePath ?? '未下载'" />
        <Detail label="错误原因" :value="reasonLabel(appStore.updateStatus?.errorReason ?? appStore.latestUpdateCheck?.errorReason ?? appStore.latestUpdateCheck?.skipReason)" />
        <Detail label="最近检查" :value="formatDateTime(appStore.latestUpdateCheck?.checkedAt)" />
      </div>

      <footer class="dialog-footer">
        <UiButton variant="ghost" @click="openExternalURL(projectMetadata.repositoryUrl)">
          <ExternalLink :size="18" />
          打开仓库
        </UiButton>
      </footer>
  </UiDialog>
</template>

<style scoped src="./UpdateStatusDialog.css"></style>
