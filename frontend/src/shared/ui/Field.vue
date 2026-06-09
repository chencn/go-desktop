<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()

// delegatedAttrs 排除 class，避免 attrs.class 被 v-bind 再写一次并覆盖 cn() 合并后的样式。
const delegatedAttrs = computed(() => {
  const { class: _class, ...rest } = attrs
  return rest
})
</script>

<template>
  <label :class="cn('ui-field', attrs.class)" v-bind="delegatedAttrs">
    <slot />
  </label>
</template>

<style scoped>
.ui-field {
  display: grid;
  min-width: 0;
  gap: 7px;
  color: var(--foreground);
  font-size: var(--fs-caption);
  font-weight: 650;
}
</style>
