// 文件职责：集中维护前端视图到组件的映射关系。

import type { Component } from 'vue'
import type { ViewKey } from '@/shared/views'
import AboutPage from '@/features/about/AboutPage.vue'
import HomePage from '@/features/home/HomePage.vue'
import LogsPage from '@/features/logs/LogsPage.vue'
import SettingsPage from '@/features/settings/SettingsPage.vue'

// 桌面端不使用 URL 路由；App.vue 根据当前 ViewKey 从这里取页面组件。
export const viewComponents: Record<ViewKey, Component> = {
  home: HomePage,
  logs: LogsPage,
  settings: SettingsPage,
  about: AboutPage,
}
