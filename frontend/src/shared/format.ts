// ============================================================================
// 文件: shared/format.ts
// 描述: 格式化工具函数
//
// 功能概述:
// - 日期时间格式化（UTC 转北京时间）
// - 字节大小格式化（B/KB/MB）
// - SHA256 哈希值截断显示
// ============================================================================

// 从 dayjs 导入核心库
import dayjs from 'dayjs'
// 导入 UTC 时区插件
import utc from 'dayjs/plugin/utc'
// 导入时区转换插件
import timezone from 'dayjs/plugin/timezone'
// 导入简体中文语言包
import 'dayjs/locale/zh-cn'

// 扩展 dayjs 支持 UTC 时区
dayjs.extend(utc)
// 扩展 dayjs 支持时区转换
dayjs.extend(timezone)
// 设置默认语言为简体中文
dayjs.locale('zh-cn')
// 设置默认时区为北京时间（Asia/Shanghai）
dayjs.tz.setDefault('Asia/Shanghai')

// ============================================================================
// 日期时间格式化
// ============================================================================

// 格式化日期时间字符串
// 将 UTC 时间字符串转换为北京时间并格式化显示
// 参数:
//   - value: UTC 时间字符串（ISO 8601 格式）
// 返回:
//   - string: 格式化后的北京时间字符串，格式如 "2026年06月04日 12:00:00"
//             如果输入为空或无效，返回 "未记录" 或 "时间无效"
export function formatDateTime(value?: string) {
  // 空值检查
  if (!value) {
    return '未记录'
  }
  // 解析 UTC 时间
  const parsed = dayjs.utc(value)
  // 有效性检查
  if (!parsed.isValid()) {
    return '时间无效'
  }
  // 转换为北京时间并格式化
  return parsed.tz('Asia/Shanghai').format('YYYY年MM月DD日 HH:mm:ss')
}

// ============================================================================
// 字节大小格式化
// ============================================================================

// 格式化字节大小为人类可读格式
// 参数:
//   - value: 字节数
// 返回:
//   - string: 格式化后的大小字符串，如 "1.5 KB"、"10.3 MB"
//             如果输入为空或无效，返回 "未提供"
export function formatBytes(value?: number) {
  // 空值或无效值检查
  if (!value || value <= 0) {
    return '未提供'
  }
  // 小于 1KB 显示为字节
  if (value < 1024) {
    return `${value} B`
  }
  // 小于 1MB 显示为 KB
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`
  }
  // 大于等于 1MB 显示为 MB
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

// ============================================================================
// SHA256 哈希截断显示
// ============================================================================

// 截断显示 SHA256 哈希值
// 保留前 12 位和后 8 位，中间用省略号连接
// 参数:
//   - value: 完整的 SHA256 哈希字符串
// 返回:
//   - string: 截断后的哈希字符串，如 "abcdef123456...12345678"
//             如果输入为空，返回 "未提供"
export function shortSha(value?: string) {
  // 空值检查
  if (!value) {
    return '未提供'
  }
  // 截取前 12 位 + 省略号 + 后 8 位
  return `${value.slice(0, 12)}...${value.slice(-8)}`
}
