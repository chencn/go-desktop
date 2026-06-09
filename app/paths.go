package app

import (
	"time"

	appruntime "github.com/chencn/go-desktop/internal/desktopapp/runtime"
)

// DefaultDatabasePath 返回 appName 对应的当前用户 SQLite 配置库路径。
func DefaultDatabasePath(appName string) string {
	return appruntime.DefaultDatabasePath(appName)
}

// DefaultDataDir 返回 appName 对应的当前用户应用数据目录。
func DefaultDataDir(appName string) string {
	return appruntime.DefaultDataDir(appName)
}

// DefaultCachePath 返回已校验更新包使用的缓存目录。
func DefaultCachePath(appName string) string {
	return appruntime.DefaultCachePath(appName)
}

// DefaultLogDirPath 返回每日日志 JSONL 文件所在目录。
func DefaultLogDirPath(appName string) string {
	return appruntime.DefaultLogDirPath(appName)
}

// DefaultLogFilePattern 返回诊断信息展示用的每日日志文件名模式。
func DefaultLogFilePattern(appName string) string {
	return appruntime.DefaultLogFilePattern(appName)
}

// CurrentLogFilePath 返回指定时间对应的每日日志文件路径。
func CurrentLogFilePath(appName string, now time.Time) string {
	return appruntime.CurrentLogFilePath(appName, now)
}

// DefaultLogFilePath 返回当天每日日志文件路径。
func DefaultLogFilePath(appName string) string {
	return appruntime.DefaultLogFilePath(appName)
}

// DefaultCrashLogPath 返回早期崩溃 breadcrumb 日志路径。
func DefaultCrashLogPath(appName string) string {
	return appruntime.DefaultCrashLogPath(appName)
}

// DefaultCrashStatePath 返回早期崩溃状态文件路径。
func DefaultCrashStatePath(appName string) string {
	return appruntime.DefaultCrashStatePath(appName)
}

// DefaultCrashDir 返回崩溃日志和崩溃状态共享的目录。
func DefaultCrashDir(appName string) string {
	return appruntime.DefaultCrashDir(appName)
}
