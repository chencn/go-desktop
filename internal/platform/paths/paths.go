// Package paths 生成应用数据、缓存、日志和崩溃哨兵文件路径。
// 当前策略优先使用可执行文件所在目录下的 data，保证不同启动入口共享同一份状态。

package paths

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// fallbackAppName 是路径生成的空 appName 兜底值。
const fallbackAppName = "go-desktop"

// DefaultDatabasePath 返回 SQLite 数据库默认路径。
// appName 为空时使用 go-desktop；目录来自 DefaultDataDir。
func DefaultDatabasePath(appName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = fallbackAppName
	}
	return filepath.Join(DefaultDataDir(appName), appName+".db")
}

// DefaultDataDir 生成应用数据目录
// 优先使用可执行文件所在目录下的 data，保证快捷方式、开机自启和安装器启动都指向同一份数据。
// 当开发态无法解析可执行文件目录时，兜底使用当前工作目录下的 data。
func DefaultDataDir(appName string) string {
	executable, err := os.Executable()
	if err == nil && strings.TrimSpace(executable) != "" {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil && strings.TrimSpace(resolved) != "" {
			executable = resolved
		}
		return filepath.Join(filepath.Dir(executable), "data")
	}
	workingDir, err := os.Getwd()
	if err != nil || strings.TrimSpace(workingDir) == "" {
		return filepath.Join(".", "data")
	}
	return filepath.Join(workingDir, "data")
}

// DefaultCachePath 返回更新包缓存目录。
// appName 只影响 DefaultDataDir 兜底策略，当前不会写入目录名。
func DefaultCachePath(appName string) string {
	return filepath.Join(DefaultDataDir(appName), "updates")
}

// DefaultLogDirPath 生成文件日志目录的默认路径。
func DefaultLogDirPath(appName string) string {
	return filepath.Join(DefaultDataDir(appName), "logs")
}

// DefaultLogFilePattern 返回日志库使用的每日文件 pattern。
func DefaultLogFilePattern(appName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = fallbackAppName
	}
	return filepath.Join(DefaultLogDirPath(appName), appName+"-%Y-%m-%d.log")
}

// CurrentLogFilePath 返回指定日期对应的每日文件日志路径；now 为零值时使用当前时间。
func CurrentLogFilePath(appName string, now time.Time) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = fallbackAppName
	}
	if now.IsZero() {
		now = time.Now()
	}
	return filepath.Join(DefaultLogDirPath(appName), appName+"-"+now.Format("2006-01-02")+".log")
}

// DefaultLogFilePath 返回当前日期的文件日志路径。
func DefaultLogFilePath(appName string) string {
	return CurrentLogFilePath(appName, time.Now())
}

// DefaultCrashLogPath 生成最早期崩溃日志路径。
// 该路径与普通运行日志同目录，统一落到可执行文件所在目录的 data/logs。
func DefaultCrashLogPath(appName string) string {
	return filepath.Join(DefaultCrashDir(appName), "crash.log")
}

// DefaultCrashStatePath 生成运行状态哨兵文件路径。
// 如果上次启动没有正常清理该文件，下次启动会把它导入应用日志。
func DefaultCrashStatePath(appName string) string {
	return filepath.Join(DefaultCrashDir(appName), "crash-state.json")
}

// DefaultCrashDir 返回 crash 日志目录。
func DefaultCrashDir(appName string) string {
	return DefaultLogDirPath(appName)
}

// nowRFC3339 返回当前 UTC 时间的 RFC3339 字符串。
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
