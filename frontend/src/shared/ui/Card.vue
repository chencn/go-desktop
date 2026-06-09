<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

const props = defineProps<{
  // class 继续向 primitive 透传，wrapper 只追加项目级 app-card 边框 token。
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <Card :class="cn('app-card', props.class)">
    <slot />
  </Card>
</template>

<style scoped>
.app-card {
  border-color: var(--card-border);
}
</style>
