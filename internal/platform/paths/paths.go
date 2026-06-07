// ============================================================================
// 文件: paths.go
// 描述: 跨平台路径管理模块
//
// 功能概述:
// - 提供各平台标准路径（数据库、缓存、日志）
// - 自动适配 Windows、macOS、Linux 路径差异
// - 统一的路径生成接口
// ============================================================================

package paths

import (
	"os"            // 操作系统接口，获取用户配置/缓存目录
	"path/filepath" // 路径拼接，自动处理平台分隔符
	"strings"       // 字符串处理，清理应用名称
	"time"          // 时间包，用于时间戳
)

const fallbackAppName = "go-desktop"

// DefaultDatabasePath 生成 SQLite 数据库文件的默认存储路径
// 路径格式:
//   - 所有平台: {可执行文件所在目录}/data/{appName}.db
//   - 开发兜底: {当前工作目录}/data/{appName}.db
//
// 参数:
//   - appName: 应用名称，为空时使用默认名称
//
// 返回:
//   - string: 数据库文件完整路径，错误时返回空字符串
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

// DefaultCachePath 生成缓存目录的默认路径
// 路径格式:
//   - 所有平台: {可执行文件所在目录}/data/updates
//   - 开发兜底: {当前工作目录}/data/updates
//
// 参数:
//   - appName: 应用名称，为空时使用默认名称
//
// 返回:
//   - string: 缓存目录完整路径，错误时返回空字符串
func DefaultCachePath(appName string) string {
	return filepath.Join(DefaultDataDir(appName), "updates")
}

// DefaultLogDirPath 生成文件日志目录的默认路径。
func DefaultLogDirPath(appName string) string {
	return filepath.Join(DefaultDataDir(appName), "logs")
}

// DefaultLogFilePattern 生成每日文件日志的默认 pattern。
func DefaultLogFilePattern(appName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = fallbackAppName
	}
	return filepath.Join(DefaultLogDirPath(appName), appName+"-%Y-%m-%d.log")
}

// CurrentLogFilePath 生成指定日期对应的每日文件日志路径。
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

// DefaultLogFilePath 生成当前每日文件日志路径
// 路径格式:
//   - 所有平台: {可执行文件所在目录}/data/logs/{appName}-YYYY-MM-DD.log
//   - 开发兜底: {当前工作目录}/data/logs/{appName}-YYYY-MM-DD.log
//
// 参数:
//   - appName: 应用名称，为空时使用默认名称
//
// 返回:
//   - string: 文件日志完整路径，错误时返回空字符串
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

// nowRFC3339 生成当前时间的 RFC3339 格式字符串
// 用于日志时间戳、事件记录等场景
// 返回:
//   - string: UTC 时间字符串，格式如 "2026-06-04T12:00:00Z"
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
