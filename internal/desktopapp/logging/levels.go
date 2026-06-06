package logging

import (
	"log/slog"
	"strings"

	"github.com/chencn/go-desktop/internal/desktopapp/settings"
)

func NormalizeSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "debug":
		return "debug"
	case "error", "panic", "fatal", "critical":
		return "error"
	case "warning", "warn":
		return "warning"
	default:
		return "info"
	}
}

func SeverityRank(severity string) int {
	switch NormalizeSeverity(severity) {
	case "debug":
		return 0
	case "info":
		return 1
	case "warning":
		return 2
	case "error":
		return 3
	default:
		return 1
	}
}

func SlogLevelFromLogLevel(level string) slog.Level {
	switch settings.NormalizeLogLevel(level) {
	case "debug":
		return slog.LevelDebug
	case "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func SeverityFromSlogLevel(level slog.Level) string {
	if level >= slog.LevelError {
		return "error"
	}
	if level >= slog.LevelWarn {
		return "warning"
	}
	if level <= slog.LevelDebug {
		return "debug"
	}
	return "info"
}

func SeverityFromProcessMessage(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return "info"
	}
	if looksLikeStackFrame(normalized) {
		return "info"
	}
	if hasExplicitErrorSignal(normalized) {
		return "error"
	}
	if strings.HasPrefix(normalized, "warn") || strings.Contains(normalized, "warning") || strings.Contains(normalized, "警告") {
		return "warning"
	}
	return "info"
}

func looksLikeStackFrame(message string) bool {
	message = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(message, "stdout:"), "stderr:"))
	message = strings.TrimSpace(message)
	if message == "stack trace:" || message == "stack trace" {
		return true
	}
	index := strings.Index(message, ":")
	if index <= 0 {
		return false
	}
	for _, char := range message[:index] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return strings.Contains(message, ".go:") ||
		strings.Contains(message, "github.com/") ||
		strings.Contains(message, "runtime/")
}

func hasExplicitErrorSignal(message string) bool {
	message = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(message, "stdout:"), "stderr:"))
	message = strings.TrimSpace(message)
	if strings.Contains(message, "失败") || strings.Contains(message, "fatal error") || strings.Contains(message, "runtime error") {
		return true
	}
	for _, prefix := range []string{
		"panic:",
		"fatal:",
		"error:",
		"panic ",
		"fatal ",
		"error ",
		"wails run failed:",
	} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	return false
}
