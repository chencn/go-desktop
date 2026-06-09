<!--
  文件职责：装配 Pinia 应用状态、顶层页面切换和桌面壳布局。
  说明：注释覆盖组件脚本状态、方法、生命周期和模板结构；不改变渲染逻辑。
-->

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { viewComponents } from '@/app/routes'
import { useAppStore } from './stores/app'
import AppChrome from './features/layout/AppChrome.vue'
import LicensePage from './features/license/LicensePage.vue'
import type { ViewKey } from './shared/views'

// appStore 保存 Pinia store 实例，集中访问应用共享状态和动作。
const appStore = useAppStore()
// activeView 保存当前前端页面；桌面端不走 URL 路由，避免刷新时依赖浏览器历史。
const activeView = ref<ViewKey>('home')
// activeViewComponent 从集中路由表取组件，让 App.vue 不再维护页面 import/switch。
const activeViewComponent = computed(() => viewComponents[activeView.value])

// navigate 处理装配 Pinia 应用状态、顶层页面切换和桌面壳布局 中的用户动作、生命周期动作或数据转换。
function navigate(view: ViewKey) {
  activeView.value = view
}

// onMounted 在组件挂载后启动页面初始化、事件订阅或异步加载。
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
  <!-- 模板结构：声明当前组件对外呈现的布局、插槽和交互入口。 -->
  <LicensePage v-if="appStore.licenseStatus?.required && !appStore.licenseStatus?.authorized" />
  <AppChrome v-else-if="appStore.licenseStatus" :active-view="activeView" @navigate="navigate">
    <component :is="activeViewComponent" />
  </AppChrome>
</template>
