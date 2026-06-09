<script setup lang="ts">
import { computed } from 'vue'
import { Switch } from '@/components/ui/switch'

const props = defineProps<{
  // checked 是业务设置的受控值，wrapper 负责映射到 primitive 的 model-value。
  checked?: boolean
  disabled?: boolean
  // ariaLabel 用于无可见文本的开关，页面有可见 label 时可不传。
  ariaLabel?: string
}>()

const emit = defineEmits<{
  'update:checked': [value: boolean]
}>()

const checkedModel = computed({
  get: () => Boolean(props.checked),
  // primitive 可能传回 truthy/falsy 值，这里统一收敛成 boolean 再通知业务层。
  set: (value) => emit('update:checked', Boolean(value)),
})
</script>

<template>
  <Switch class="ui-switch" :model-value="checkedModel" :aria-label="ariaLabel" :disabled="disabled" @update:model-value="checkedModel = $event" />
</template>

<style scoped>
.ui-switch[data-state="checked"] {
  background: var(--primary);
}
</style>
