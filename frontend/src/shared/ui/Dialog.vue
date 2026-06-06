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
      @pointer-down-outside="emit('close')"
    >
      <DialogTitle class="sr-only">{{ label }}</DialogTitle>
      <slot />
    </DialogContent>
  </Dialog>
</template>
