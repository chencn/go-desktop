// 文件职责：提供 runtime 统一日志框架与每日文件 writer 的装配。
// 说明：日志长期保存到每日文件，内存 ring buffer 只服务当前前端视图（见 log_memory.go），
// 查询与分页见 log_query.go。

package runtime

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/adapters/filelog"
)

const maxRuntimeMemoryLogs = 200

// runtimeLogHandler 同时把 slog 记录写入内存 ring buffer 和每日 JSONL 文件。
type runtimeLogHandler struct {
	runtime *Runtime     // runtime 是内存视图写入目标。
	file    slog.Handler // file 是实际文件 handler；可能写入 io.Discard。
	attrs   []slog.Attr  // attrs 是 WithAttrs 累积字段，用于还原内存 LogEntry。
	group   string       // group 是 WithGroup 累积字段前缀。
}

// Enabled 复用文件 handler 的级别过滤规则。
func (h *runtimeLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.file.Enabled(ctx, level)
}

// Handle 把日志记录追加到当前视图内存，再写入 JSONL 文件。
func (h *runtimeLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.runtime != nil {
		h.runtime.appendMemoryLog(logEntryFromRecord(record, h.attrs, h.group))
	}
	return h.file.Handle(ctx, record)
}

// WithAttrs 返回携带结构化字段的新 handler。
func (h *runtimeLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := &runtimeLogHandler{
		runtime: h.runtime,
		file:    h.file.WithAttrs(attrs),
		attrs:   append(append([]slog.Attr(nil), h.attrs...), attrs...),
		group:   h.group,
	}
	return next
}

// WithGroup 返回携带字段分组的新 handler。
func (h *runtimeLogHandler) WithGroup(name string) slog.Handler {
	next := &runtimeLogHandler{
		runtime: h.runtime,
		file:    h.file.WithGroup(name),
		attrs:   append([]slog.Attr(nil), h.attrs...),
		group:   h.group,
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return next
	}
	if next.group == "" {
		next.group = name
	} else {
		next.group += "." + name
	}
	return next
}

// initRuntimeLogger 初始化 Runtime logger；文件不可写时保留内存视图，不阻断启动。
func (s *Runtime) initRuntimeLogger() {
	if s.logLevel == nil {
		s.logLevel = &slog.LevelVar{}
	}
	s.logLevel.Set(SlogLevelFromLogLevel(s.SettingsSnapshot().LogLevel))

	var writer io.Writer = io.Discard
	var openedWriter *filelog.DailyWriter
	if strings.TrimSpace(s.logDirPath) != "" {
		logWriter, err := filelog.NewDailyWriter(s.logDirPath, s.options.AppName, time.Now)
		if err != nil {
			s.appendMemoryLog(LogEntry{
				Time:     nowRFC3339(),
				Scope:    "log-file",
				Severity: "warning",
				Message:  fmt.Sprintf("打开每日日志文件失败：%s", err),
			})
		} else {
			openedWriter = logWriter
			writer = logWriter
		}
	}

	fileHandler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: s.logLevel})
	logger := slog.New(&runtimeLogHandler{runtime: s, file: fileHandler})
	s.lock.Lock()
	// 并发懒初始化时只保留先完成的一方，避免后到者覆盖导致已打开的文件句柄泄漏。
	if s.shuttingDown || s.logger != nil {
		s.lock.Unlock()
		if openedWriter != nil {
			_ = openedWriter.Close()
		}
		return
	}
	s.logWriter = openedWriter
	s.logger = logger
	s.lock.Unlock()
}

// closeRuntimeLogger 停止当前 logger 并关闭每日文件 writer；内存日志不在这里清空。
func (s *Runtime) closeRuntimeLogger() {
	s.lock.Lock()
	writer := s.logWriter
	s.logWriter = nil
	s.logger = nil
	s.lock.Unlock()
	if writer != nil {
		_ = writer.Close()
	}
}

// currentLogFilePath 返回当前每日文件路径；writer 尚未打开时按今天日期推导。
func (s *Runtime) currentLogFilePath() string {
	s.lock.RLock()
	writer := s.logWriter
	appName := s.options.AppName
	logDirPath := s.logDirPath
	s.lock.RUnlock()
	if writer != nil {
		if name := writer.CurrentFileName(); name != "" {
			return name
		}
	}
	if logDirPath == "" {
		return ""
	}
	return filepath.Join(logDirPath, appName+"-"+time.Now().Format("2006-01-02")+".log")
}
