// ============================================================================
// 文件: shared/labels.ts
// 描述: 标签和消息本地化工具
//
// 功能概述:
// - 将英文错误代码转换为中文描述
// - 将英文错误消息翻译为中文
// - 处理网络、权限、文件等常见错误类型
// ============================================================================

// ============================================================================
// 错误原因标签映射
// ============================================================================

// Go 更新链返回的 errorReason/skipReason 代码在这里统一转换为可读中文。
const reasonLabels: Record<string, string> = {
  offline: '当前无网络',
  rate_limited: 'GitHub API 暂时受限',

  disabled: '更新检查已关闭',
  preview: '前端预览模式',

  request_create_failed: '创建更新检查请求失败',
  request_failed: '更新检查请求失败',
  http_error: 'GitHub API 返回异常',

  response_read_failed: '读取 GitHub 响应失败',
  response_parse_failed: '解析 GitHub 响应失败',

  no_available_release: '没有可用的发布版本',
  windows_asset_missing: '未找到 Windows 安装资产',

  sha256_missing: '缺少 SHA256 校验信息',
  sha256_mismatch: '安装包 SHA256 校验失败',
  sha256_asset_offline: '当前无网络，无法读取 .sha256 文件',
  sha256_asset_read_failed: '读取 .sha256 文件失败',

  asset_missing: '安装资产信息不完整',
  download_failed: '下载安装包失败',
  download_skipped: '已跳过安装包下载',

  not_verified: '没有可安装的已校验更新包',
  verified_file_missing: '已校验安装包文件缺失',
  installing: '正在启动静默安装器',
  install_failed: '启动静默安装器失败',
  install_started: '静默安装器已启动',
}

// ============================================================================
// 英文错误片段匹配规则
// ============================================================================

// 只翻译明确的底层错误片段；普通英文进程日志保留原文。
const englishErrorFragments: Array<[RegExp, string]> = [
  [/no such host|network is unreachable|connection refused|connection reset|dial tcp|connectex|i\/o timeout/i, '网络连接失败'],
  [/context deadline exceeded|timeout/i, '请求超时'],
  [/permission denied|access is denied/i, '权限不足'],
  [/file does not exist|cannot find the file|no such file/i, '文件不存在'],
]

// ============================================================================
// 导出函数
// ============================================================================

// 面向 reason/skip code：未知英文代码返回“未本地化的底层错误”，避免裸露内部枚举。
export function reasonLabel(value?: string) {
  const key = String(value ?? '').trim()
  if (!key) {
    return '无'
  }
  return reasonLabels[key] ?? chineseOrFallback(key)
}

// 面向日志/状态消息：只替换已知错误，未知英文文本保留原文供排障。
export function displayMessage(value?: string) {
  const text = String(value ?? '').trim()
  if (!text) {
    return '无内容'
  }
  for (const [pattern, label] of englishErrorFragments) {
    if (pattern.test(text)) {
      return label
    }
  }
  // 普通英文进程日志（例如 WebView2 成功信息）应保留原文，避免被误判为错误。
  return readableMessage(text)
}

// ============================================================================
// 内部辅助函数
// ============================================================================

// 中文句子里可能夹带 errorReason code，此时只替换 code 片段。
function chineseOrFallback(value: string) {
  if (/[一-龥]/.test(value)) {
    return replaceKnownReasonCodes(value)
  }
  // 返回默认的未本地化提示
  return reasonLabels[value] ?? '未本地化的底层错误'
}

// readableMessage 只本地化已知代码或中文消息中的代码，未知英文日志保持可读原文。
function readableMessage(value: string) {
  if (/[一-龥]/.test(value)) {
    return replaceKnownReasonCodes(value)
  }
  return reasonLabels[value] ?? value
}

// 后端有些 message 会拼接多个 code，这里按完整 code 字符串逐个替换。
function replaceKnownReasonCodes(value: string) {
  let result = value
  for (const [code, label] of Object.entries(reasonLabels)) {
    result = result.replaceAll(code, label)
  }
  return result
}
