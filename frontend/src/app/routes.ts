// 文件职责：集中维护前端视图到组件的映射关系。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

import type { Component } from 'vue'
import type { ViewKey } from '@/shared/views'
import AboutPage from '@/features/about/AboutPage.vue'
import HomePage from '@/features/home/HomePage.vue'
import LogsPage from '@/features/logs/LogsPage.vue'
import SettingsPage from '@/features/settings/SettingsPage.vue'

// viewComponents 是前端页面路由表；新增页面时只改这里和 shared/views.ts。
export const viewComponents: Record<ViewKey, Component> = {
  home: HomePage,
  logs: LogsPage,
  settings: SettingsPage,
  about: AboutPage,
}
