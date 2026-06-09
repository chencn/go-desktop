//go:build !ios

// 文件职责：为非 iOS 构建提供空的 iOS option hook，保持 main.go 调用点跨平台一致。

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS 在非 iOS 平台不修改 Wails options。
func modifyOptionsForIOS(opts *application.Options) {
	// 非 iOS 平台使用 Wails 默认 signal handler。
}
