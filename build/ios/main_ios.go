//go:build ios

// 文件职责：为 iOS Objective-C 启动层导出 Wails 应用入口。

package main

import (
	"C"
)

//export WailsIOSMain
func WailsIOSMain() {
	// iOS 禁止把入口 goroutine 锁到当前 OS thread，否则 Go signal handler 会触发：
	// "signal 16 received on thread with no signal stack"
	// "fatal error: non-Go code disabled sigaltstack"
	// 因此这里只做 Objective-C 可调用 wrapper，真正启动逻辑仍复用 main.go。
	main()
}
