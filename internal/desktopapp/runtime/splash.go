// 文件职责：启动期透明加载窗口的 HTML。内联而非引用前端产物，
// 因为 splash 必须在 Wails 主窗口加载任何资源之前就能渲染。

package runtime

// SplashHTML 返回启动期透明加载窗口的完整 HTML。
func SplashHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <style>
      html,
      body {
        width: 100%;
        height: 100%;
        margin: 0;
        overflow: hidden;
        background: transparent;
        font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        -webkit-font-smoothing: antialiased;
        text-rendering: optimizeLegibility;
      }

      body {
        display: grid;
        place-items: center;
      }

      .splash {
        display: inline-flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 8px;
        width: 120px;
        height: 72px;
        color: #1890ff;
      }

      .ant-spin-dot {
        position: relative;
        display: inline-block;
        width: 32px;
        height: 32px;
        animation: ant-spin-rotate 1.2s linear infinite;
      }

      .ant-spin-dot-item {
        position: absolute;
        display: block;
        width: 14px;
        height: 14px;
        border-radius: 100%;
        background-color: currentColor;
        opacity: 0.3;
        animation: ant-spin-move 1s linear infinite alternate;
      }

      .ant-spin-dot-item:nth-child(1) {
        top: 0;
        left: 0;
      }

      .ant-spin-dot-item:nth-child(2) {
        top: 0;
        right: 0;
        animation-delay: 0.4s;
      }

      .ant-spin-dot-item:nth-child(3) {
        right: 0;
        bottom: 0;
        animation-delay: 0.8s;
      }

      .ant-spin-dot-item:nth-child(4) {
        bottom: 0;
        left: 0;
        animation-delay: 1.2s;
      }

      .ant-spin-text {
        color: rgba(0, 0, 0, 0.65);
        font-size: 14px;
        line-height: 1.5715;
        letter-spacing: 0;
        white-space: nowrap;
      }

      @keyframes ant-spin-rotate {
        to {
          transform: rotate(360deg);
        }
      }

      @keyframes ant-spin-move {
        to {
          opacity: 1;
          transform: scale(1);
        }
      }
    </style>
  </head>
  <body>
    <main class="splash" role="status" aria-live="polite">
      <span class="ant-spin-dot" aria-hidden="true">
        <i class="ant-spin-dot-item"></i>
        <i class="ant-spin-dot-item"></i>
        <i class="ant-spin-dot-item"></i>
        <i class="ant-spin-dot-item"></i>
      </span>
      <span class="ant-spin-text">正在加载</span>
    </main>
  </body>
</html>`
}

// SplashHTML 是 Runtime 实例上的便捷方法，供 app facade 通过嵌入直接访问。
func (s *Runtime) SplashHTML() string {
	return SplashHTML()
}
