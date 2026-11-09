// Package installer 抽象「启动已校验安装包」的平台实现。
// Windows 使用 NSIS /S 静默参数；其他平台由 installer_other.go 提供占位实现。
package installer

import "context"

// Runner 启动已经校验过的安装器。
// 实现只负责拉起安装进程；等待安装完成和文件安全校验由调用方处理。
type Runner func(ctx context.Context, installerPath string) error
