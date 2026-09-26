// ============================================================================
// 文件: shared/views.ts
// 描述: 视图导航配置
//
// 功能概述:
// - 侧栏分组、导航图标与页面标题/副标题的唯一来源
// - 同一份配置同时驱动桌面侧栏、手机端 TabBar 和顶栏页头
// ============================================================================

import type { Component } from 'vue'
import { Activity, File, Info, Settings } from '@lucide/vue'

// 对应侧边栏导航的四个主要页面；更新入口固定在顶栏右侧弹窗。
export type ViewKey = 'home' | 'logs' | 'settings' | 'about'

// IconTone 对应设计稿 .sq-icon-badge 的渐变修饰类，外壳与页面徽标共用同一组取值。
export type IconTone = 'blue' | 'cyan' | 'gray' | 'green' | 'indigo' | 'orange' | 'pink' | 'purple' | 'red' | 'teal' | 'yellow'

// 导航项是侧栏、TabBar 和页头的单一文案来源。
export type NavigationItem = {
  icon: Component
  key: ViewKey
  // label 是手机端 TabBar 的窄文案。
  label: string
  // subtitle 同时用于页头副标题和侧栏提示。
  subtitle: string
  // tone 决定侧栏渐变徽标的配色。
  tone: IconTone
  // title 用于侧栏导航项文字和页头主标题。
  title: string
}

export type NavigationGroup = {
  items: NavigationItem[]
  title: string
}

// navigationGroups 的声明顺序就是侧栏分组和组内条目的渲染顺序。
export const navigationGroups: NavigationGroup[] = [
  {
    title: '核心功能',
    items: [
      { icon: Activity, key: 'home', label: '概览', subtitle: '软件运行状态、业务统计和样例图表', title: '概览', tone: 'blue' },
      { icon: File, key: 'logs', label: '日志', subtitle: '检索运行记录、定位异常和清理日志', title: '应用日志', tone: 'orange' },
    ],
  },
  {
    title: '配置与系统',
    items: [
      { icon: Settings, key: 'settings', label: '设置', subtitle: '显示偏好和业务设置', title: '应用设置', tone: 'gray' },
      { icon: Info, key: 'about', label: '关于', subtitle: '版本、Release、技术栈和本地路径', title: '关于应用', tone: 'teal' },
    ],
  },
]

export const navigation: NavigationItem[] = navigationGroups.flatMap((group) => group.items)

export function navigationItem(view: ViewKey) {
  return navigation.find((item) => item.key === view)
}

// 页头主标题与侧栏条目文字同源，避免两处漂移。
export function pageTitle(view: ViewKey) {
  return navigationItem(view)?.title ?? '概览'
}

// pageSubtitle 从导航配置读取页面副标题，保证侧栏、TabBar 和页头共用一份文案。
export function pageSubtitle(view: ViewKey) {
  return navigationItem(view)?.subtitle ?? ''
}
