// ============================================================================
// 文件: logs.go
// 描述: 日志管理模块
//
// 功能概述:
// - 提供日志记录、查询、清空功能
// - 支持按作用域、严重级别、关键词过滤
// - 日志写入每日 JSONL 文件并同步当前内存视图
// ============================================================================

package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	applogging "github.com/chencn/go-desktop/internal/desktopapp/logging"
	appsettings "github.com/chencn/go-desktop/internal/desktopapp/settings"
)

// ListLogs API 方法，返回当前内存视图中的日志副本。
func (api *API) ListLogs() (logs []LogEntry, err error) {
	defer api.recoverError("读取日志", &err)
	if err := api.requireAuthorized(); err != nil {
		return nil, err
	}
	return api.runtime.ListLogs(), nil
}

// ListLogs 返回内存 ring buffer 副本；不会读取历史每日文件日志。
func (s *Runtime) ListLogs() []LogEntry {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return append([]LogEntry(nil), s.logs...)
}

// QueryLogs API 方法，支持单文件来源的分页和过滤查询。
// 这里不写 api-trace，避免日志页刷新自身制造 QueryLogs 噪音。
func (api *API) QueryLogs(query LogQuery) (response LogResponse, err error) {
	defer api.recoverError("查询日志", &err)
	if err := api.requireAuthorized(); err != nil {
		return LogResponse{}, err
	}
	response = api.runtime.QueryLogs(query)
	return response, nil
}

// QueryLogs 查询单个每日文件日志；文件不可用时才降级到当前内存视图。
func (s *Runtime) QueryLogs(query LogQuery) LogResponse {
	logs, source, fileName, filePath := s.logEntriesForQuery(query)
	response := s.filterSortAndPageLogs(logs, query, true)
	response.Source = source
	response.FileName = fileName
	response.FilePath = filePath
	return response
}

// ListLogFiles API 方法，返回日志目录下可选择的每日文件。
func (api *API) ListLogFiles() (files []LogFileInfo, err error) {
	defer api.recoverError("读取日志文件列表", &err)
	if err := api.requireAuthorized(); err != nil {
		return nil, err
	}
	api.runtime.RecordLogWithSeverity("api-trace", "ListLogFiles：后端收到请求", "debug")
	files = api.runtime.ListLogFiles()
	api.runtime.RecordLogWithSeverity("api-trace", fmt.Sprintf("ListLogFiles：后端返回成功 files=%d", len(files)), "debug")
	return files, nil
}

// ClearLogs API 方法，清空当前前端日志视图；文件日志保留给历史查询和保留策略处理。
func (api *API) ClearLogs(scope string) (cleared bool, err error) {
	defer api.recoverError("清空日志", &err)
	if err := api.requireAuthorized(); err != nil {
		return false, err
	}
	return api.runtime.ClearLogs(scope), nil
}

// ClearLogs 清空当前前端视图，不删除或截断每日文件日志。
// 为了让后续文件查询也尊重“已清空视图”，这里记录按 scope 的清空时间。
func (s *Runtime) ClearLogs(scope string) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = "all"
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.logViewClearedAt == nil {
		s.logViewClearedAt = map[string]time.Time{}
	}
	s.logViewClearedAt[scope] = time.Now().UTC()
	if scope == "all" {
		s.logs = nil
		return true
	}

	remaining := make([]LogEntry, 0, len(s.logs))
	for _, entry := range s.logs {
		if strings.ToLower(entry.Scope) != scope {
			remaining = append(remaining, entry)
		}
	}
	s.logs = remaining
	return true
}

// RecordLog 记录 info 级别日志。
func (s *Runtime) RecordLog(scope, message string) {
	s.RecordLogWithSeverity(scope, message, "info")
}

// RecordLogWithSeverity 记录指定严重级别的日志。
// 低于当前 LogLevel 的日志会被丢弃；文件不可写或 logger 未就绪时降级写入内存视图。
func (s *Runtime) RecordLogWithSeverity(scope, message, severity string) {
	severity = normaliseLogSeverity(severity)
	if !s.shouldRecordLogSeverity(severity) {
		return
	}

	scope = normaliseLogScope(scope)
	message = strings.TrimSpace(message)
	crashReporter := s.crashReporter

	s.lock.RLock()
	logger := s.logger
	shuttingDown := s.shuttingDown
	s.lock.RUnlock()
	if logger == nil && !shuttingDown {
		s.initRuntimeLogger()
		s.lock.RLock()
		logger = s.logger
		shuttingDown = s.shuttingDown
		s.lock.RUnlock()
	}
	if logger == nil {
		s.appendMemoryLog(LogEntry{
			Time:     time.Now().UTC().Format(time.RFC3339Nano),
			Scope:    scope,
			Message:  message,
			Severity: severity,
		})
	} else {
		logger.LogAttrs(context.Background(), slogLevelFromSeverity(severity), message,
			slog.String("scope", scope),
			slog.String("severity", severity),
		)
	}

	if severity == "error" && crashReporter != nil && strings.ToLower(strings.TrimSpace(scope)) != "crash" {
		crashReporter.Append(scope, "%s", message)
	}
}

// shouldRecordLogSeverity 按当前设置中的最小日志级别过滤应用日志。
func (s *Runtime) shouldRecordLogSeverity(severity string) bool {
	s.lock.RLock()
	level := s.settings.LogLevel
	s.lock.RUnlock()
	return logSeverityRank(severity) >= logSeverityRank(normaliseLogLevel(level))
}

// normaliseLogSeverity 标准化严重级别字符串，兼容 panic/fatal/critical/warn 等输入。
func normaliseLogSeverity(severity string) string {
	return applogging.NormalizeSeverity(severity)
}

// normaliseLogLevel 标准化设置里的最小记录级别。
func normaliseLogLevel(level string) string {
	return appsettings.NormalizeLogLevel(level)
}

// SlogLevelFromLogLevel 把应用设置中的日志级别映射到 Wails/slog 启动日志级别。
func SlogLevelFromLogLevel(level string) slog.Level {
	return applogging.SlogLevelFromLogLevel(level)
}

// logSeverityRank 返回日志级别排序值，用于最小级别过滤。
func logSeverityRank(severity string) int {
	return applogging.SeverityRank(severity)
}

// calculateLogStats 基于过滤后的日志集合计算统计信息。
func calculateLogStats(logs []LogEntry) LogStats {
	stats := LogStats{Total: len(logs)}
	for _, entry := range logs {
		switch normaliseLogSeverity(entry.Severity) {
		case "debug":
			stats.Debug++
		case "error":
			stats.Error++
		case "warning":
			stats.Warning++
		default:
			stats.Info++
		}
	}
	return stats
}
