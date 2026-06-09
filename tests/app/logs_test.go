// 文件职责：验证运行日志查询、文件日志、进程级捕获、崩溃状态和保留清理。

package app_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chencn/go-desktop/app"
)

// TestQueryLogsFiltersByScopeSeverityKeywordAndPaginates 验证日志页核心查询条件。
// 过滤和分页是前端日志排障入口的基础能力，避免大日志量时只能靠前端全量筛。
func TestQueryLogsFiltersByScopeSeverityKeywordAndPaginates(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{})
	runtime.RecordLog("window", "窗口已隐藏到托盘")
	runtime.RecordLogWithSeverity("update", "更新检查失败", "error")
	runtime.RecordLogWithSeverity("update", "更新检查完成", "info")
	runtime.RecordLogWithSeverity("settings", "设置已保存", "warning")

	response := runtime.QueryLogs(app.LogQuery{
		Scope:    "update",
		Severity: "error",
		Keyword:  "失败",
		Page:     1,
		PageSize: 10,
	})

	if response.Total != 1 {
		t.Fatalf("expected 1 filtered log, got %d", response.Total)
	}
	if len(response.Logs) != 1 {
		t.Fatalf("expected 1 log on first page, got %d", len(response.Logs))
	}
	if response.Logs[0].Scope != "update" || response.Logs[0].Severity != "error" || response.Logs[0].Message != "更新检查失败" {
		t.Fatalf("unexpected log entry: %#v", response.Logs[0])
	}
	if response.Stats.Total != 1 || response.Stats.Error != 1 || response.Stats.Info != 0 || response.Stats.Warning != 0 {
		t.Fatalf("unexpected stats: %#v", response.Stats)
	}
	if response.HasMore {
		t.Fatal("expected no more pages")
	}
}

// TestQueryLogsDefaultsAndHasMore 验证分页默认值和下一页标记。
// 日志页每次只拉取一页，HasMore 是“下一页”按钮的唯一后端依据。
func TestQueryLogsDefaultsAndHasMore(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{})
	runtime.RecordLog("update", "第一条")
	runtime.RecordLog("update", "第二条")
	runtime.RecordLog("update", "第三条")

	firstPage := runtime.QueryLogs(app.LogQuery{Page: 1, PageSize: 2})
	if firstPage.Total != 3 {
		t.Fatalf("expected total 3, got %d", firstPage.Total)
	}
	if len(firstPage.Logs) != 2 {
		t.Fatalf("expected 2 logs on first page, got %d", len(firstPage.Logs))
	}
	if !firstPage.HasMore {
		t.Fatal("expected first page to have more logs")
	}

	secondPage := runtime.QueryLogs(app.LogQuery{Page: 2, PageSize: 2})
	if len(secondPage.Logs) != 1 {
		t.Fatalf("expected 1 log on second page, got %d", len(secondPage.Logs))
	}
	if secondPage.HasMore {
		t.Fatal("expected second page to be the last page")
	}
}

func TestQueryLogsAPIDoesNotRecordQueryTraceLogs(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{})
	runtime.RecordLog("update", "已有日志")

	response, err := runtime.API().QueryLogs(app.LogQuery{Scope: "all", Severity: "all", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("query logs through API: %v", err)
	}
	if response.Total != 1 {
		t.Fatalf("expected API query to leave total unchanged, got %d", response.Total)
	}

	logs := runtime.QueryLogs(app.LogQuery{Scope: "api-trace", Page: 1, PageSize: 50})
	if logs.Total != 0 {
		t.Fatalf("expected QueryLogs API to avoid trace noise, got %#v", logs.Logs)
	}
}

func TestQueryLogsClampsHugePageWithoutPanic(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{})
	runtime.RecordLog("update", "第一条")

	maxInt := int(^uint(0) >> 1)
	response := runtime.QueryLogs(app.LogQuery{Page: maxInt, PageSize: 200})
	if response.Total != 1 {
		t.Fatalf("expected total 1, got %d", response.Total)
	}
	if len(response.Logs) != 0 {
		t.Fatalf("expected oversized page to return no logs, got %#v", response.Logs)
	}
	if response.HasMore {
		t.Fatal("expected oversized page to have no next page")
	}
}

