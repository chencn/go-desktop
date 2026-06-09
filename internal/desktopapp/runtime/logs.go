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
	"strings" // 字符串处理，用于日志过滤
	"time"    // 时间包，用于日志时间戳

	applogging "github.com/chencn/go-desktop/internal/desktopapp/logging"
	appsettings "github.com/chencn/go-desktop/internal/desktopapp/settings"
)

// ListLogs API 方法，返回所有日志条目
// 返回值的副本，避免外部修改内部状态
func (api *API) ListLogs() (logs []LogEntry, err error) {
	defer api.recoverError("读取日志", &err)
	if err := api.requireAuthorized(); err != nil {
		return nil, err
	}
	return api.runtime.ListLogs(), nil
}

// ListLogs 返回内存中所有日志的副本
// 使用 RLock 保护并发读取
func (s *Runtime) ListLogs() []LogEntry {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return append([]LogEntry(nil), s.logs...)
}

// QueryLogs API 方法，支持分页和过滤查询。
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

// ClearLogs API 方法，清空指定作用域的日志
// 参数:
//   - scope: 日志作用域，为空或 "all" 时清空所有日志
//
// 返回:
//   - bool: 操作是否成功
func (api *API) ClearLogs(scope string) (cleared bool, err error) {
	defer api.recoverError("清空日志", &err)
	if err := api.requireAuthorized(); err != nil {
		return false, err
	}
	return api.runtime.ClearLogs(scope), nil
}

// ClearLogs 清空当前前端视图，不删除或截断每日文件日志。
func (s *Runtime) ClearLogs(scope string) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = "all"
	}

	// 清空内存日志
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

	// 保留非指定作用域的日志
	remaining := make([]LogEntry, 0, len(s.logs))
	for _, entry := range s.logs {
		if strings.ToLower(entry.Scope) != scope {
			remaining = append(remaining, entry)
		}
	}
	s.logs = remaining
	return true
}

// RecordLog 记录信息级别日志
// 参数:
//   - scope: 日志作用域（如 "app", "window", "storage"）
//   - message: 日志消息
func (s *Runtime) RecordLog(scope, message string) {
	s.RecordLogWithSeverity(scope, message, "info")
}

// RecordLogWithSeverity 记录指定严重级别的日志。
// 日志通过 slog 写入每日 JSONL 文件，并由 runtime handler 同步到当前内存视图。
func (s *Runtime) RecordLogWithSeverity(scope, message, severity string) {
	// 标准化严重级别
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

func (s *Runtime) shouldRecordLogSeverity(severity string) bool {
	s.lock.RLock()
	level := s.settings.LogLevel
	s.lock.RUnlock()
	return logSeverityRank(severity) >= logSeverityRank(normaliseLogLevel(level))
}

// normaliseLogSeverity 标准化严重级别字符串
// 将各种格式统一为 "error", "warning", "info"
func normaliseLogSeverity(severity string) string {
	return applogging.NormalizeSeverity(severity)
}

func normaliseLogLevel(level string) string {
	return appsettings.NormalizeLogLevel(level)
}

// SlogLevelFromLogLevel 把应用设置中的日志级别映射到 Wails/slog 启动日志级别。
func SlogLevelFromLogLevel(level string) slog.Level {
	return applogging.SlogLevelFromLogLevel(level)
}

func logSeverityRank(severity string) int {
	return applogging.SeverityRank(severity)
}

// calculateLogStats 计算日志统计信息
// 统计总数量、信息数量、警告数量、错误数量
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
