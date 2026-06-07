<script setup lang="ts">
import { computed } from 'vue'
import { Switch } from '@/components/ui/switch'

const props = defineProps<{
  checked?: boolean
  disabled?: boolean
  ariaLabel?: string
}>()

const emit = defineEmits<{
  'update:checked': [value: boolean]
}>()

const checkedModel = computed({
  get: () => Boolean(props.checked),
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
