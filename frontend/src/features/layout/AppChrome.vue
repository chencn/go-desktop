<!--
  文件职责：渲染桌面应用外壳、侧栏导航、页头和更新入口。
  说明：注释覆盖组件脚本状态、方法、生命周期和模板结构；不改变渲染逻辑。
-->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { AppWindow, Bell, Moon, Sun } from '@lucide/vue'
import { useDisplayPreferences } from '@/app/display'
import { toMessage } from '@/app/state'
import { useAppStore } from '@/stores/app'
import { cn } from '@/lib/utils'
import { projectMetadata } from '@/shared/project'
import { navigation, pageSubtitle, pageTitle, type ViewKey } from '@/shared/views'
import UpdateStatusDialog from '@/features/update/UpdateStatusDialog.vue'

// props 描述组件从父级接收的入参，保证模板和脚本使用同一份契约。
const props = defineProps<{
  activeView: ViewKey
}>()

// emit 描述组件向父级抛出的事件，保证导航和动作回调具备明确边界。
const emit = defineEmits<{
  navigate: [view: ViewKey]
}>()

// appStore 保存 Pinia store 实例，集中访问应用共享状态和动作。
const appStore = useAppStore()
// display 保存 渲染桌面应用外壳、侧栏导航、页头和更新入口 使用的配置、引用或中间结果。
const display = useDisplayPreferences()
// updateOpen 保存组件本地响应式状态，是模板渲染和事件处理的状态源。
const updateOpen = ref(false)
// activeTitle 保存由响应式状态推导出的只读结果，模板直接消费该值。
const activeTitle = computed(() => pageTitle(props.activeView))
// activeSubtitle 保存由响应式状态推导出的只读结果，模板直接消费该值。
const activeSubtitle = computed(() => pageSubtitle(props.activeView))
// updateTone 保存由响应式状态推导出的只读结果，模板直接消费该值。
const updateTone = computed(() => updateIconTone(appStore.updateStatus?.status))

// toggleTheme 处理 渲染桌面应用外壳、侧栏导航、页头和更新入口 中的用户动作、生命周期动作或数据转换。
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

// updateIconTone 处理 渲染桌面应用外壳、侧栏导航、页头和更新入口 中的用户动作、生命周期动作或数据转换。
function updateIconTone(status?: string) {
  // value 保存 渲染桌面应用外壳、侧栏导航、页头和更新入口 使用的配置、引用或中间结果。
  const value = String(status ?? 'idle')
  if (value === 'error') return 'is-danger'
  if (['downloading', 'verifying', 'installing'].includes(value)) return 'is-busy'
  if (['update_available', 'verified', 'pending_install'].includes(value)) return 'is-ready'
  return ''
}
</script>

<template>
  <!-- 模板结构：声明当前组件对外呈现的布局、插槽和交互入口。 -->
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
                <Bell :size="18" />
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
