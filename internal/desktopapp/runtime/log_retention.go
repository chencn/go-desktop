// 文件职责：按设置周期性清理过期的文件日志。
// 说明：清理只处理每日 appName-YYYY-MM-DD.log 和旧版单文件 appName.log，
// 不触碰数据库、crash 日志或其他文件。

package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chencn/go-desktop/internal/adapters/filelog"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

// logRetentionCheckInterval 是常驻清理的检查周期。
// 每日日志按天轮转、保留以天计，每小时扫一次足以让运行期新过期的文件当天被清掉，
// 又不会频繁读目录；只在启动和改设置时清理会让长期不重启的进程一直攒文件。
const logRetentionCheckInterval = time.Hour

// startLogRetentionCleanup 启动常驻日志保留清理任务，并替换上一次未完成的清理上下文。
func (s *Runtime) startLogRetentionCleanup() {
	s.startLogRetentionCleanupWithInterval(logRetentionCheckInterval)
}

// startLogRetentionCleanupWithInterval 允许测试注入短周期；非正周期回退到默认值。
func (s *Runtime) startLogRetentionCleanupWithInterval(interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	s.lock.Lock()
	if s.logCleanupStop != nil {
		s.logCleanupStop()
	}
	s.logCleanupStop = cancel
	s.lock.Unlock()

	go s.runLogRetentionCleanup(ctx, interval)
}

// runLogRetentionCleanup 周期性清理过期日志，直到 Runtime.Shutdown 取消 context。
// 首次进入立即清理一次，保持与旧的一次性任务一致的启动行为。
func (s *Runtime) runLogRetentionCleanup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = logRetentionCheckInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		s.cleanupExpiredLogFilesSafely(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// cleanupExpiredLogFilesSafely 执行单轮清理并隔离 panic，避免常驻循环被一次异常打断。
func (s *Runtime) cleanupExpiredLogFilesSafely(ctx context.Context) {
	defer s.RecoverPanic("日志保留清理")
	s.cleanupExpiredLogFiles(ctx, s.SettingsSnapshot().LogRetentionDays)
}

// cleanupExpiredLogFiles 删除超过保留天数的文件日志。
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
		s.removeExpiredLogFile(entry, logDirPath, appName, cutoff)
	}
}

// removeExpiredLogFile 判断单个目录项是否过期并删除；非日志文件一律跳过。
// 每日日志的文件名自带日期，旧版单文件 appName.log 没有日期，只能按最后修改时间判断。
func (s *Runtime) removeExpiredLogFile(entry os.DirEntry, logDirPath, appName string, cutoff time.Time) {
	name := entry.Name()

	if date, ok := filelog.DailyLogDate(appName, name); ok {
		if date.Before(cutoff) {
			s.removeLogFile(logDirPath, name)
		}
		return
	}

	if !filelog.LegacyLogFileName(appName, name) {
		return
	}
	info, err := entry.Info()
	if err != nil {
		s.RecordLogWithSeverity("log-file", fmt.Sprintf("读取日志文件信息失败：%s", err), "warning")
		return
	}
	if info.ModTime().Before(cutoff) {
		s.removeLogFile(logDirPath, name)
	}
}

// removeLogFile 删除单个日志文件；失败只记录 warning，不中断本轮清理。
func (s *Runtime) removeLogFile(logDirPath, name string) {
	if err := os.Remove(filepath.Join(logDirPath, name)); err != nil {
		s.RecordLogWithSeverity("log-file", fmt.Sprintf("删除过期日志失败：%s", err), "warning")
	}
}
