<script setup lang="ts">
import { computed } from 'vue'
import { Progress } from '@/components/ui/progress'

const props = defineProps<{
  // modelValue 兼容 v-model，value 兼容 shadcn Progress 的常规调用写法；两者都缺失时显示 0。
  modelValue?: number | null
  value?: number | null
}>()

// progressValue 负责把空值归一化，避免 primitive 收到 null 后出现不确定宽度。
const progressValue = computed(() => props.modelValue ?? props.value ?? 0)
</script>

<template>
  <Progress v-bind="$attrs" :model-value="progressValue" />
</template>