// TestClearLogsByScopeAndAll 验证日志清理不会误删其他作用域。
// 设置页的日志保留和日志页的手动清理都依赖这个作用域边界。
func TestClearLogsByScopeAndAll(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{})
	runtime.RecordLog("window", "窗口日志")
	runtime.RecordLog("update", "更新日志")

	if !runtime.ClearLogs("update") {
		t.Fatal("expected clear by scope to succeed")
	}

	remaining := runtime.QueryLogs(app.LogQuery{})
	if remaining.Total != 1 || remaining.Logs[0].Scope != "window" {
		t.Fatalf("expected only window log to remain, got %#v", remaining.Logs)
	}

	if !runtime.ClearLogs("all") {
		t.Fatal("expected clear all to succeed")
	}

	empty := runtime.QueryLogs(app.LogQuery{})
	if empty.Total != 0 || len(empty.Logs) != 0 {
		t.Fatalf("expected logs to be empty, got %#v", empty)
	}
}

// TestClearLogsOnlyClearsCurrentViewKeepsDailyFile 验证清空按钮不删除也不截断每日文件日志。
func TestClearLogsOnlyClearsCurrentViewKeepsDailyFile(t *testing.T) {
	logDir := t.TempDir()
	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   logDir,
	})
	defer runtime.Shutdown()

	runtime.RecordLogWithSeverity("update", "开始下载更新", "info")
	logPath := runtime.GetEnvironmentInfo().LogFilePath
	before, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read daily log before clear: %v", err)
	}
	if !strings.Contains(string(before), "开始下载更新") {
		t.Fatalf("expected daily log to contain update message, got %q", string(before))
	}

	if !runtime.ClearLogs("all") {
		t.Fatal("expected clear current view to succeed")
	}
	response := runtime.QueryLogs(app.LogQuery{})
	if response.Total != 0 || len(response.Logs) != 0 {
		t.Fatalf("expected clear current view to hide file logs, got %#v", response)
	}

	after, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read daily log after clear: %v", err)
	}
	if !strings.Contains(string(after), "开始下载更新") {
		t.Fatalf("expected daily log file to stay intact, got %q", string(after))
	}
}

// TestQueryLogsDefaultsToCurrentDailyFileAndSelectsHistoricalFiles 验证日志查询不会把多个文件混在一起。
func TestQueryLogsDefaultsToCurrentDailyFileAndSelectsHistoricalFiles(t *testing.T) {
	logDir := t.TempDir()
	currentName := "go-desktop-" + time.Now().Format("2006-01-02") + ".log"
	previousName := "go-desktop-" + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + ".log"
	writeJSONLogFixture(t, filepath.Join(logDir, currentName), "current", 7)
	writeJSONLogFixture(t, filepath.Join(logDir, previousName), "previous", 4)

	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   logDir,
	})
	defer runtime.Shutdown()

	defaultResponse := runtime.QueryLogs(app.LogQuery{Page: 1, PageSize: 20})
	if defaultResponse.Total != 7 {
		t.Fatalf("expected default query to read only current daily file, got %d logs: %#v", defaultResponse.Total, defaultResponse.Logs)
	}
	for _, entry := range defaultResponse.Logs {
		if !strings.HasPrefix(entry.Message, "current-") {
			t.Fatalf("default query should not include historical file entries, got %#v", entry)
		}
	}
	if defaultResponse.FileName != currentName || defaultResponse.Source != "file" {
		t.Fatalf("expected current file metadata, got source=%q file=%q", defaultResponse.Source, defaultResponse.FileName)
	}

	historicalResponse := runtime.QueryLogs(app.LogQuery{FileName: previousName, Page: 1, PageSize: 20})
	if historicalResponse.Total != 4 {
		t.Fatalf("expected explicit historical query to read previous file, got %d logs: %#v", historicalResponse.Total, historicalResponse.Logs)
	}
	for _, entry := range historicalResponse.Logs {
		if !strings.HasPrefix(entry.Message, "previous-") {
			t.Fatalf("historical query should not include current file entries, got %#v", entry)
		}
	}
	if historicalResponse.FileName != previousName {
		t.Fatalf("expected selected historical file name %q, got %q", previousName, historicalResponse.FileName)
	}

	files := runtime.ListLogFiles()
	if len(files) != 2 {
		t.Fatalf("expected two selectable log files, got %#v", files)
	}
	if files[0].FileName != currentName || !files[0].Current {
		t.Fatalf("expected current daily file first, got %#v", files)
	}
}

