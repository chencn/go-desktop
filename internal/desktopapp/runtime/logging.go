// 文件职责：提供 runtime 统一日志框架、每日文件 writer 和 JSONL 日志读取能力。
// 说明：日志长期保存到每日文件，内存 ring buffer 只服务当前前端视图。

package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chencn/go-desktop/internal/adapters/filelog"
	applogging "github.com/chencn/go-desktop/internal/desktopapp/logging"
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

// logEntriesForQuery 返回单一日志来源的数据；文件日志可用时不合并内存日志。
// query.FileName 会被限制为 basename，并且必须匹配当前 app 的每日或旧版日志文件名。
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

// readLogFile 读取单个日志文件；优先解析 JSONL，兼容旧 TSV 文件。
// 保留该函数用于旧调用路径，Runtime 查询当前通过 filelog adapter 读取。
func readLogFile(path string) []LogEntry {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	logs := make([]LogEntry, 0)
	for scanner.Scan() {
		if entry, ok := parseLogLine(scanner.Bytes()); ok {
			logs = append(logs, entry)
		}
	}
	return logs
}

// parseLogLine 将当前 JSONL 和旧 TSV 两种文件格式解析为前端日志结构。
func parseLogLine(line []byte) (LogEntry, bool) {
	if entry, ok := parseJSONLogLine(line); ok {
		return entry, true
	}
	return parseLegacyTSVLogLine(line)
}

// parseJSONLogLine 将 slog JSONHandler 输出解析为前端日志结构。
func parseJSONLogLine(line []byte) (LogEntry, bool) {
	var record map[string]any
	if err := json.Unmarshal(line, &record); err != nil {
		return LogEntry{}, false
	}
	entry := LogEntry{
		Time:     stringValue(record["time"]),
		Scope:    stringValue(record["scope"]),
		Severity: stringValue(record["severity"]),
		Message:  stringValue(record["msg"]),
	}
	if entry.Severity == "" {
		entry.Severity = severityFromSlogLevelName(stringValue(record["level"]))
	}
	entry.Scope = normaliseLogScope(entry.Scope)
	entry.Severity = normaliseLogSeverity(entry.Severity)
	entry.Message = strings.TrimSpace(entry.Message)
	return entry, entry.Time != "" && entry.Message != ""
}

// parseLegacyTSVLogLine 兼容旧 time/scope/severity/message 制表符日志。
func parseLegacyTSVLogLine(line []byte) (LogEntry, bool) {
	parts := strings.SplitN(strings.TrimSpace(string(line)), "\t", 4)
	if len(parts) != 4 {
		return LogEntry{}, false
	}
	entry := LogEntry{
		Time:     strings.TrimSpace(parts[0]),
		Scope:    normaliseLogScope(parts[1]),
		Severity: normaliseLogSeverity(parts[2]),
		Message:  strings.TrimSpace(parts[3]),
	}
	if entry.Time == "" || entry.Message == "" {
		return LogEntry{}, false
	}
	if _, err := time.Parse(time.RFC3339Nano, entry.Time); err != nil {
		if _, err := time.Parse(time.RFC3339, entry.Time); err != nil {
			return LogEntry{}, false
		}
	}
	return entry, true
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

// legacyDailyLogWriter 是旧版最小按天文件 writer；保留给旧测试和兼容路径，只轮转不清理。
type legacyDailyLogWriter struct {
	mu       sync.Mutex
	dir      string
	appName  string
	now      func() time.Time
	date     string
	file     *os.File
	filePath string
}

// newDailyLogWriter 创建旧版每日文件 writer；生产日志写入优先使用 filelog.DailyWriter。
func newDailyLogWriter(dir, appName string, now func() time.Time) (*legacyDailyLogWriter, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("日志目录为空")
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "go-desktop"
	}
	if now == nil {
		now = time.Now
	}
	writer := &legacyDailyLogWriter{dir: dir, appName: appName, now: now}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := writer.rotateLocked(now()); err != nil {
		return nil, err
	}
	return writer, nil
}

// Write 写入当前日期对应的日志文件。
func (w *legacyDailyLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureFileLocked(w.now()); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

// CurrentFileName 返回当前打开的每日日志文件路径。
func (w *legacyDailyLogWriter) CurrentFileName() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.filePath
}

// Close 关闭当前文件。
func (w *legacyDailyLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// ensureFileLocked 在日期变化时切换到新的每日文件；调用方必须持有 w.mu。
func (w *legacyDailyLogWriter) ensureFileLocked(now time.Time) error {
	date := now.In(time.Local).Format("2006-01-02")
	if w.file != nil && w.date == date {
		return nil
	}
	return w.rotateLocked(now)
}

// rotateLocked 关闭旧文件并打开指定日期的追加写入文件；调用方必须持有 w.mu。
func (w *legacyDailyLogWriter) rotateLocked(now time.Time) error {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	date := now.In(time.Local).Format("2006-01-02")
	path := filepath.Join(w.dir, w.appName+"-"+date+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.date = date
	w.filePath = path
	w.file = file
	return nil
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

// severityFromSlogLevelName 兼容 slog JSON 中只有 level、没有 severity 字段的记录。
func severityFromSlogLevelName(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return "debug"
	case "WARN", "WARNING":
		return "warning"
	case "ERROR":
		return "error"
	default:
		return "info"
	}
}

// stringValue 只接受 JSON 字符串字段，其他类型按缺失处理。
func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

// selectableLogFileName 判断文件名是否属于当前 app 的每日日志或旧版日志。
func selectableLogFileName(appName, name string) bool {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		return false
	}
	if _, ok := dailyLogDate(appName, name); ok {
		return true
	}
	return legacyLogFileName(appName, name)
}

// legacyLogFileName 判断旧版单文件日志名。
func legacyLogFileName(appName, name string) bool {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "go-desktop"
	}
	return filepath.Base(strings.TrimSpace(name)) == appName+".log"
}

// fileExists 判断路径是否存在且不是目录。
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// normaliseLogScope 标准化日志作用域，空值归入 app。
func normaliseLogScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "app"
	}
	return scope
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
