<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()

// delegatedAttrs 排除 class，Label 的 muted 语义和调用方 class 统一交给 cn() 合并。
const delegatedAttrs = computed(() => {
  const { class: _class, ...rest } = attrs
  return rest
})
</script>

<template>
  <span :class="cn('ui-label', attrs.class)" v-bind="delegatedAttrs">
    <slot />
  </span>
</template>

<style scoped>
.ui-label {
  color: var(--muted-foreground);
}
</style>
