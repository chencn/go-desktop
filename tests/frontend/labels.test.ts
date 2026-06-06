// 文件职责：验证共享标签和日志消息展示的本地化边界。

import { describe, expect, it } from 'vitest'
import { displayMessage } from '../../frontend/src/shared/labels'

describe('shared labels', () => {
  it('keeps English informational process logs readable', () => {
    expect(displayMessage('[WebView2] Environment created successfully')).toBe('[WebView2] Environment created successfully')
  })

  it('localises known English error fragments', () => {
    expect(displayMessage('dial tcp: no such host')).toBe('网络连接失败')
  })
})
