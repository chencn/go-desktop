package crash_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chencn/go-desktop/internal/desktopapp/crash"
)

// TestFinishRemovesStateAfterMarkClean 验证标记正常退出后状态文件被删除。
// 状态文件残留会让下次启动把本次正常退出误报为"上次未正常结束"，
// 并在日志页刷出一批历史 crash.log 面包屑。
func TestFinishRemovesStateAfterMarkClean(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "crash.log")
	statePath := filepath.Join(t.TempDir(), "crash-state.json")

	reporter := crash.NewReporter(logPath, statePath)
	reporter.Start([]string{"go-desktop.exe"})
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected Start to persist state file: %v", err)
	}

	reporter.MarkClean("Wails 主循环返回")
	reporter.Finish("主入口")

	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("expected clean exit to remove state file, stat err = %v", err)
	}
}

// TestFinishKeepsStateWhenNotMarkedClean 验证未标记正常退出时状态文件保留，供下次启动导入线索。
// 这里直接检查文件内容：ReadPreviousState 会按 PID 过滤，测试进程自身 PID 会被判为存活。
func TestFinishKeepsStateWhenNotMarkedClean(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "crash.log")
	statePath := filepath.Join(t.TempDir(), "crash-state.json")

	reporter := crash.NewReporter(logPath, statePath)
	reporter.Start([]string{"go-desktop.exe"})
	reporter.Finish("主入口")

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("expected unclean exit to keep state file for next startup: %v", err)
	}
	if !strings.Contains(string(data), "未标记正常退出") {
		t.Fatalf("expected state to record unclean exit, got %q", string(data))
	}
}

func TestTrimLogFileKeepsRecentWholeLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crash.log")
	content := strings.Join([]string{
		"2026-06-01T00:00:00Z\tcrash\terror\told boot",
		"2026-06-02T00:00:00Z\tpanic\terror\told panic",
		"2026-06-03T00:00:00Z\tcrash\terror\trecent boot",
		"2026-06-04T00:00:00Z\tpanic\terror\trecent panic",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write crash log: %v", err)
	}

	if err := crash.TrimLogFile(path, 128); err != nil {
		t.Fatalf("trim crash log: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trimmed crash log: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "old boot") || strings.Contains(text, "old panic") {
		t.Fatalf("expected old crash lines to be trimmed, got %q", text)
	}
	if !strings.Contains(text, "recent boot") || !strings.Contains(text, "recent panic") {
		t.Fatalf("expected recent crash lines to remain, got %q", text)
	}
	if strings.HasPrefix(text, "panic\terror") {
		t.Fatalf("expected trim to start at a whole line, got %q", text)
	}
}
