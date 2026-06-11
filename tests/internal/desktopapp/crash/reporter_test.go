package crash_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chencn/go-desktop/internal/desktopapp/crash"
)

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
