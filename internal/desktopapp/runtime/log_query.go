// 文件职责：日志查询与分页。负责选择日志来源（文件/内存）、过滤排序分页，
// 以及 filelog 结构到 API DTO 的转换。

package runtime

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/adapters/filelog"
)

// logEntriesForQuery 返回单一日志来源的数据；文件日志可用时不合并内存日志。
// query.FileName 会被限制为 basename，并且必须匹配当前 app 的每日日志文件名。
func (s *Runtime) logEntriesForQuery(query LogQuery) ([]LogEntry, string, string, string) {
	s.lock.RLock()
	logDirPath := s.logDirPath
	appName := s.options.AppName
	writer := s.logWriter
	memoryLogs := append([]LogEntry(nil), s.logs...)
	s.lock.RUnlock()

	currentPath := s.currentLogFilePath()
	fileName := strings.TrimSpace(query.FileName)
	if fileName == "" && currentPath != "" {
		fileName = filepath.Base(currentPath)
	}
	if fileName != "" {
		fileName = filepath.Base(fileName)
	}

	if logDirPath == "" {
		return memoryLogs, "memory", fileName, currentPath
	}
	if !filelog.SelectableName(appName, fileName) {
		filePath := filepath.Join(logDirPath, fileName)
		return nil, "file", fileName, filePath
	}

	filePath := filepath.Join(logDirPath, fileName)
	if filelog.Exists(filePath) {
		return logEntriesFromFileLog(filelog.ReadFile(filePath)), "file", fileName, filePath
	}
	if writer == nil {
		return memoryLogs, "memory", fileName, currentPath
	}
	return nil, "file", fileName, filePath
}

// ListLogFiles 返回日志目录下所有可选择文件日志，按日期倒序排列。
func (s *Runtime) ListLogFiles() []LogFileInfo {
	s.lock.RLock()
	logDirPath := s.logDirPath
	appName := s.options.AppName
	s.lock.RUnlock()
	if logDirPath == "" {
		return nil
	}

	return logFileInfosFromFileLog(filelog.ListFiles(logDirPath, appName, s.currentLogFilePath()))
}

// logEntriesFromFileLog 把 adapter 层日志结构转换为 runtime API DTO。
func logEntriesFromFileLog(entries []filelog.Entry) []LogEntry {
	logs := make([]LogEntry, 0, len(entries))
	for _, entry := range entries {
		logs = append(logs, LogEntry{
			Time:     entry.Time,
			Scope:    entry.Scope,
			Message:  entry.Message,
			Severity: entry.Severity,
		})
	}
	return logs
}

// logFileInfosFromFileLog 把 adapter 层文件描述转换为 runtime API DTO。
func logFileInfosFromFileLog(files []filelog.FileInfo) []LogFileInfo {
	result := make([]LogFileInfo, 0, len(files))
	for _, file := range files {
		result = append(result, LogFileInfo{
			Date:       file.Date,
			FileName:   file.FileName,
			FilePath:   file.FilePath,
			SizeBytes:  file.SizeBytes,
			ModifiedAt: file.ModifiedAt,
			Current:    file.Current,
		})
	}
	return result
}

// filterSortAndPageLogs 统一处理日志过滤、排序、去重与分页；PageSize 最大限制为 200。
// honorViewCleared 为 true 时，会应用 ClearLogs 记录的视图清空时间。
func (s *Runtime) filterSortAndPageLogs(logs []LogEntry, query LogQuery, honorViewCleared bool) LogResponse {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 50
	}
	if query.PageSize > 200 {
		query.PageSize = 200
	}

	scope := strings.ToLower(strings.TrimSpace(query.Scope))
	severity := strings.ToLower(strings.TrimSpace(query.Severity))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))

	seen := map[string]bool{}
	filtered := make([]LogEntry, 0, len(logs))
	for _, entry := range logs {
		entry.Scope = normaliseLogScope(entry.Scope)
		entry.Severity = normaliseLogSeverity(entry.Severity)
		if honorViewCleared && !s.logEntryVisible(entry) {
			continue
		}
		if scope != "" && scope != "all" && strings.ToLower(entry.Scope) != scope {
			continue
		}
		if severity != "" && severity != "all" && strings.ToLower(entry.Severity) != severity {
			continue
		}
		if keyword != "" {
			haystack := strings.ToLower(entry.Scope + " " + entry.Severity + " " + entry.Message)
			if !strings.Contains(haystack, keyword) {
				continue
			}
		}
		key := logEntryKey(entry)
		if seen[key] {
			continue
		}
		seen[key] = true
		filtered = append(filtered, entry)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return logEntryTime(filtered[i]).After(logEntryTime(filtered[j]))
	})

	stats := calculateLogStats(filtered)
	total := len(filtered)
	start := total
	if pageIndex := query.Page - 1; pageIndex <= total/query.PageSize {
		start = pageIndex * query.PageSize
	}
	if start > total {
		start = total
	}
	end := start + query.PageSize
	if end > total {
		end = total
	}

	return LogResponse{
		Logs:     append([]LogEntry(nil), filtered[start:end]...),
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
		HasMore:  end < total,
		Stats:    stats,
	}
}

// logEntryVisible 判断日志是否晚于全局或对应 scope 的最近清空时间。
func (s *Runtime) logEntryVisible(entry LogEntry) bool {
	entryTime := logEntryTime(entry)
	if entryTime.IsZero() {
		return true
	}
	s.lock.RLock()
	clearedAt := s.logViewClearedAt["all"]
	if scopeClearedAt := s.logViewClearedAt[strings.ToLower(entry.Scope)]; scopeClearedAt.After(clearedAt) {
		clearedAt = scopeClearedAt
	}
	s.lock.RUnlock()
	return clearedAt.IsZero() || entryTime.After(clearedAt)
}

// logEntryKey 返回日志去重键。
func logEntryKey(entry LogEntry) string {
	return strings.Join([]string{entry.Time, entry.Scope, entry.Severity, entry.Message}, "\x00")
}

// logEntryTime 解析日志时间；无法解析时返回零值并排到较后位置。
func logEntryTime(entry LogEntry) time.Time {
	if parsed, err := time.Parse(time.RFC3339Nano, entry.Time); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, entry.Time); err == nil {
		return parsed
	}
	return time.Time{}
}
