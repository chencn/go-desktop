package runtime

import (
	"time"

	"github.com/chencn/go-desktop/internal/platform/paths"
)

// DefaultDatabasePath 返回当前平台的 SQLite 配置库默认路径。
func DefaultDatabasePath(appName string) string {
	return paths.DefaultDatabasePath(appName)
}

// DefaultDataDir 返回当前平台的应用数据目录。
func DefaultDataDir(appName string) string {
	return paths.DefaultDataDir(appName)
}

// DefaultCachePath 返回更新包等运行时缓存的默认目录。
func DefaultCachePath(appName string) string {
	return paths.DefaultCachePath(appName)
}

// DefaultLogDirPath 返回每日运行日志目录。
func DefaultLogDirPath(appName string) string {
	return paths.DefaultLogDirPath(appName)
}

// DefaultLogFilePattern 返回每日运行日志命名模式。
func DefaultLogFilePattern(appName string) string {
	return paths.DefaultLogFilePattern(appName)
}

// CurrentLogFilePath 返回指定时间对应的每日运行日志文件路径。
func CurrentLogFilePath(appName string, now time.Time) string {
	return paths.CurrentLogFilePath(appName, now)
}

// DefaultLogFilePath 返回当前时间对应的每日运行日志文件路径。
func DefaultLogFilePath(appName string) string {
	return paths.DefaultLogFilePath(appName)
}

// DefaultCrashLogPath 返回最早期 crash.log 路径。
func DefaultCrashLogPath(appName string) string {
	return paths.DefaultCrashLogPath(appName)
}

// DefaultCrashStatePath 返回用于检测上次异常退出的状态文件路径。
func DefaultCrashStatePath(appName string) string {
	return paths.DefaultCrashStatePath(appName)
}

// DefaultCrashDir 返回最早期 crash 日志和状态文件所在目录。
func DefaultCrashDir(appName string) string {
	return paths.DefaultCrashDir(appName)
}
