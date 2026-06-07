<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'

const props = defineProps<{
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
