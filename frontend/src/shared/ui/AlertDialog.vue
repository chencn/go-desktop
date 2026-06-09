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
  confirmText?: string
  description: string
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
    if (!value) emit('close')
  },
})
</script>

<template>
  <AlertDialog v-model:open="dialogOpen">
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
