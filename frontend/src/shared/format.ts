// ============================================================================
// 文件: shared/format.ts
// 描述: 格式化工具函数
//
// 功能概述:
// - 日期时间格式化（UTC 转北京时间）
// - 字节大小格式化（B/KB/MB）
// - SHA256 哈希值截断显示
// ============================================================================

import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
import 'dayjs/locale/zh-cn'

// 后端时间统一按 ISO/UTC 传入，展示层统一转为北京时间。
dayjs.extend(utc)
dayjs.extend(timezone)
dayjs.locale('zh-cn')
dayjs.tz.setDefault('Asia/Shanghai')

// ============================================================================
// 日期时间格式化
// ============================================================================

// 将后端 ISO/UTC 时间转换为北京时间；空值和非法值分别给出可展示占位。
export function formatDateTime(value?: string) {
  if (!value) {
    return '未记录'
  }
  const parsed = dayjs.utc(value)
  if (!parsed.isValid()) {
    return '时间无效'
  }
  return parsed.tz('Asia/Shanghai').format('YYYY年MM月DD日 HH:mm:ss')
}

// ============================================================================
// 字节大小格式化
// ============================================================================

// 安装包大小展示：0、负数和缺失值都视为未提供。
export function formatBytes(value?: number) {
  if (!value || value <= 0) {
    return '未提供'
  }
  if (value < 1024) {
    return `${value} B`
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`
  }
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

// ============================================================================
// SHA256 哈希截断显示
// ============================================================================

// SHA256 在窄弹窗里只展示首尾，完整值仍保存在更新状态中。
export function shortSha(value?: string) {
  if (!value) {
    return '未提供'
  }
  return `${value.slice(0, 12)}...${value.slice(-8)}`
}
