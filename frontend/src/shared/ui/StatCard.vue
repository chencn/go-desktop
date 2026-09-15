<!--
  文件职责：通用指标统计卡片（UiStatCard）。
  规范统一：
  1. 水平布局（左图标、右上方 Label、右下方 Value）
  2. 图标容器固定 40px × 40px，内部 SVG 标准尺寸 18px，配合 icon-tone-* 语义色
  3. 阴影与微动统一承接全局 card.css 与 artistic-shadow token，禁止页面层随意覆盖
  4. 支持长文本自适应缩小、语义状态点（ok / warning / error / pending）
-->

<script setup lang="ts">
import type { Component, HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import CardCompat from './Card.vue'

export type StatTone =
  | 'icon-tone-indigo'
  | 'icon-tone-green'
  | 'icon-tone-orange'
  | 'icon-tone-blue'
  | 'icon-tone-purple'
  | 'icon-tone-amber'
  | 'icon-tone-red'
  | 'icon-tone-cyan'
  | string

export type StatStatus = 'ok' | 'warning' | 'error' | 'pending'

const props = withDefaults(
  defineProps<{
    label: string
    value?: string | number
    icon?: Component
    tone?: StatTone
    // text 标记值是不定长文本（如打印机名、主题名），字号降到正文大小避免撑高
    text?: boolean
    clickable?: boolean
    status?: StatStatus
    statusLabel?: string
    // hint 是 value 下方的次要说明；状态卡用它承载接口返回值等辅助信息
    hint?: string
    // valueClass 允许为数值追加语义色（如 is-ok / is-error / is-pending）
    valueClass?: string
    class?: HTMLAttributes['class']
  }>(),
  {
    value: '',
    icon: undefined,
    tone: 'icon-tone-indigo',
    text: false,
    clickable: false,
    status: undefined,
    statusLabel: '',
    hint: '',
    valueClass: '',
    class: '',
  },
)

defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()
</script>

<template>
  <CardCompat
    :class="cn('app-stat-card', clickable && 'is-clickable', props.class)"
    @click="(e) => clickable && $emit('click', e)"
  >
    <div class="stat-card-body">
      <!-- 左侧图标盒子：固定 40x40，配合 icon-tone-* 语义色 -->
      <slot name="icon">
        <span
          v-if="icon"
          :class="cn('stat-card-icon data-icon', tone)"
          aria-hidden="true"
        >
          <component :is="icon" :size="18" />
        </span>
      </slot>

      <!-- 右侧文案：Label 在上，Value 在下 -->
      <div class="stat-card-copy">
        <div class="stat-card-label-row">
          <slot name="label">
            <small class="stat-card-label">{{ label }}</small>
          </slot>

          <!-- action 承载标签行右侧的徽标等附加内容；不传时不占位 -->
          <slot name="action" />
        </div>

        <slot name="value">
          <span v-if="status" :class="cn('stat-card-status-inline', `is-${status}`)">
            <span class="stat-card-status-dot" aria-hidden="true" />
            {{ statusLabel || value }}
          </span>
          <strong
            v-else
            :class="cn('stat-card-value', text && 'is-text', valueClass)"
            :title="String(value)"
          >
            {{ value }}
          </strong>
        </slot>

        <small v-if="hint" class="stat-card-hint" :title="hint">{{ hint }}</small>
      </div>
    </div>
  </CardCompat>
</template>

<style scoped>
.app-stat-card {
  min-width: 0;
  min-height: 84px;
  gap: 0 !important;
  padding: 0 !important;
  transition: transform var(--duration-slow, 200ms) ease,
              box-shadow var(--duration-slow, 200ms) ease,
              border-color var(--duration-slow, 200ms) ease !important;
}

.app-stat-card.is-clickable {
  cursor: pointer;
}

.app-stat-card.is-clickable:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--primary) 35%, var(--border)) !important;
  box-shadow: var(--artistic-shadow-lg) !important;
}

.stat-card-body {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 12px;
  padding: 14px 18px !important;
}

/* 图标容器：40px x 40px，色彩完全继承 .data-icon 与 data-icon-tone 规则 */
.stat-card-icon {
  width: 40px;
  height: 40px;
  transition: transform 200ms ease;
}

.app-stat-card.is-clickable:hover .stat-card-icon {
  transform: scale(1.06);
}

.stat-card-copy {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 4px;
}

.stat-card-label-row {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.stat-card-label {
  overflow: hidden;
  color: var(--muted-foreground);
  font-size: var(--fs-caption);
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stat-card-value {
  overflow: hidden;
  color: var(--foreground);
  font-size: calc(var(--fs-section) + 4px);
  font-weight: 650;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 不定长长文本值降级为正文字号，保证卡片高度一致 */
.stat-card-value.is-text {
  font-size: var(--fs-body);
  line-height: 1.35;
}

.stat-card-value.is-ok {
  color: var(--icon-green);
}

.stat-card-value.is-error {
  color: var(--destructive);
}

.stat-card-value.is-pending,
.stat-card-value.is-warning {
  color: var(--icon-amber);
}

.stat-card-hint {
  overflow: hidden;
  color: var(--muted-foreground);
  font-size: var(--fs-caption);
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 状态药丸与脉冲点 */
.stat-card-status-inline {
  display: inline-flex;
  min-width: max-content;
  align-items: center;
  gap: 5px;
  font-size: var(--fs-caption);
  font-weight: 650;
  line-height: 1;
}

.stat-card-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: currentColor;
}

.stat-card-status-inline.is-ok {
  color: var(--icon-green);
}

.stat-card-status-inline.is-warning,
.stat-card-status-inline.is-pending {
  color: var(--icon-orange);
}

.stat-card-status-inline.is-error {
  color: var(--destructive);
}
</style>
