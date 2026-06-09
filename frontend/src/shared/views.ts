// ============================================================================
// 文件: shared/views.ts
// 描述: 视图导航配置
//
// 功能概述:
// - 定义所有页面视图的键名、标签和图标
// - 提供页面标题工具函数
// ============================================================================

import type { Component } from 'vue'
import { Activity, Info, Logs, Settings2 } from '@lucide/vue'

// 对应侧边栏导航的四个主要页面；更新入口固定在右上角弹窗。
export type ViewKey = 'home' | 'logs' | 'settings' | 'about'
// 导航图标的语义色 class，实际色值由全局样式定义。
export type IconTone = 'icon-tone-indigo' | 'icon-tone-blue' | 'icon-tone-green' | 'icon-tone-orange' | 'icon-tone-gray' | 'icon-tone-purple'
// 导航项是侧栏、窄屏导航和页头副标题的单一文案来源。
export type NavigationItem = {
  icon: Component
  key: ViewKey
  label: string
  subtitle: string
  tone: IconTone
}

export const navigation: NavigationItem[] = [
  { key: 'home' as const, label: '概览', subtitle: '软件运行状态、业务统计和样例图表', icon: Activity, tone: 'icon-tone-indigo' as IconTone },
  { key: 'logs' as const, label: '日志', subtitle: '检索运行记录、定位异常和清理日志', icon: Logs, tone: 'icon-tone-orange' as IconTone },
  { key: 'settings' as const, label: '设置', subtitle: '显示偏好和业务设置', icon: Settings2, tone: 'icon-tone-blue' as IconTone },
  { key: 'about' as const, label: '关于', subtitle: '版本、Release、技术栈和本地路径', icon: Info, tone: 'icon-tone-purple' as IconTone },
]

// 页面标题和导航 label 不完全相同，例如 logs 在页头显示为“应用日志”。
export function pageTitle(view: ViewKey) {
  if (view === 'logs') return '应用日志'
  if (view === 'settings') return '应用设置'
  if (view === 'about') return '关于应用'
  return '概览'
}

// pageSubtitle 从导航配置读取页面副标题，保证侧栏、窄屏导航和页头共用一份文案。
export function pageSubtitle(view: ViewKey) {
  return navigation.find((item) => item.key === view)?.subtitle ?? ''
}
