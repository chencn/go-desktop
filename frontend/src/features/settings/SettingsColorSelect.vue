<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ChevronDown } from '@lucide/vue'

const props = defineProps<{
  // disabled 由设置页根据加载状态和显示方案托管规则传入；禁用时下拉必须收起。
  disabled?: boolean
  // label 同时作为按钮和 listbox 的无障碍名称。
  label: string
  // modelValue 是持久化偏好值，必须能在 options 中找到对应展示名。
  modelValue: string
  // options 使用 [value, label] 结构，value 同时驱动色板 data-accent。
  options: Array<[string, string]>
}>()

const emit = defineEmits<{
  // update:modelValue 只上报新值，保存动作由父组件负责。
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const selectedLabel = computed(() => props.options.find(([value]) => value === props.modelValue)?.[1] ?? props.modelValue)

// 父组件切换到托管或未加载状态时，立刻关闭已经展开的菜单。
watch(() => props.disabled, (disabled) => {
  if (disabled) {
    open.value = false
  }
})

// focusout 只在焦点离开整个控件时关闭，允许用户在菜单按钮之间移动焦点。
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

<style scoped src="./SettingsColorSelect.css"></style>
