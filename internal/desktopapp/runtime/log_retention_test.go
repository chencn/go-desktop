// 文件职责：验证日志保留清理的常驻循环、旧版单文件日志清理与目录保护边界。

package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCleanupExpiredLogFilesRemovesLegacyLogByModTime 验证旧版单文件 appName.log 按修改时间清理。
// 旧版日志文件名不含日期，只能靠 ModTime 判断；这条覆盖避免它成为永不清理的遗留文件。
func TestCleanupExpiredLogFilesRemovesLegacyLogByModTime(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "go-desktop.log")
	crashPath := filepath.Join(dir, "crash.log")
	dbPath := filepath.Join(dir, "go-desktop.db")
	writeTestFile(t, legacyPath, "{}\n")
	writeTestFile(t, crashPath, "crash\n")
	writeTestFile(t, dbPath, "db\n")
	// 把旧版日志的修改时间推回到保留窗口之外，其余文件保持当前时间。
	oldTime := time.Now().AddDate(0, 0, -40)
	if err := os.Chtimes(legacyPath, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes legacy log: %v", err)
	}

	runtime := newRetentionTestRuntime(dir)
	runtime.cleanupExpiredLogFiles(context.Background(), 30)

	requireFileMissing(t, legacyPath)
	for _, path := range []string{crashPath, dbPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected non-log file to stay %s: %v", path, err)
		}
	}
}

// TestCleanupExpiredLogFilesKeepsRecentLegacyLog 验证未过期的旧版单文件日志不会被误删。
func TestCleanupExpiredLogFilesKeepsRecentLegacyLog(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "go-desktop.log")
	writeTestFile(t, legacyPath, "{}\n")

	runtime := newRetentionTestRuntime(dir)
	runtime.cleanupExpiredLogFiles(context.Background(), 30)

	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("expected recent legacy log to stay: %v", err)
	}
}

// TestRunLogRetentionCleanupRepeatsWhileRunning 验证常驻循环会反复清理，而不是只在启动时跑一次。
// 运行中新增的过期文件必须在后续轮次被清掉，否则长期不重启的进程会一直攒文件。
func TestRunLogRetentionCleanupRepeatsWhileRunning(t *testing.T) {
	dir := t.TempDir()
	runtime := newRetentionTestRuntime(dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runtime.runLogRetentionCleanup(ctx, 10*time.Millisecond)

	firstPath := filepath.Join(dir, "go-desktop-2026-01-01.log")
	writeTestFile(t, firstPath, "{}\n")
	requireEventuallyMissing(t, firstPath)

	// 首轮之后才出现第二个过期文件，只有常驻循环才能清掉它。
	secondPath := filepath.Join(dir, "go-desktop-2026-01-02.log")
	writeTestFile(t, secondPath, "{}\n")
	requireEventuallyMissing(t, secondPath)
}

// TestCleanupExpiredLogFilesStopsOnCancelledContext 验证清理在 context 取消后立即返回。
func TestCleanupExpiredLogFilesStopsOnCancelledContext(t *testing.T) {
	dir := t.TempDir()
	expiredPath := filepath.Join(dir, "go-desktop-2026-01-01.log")
	writeTestFile(t, expiredPath, "{}\n")

	runtime := newRetentionTestRuntime(dir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runtime.cleanupExpiredLogFiles(ctx, 30)

	if _, err := os.Stat(expiredPath); err != nil {
		t.Fatalf("expected cancelled cleanup to leave files untouched: %v", err)
	}
}

// newRetentionTestRuntime 构造只含日志目录与保留天数的 Runtime，避免依赖数据库和完整装配。
func newRetentionTestRuntime(logDirPath string) *Runtime {
	settings := defaultSettings()
	settings.LogRetentionDays = 30
	return &Runtime{
		options:    ServiceOptions{AppName: "go-desktop"},
		logDirPath: logDirPath,
		settings:   settings,
	}
}

// writeTestFile 写入测试用日志文件。
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

// requireFileMissing 断言文件不存在。
func requireFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to be removed, stat err=%v", path, err)
	}
}

// requireEventuallyMissing 等待后台清理任务删除指定文件。
func requireEventuallyMissing(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %s to be removed", path)
}
