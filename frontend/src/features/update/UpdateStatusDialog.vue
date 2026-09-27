<!--
  文件职责：渲染更新状态弹窗并驱动检查、下载、安装动作。
  边界：打开弹窗只读取当前状态；安装只能由用户点击主按钮显式触发。
  布局对应设计稿 .update-dialog：页头（图标标题 + 红绿灯关闭）/ 页身（版本横幅 + 状态说明 + 进度）/ 页脚（两枚按钮）。
  更新状态只讲「当前能不能升级、进度如何」，不渲染 Release 正文。
-->

<script setup lang="ts">
import { computed, watch } from 'vue'
import { Info, Loader2, RefreshCw, X } from '@lucide/vue'
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
const latestVersion = computed(() => appStore.latestUpdateCheck?.latestVersion ?? appStore.updateStatus?.version ?? '')
// hasLatest 决定版本横幅里要不要画「→ 最新版本」这一段：只有服务端版本严格更高才算有更新，
// 版本相同或更低时后端已经回 no_update，横幅不能再摆出一个像是待升级的目标版本。
const hasLatest = computed(() => isVersionAhead(latestVersion.value, currentVersion.value))
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

// versionParts 按后端 semver 包的口径解析 v?N(.N){0,2}：缺段补 0，出现非数字段即判为非法版本。
function versionParts(value: string) {
  const core = value.trim().replace(/^[vV]/, '')
  if (core === '') return null
  const segments = core.split('.')
  if (segments.length > 3) return null
  const numbers = segments.map((segment) => (/^\d+$/.test(segment) ? Number(segment) : Number.NaN))
  if (numbers.some((item) => Number.isNaN(item))) return null
  while (numbers.length < 3) numbers.push(0)
  return numbers
}

// isVersionAhead 判断服务端版本是否严格高于本地版本：版本相同或更低都不算有更新。
// 兜底方向与后端 semver.Compare 一致（非法版本低于合法版本），保证前端横幅不会比后端多喊一次「可更新」。
function isVersionAhead(candidate: string, baseline: string) {
  const left = versionParts(candidate)
  const right = versionParts(baseline)
  if (!left || !right) return Boolean(left)
  for (let index = 0; index < 3; index += 1) {
    if (left[index] !== right[index]) return left[index] > right[index]
  }
  return false
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

function userStatusDescription() {
  if (canInstall.value) return '安装包已下载并通过 SHA-256 校验，可以立即安装。'
  if (status.value === 'no_update') return '当前版本已经是最新，无需操作。'
  // 服务端版本更高才走这条分支（后端已按 semver 严格比较），文案直接给出目标版本和下一步动作。
  if (status.value === 'update_available') return `发现新版本 v${latestVersion.value}，点击「立即更新」开始下载并校验。`
  if (status.value === 'error') return message.value || '更新过程中遇到问题，请稍后重试。'
  if (status.value === 'install_started') return '安装器已经打开，请按安装器提示完成更新。'
  if (showProgress.value) return message.value
  return '点击「检查更新」，应用会自动确认是否有新版本。'
}

function primaryActionLabel() {
  if (canInstall.value) return '立即下载并安装'
  if (status.value === 'error') return '重新检查'
  // 已知有更高版本时不能再喊「检查更新」，否则用户以为要点两次才知道有更新。
  if (status.value === 'update_available') return '立即更新'
  return '检查更新'
}

async function runPrimaryAction() {
  if (canInstall.value) {
    await installNow()
    return
  }
  await checkAndDownload()
}

// secondaryActionLabel 把设计稿的「稍后处理」让位给真实分支：包已就绪时这一格用来延后安装。
function secondaryActionLabel() {
  return canInstall.value ? '下次启动再更新' : '稍后处理'
}

async function runSecondaryAction() {
  if (canInstall.value) {
    await scheduleOnStartup()
    return
  }
  closeDialog()
}
</script>

<template>
  <!-- 外部点击不关闭弹窗（close-on-click-modal=false），Esc 仍可关闭，与项目级弹窗策略一致。 -->
  <!-- 不传 width：480px 的弹窗宽度由 styles/element-plus.css 的 --dialog-width 决定。 -->
  <el-dialog
    :model-value="props.open"
    align-center
    :close-on-click-modal="false"
    :show-close="false"
    class="apple-dialog update-status-dialog"
    aria-label="应用更新状态"
    @close="closeDialog"
  >
    <template #header>
      <span class="dialog-title">
        <RefreshCw :size="18" aria-hidden="true" />
        应用更新状态
      </span>
      <button type="button" class="modal-close-btn" title="关闭" aria-label="关闭更新弹窗" @click="closeDialog">
        <X :size="12" aria-hidden="true" />
      </button>
    </template>

    <div class="update-info-banner">
      <Info :size="24" :stroke-width="2" aria-hidden="true" />
      <div class="update-versions">
        <span>当前版本 <strong class="ver-badge">v{{ currentVersion }}</strong></span>
        <template v-if="hasLatest">
          <span aria-hidden="true">→</span>
          <span>最新版本 <strong class="ver-badge ok">v{{ latestVersion }}</strong></span>
        </template>
      </div>
    </div>

    <p v-if="description" class="update-message">{{ description }}</p>

    <div v-if="showProgress" class="update-progress">
      <!-- EP 把 stroke-width 和填充色写进行内样式，只能走 props：色值传 CSS 变量以跟随亮暗模式 -->
      <el-progress
        class="apple-progress"
        color="var(--accent)"
        :percentage="Math.min(100, Math.max(appStore.checking ? 12 : 4, progress))"
        :show-text="false"
        :stroke-width="8"
      />
      <div class="progress-meta">
        <span>{{ updateStatusLabel(status) }}</span>
        <span>{{ progressText(appStore.updateStatus, progress) }}</span>
      </div>
    </div>

    <template #footer>
      <el-button class="btn-apple" @click="runSecondaryAction">{{ secondaryActionLabel() }}</el-button>
      <el-button class="btn-apple" type="primary" :disabled="isBusy" @click="runPrimaryAction">
        <Loader2 v-if="isBusy" class="animate-spin" :size="15" aria-hidden="true" />
        {{ primaryActionLabel() }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped src="./UpdateStatusDialog.css"></style>
