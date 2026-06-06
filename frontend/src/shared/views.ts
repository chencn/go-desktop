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

// ============================================================================
// 类型定义
// ============================================================================

// 页面视图键名类型
// 对应侧边栏导航的四个主要页面；更新入口固定在右上角弹窗。
export type ViewKey = 'home' | 'logs' | 'settings' | 'about'
// IconTone 定义定义前端导航视图键、图标、标题和副标题的单一配置来源 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type IconTone = 'icon-tone-indigo' | 'icon-tone-green' | 'icon-tone-orange' | 'icon-tone-gray' | 'icon-tone-purple'
// NavigationItem 定义定义前端导航视图键、图标、标题和副标题的单一配置来源 使用的类型契约，限制跨组件或跨模块传递的数据形状。
export type NavigationItem = {
  icon: Component
  key: ViewKey
  label: string
  subtitle: string
  tone: IconTone
}

// ============================================================================
// 导航配置
// ============================================================================

// 导航项配置数组
// 定义侧边栏导航的所有页面项
export const navigation: NavigationItem[] = [
  // 概览 - 软件运行状态和业务统计
  { key: 'home' as const, label: '概览', subtitle: '软件运行状态、业务统计和样例图表', icon: Activity, tone: 'icon-tone-indigo' as IconTone },
  // 日志 - 应用日志列表
  { key: 'logs' as const, label: '日志', subtitle: '检索运行记录、定位异常和清理日志', icon: Logs, tone: 'icon-tone-orange' as IconTone },
  // 设置 - 应用偏好设置
  { key: 'settings' as const, label: '设置', subtitle: '显示偏好和业务设置', icon: Settings2, tone: 'icon-tone-gray' as IconTone },
  // 关于 - 应用信息视图
  { key: 'about' as const, label: '关于', subtitle: '版本、Release、技术栈和本地路径', icon: Info, tone: 'icon-tone-purple' as IconTone },
]

// ============================================================================
// 工具函数
// ============================================================================

// 根据视图键名获取页面标题
// 参数:
//   - view: 视图键名
// 返回:
//   - string: 页面标题
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
