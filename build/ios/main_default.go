//go:build !ios

// 文件职责：让非 iOS 平台枚举 build/ios 包时具备 main 入口。
// 说明：真正 iOS 构建使用 main_ios.go；此文件只服务 go build ./... 的本地编译校验。

package main

func main() {}
