<!-- 文件职责：授权闸门视图，未授权时全屏承载 LicenseCard 并让它保持可读宽度。 -->
<script setup lang="ts">
import LicenseCard from './LicenseCard.vue'
</script>

<template>
  <main class="license-shell">
    <div class="license-stage">
      <LicenseCard />
    </div>
  </main>
</template>

<style scoped src="./LicensePage.css"></style>
