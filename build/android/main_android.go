//go:build android

// 文件职责：在 Android c-shared 构建模式下把 Wails 应用入口注册给运行时。

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// init 在包加载阶段完成平台运行时注册，保证真正入口执行前依赖已接入。
func init() {
	// c-shared 构建不会自动调用 main()，必须把桌面入口交给 Android runtime 主动触发。
	application.RegisterAndroidMain(main)
}
