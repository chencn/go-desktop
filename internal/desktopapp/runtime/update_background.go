package runtime

import (
	"context"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

// initialUpdateCheckDelay 避免应用刚启动、窗口和日志管线尚在初始化时立即发起网络请求。
const initialUpdateCheckDelay = time.Minute

// StartUpdateBackgroundTasks 启动单个后台更新检查循环。
// 循环只调用 CheckUpdate；即使发现新版本并完成校验，也不会自动安装。
func (s *Runtime) StartUpdateBackgroundTasks() {
	s.startUpdateBackgroundTasks(initialUpdateCheckDelay)
}

// startUpdateBackgroundTasks 用于生产默认延迟和测试注入延迟。
// 若调度器已经存在，本次调用会直接取消新建 context，避免重复后台检查。
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

// runUpdateBackgroundChecks 按设置间隔重复检查更新，直到 Runtime.Shutdown 取消 context。
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

// nextUpdateCheckInterval 从当前设置读取后台检查间隔；非法值回退到 metadata 默认值。
func (s *Runtime) nextUpdateCheckInterval() time.Duration {
	hours := s.SettingsSnapshot().UpdateCheckIntervalHours
	if hours <= 0 {
		hours = metadata.DefaultUpdateCheckIntervalHours
	}
	return time.Duration(hours) * time.Hour
}

// waitUpdateInterval 等待下一次检查或取消信号；非正间隔表示立即触发一次。
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
