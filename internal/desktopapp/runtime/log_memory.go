// 文件职责：内存日志视图。负责 ring buffer 写入、slog 记录到 LogEntry 的还原，
// 以及作用域/级别的归一化。每日文件日志才是长期存储，内存视图只服务当前前端会话。

package runtime

import (
	"log/slog"
	"strings"
	"time"

	applogging "github.com/chencn/go-desktop/internal/desktopapp/logging"
)

// appendMemoryLog 把日志加入当前前端视图 ring buffer；最新日志在前，最多保留 maxRuntimeMemoryLogs 条。
func (s *Runtime) appendMemoryLog(entry LogEntry) {
	entry.Time = strings.TrimSpace(entry.Time)
	if entry.Time == "" {
		entry.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	entry.Scope = normaliseLogScope(entry.Scope)
	entry.Severity = normaliseLogSeverity(entry.Severity)
	entry.Message = strings.TrimSpace(entry.Message)
	if entry.Message == "" {
		entry.Message = "空日志"
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.logs = append([]LogEntry{entry}, s.logs...)
	if len(s.logs) > maxRuntimeMemoryLogs {
		s.logs = s.logs[:maxRuntimeMemoryLogs]
	}
}

// logEntryFromRecord 从 slog.Record 中提取日志页需要的 scope、severity、message 和 UTC 时间。
func logEntryFromRecord(record slog.Record, handlerAttrs []slog.Attr, _ string) LogEntry {
	attrs := append([]slog.Attr(nil), handlerAttrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	scope := "app"
	severity := applogging.SeverityFromSlogLevel(record.Level)
	for _, attr := range attrs {
		key := strings.TrimSpace(attr.Key)
		if key == "" {
			continue
		}
		value := strings.TrimSpace(attr.Value.String())
		switch key {
		case "scope":
			if value != "" {
				scope = value
			}
		case "severity":
			if value != "" {
				severity = value
			}
		}
	}

	message := strings.TrimSpace(record.Message)
	if message == "" {
		message = "空日志"
	}

	logTime := record.Time
	if logTime.IsZero() {
		logTime = time.Now().UTC()
	}
	return LogEntry{
		Time:     logTime.UTC().Format(time.RFC3339Nano),
		Scope:    scope,
		Severity: severity,
		Message:  message,
	}
}

// slogLevelFromSeverity 把日志页严重级别映射为 slog 级别。
func slogLevelFromSeverity(severity string) slog.Level {
	switch normaliseLogSeverity(severity) {
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

// normaliseLogScope 标准化日志作用域，空值归入 app。
func normaliseLogScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "app"
	}
	return scope
}
