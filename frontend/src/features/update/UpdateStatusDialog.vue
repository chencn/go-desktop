<!--
  文件职责：渲染更新状态弹窗并驱动检查、下载、安装动作。
  说明：注释覆盖组件脚本状态、方法、生命周期和模板结构；不改变渲染逻辑。
-->

<script setup lang="ts">
import { computed, watch } from 'vue'
import { Download, Loader2, RefreshCw, X } from '@lucide/vue'
import { type UpdateStatus } from '@/api/wails'
import { useAppStore } from '@/stores/app'
import { formatBytes } from '@/shared/format'
import { displayMessage } from '@/shared/labels'
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
const currentVersion = computed(() => appStore.latestUpdateCheck?.currentVersion ?? appStore.appInfo?.version ?? projectMetadata.defaultVersion)
const latestVersion = computed(() => appStore.latestUpdateCheck?.latestVersion ?? appStore.updateStatus?.version ?? '未获取')
const showProgress = computed(() => appStore.checking || isTransferState(status.value) || status.value === 'installing')
const description = computed(() => userStatusDescription())
let openRevision = 0

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
  closeDialog()
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
  if (appStore.checking) return '正在检查'
  if (progress > 0) {
    return `${progress}%（${formatBytes(status?.downloadedBytes)} / ${formatBytes(status?.totalBytes)}）`
  }
  if (status?.status === 'downloading') return '正在下载'
  if (status?.status === 'verifying') return '正在校验'
  if (status?.status === 'installing') return '正在启动安装'
  return '未开始'
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

function userStatusTitle() {
  if (appStore.checking) return '正在检查更新'
  if (status.value === 'downloading') return '正在下载更新'
  if (status.value === 'verifying') return '正在校验更新包'
  if (status.value === 'installing') return '正在启动安装'
  if (canInstall.value) return '更新包已准备好'
  if (status.value === 'install_started') return '安装器已启动'
  if (status.value === 'no_update') return '当前已是最新版本'
  if (status.value === 'update_available') return latestVersion.value === '未获取' ? '发现新版本' : `发现新版本 ${latestVersion.value}`
  if (status.value === 'error') return '更新失败'
  if (status.value === 'skipped' || status.value === 'ignored') return '本次检查已跳过'
  return '尚未检查更新'
}

function userStatusDescription() {
  if (canInstall.value) return ''
  if (status.value === 'no_update') return '当前版本已经是最新，无需操作。'
  if (status.value === 'update_available') return message.value
  if (status.value === 'error') return message.value || '更新过程中遇到问题，请稍后重试。'
  if (status.value === 'install_started') return '安装器已经打开，请按安装器提示完成更新。'
  if (showProgress.value) return message.value
  return '点击检查更新，应用会自动确认是否有新版本。'
}

function versionLine() {
  if (latestVersion.value === '未获取') return `当前版本 ${currentVersion.value}`
  return `当前版本 ${currentVersion.value} · 最新版本 ${latestVersion.value}`
}

function primaryActionLabel() {
  if (canInstall.value) return '马上更新'
  if (status.value === 'error') return '重新检查'
  return '检查更新'
}

async function runPrimaryAction() {
  if (canInstall.value) {
    await installNow()
    return
  }
  await checkAndDownload()
}

function primaryActionIcon() {
  if (isBusy.value) return Loader2
  if (canInstall.value) return Download
  return RefreshCw
}
</script>

<template>
  <!-- 模板结构：声明当前组件对外呈现的布局、插槽和交互入口。 -->
  <UiDialog :open="props.open" label="更新状态" placement="top-right" @close="closeDialog">
      <header class="dialog-header">
        <div>
          <h2>{{ userStatusTitle() }}</h2>
          <p v-if="description">{{ description }}</p>
        </div>
        <div class="dialog-header-actions">
          <UiButton aria-label="关闭更新弹窗" size="icon-sm" variant="ghost" @click="closeDialog">
            <X :size="17" />
          </UiButton>
        </div>
      </header>

      <p class="user-version-line">{{ versionLine() }}</p>

      <div v-if="showProgress" class="progress-block">
        <div>
          <span>{{ updateStatusLabel(status) }}</span>
          <strong>{{ progressText(appStore.updateStatus, progress) }}</strong>
        </div>
        <UiProgress :value="Math.max(appStore.checking ? 12 : 4, progress)" />
      </div>

      <div class="dialog-actions">
        <UiButton class="antd-primary-action" :disabled="isBusy" @click="runPrimaryAction">
          <component :is="primaryActionIcon()" :class="{ 'animate-spin': isBusy }" :size="18" />
          {{ primaryActionLabel() }}
        </UiButton>
        <UiButton v-if="canInstall" variant="secondary" @click="scheduleOnStartup">
          下次启动再更新
        </UiButton>
      </div>
  </UiDialog>
</template>

<style scoped src="./UpdateStatusDialog.css"></style>
