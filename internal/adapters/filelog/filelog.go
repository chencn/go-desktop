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

type Entry struct {
	Time     string
	Scope    string
	Message  string
	Severity string
}

type FileInfo struct {
	Date       string
	FileName   string
	FilePath   string
	SizeBytes  int64
	ModifiedAt string
	Current    bool
}

type DailyWriter struct {
	mu       sync.Mutex
	dir      string
	appName  string
	now      func() time.Time
	date     string
	file     *os.File
	filePath string
}

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

func (w *DailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureFileLocked(w.now()); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *DailyWriter) CurrentFileName() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.filePath
}

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

func (w *DailyWriter) ensureFileLocked(now time.Time) error {
	date := now.In(time.Local).Format("2006-01-02")
	if w.file != nil && w.date == date {
		return nil
	}
	return w.rotateLocked(now)
}

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

func Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func parseLogLine(line []byte) (Entry, bool) {
	if entry, ok := parseJSONLogLine(line); ok {
		return entry, true
	}
	return parseLegacyTSVLogLine(line)
}

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

func legacyLogFileName(appName, name string) bool {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "go-desktop"
	}
	return filepath.Base(strings.TrimSpace(name)) == appName+".log"
}

func normaliseScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "app"
	}
	return scope
}

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

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}
