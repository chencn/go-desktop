<!--
  文件职责：渲染设计稿 .apple-alert-dialog 的居中二次确认弹窗（图标方块 + 标题 + 说明 + 等宽按钮）。
  边界：只负责展示与回传确认/关闭，业务动作由调用方执行；材质与尺寸由 el-dialog.apple-dialog.is-alert 皮肤决定。
-->

<script setup lang="ts">
import { TriangleAlert } from '@lucide/vue'

const props = withDefaults(
  defineProps<{
    cancelText?: string
    confirmText?: string
    // destructive 决定确认键是红色危险填充（设计稿 .btn-apple.danger）还是蓝色主色填充。
    destructive?: boolean
    open: boolean
    title: string
  }>(),
  { cancelText: '取消', confirmText: '确认', destructive: true },
)

const emit = defineEmits<{
  close: []
  confirm: []
}>()
</script>

<template>
  <!-- 不传 width：宽度交给皮肤的 --alert-dialog-width，避免 EP 行内自定义属性覆盖设计稿尺寸。 -->
  <el-dialog
    :model-value="props.open"
    class="apple-dialog is-alert"
    :show-close="false"
    :close-on-click-modal="false"
    :aria-label="props.title"
    @close="emit('close')"
  >
    <div class="alert-icon-sq" :class="{ 'is-neutral': !props.destructive }" aria-hidden="true">
      <TriangleAlert :size="26" :stroke-width="2.2" />
    </div>
    <div class="alert-header-text" :class="{ 'is-neutral': !props.destructive }">
      <h3>{{ props.title }}</h3>
      <p>
        <slot />
      </p>
    </div>
    <div class="alert-actions-row">
      <el-button class="btn-apple is-alert-action" @click="emit('close')">{{ props.cancelText }}</el-button>
      <el-button
        class="btn-apple is-alert-action"
        :type="props.destructive ? 'danger' : 'primary'"
        @click="emit('confirm')"
      >
        {{ props.confirmText }}
      </el-button>
    </div>
  </el-dialog>
</template>

<style scoped src="./AlertDialog.css"></style>
