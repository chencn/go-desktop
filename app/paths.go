package app

import (
	"time"

	appruntime "github.com/chencn/go-desktop/internal/desktopapp/runtime"
)

func DefaultDatabasePath(appName string) string {
	return appruntime.DefaultDatabasePath(appName)
}

func DefaultDataDir(appName string) string {
	return appruntime.DefaultDataDir(appName)
}

func DefaultCachePath(appName string) string {
	return appruntime.DefaultCachePath(appName)
}

func DefaultLogDirPath(appName string) string {
	return appruntime.DefaultLogDirPath(appName)
}

func DefaultLogFilePattern(appName string) string {
	return appruntime.DefaultLogFilePattern(appName)
}

func CurrentLogFilePath(appName string, now time.Time) string {
	return appruntime.CurrentLogFilePath(appName, now)
}

func DefaultLogFilePath(appName string) string {
	return appruntime.DefaultLogFilePath(appName)
}

func DefaultCrashLogPath(appName string) string {
	return appruntime.DefaultCrashLogPath(appName)
}

func DefaultCrashStatePath(appName string) string {
	return appruntime.DefaultCrashStatePath(appName)
}

func DefaultCrashDir(appName string) string {
	return appruntime.DefaultCrashDir(appName)
}
