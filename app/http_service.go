// Package app 的 HTTP 门面：把本地 /api 请求转到 app.API 的方法集。
package app

import (
	"net/http"

	"github.com/chencn/go-desktop/internal/desktopapp/httpapi"
)

// HTTPServiceOptions 配置本地 /api HTTP 门面。
type HTTPServiceOptions struct {
	// API 是待暴露的 Wails service receiver，通常是 Runtime.API()。
	API *API
}

// HTTPService 是委托 internal/desktopapp/httpapi 的 HTTP 门面。
// 它只在本地开发链路（scripts/devapi）里被真正监听，生产窗口内的前端继续走 Wails 绑定。
type HTTPService struct {
	inner *httpapi.Service
}

// NewHTTPService 创建本地 HTTP API 门面。
func NewHTTPService(options HTTPServiceOptions) *HTTPService {
	return &HTTPService{inner: httpapi.New(httpapi.Options{Target: options.API})}
}

// ServeHTTP 将请求委托给内部 httpapi.Service。
func (s *HTTPService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.inner.ServeHTTP(w, r)
}

// Methods 返回门面已登记的方法名，供 devapi 启动日志与前端调用清单核对。
func (s *HTTPService) Methods() []string {
	return s.inner.Methods()
}
