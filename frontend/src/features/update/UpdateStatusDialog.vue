<!--
  文件职责：渲染更新状态弹窗并驱动检查、下载、安装动作。
  边界：打开弹窗只读取当前状态；安装只能由用户点击主按钮显式触发。
-->

<script setup lang="ts">
import { computed, watch } from 'vue'
import { Download, Loader2, RefreshCw, X } from '@lucide/vue'
import { type UpdateStatus } from '@/api/wails'
import { useAppStore } from '@/stores/app'
import { formatBytes } from '@/shared/format'
import { displayMessage } from '@/shared/labels'
import { projectMetadata } from '@/shared/project'

const props = defineProps<{
  // 由 AppChrome 控制弹窗可见性；组件内部不会自行保持打开状态。
  open: boolean
}>()
const emit = defineEmits<{
  // close 只通知父级收起弹窗，不取消正在进行的后端下载或安装流程。
  close: []
}>()

const appStore = useAppStore()
// status 统一把缺失状态视为 idle，避免后端状态尚未返回时按钮文案抖动。
const status = computed(() => String(appStore.updateStatus?.status ?? 'idle'))
// progress 只做展示取整；真实字节数仍从 updateStatus 读取。
const progress = computed(() => Math.round(appStore.updateStatus?.progressPercent ?? 0))
// isBusy 聚合 store 标志和后端生命周期状态，统一锁住检查/安装入口。
const isBusy = computed(() => appStore.checking || appStore.downloading || ['downloading', 'verifying', 'installing'].includes(status.value))
// canInstall 要求后端返回已校验文件路径；只看状态名不足以证明安装包可用。
const canInstall = computed(() => Boolean(appStore.updateStatus?.verified && appStore.updateStatus?.filePath && ['verified', 'pending_install'].includes(status.value)))
// message 优先用生命周期状态消息，其次用最近一次检查消息，最后才给空状态提示。
const message = computed(() => displayMessage(appStore.updateStatus?.message ?? appStore.latestUpdateCheck?.message ?? '尚未执行更新检查。'))
const currentVersion = computed(() => appStore.latestUpdateCheck?.currentVersion ?? appStore.appInfo?.version ?? projectMetadata.defaultVersion)
const latestVersion = computed(() => appStore.latestUpdateCheck?.latestVersion ?? appStore.updateStatus?.version ?? '未获取')
const showProgress = computed(() => appStore.checking || isTransferState(status.value) || status.value === 'installing')
const description = computed(() => userStatusDescription())
// openRevision 用来丢弃过期的打开刷新结果，避免快速开关弹窗后旧请求覆盖错误状态。
let openRevision = 0

// checkAndDownload 调用后端 CheckUpdate；store 会随后读取 GetUpdateStatus 并刷新日志。
async function checkAndDownload() {
  await appStore.checkUpdate()
}

// installNow 请求后端 InstallDownloadedUpdate，通常会启动安装器并进入安装生命周期。
async function installNow() {
  await appStore.installDownloadedUpdate()
}

// scheduleOnStartup 把已下载安装包标记为下次启动时安装，避免当前进程立即退出。
async function scheduleOnStartup() {
  await appStore.scheduleDownloadedUpdateOnStartup()
  closeDialog()
}

// closeDialog 只关闭 UI；下载进度仍靠 Wails 事件继续同步到 store。
function closeDialog() {
  emit('close')
}

// 弹窗打开时只刷新 GetUpdateStatus，不能在这里自动 install，避免“查看状态”变成隐式升级。
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

// isTransferState 用于进度条展示；installing 单独处理，因为它没有下载字节进度语义。
function isTransferState(status?: string) {
  return status === 'downloading' || status === 'verifying'
}

// progressText 在没有百分比时回退到阶段文案，避免 0% 被误读为下载失败。
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

// updateStatusLabel 映射后端状态机值；未知状态回退为未检查，避免把内部状态直接暴露到 UI。
function updateStatusLabel(status?: string) {
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
  <UiDialog :open="props.open" label="更新状态" placement="top-right" @close="closeDialog">
      <header class="dialog-header" :class="{ 'is-no-update': status === 'no_update' }">
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
        <UiButton :disabled="isBusy" @click="runPrimaryAction">
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
