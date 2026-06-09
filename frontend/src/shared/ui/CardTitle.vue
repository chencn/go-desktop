<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'

const props = defineProps<{
  // class 保留页面局部微调能力，字体族统一由 wrapper 的 app-card-title 控制。
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <CardTitle :class="cn('app-card-title', props.class)">
    <slot />
  </CardTitle>
</template>

<style scoped>
.app-card-title {
  font-family: var(--font-heading);
}
</style>
