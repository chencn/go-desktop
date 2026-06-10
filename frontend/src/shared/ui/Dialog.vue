<script setup lang="ts">
import { computed } from 'vue'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  // label 只供无可见标题的弹窗写入 sr-only 标题，保证 Wails 桌面弹窗仍有可访问名称。
  label: string
  // open 由业务页托管，wrapper 只把 primitive 的关闭请求翻译成 close 事件。
  open: boolean
  // placement 目前只服务更新弹窗的右上角浮层，其它弹窗保持居中。
  placement?: 'center' | 'top-right'
}>(), {
  placement: 'center',
})

const emit = defineEmits<{
  close: []
}>()

const dialogOpen = computed({
  get: () => props.open,
  set: (value) => {
    // primitive 在 Esc、关闭按钮或受控 open=false 时都会写回；业务层统一通过 close 收敛状态。
    if (!value) emit('close')
  },
})
</script>

<template>
  <Dialog v-model:open="dialogOpen">
    <!-- 外部点击只阻止关闭，不拦截 Esc；项目级策略放 wrapper，避免改 shadcn-vue primitive。 -->
    <DialogContent
      :show-close-button="false"
      :class="cn(
        'ui-dialog-content',
        placement === 'top-right' && 'ui-dialog-content-top-right',
      )"
      @escape-key-down="emit('close')"
      @pointer-down-outside="(event) => event.preventDefault()"
    >
      <DialogTitle class="sr-only">{{ label }}</DialogTitle>
      <slot />
    </DialogContent>
  </Dialog>
</template>

<style scoped>
.ui-dialog-content {
  display: grid;
  width: min(100%, 560px);
  max-height: calc(100vh - 112px);
  gap: 16px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  background: var(--popover);
  color: var(--popover-foreground);
  box-shadow: 0 24px 70px color-mix(in oklch, var(--foreground) 18%, var(--color-transparent));
  padding: 18px;
}

.ui-dialog-content-top-right {
  top: 84px;
  right: 28px;
  left: auto;
  width: min(100% - 56px, 560px);
  max-height: calc(100vh - 112px);
  transform: none;
}

@media (max-width: 980px) {
  .ui-dialog-content {
    max-height: calc(100vh - 86px);
    width: 100%;
  }
}
</style>
