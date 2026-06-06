<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ChevronDown } from '@lucide/vue'

const props = defineProps<{
  disabled?: boolean
  label: string
  modelValue: string
  options: Array<[string, string]>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const selectedLabel = computed(() => props.options.find(([value]) => value === props.modelValue)?.[1] ?? props.modelValue)

watch(() => props.disabled, (disabled) => {
  if (disabled) {
    open.value = false
  }
})

function closeOnBlur(event: FocusEvent) {
  const current = event.currentTarget as HTMLElement
  const next = event.relatedTarget as Node | null
  if (!next || !current.contains(next)) {
    open.value = false
  }
}

function toggleOpen() {
  if (props.disabled) return
  open.value = !open.value
}

function selectValue(value: string) {
  if (props.disabled) return
  emit('update:modelValue', value)
  open.value = false
}
</script>

<template>
  <div :class="['preference-color-select', open && 'is-open', disabled && 'is-disabled']" @focusout="closeOnBlur">
    <button
      type="button"
      class="preference-color-trigger"
      :aria-expanded="open ? 'true' : 'false'"
      aria-haspopup="listbox"
      :aria-label="label"
      :disabled="disabled"
      @click="toggleOpen"
      @keydown.escape="open = false"
    >
      <span class="preference-color-value">
        <span class="preference-swatch" :data-accent="modelValue" aria-hidden="true" />
        <span>{{ selectedLabel }}</span>
      </span>
      <span class="preference-color-caret" aria-hidden="true"><ChevronDown :size="14" /></span>
    </button>

    <div v-if="open && !disabled" class="preference-color-menu" role="listbox" :aria-label="label">
      <button
        v-for="[value, label] in options"
        :key="value"
        type="button"
        :class="['preference-color-option', value === modelValue && 'is-selected']"
        role="option"
        :aria-selected="value === modelValue ? 'true' : 'false'"
        @click="selectValue(value)"
      >
        <span class="preference-swatch" :data-accent="value" aria-hidden="true" />
        <span>{{ label }}</span>
      </button>
    </div>
  </div>
</template>
