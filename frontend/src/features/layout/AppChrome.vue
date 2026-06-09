<!--
  文件职责：渲染桌面应用外壳、侧栏导航、页头和更新入口。
  更新按钮只负责打开状态弹窗；检查、安装和下次启动安装由弹窗内显式动作触发。
-->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { AppWindow, Moon, RefreshCw, Sun } from '@lucide/vue'
import { useDisplayPreferences } from '@/app/display'
import { toMessage } from '@/app/state'
import { useAppStore } from '@/stores/app'
import { cn } from '@/lib/utils'
import { projectMetadata } from '@/shared/project'
import { navigation, pageSubtitle, pageTitle, type ViewKey } from '@/shared/views'
import UpdateStatusDialog from '@/features/update/UpdateStatusDialog.vue'

const props = defineProps<{
  // activeView 由 App.vue 持有，保证侧栏和窄屏导航共享同一个页面状态。
  activeView: ViewKey
}>()

const emit = defineEmits<{
  // navigate 只通知父级切换前端页面，不触发后端路由或浏览器历史变更。
  navigate: [view: ViewKey]
}>()

const appStore = useAppStore()
// display 控制全局主题 facade；保存失败时要回滚该 facade，而不是只报错。
const display = useDisplayPreferences()
// updateOpen 是弹窗开关，关闭弹窗不会取消 store 中的更新任务。
const updateOpen = ref(false)
// activeTitle/activeSubtitle 从共享导航配置读取，避免侧栏文案和页头漂移。
const activeTitle = computed(() => pageTitle(props.activeView))
const activeSubtitle = computed(() => pageSubtitle(props.activeView))
// updateTone 把后端更新状态压缩成图标状态类，避免模板散落状态机判断。
const updateTone = computed(() => updateIconTone(appStore.updateStatus?.status))

// toggleTheme 先乐观切换 DOM 主题，再调用 SaveDisplayPreferences；失败后恢复原主题。
async function toggleTheme() {
  const previous = display.themeMode.value
  const next = previous === 'dark' ? 'light' : 'dark'
  display.setThemeMode(next)
  try {
    await appStore.persistDisplayPreferences()
  } catch (error) {
    display.setThemeMode(previous)
    appStore.applyAction({ type: 'errorSet', payload: toMessage(error) || '主题保存失败' })
  }
}

// updateIconTone 只关心用户可感知阶段：错误、忙碌、已准备，其余保持默认。
function updateIconTone(status?: string) {
  const value = String(status ?? 'idle')
  if (value === 'error') return 'is-danger'
  if (['downloading', 'verifying', 'installing'].includes(value)) return 'is-busy'
  if (['update_available', 'verified', 'pending_install'].includes(value)) return 'is-ready'
  return ''
}
</script>

<template>
  <div class="app-shell">
    <aside class="app-sidebar" aria-label="主导航">
      <div class="sidebar-brand">
        <span class="brand-mark" aria-hidden="true">
          <AppWindow :size="19" />
        </span>
        <span class="brand-copy">
          <strong>{{ appStore.appInfo?.name ?? projectMetadata.appName }}</strong>
        </span>
      </div>

      <nav class="sidebar-nav">
        <button
          v-for="item in navigation"
          :key="item.key"
          :class="cn('sidebar-item', props.activeView === item.key && 'is-active')"
          :title="item.label"
          type="button"
          @click="emit('navigate', item.key)"
        >
          <span :class="cn('nav-icon', props.activeView !== item.key && item.tone)" aria-hidden="true">
            <component :is="item.icon" :size="18" />
          </span>
          <span>{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <main class="app-main">
      <header class="topbar">
        <div class="topbar-title">
          <h1>{{ activeTitle }}</h1>
          <p>{{ activeSubtitle }}</p>
        </div>
        <div class="topbar-utility">
          <div class="topbar-actions" aria-label="全局操作">
            <UiTooltip :content="display.themeMode.value === 'dark' ? '切换到日间模式' : '切换到夜间模式'">
              <UiButton
                :aria-label="display.themeMode.value === 'dark' ? '切换到日间模式' : '切换到夜间模式'"
                size="icon"
                variant="ghost"
                @click="toggleTheme"
              >
                <Sun v-if="display.themeMode.value === 'dark'" class="icon-tone-orange" :size="18" />
                <Moon v-else class="icon-tone-purple" :size="18" />
              </UiButton>
            </UiTooltip>
            <UiTooltip content="更新状态">
              <UiButton
                aria-label="查看更新状态"
                :class="cn('update-icon-button', updateTone)"
                size="icon"
                variant="ghost"
                @click="updateOpen = true"
              >
                <RefreshCw :class="cn(updateTone === 'is-danger' ? 'icon-tone-red' : updateTone === 'is-ready' ? 'icon-tone-green' : 'icon-tone-blue')" :size="18" />
              </UiButton>
            </UiTooltip>
          </div>
          <nav class="compact-nav" aria-label="窄窗口导航">
            <button
              v-for="item in navigation"
              :key="item.key"
              :class="cn('compact-nav-item', props.activeView === item.key && 'is-active')"
              type="button"
              @click="emit('navigate', item.key)"
            >
              <component :is="item.icon" :class="cn(props.activeView !== item.key && item.tone)" :size="17" />
              <span>{{ item.label }}</span>
            </button>
          </nav>
        </div>
      </header>

      <section class="content-scroll">
        <div v-if="appStore.errorMessage" class="app-error">{{ appStore.errorMessage }}</div>
        <slot />
      </section>
    </main>
    <UpdateStatusDialog :open="updateOpen" @close="updateOpen = false" />
  </div>
</template>

<style scoped src="./AppChrome.css"></style>
