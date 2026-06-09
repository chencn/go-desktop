// 文件职责：按设置清理过期的每日文件日志。
// 说明：清理只处理 appName-YYYY-MM-DD.log，不触碰数据库、crash 日志或其他文件。

package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

// startLogRetentionCleanup 启动一次后台日志保留清理任务，并替换上一次未完成的清理上下文。
func (s *Runtime) startLogRetentionCleanup() {
	ctx, cancel := context.WithCancel(context.Background())
	s.lock.Lock()
	if s.logCleanupStop != nil {
		s.logCleanupStop()
	}
	s.logCleanupStop = cancel
	s.lock.Unlock()

	go func() {
		s.cleanupExpiredLogFiles(ctx, s.SettingsSnapshot().LogRetentionDays)
	}()
}

// cleanupExpiredLogFiles 删除超过保留天数的每日文件日志。
// retentionDays=-1 表示永不清理，0 使用 metadata 默认值；失败只记录 warning，不影响 Runtime。
func (s *Runtime) cleanupExpiredLogFiles(ctx context.Context, retentionDays int) {
	if retentionDays < 0 {
		return
	}
	if retentionDays == 0 {
		retentionDays = metadata.DefaultLogRetentionDays
	}
	s.lock.RLock()
	logDirPath := s.logDirPath
	appName := s.options.AppName
	s.lock.RUnlock()
	if logDirPath == "" {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	entries, err := os.ReadDir(logDirPath)
	if err != nil {
		s.RecordLogWithSeverity("log-file", fmt.Sprintf("读取日志目录失败：%s", err), "warning")
		return
	}
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if entry.IsDir() {
			continue
		}
		date, ok := dailyLogDate(appName, entry.Name())
		if !ok || !date.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(logDirPath, entry.Name())); err != nil {
			s.RecordLogWithSeverity("log-file", fmt.Sprintf("删除过期日志失败：%s", err), "warning")
		}
	}
}

// dailyLogDate 只识别 appName-YYYY-MM-DD.log，用于避免误删 crash.log、SQLite 或其他文件。
func dailyLogDate(appName, name string) (time.Time, bool) {
	prefix := appName + "-"
	suffix := ".log"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return time.Time{}, false
	}
	value := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	date, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return date, true
}
