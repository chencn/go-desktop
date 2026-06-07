package runtime

import (
	"context"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

const initialUpdateCheckDelay = time.Minute

// StartUpdateBackgroundTasks 启动后台更新检查任务。
func (s *Runtime) StartUpdateBackgroundTasks() {
	s.startUpdateBackgroundTasks(initialUpdateCheckDelay)
}

func (s *Runtime) startUpdateBackgroundTasks(initialDelay time.Duration) {
	if s == nil {
		return
	}
	if initialDelay < 0 {
		initialDelay = 0
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.lock.Lock()
	if s.updateSchedulerStop != nil {
		s.lock.Unlock()
		cancel()
		return
	}
	s.updateSchedulerStop = cancel
	s.lock.Unlock()
	go s.runUpdateBackgroundChecks(ctx, initialDelay)
}

func (s *Runtime) runUpdateBackgroundChecks(ctx context.Context, initialDelay time.Duration) {
	if !waitUpdateInterval(ctx, initialDelay) {
		return
	}
	for {
		func() {
			defer s.RecoverPanic("后台更新检查")
			s.RecordLog("update", "后台更新检查开始")
			s.CheckUpdate()
		}()
		if !waitUpdateInterval(ctx, s.nextUpdateCheckInterval()) {
			return
		}
	}
}

func (s *Runtime) nextUpdateCheckInterval() time.Duration {
	hours := s.SettingsSnapshot().UpdateCheckIntervalHours
	if hours <= 0 {
		hours = metadata.DefaultUpdateCheckIntervalHours
	}
	return time.Duration(hours) * time.Hour
}

func waitUpdateInterval(ctx context.Context, interval time.Duration) bool {
	if interval <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
