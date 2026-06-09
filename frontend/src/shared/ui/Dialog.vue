<script setup lang="ts">
import { computed } from 'vue'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  label: string
  open: boolean
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
    if (!value) emit('close')
  },
})
</script>

<template>
  <Dialog v-model:open="dialogOpen">
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
  box-shadow: 0 24px 70px color-mix(in oklch, var(--foreground) 18%, transparent);
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
