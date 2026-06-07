<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()
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
