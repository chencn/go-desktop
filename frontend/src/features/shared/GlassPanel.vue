<!--
  文件职责：Apple 液态玻璃材质宿主——渲染三层光学图层（磨砂折射 / 边缘反射 / 色散漫射）与独立内容层。
  材质规则全部在 styles/liquid-glass.css，本组件只维护 DOM 契约和指针随动光学。
  用法：<GlassPanel class="service-card">内容</GlassPanel>，附加 class 会合并到宿主 .lg 上。
-->

<script setup lang="ts">
import { ref } from 'vue'
import { useGlassPointerOptics } from '@/app/glass'

const host = ref<HTMLElement>()
useGlassPointerOptics(() => host.value)
</script>

<template>
  <div ref="host" class="lg" data-glass>
    <div class="lg-surface"></div>
    <div class="lg-light"></div>
    <div class="lg-sheen"></div>
    <div class="lg-content">
      <slot />
    </div>
  </div>
</template>
