package filelog

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry 是文件日志读取后的标准结构，对应 runtime.LogEntry 的 adapter 层模型。
type Entry struct {
	Time     string
	Scope    string
	Message  string
	Severity string
}

// FileInfo 描述一个可供前端选择的日志文件。
type FileInfo struct {
	Date       string
	FileName   string
	FilePath   string
	SizeBytes  int64
	ModifiedAt string
	Current    bool
}

// DailyWriter 按本地日期轮转 appName-YYYY-MM-DD.log，只负责写入和轮转，不负责清理。
type DailyWriter struct {
	mu       sync.Mutex
	dir      string
	appName  string
	now      func() time.Time
	date     string
	file     *os.File
	filePath string
}

// NewDailyWriter 创建每日文件 writer；目录不存在时会创建，appName 为空时使用 go-desktop。
func NewDailyWriter(dir, appName string, now func() time.Time) (*DailyWriter, error) {
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
	writer := &DailyWriter{dir: dir, appName: appName, now: now}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := writer.rotateLocked(now()); err != nil {
		return nil, err
	}
	return writer, nil
}

// Write 写入当前本地日期对应的日志文件；跨天时会先轮转。
func (w *DailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureFileLocked(w.now()); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

// CurrentFileName 返回当前打开的日志文件完整路径。
func (w *DailyWriter) CurrentFileName() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.filePath
}

// Close 关闭当前日志文件；重复调用安全。
func (w *DailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// ensureFileLocked 确保 writer 指向当前日期文件；调用方必须持有 w.mu。
func (w *DailyWriter) ensureFileLocked(now time.Time) error {
	date := now.In(time.Local).Format("2006-01-02")
	if w.file != nil && w.date == date {
		return nil
	}
	return w.rotateLocked(now)
}

// rotateLocked 切换到指定日期的追加写入文件；调用方必须持有 w.mu。
func (w *DailyWriter) rotateLocked(now time.Time) error {
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

// ReadFile 读取单个日志文件；打不开或无法解析的行会被跳过。
func ReadFile(path string) []Entry {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	logs := make([]Entry, 0)
	for scanner.Scan() {
		if entry, ok := parseLogLine(scanner.Bytes()); ok {
			logs = append(logs, entry)
		}
	}
	return logs
}

// ListFiles 列出当前 app 可选择的每日日志和旧版单文件日志，按日期倒序返回。
func ListFiles(dir, appName, currentPath string) []FileInfo {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	currentName := filepath.Base(currentPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		date, ok := dailyLogDate(appName, entry.Name())
		if !ok {
			if !legacyLogFileName(appName, entry.Name()) {
				continue
			}
			date = info.ModTime()
		}
		files = append(files, FileInfo{
			Date:       date.Format("2006-01-02"),
			FileName:   entry.Name(),
			FilePath:   filepath.Join(dir, entry.Name()),
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339Nano),
			Current:    entry.Name() == currentName,
		})
	}
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Date == files[j].Date {
			return files[i].FileName > files[j].FileName
		}
		return files[i].Date > files[j].Date
	})
	return files
}

// SelectableName 限制前端可查询文件名，避免越权读取日志目录外文件。
func SelectableName(appName, name string) bool {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		return false
	}
	if _, ok := dailyLogDate(appName, name); ok {
		return true
	}
	return legacyLogFileName(appName, name)
}

// Exists 判断路径是否存在且不是目录。
func Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// parseLogLine 兼容当前 slog JSONL 和旧版 TSV 两种格式。
func parseLogLine(line []byte) (Entry, bool) {
	if entry, ok := parseJSONLogLine(line); ok {
		return entry, true
	}
	return parseLegacyTSVLogLine(line)
}

// parseJSONLogLine 解析 slog.JSONHandler 输出的一行 JSON。
func parseJSONLogLine(line []byte) (Entry, bool) {
	var record map[string]any
	if err := json.Unmarshal(line, &record); err != nil {
		return Entry{}, false
	}
	entry := Entry{
		Time:     stringValue(record["time"]),
		Scope:    stringValue(record["scope"]),
		Severity: stringValue(record["severity"]),
		Message:  stringValue(record["msg"]),
	}
	if entry.Severity == "" {
		entry.Severity = severityFromSlogLevelName(stringValue(record["level"]))
	}
	entry.Scope = normaliseScope(entry.Scope)
	entry.Severity = normaliseSeverity(entry.Severity)
	entry.Message = strings.TrimSpace(entry.Message)
	return entry, entry.Time != "" && entry.Message != ""
}

// parseLegacyTSVLogLine 兼容旧 time/scope/severity/message 制表符日志。
func parseLegacyTSVLogLine(line []byte) (Entry, bool) {
	parts := strings.SplitN(strings.TrimSpace(string(line)), "\t", 4)
	if len(parts) != 4 {
		return Entry{}, false
	}
	entry := Entry{
		Time:     strings.TrimSpace(parts[0]),
		Scope:    normaliseScope(parts[1]),
		Severity: normaliseSeverity(parts[2]),
		Message:  strings.TrimSpace(parts[3]),
	}
	if entry.Time == "" || entry.Message == "" {
		return Entry{}, false
	}
	if _, err := time.Parse(time.RFC3339Nano, entry.Time); err != nil {
		if _, err := time.Parse(time.RFC3339, entry.Time); err != nil {
			return Entry{}, false
		}
	}
	return entry, true
}

// dailyLogDate 解析 appName-YYYY-MM-DD.log 中的日期。
func dailyLogDate(appName, name string) (time.Time, bool) {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "go-desktop"
	}
	name = filepath.Base(strings.TrimSpace(name))
	prefix := appName + "-"
	suffix := ".log"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return time.Time{}, false
	}
	dateText := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	parsed, err := time.ParseInLocation("2006-01-02", dateText, time.Local)
	return parsed, err == nil
}

// legacyLogFileName 判断旧版 appName.log 文件名。
func legacyLogFileName(appName, name string) bool {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "go-desktop"
	}
	return filepath.Base(strings.TrimSpace(name)) == appName+".log"
}

// normaliseScope 标准化日志作用域，空值归入 app。
func normaliseScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "app"
	}
	return scope
}

// normaliseSeverity 标准化日志严重级别。
func normaliseSeverity(severity string) string {
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

// severityFromSlogLevelName 兼容缺少 severity 字段的 slog JSON。
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
