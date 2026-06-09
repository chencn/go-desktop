//go:build ios

// 文件职责：为 iOS 构建调整 Wails options，避开移动端 signal handler 限制。

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS 关闭默认 signal handler，避免 Go runtime 在 iOS 上触发 sigaltstack 崩溃。
func modifyOptionsForIOS(opts *application.Options) {
	opts.DisableDefaultSignalHandler = true
}