// TestQueryLogsCanReadLegacyTSVLogFile 验证旧 go-desktop.log 可选择、可读取。
func TestQueryLogsCanReadLegacyTSVLogFile(t *testing.T) {
	logDir := t.TempDir()
	legacyName := "go-desktop.log"
	legacyPath := filepath.Join(logDir, legacyName)
	legacyContent := strings.Join([]string{
		"2026-06-05T14:49:14Z\tshortcut\tinfo\t快捷方式检查完成",
		"2026-06-05T14:49:15Z\tstorage\twarning\t配置目录不可写",
		"2026-06-05T14:49:16Z\tpanic\terror\tWails panic",
	}, "\n") + "\n"
	if err := os.WriteFile(legacyPath, []byte(legacyContent), 0o644); err != nil {
		t.Fatalf("write legacy log fixture: %v", err)
	}

	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   logDir,
	})
	defer runtime.Shutdown()

	files := runtime.ListLogFiles()
	if !logFileListContains(files, legacyName) {
		t.Fatalf("expected legacy log file to be selectable, got %#v", files)
	}

	response := runtime.QueryLogs(app.LogQuery{FileName: legacyName, Page: 1, PageSize: 20})
	if response.Source != "file" || response.FileName != legacyName {
		t.Fatalf("expected legacy query to use file source, got source=%q file=%q", response.Source, response.FileName)
	}
	if response.Total != 3 || response.Stats.Info != 1 || response.Stats.Warning != 1 || response.Stats.Error != 1 {
		t.Fatalf("expected legacy TSV severities to stay intact, got %#v", response)
	}
	assertLogResponseContains(t, response, "shortcut", "info", "快捷方式检查完成")
	assertLogResponseContains(t, response, "panic", "error", "Wails panic")
}

// TestRuntimeWritesDailyJSONLogFile 验证业务日志会写入每日 JSONL 文件。
func TestRuntimeWritesDailyJSONLogFile(t *testing.T) {
	logDir := t.TempDir()
	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   logDir,
	})
	defer runtime.Shutdown()

	runtime.RecordLogWithSeverity("process", "启动参数校验失败", "error")

	logPath := runtime.GetEnvironmentInfo().LogFilePath
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read daily log file: %v", err)
	}
	content := string(data)
	for _, required := range []string{`"scope":"process"`, `"severity":"error"`, "启动参数校验失败"} {
		if !strings.Contains(content, required) {
			t.Fatalf("expected file log to include %q, got %q", required, content)
		}
	}
}

// TestInstallProcessLogCaptureRecordsStandardStructuredAndStreamLogs 验证进程级日志入口。
// 用户排障时不能只看到手工 RecordLog，log/slog/stdout/stderr 都必须进入查询和文件日志。
func TestInstallProcessLogCaptureRecordsStandardStructuredAndStreamLogs(t *testing.T) {
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldSlog := slog.Default()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		slog.SetDefault(oldSlog)
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "go-desktop.db")
	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: dbPath,
		LogDirPath:   tempDir,
	})
	t.Cleanup(runtime.Shutdown)
	runtime.InstallProcessLogCapture()

	log.Print("标准库 fatal error: 静默安装器启动失败")
	slog.Warn("结构化窗口警告", "component", "wails", "stage", "create-window")
	fmt.Fprintln(os.Stdout, "stdout update pipeline started")
	fmt.Fprintln(os.Stderr, "stderr panic: webview 初始化失败")

	processLogs := waitForProcessLogs(t, runtime, "标准库 fatal error", "结构化窗口警告", "stdout update pipeline started", "stderr panic")
	assertLogResponseContains(t, processLogs, "process", "error", "标准库 fatal error")
	assertLogResponseContains(t, processLogs, "process", "warning", "结构化窗口警告", "component=wails")
	assertLogResponseContains(t, processLogs, "process", "info", "stdout: stdout update pipeline started")
	assertLogResponseContains(t, processLogs, "process", "error", "stderr: stderr panic")

	runtime.Shutdown()
	if os.Stdout != oldStdout || os.Stderr != oldStderr {
		t.Fatal("expected runtime shutdown to restore stdout and stderr")
	}
	if log.Writer() != oldWriter || log.Flags() != oldFlags || slog.Default() != oldSlog {
		t.Fatal("expected runtime shutdown to restore log and slog globals")
	}

	data, err := os.ReadFile(runtime.GetEnvironmentInfo().LogFilePath)
	if err != nil {
		t.Fatalf("read mirrored process log file: %v", err)
	}
	content := string(data)
	for _, required := range []string{"标准库 fatal error", "结构化窗口警告", "component=wails", "stdout update pipeline started", "stderr panic"} {
		if !strings.Contains(content, required) {
			t.Fatalf("expected file log to include %q, got %q", required, content)
		}
	}
}

