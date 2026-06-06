//go:build android

// 文件职责：在 Android c-shared 构建模式下把 Wails 应用入口注册给运行时。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// init 在包加载阶段完成平台运行时注册，保证真正入口执行前依赖已接入。
func init() {
	// Register main function to be called when the Android app initializes
	// This is necessary because in c-shared build mode, main() is not automatically called
	application.RegisterAndroidMain(main)
}
