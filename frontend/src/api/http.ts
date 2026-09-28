/**
 * ============================================================================
 * 文件: api/http.ts
 * 描述: 开发期浏览器预览使用的本地 /api 门面传输层
 *
 * 功能概述:
 * - 把 app.API 的方法调用转成 POST /api/<Method>，请求体为按位置传参的 JSON 数组
 * - 区分「门面不可达」与「后端已应答但报错」两类失败，前者允许上层回落兜底数据
 * - 探测到门面不可达后本页面会话内不再重试，避免每个 API 都白等一次
 *
 * 架构说明:
 * - 门面由 `go run ./scripts/envrun go run ./scripts/devapi` 提供，Vite 通过
 *   GO_DESKTOP_LOCAL_API_URL 把 /api 代理过去；原生窗口不使用本模块（继续走 Wails 绑定）
 * ============================================================================
 */

const apiPrefix = '/api'
// 实测 CheckUpdate 走 GitHub 代理要 4~5s，下载安装包更久：门面在本地，超时只用来兜住卡死的进程。
const requestTimeoutMs = 60_000

/** 门面没有响应（未启动、被 Vite 的 SPA fallback 用 index.html 兜住、超时）。 */
export class LocalApiUnavailableError extends Error {
  constructor(method: string, reason: string) {
    super(`本地 /api 门面不可用：${method}（${reason}）`)
    this.name = 'LocalApiUnavailableError'
  }
}

/** 门面已应答且后端明确失败；上层必须把错误暴露给用户，不得改用假数据。 */
export class LocalApiRejectedError extends Error {
  constructor(method: string, message: string) {
    super(`${method} 调用失败：${message}`)
    this.name = 'LocalApiRejectedError'
  }
}

// undefined 表示还没探测过；false 表示本会话内不再尝试，true 表示门面在线。
let localApiProbe: boolean | undefined

/** 是否还值得尝试本地门面：探测失败过一次后直接回落到原有兜底。 */
export function localApiLooksAvailable() {
  return localApiProbe !== false
}

/** 调用一个 app.API 方法；返回值即该方法的第一个返回类型。 */
export async function callLocalApi<T>(method: string, args: unknown[]): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${apiPrefix}/${method}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify(args),
      signal: AbortSignal.timeout(requestTimeoutMs),
    })
  } catch (error) {
    localApiProbe = false
    throw new LocalApiUnavailableError(method, error instanceof Error ? error.message : '网络失败')
  }

  const payload = await readEnvelope(response)
  if (!payload) {
    localApiProbe = false
    throw new LocalApiUnavailableError(method, `响应不是门面 JSON（HTTP ${response.status}）`)
  }
  if (!payload.ok) {
    // 404 有两种来源：门面起来了但方法未登记（按后端错误暴露），以及代理缺位（上面已判为不可达）。
    localApiProbe = true
    throw new LocalApiRejectedError(method, String(payload.error ?? '未知错误'))
  }
  localApiProbe = true
  return payload.data as T
}

/** 解析门面的 {"ok":...} 包；HTML/空响应等非 JSON 内容返回 null。 */
async function readEnvelope(response: Response): Promise<{ ok?: boolean; data?: unknown; error?: string } | null> {
  const text = await response.text()
  if (!text.trim().startsWith('{')) return null
  try {
    return JSON.parse(text) as { ok?: boolean; data?: unknown; error?: string }
  } catch {
    return null
  }
}