// TestProcessLogCaptureDoesNotPromoteStackFramesToErrors 验证堆栈函数名不应因包含 error 字样被标红。
func TestProcessLogCaptureDoesNotPromoteStackFramesToErrors(t *testing.T) {
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldSlog := slog.Default()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		slog.SetDefault(oldSlog)
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	tempDir := t.TempDir()
	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(tempDir, "go-desktop.db"),
		LogDirPath:   tempDir,
	})
	t.Cleanup(runtime.Shutdown)
	runtime.InstallProcessLogCapture()

	fmt.Fprintln(os.Stderr, "1: github.com/example/app.errorCallback  github.com/example/app/window.go:38")
	fmt.Fprintln(os.Stderr, "fatal: install failed")

	processLogs := waitForProcessLogs(t, runtime, "errorCallback", "fatal: install failed")
	assertLogResponseContains(t, processLogs, "process", "info", "errorCallback")
	assertLogResponseContains(t, processLogs, "process", "error", "fatal: install failed")
}

// TestReadPreviousCrashStateIgnoresLiveAndCleanStates 验证第二实例不会把正在运行的主进程误判为上次崩溃。
func TestReadPreviousCrashStateIgnoresLiveAndCleanStates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crash-state.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)

	writeCrashStateFixture(t, path, app.CrashState{
		PID:       os.Getpid(),
		StartedAt: now,
		UpdatedAt: now,
		Phase:     "运行 Wails",
	})
	if state, ok := app.ReadPreviousCrashState(path); ok {
		t.Fatalf("expected live process crash state to be ignored, got %#v", state)
	}

	writeCrashStateFixture(t, path, app.CrashState{
		PID:       999999,
		StartedAt: now,
		UpdatedAt: now,
		Phase:     "正常退出：Runtime.QuitApp",
	})
	if state, ok := app.ReadPreviousCrashState(path); ok {
		t.Fatalf("expected clean crash state to be ignored, got %#v", state)
	}

	writeCrashStateFixture(t, path, app.CrashState{
		PID:       999999,
		StartedAt: now,
		UpdatedAt: now,
		Phase:     "运行 Wails",
	})
	state, ok := app.ReadPreviousCrashState(path)
	if !ok || state.PID != 999999 || state.Phase != "运行 Wails" {
		t.Fatalf("expected stale unclean crash state to be imported, got ok=%t state=%#v", ok, state)
	}
}

// TestStartupDeletesExpiredDailyLogFilesByRetentionDays 验证启动/设置保留清理只删除过期每日文件日志。
func TestStartupDeletesExpiredDailyLogFilesByRetentionDays(t *testing.T) {
	logDir := t.TempDir()
	oldPath := filepath.Join(logDir, "go-desktop-2026-01-01.log")
	newPath := filepath.Join(logDir, "go-desktop-2999-01-01.log")
	crashPath := filepath.Join(logDir, "crash.log")
	dbPath := filepath.Join(logDir, "go-desktop.db")
	for path, content := range map[string]string{
		oldPath:   "{}\n",
		newPath:   "{}\n",
		crashPath: "crash\n",
		dbPath:    "db\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   logDir,
	})
	defer runtime.Shutdown()
	if _, err := runtime.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 12,
		MinimizeToTray:           true,
		LogRetentionDays:         7,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	requireEventuallyMissing(t, oldPath)
	for _, path := range []string{newPath, crashPath, dbPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected non-expired or unrelated file to stay %s: %v", path, err)
		}
	}
}

