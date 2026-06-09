<!--
  文件职责：装配 Pinia 应用状态、顶层页面切换和桌面壳布局。
  授权未通过时只渲染授权页；其余页面由内存中的 ViewKey 控制，不接入 URL 路由。
-->

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { viewComponents } from '@/app/routes'
import { useAppStore } from './stores/app'
import AppChrome from './features/layout/AppChrome.vue'
import LicensePage from './features/license/LicensePage.vue'
import type { ViewKey } from './shared/views'

const appStore = useAppStore()
// activeView 保存当前前端页面；桌面端不走 URL 路由，避免刷新时依赖浏览器历史。
const activeView = ref<ViewKey>('home')
// activeViewComponent 从集中路由表取组件，让 App.vue 不再维护页面 import/switch。
const activeViewComponent = computed(() => viewComponents[activeView.value])

// navigate 只更新本地视图状态；页面数据仍由 store 初始化和各页面自己的后端调用维护。
function navigate(view: ViewKey) {
  activeView.value = view
}

// initialise 会先读取授权状态；未授权时 store 提前结束初始化，避免继续请求设置、日志和更新信息。
onMounted(() => {
  void appStore.initialise()
  appStore.subscribeRuntimeUpdates()
})

// onUnmounted 在组件卸载前释放订阅和运行时资源，避免重复监听。
onUnmounted(() => {
  appStore.unsubscribeRuntimeUpdates()
})
</script>

<template>
  <LicensePage v-if="appStore.licenseStatus?.required && !appStore.licenseStatus?.authorized" />
  <AppChrome v-else-if="appStore.licenseStatus" :active-view="activeView" @navigate="navigate">
    <component :is="activeViewComponent" />
  </AppChrome>
</template>
