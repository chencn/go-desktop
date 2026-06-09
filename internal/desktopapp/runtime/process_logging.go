// 文件职责：把进程级输出和错误接入应用日志存储。

package runtime

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	applogging "github.com/chencn/go-desktop/internal/desktopapp/logging"
	processutil "github.com/chencn/go-desktop/internal/platform/process"
)

// InstallProcessLogCapture 将 log/slog/stdout/stderr 接入应用日志管线。
// 这样启动阶段、框架层和非业务模块的输出也会进入日志页和文件日志。
func (s *Runtime) InstallProcessLogCapture() {
	s.closeProcessLogCapture()
	restore := &processLogRestore{
		logWriter:  log.Writer(),
		logFlags:   log.Flags(),
		slogLogger: slog.Default(),
	}
	s.lock.Lock()
	s.processLogRestore = restore
	s.lock.Unlock()

	originalStderr := os.Stderr
	log.SetOutput(io.MultiWriter(originalStderr, processLogWriter{runtime: s}))
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	slog.SetDefault(slog.New(processSlogHandler{runtime: s}))
	if err := s.installProcessStreamCapture(); err != nil {
		s.RecordLogWithSeverity("process", fmt.Sprintf("进程标准流捕获启用失败：%s", err), "warning")
	}
	s.RecordLog("process", "进程日志捕获已启用")
}

// closeProcessLogCapture 恢复 log/slog/stdout/stderr 的捕获前状态；Shutdown 会调用它避免全局 logger 泄漏。
func (s *Runtime) closeProcessLogCapture() {
	s.closeProcessStreamCapture()
	s.lock.Lock()
	restore := s.processLogRestore
	s.processLogRestore = nil
	s.lock.Unlock()
	if restore == nil {
		return
	}
	log.SetOutput(restore.logWriter)
	log.SetFlags(restore.logFlags)
	slog.SetDefault(restore.slogLogger)
}

// installProcessStreamCapture 安装 stdout/stderr 捕获器，并把每行输出按内容推断严重级别。
func (s *Runtime) installProcessStreamCapture() error {
	capture, err := processutil.NewStreamCapture(
		func(stream string, line string) {
			defer func() {
				if recovered := recover(); recovered != nil {
					s.RecordLogWithSeverity("panic", fmt.Sprintf("进程日志捕获异常：%v", recovered), "error")
				}
			}()
			message := strings.TrimSpace(line)
			if message != "" {
				s.RecordLogWithSeverity("process", stream+": "+message, applogging.SeverityFromProcessMessage(message))
			}
		},
		func(stream string, err error) {
			s.RecordLogWithSeverity("process", fmt.Sprintf("读取 %s 输出失败：%s", stream, err), "warning")
		},
	)
	if err != nil {
		return err
	}
	s.lock.Lock()
	s.processCapture = capture
	s.lock.Unlock()
	return nil
}

// closeProcessStreamCapture 恢复 stdout/stderr 并清空 Runtime 中的捕获器引用。
func (s *Runtime) closeProcessStreamCapture() {
	s.lock.Lock()
	capture := s.processCapture
	s.processCapture = nil
	s.lock.Unlock()
	if capture != nil {
		capture.Close()
	}
}

// processLogRestore 保存 Runtime 安装全局日志捕获前的状态。
type processLogRestore struct {
	// logWriter 是标准库 log 原输出目标。
	logWriter io.Writer
	// logFlags 是标准库 log 原 flags。
	logFlags int
	// slogLogger 是结构化日志原默认 logger。
	slogLogger *slog.Logger
}

// processLogWriter 适配标准库 log.Writer 接口。
type processLogWriter struct {
	// runtime 是日志写入目标，负责继续写入文件日志和内存 tail。
	runtime *Runtime
}

// Write 捕获标准库 log 输出；返回 len(p) 避免影响调用方。
func (w processLogWriter) Write(p []byte) (int, error) {
	message := strings.TrimSpace(string(p))
	if message != "" && w.runtime != nil {
		w.runtime.RecordLogWithSeverity("process", message, applogging.SeverityFromProcessMessage(message))
	}
	return len(p), nil
}

// processSlogHandler 适配 slog.Handler，把结构化日志写入 Runtime。
type processSlogHandler struct {
	// runtime 是日志写入目标。
	runtime *Runtime
	// attrs 是 WithAttrs 累积的结构化字段，会被拼到日志消息末尾。
	attrs []slog.Attr
	// group 是 WithGroup 累积的字段前缀。
	group string
}

// Enabled 接受所有级别；最终是否记录由 RecordLogWithSeverity 按 Runtime 设置过滤。
func (h processSlogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle 将 slog.Record 转成单行消息并写入 Runtime。
func (h processSlogHandler) Handle(_ context.Context, record slog.Record) error {
	if h.runtime == nil {
		return nil
	}
	message := strings.TrimSpace(record.Message)
	fields := h.formatAttrs(record)
	if fields != "" {
		message = strings.TrimSpace(message + " " + fields)
	}
	if message == "" {
		message = "空 slog 记录"
	}
	h.runtime.RecordLogWithSeverity("process", message, applogging.SeverityFromSlogLevel(record.Level))
	return nil
}

// WithAttrs 返回带附加字段的新 handler。
func (h processSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h
	next.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return next
}

// WithGroup 返回带字段分组的新 handler。
func (h processSlogHandler) WithGroup(name string) slog.Handler {
	next := h
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

// formatAttrs 把 handler 字段和本条记录字段拼成可检索文本。
func (h processSlogHandler) formatAttrs(record slog.Record) string {
	attrs := append([]slog.Attr(nil), h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		key := strings.TrimSpace(attr.Key)
		if key == "" {
			continue
		}
		if h.group != "" {
			key = h.group + "." + key
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, attr.Value.String()))
	}
	return strings.Join(parts, " ")
}