// TestRuntimeLogLevelFiltersLowSeverityLogs 验证设置中的日志级别会过滤低级别日志。
func TestRuntimeLogLevelFiltersLowSeverityLogs(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtime.Shutdown()

	if _, err := runtime.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 3,
		MinimizeToTray:           true,
		LogRetentionDays:         30,
		LogLevel:                 "warning",
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	runtime.RecordLogWithSeverity("settings", "debug should be hidden", "debug")
	runtime.RecordLogWithSeverity("settings", "info should be hidden", "info")
	runtime.RecordLogWithSeverity("settings", "warning should stay", "warning")
	runtime.RecordLogWithSeverity("settings", "error should stay", "error")

	logs := runtime.QueryLogs(app.LogQuery{Scope: "settings"})
	messages := map[string]bool{}
	for _, entry := range logs.Logs {
		messages[entry.Message] = true
	}
	if messages["debug should be hidden"] || messages["info should be hidden"] {
		t.Fatalf("expected low severity logs to be filtered, got %#v", logs.Logs)
	}
	if !messages["warning should stay"] || !messages["error should stay"] {
		t.Fatalf("expected warning/error logs to remain, got %#v", logs.Logs)
	}
}

func assertLogResponseContains(t *testing.T, response app.LogResponse, required ...string) {
	t.Helper()
	for _, entry := range response.Logs {
		haystack := strings.Join([]string{entry.Scope, entry.Severity, entry.Message}, " ")
		matched := true
		for _, value := range required {
			if !strings.Contains(haystack, value) {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("expected logs to contain %q, got %#v", strings.Join(required, " / "), response.Logs)
}

// logFileListContains 判断文件列表中是否存在指定文件名。
func logFileListContains(files []app.LogFileInfo, fileName string) bool {
	for _, file := range files {
		if file.FileName == fileName {
			return true
		}
	}
	return false
}

// writeCrashStateFixture 写入 crash-state 测试夹具。
func writeCrashStateFixture(t *testing.T, path string, state app.CrashState) {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal crash state fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write crash state fixture: %v", err)
	}
}

// waitForProcessLogs 等待异步 stdout/stderr 捕获写入 Runtime 日志视图。
func waitForProcessLogs(t *testing.T, runtime *app.Runtime, required ...string) app.LogResponse {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var response app.LogResponse
	for time.Now().Before(deadline) {
		response = runtime.QueryLogs(app.LogQuery{Scope: "process", Page: 1, PageSize: 50})
		if logResponseContainsAll(response, required...) {
			return response
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected process logs to eventually include %q, got %#v", strings.Join(required, " / "), response.Logs)
	return response
}

// requireEventuallyMissing 等待后台清理任务删除指定文件。
func requireEventuallyMissing(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected %s to be removed", path)
}

// writeJSONLogFixture 写入 slog JSONHandler 兼容的一行一条 JSONL 测试文件。
func writeJSONLogFixture(t *testing.T, path string, prefix string, count int) {
	t.Helper()
	var builder strings.Builder
	base := time.Now().UTC().Add(-time.Duration(count) * time.Minute)
	for i := 0; i < count; i++ {
		fmt.Fprintf(&builder, `{"time":%q,"level":"INFO","msg":%q,"scope":"app","severity":"info"}`+"\n",
			base.Add(time.Duration(i)*time.Minute).Format(time.RFC3339Nano),
			fmt.Sprintf("%s-%02d", prefix, i+1),
		)
	}
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write log fixture %s: %v", path, err)
	}
}

func logResponseContainsAll(response app.LogResponse, required ...string) bool {
	for _, value := range required {
		found := false
		for _, entry := range response.Logs {
			if strings.Contains(entry.Message, value) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
