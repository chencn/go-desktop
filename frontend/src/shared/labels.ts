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

// 错误原因代码到中文描述的映射表
// 用于将 Go 后端返回的错误代码转换为可读的中文描述
const reasonLabels: Record<string, string> = {
  // 网络相关错误
  offline: '当前无网络',                    // 离线状态
  rate_limited: 'GitHub API 暂时受限',      // API 速率限制

  // 配置相关错误
  disabled: '更新检查已关闭',               // 功能已禁用
  preview: '前端预览模式',                  // 浏览器预览模式

  // 请求相关错误
  request_create_failed: '创建更新检查请求失败',   // HTTP 请求创建失败
  request_failed: '更新检查请求失败',             // HTTP 请求发送失败
  http_error: 'GitHub API 返回异常',              // HTTP 响应错误

  // 响应处理错误
  response_read_failed: '读取 GitHub 响应失败',    // 响应读取失败
  response_parse_failed: '解析 GitHub 响应失败',  // JSON 解析失败

  // 发布版本相关错误
  no_available_release: '没有可用的发布版本',     // 找不到发布版本
  windows_asset_missing: '未找到 Windows 安装资产', // 缺少 Windows 安装包

  // SHA256 校验相关错误
  sha256_missing: '缺少 SHA256 校验信息',          // 缺少哈希值
  sha256_mismatch: '安装包 SHA256 校验失败',      // 哈希不匹配
  sha256_asset_offline: '当前无网络，无法读取 .sha256 文件',  // 沙盒文件离线
  sha256_asset_read_failed: '读取 .sha256 文件失败',          // 沙盒文件读取失败

  // 下载相关错误
  asset_missing: '安装资产信息不完整',             // 资产信息缺失
  download_failed: '下载安装包失败',               // 下载失败
  download_skipped: '已跳过安装包下载',            // 下载被跳过

  // 安装相关错误
  not_verified: '没有可安装的已校验更新包',       // 未通过校验
  verified_file_missing: '已校验安装包文件缺失',   // 安装包文件丢失
  installing: '正在启动静默安装器',               // 正在安装
  install_failed: '启动静默安装器失败',           // 安装失败
  install_started: '静默安装器已启动',             // 安装已启动
}

// ============================================================================
// 英文错误片段匹配规则
// ============================================================================

// 英文错误消息片段到中文描述的映射数组
// 使用正则表达式匹配常见的英文错误消息
const englishErrorFragments: Array<[RegExp, string]> = [
  // 网络连接失败相关错误
  [/no such host|network is unreachable|connection refused|connection reset|dial tcp|connectex|i\/o timeout/i, '网络连接失败'],
  // 请求超时相关错误
  [/context deadline exceeded|timeout/i, '请求超时'],
  // 权限不足相关错误
  [/permission denied|access is denied/i, '权限不足'],
  // 文件不存在相关错误
  [/file does not exist|cannot find the file|no such file/i, '文件不存在'],
]

// ============================================================================
// 导出函数
// ============================================================================

// 将错误原因代码转换为中文描述
// 如果找不到对应映射，则尝试英文翻译或返回默认值
// 参数:
//   - value: 错误原因代码字符串
// 返回:
//   - string: 中文错误描述
export function reasonLabel(value?: string) {
  // key 保存 集中提供设置项、日志项和更新状态的中文标签 使用的配置、引用或中间结果。
  const key = String(value ?? '').trim()
  // 空值返回 "无"
  if (!key) {
    return '无'
  }
  // 查找映射表
  return reasonLabels[key] ?? chineseOrFallback(key)
}

// 显示本地化的消息
// 优先匹配英文错误片段，然后尝试中文转换
// 参数:
//   - value: 原始消息字符串
// 返回:
//   - string: 本地化后的消息
export function displayMessage(value?: string) {
  // text 保存 集中提供设置项、日志项和更新状态的中文标签 使用的配置、引用或中间结果。
  const text = String(value ?? '').trim()
  // 空值返回 "无内容"
  if (!text) {
    return '无内容'
  }
  // 遍历英文错误片段规则进行匹配
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

// 判断字符串是否包含中文字符
// 如果包含则替换其中的错误代码，否则返回默认值
// 参数:
//   - value: 要检查的字符串
// 返回:
//   - string: 处理后的字符串
function chineseOrFallback(value: string) {
  // 使用正则表达式检测中文字符范围
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

// 替换字符串中所有已知的错误代码为中文描述
// 参数:
//   - value: 原始字符串
// 返回:
//   - string: 替换后的字符串
function replaceKnownReasonCodes(value: string) {
  // result 保存 集中提供设置项、日志项和更新状态的中文标签 使用的配置、引用或中间结果。
  let result = value
  // 遍历所有错误代码进行替换
  for (const [code, label] of Object.entries(reasonLabels)) {
    result = result.replaceAll(code, label)
  }
  return result
}
