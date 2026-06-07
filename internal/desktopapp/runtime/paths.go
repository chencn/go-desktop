package runtime

import (
	"time"

	"github.com/chencn/go-desktop/internal/platform/paths"
)

func DefaultDatabasePath(appName string) string {
	return paths.DefaultDatabasePath(appName)
}

func DefaultDataDir(appName string) string {
	return paths.DefaultDataDir(appName)
}

func DefaultCachePath(appName string) string {
	return paths.DefaultCachePath(appName)
}

func DefaultLogDirPath(appName string) string {
	return paths.DefaultLogDirPath(appName)
}

func DefaultLogFilePattern(appName string) string {
	return paths.DefaultLogFilePattern(appName)
}

func CurrentLogFilePath(appName string, now time.Time) string {
	return paths.CurrentLogFilePath(appName, now)
}

func DefaultLogFilePath(appName string) string {
	return paths.DefaultLogFilePath(appName)
}

func DefaultCrashLogPath(appName string) string {
	return paths.DefaultCrashLogPath(appName)
}

func DefaultCrashStatePath(appName string) string {
	return paths.DefaultCrashStatePath(appName)
}

func DefaultCrashDir(appName string) string {
	return paths.DefaultCrashDir(appName)
}
