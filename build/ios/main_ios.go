//go:build ios

// 文件职责：main_ios.go 中的业务流程、状态和数据结构。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package main

import (
	"C"
)

// For iOS builds, we need to export a function that can be called from Objective-C
// This wrapper allows us to keep the original main.go unmodified

//export WailsIOSMain
func WailsIOSMain() {
	// DO NOT lock the goroutine to the current OS thread on iOS!
	// This causes signal handling issues:
	// "signal 16 received on thread with no signal stack"
	// "fatal error: non-Go code disabled sigaltstack"
	// iOS apps run in a sandboxed environment where the Go runtime's
	// signal handling doesn't work the same way as desktop platforms.

	// Call the actual main function from main.go
	// This ensures all the user's code is executed
	main()
}
