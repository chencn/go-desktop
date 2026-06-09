<script setup lang="ts">
import { computed } from 'vue'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'

const props = withDefaults(defineProps<{
  // confirmText 保留默认中文文案，业务页只在需要更具体动作时覆盖。
  confirmText?: string
  // description 必填，危险确认必须说明影响范围，避免退回 window.confirm。
  description: string
  // open 由业务状态控制，wrapper 不在内部持有确认弹窗状态。
  open: boolean
  title: string
}>(), {
  confirmText: '确认',
})

const emit = defineEmits<{
  close: []
  confirm: []
}>()

const dialogOpen = computed({
  get: () => props.open,
  set: (value) => {
    // AlertDialog primitive 的关闭请求统一抛给业务层，保持取消和确认后的状态收敛一致。
    if (!value) emit('close')
  },
})
</script>

<template>
  <AlertDialog v-model:open="dialogOpen">
    <!-- 危险确认不允许外部点击关闭，避免用户误以为操作已取消或已确认。 -->
    <AlertDialogContent @pointer-down-outside="(event) => event.preventDefault()">
      <AlertDialogHeader>
        <AlertDialogTitle>{{ title }}</AlertDialogTitle>
        <AlertDialogDescription>{{ description }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel as-child>
          <Button variant="outline" @click="emit('close')">取消</Button>
        </AlertDialogCancel>
        <Button variant="destructive" @click="emit('confirm')">{{ confirmText }}</Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
