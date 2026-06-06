<script setup lang="ts">
import { computed } from 'vue'
import { Progress } from '@/components/ui/progress'

const props = defineProps<{
  modelValue?: number | null
  value?: number | null
}>()

const progressValue = computed(() => props.modelValue ?? props.value ?? 0)
</script>

<template>
  <Progress v-bind="$attrs" :model-value="progressValue" />
</template>
