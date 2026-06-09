<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  // modelValue 接受 number 是为了兼容设置页枚举值，DOM change 仍按 select 原生规则回传字符串。
  modelValue?: string | number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const attrs = useAttrs()

// delegatedAttrs 排除 class，避免透传属性覆盖 wrapper 固定的高度、边框和禁用态样式。
const delegatedAttrs = computed(() => {
  const { class: _class, ...rest } = attrs
  return rest
})
</script>

<template>
  <select
    :class="cn('ui-native-select flex h-9 w-full min-w-0 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs transition-colors disabled:cursor-not-allowed disabled:opacity-50', attrs.class)"
    :value="props.modelValue"
    v-bind="delegatedAttrs"
    @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
  >
    <slot />
  </select>
</template>

<style scoped>
.ui-native-select {
  min-height: var(--control-height);
}
</style>
