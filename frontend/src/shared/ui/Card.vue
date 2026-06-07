<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

const props = defineProps<{
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
