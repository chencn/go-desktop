package runtime

import (
	"fmt"

	"github.com/chencn/go-desktop/internal/desktopapp/crash"
)

const previousCrashTailLines = 40

// CrashState 是 crash 包状态模型的 runtime 对外别名，供启动流程传递上次进程状态。
type CrashState = crash.State

// CrashReporter 是 crash 包 reporter 的 runtime 对外别名，负责写入早期 crash 面包屑。
type CrashReporter = crash.Reporter

// NewCrashReporter 创建早期崩溃日志器；调用方负责把它传入 Runtime。
func NewCrashReporter(logPath string, statePath string) *CrashReporter {
	return crash.NewReporter(logPath, statePath)
}

// StartCrashReporter 启动崩溃状态记录，并返回上一次未正常结束的状态。
func StartCrashReporter(logPath string, statePath string, args []string) (*CrashReporter, CrashState, bool) {
	return crash.StartReporter(logPath, statePath, args)
}

// ReadPreviousCrashState 读取未被当前启动流程消费的上次崩溃状态。
func ReadPreviousCrashState(path string) (CrashState, bool) {
	return crash.ReadPreviousState(path)
}

// RecordPreviousCrash 把上次未正常结束导入应用日志页。
// 只导入 crash.log 中 state.UpdatedAt 附近的尾部片段，避免一次性把历史崩溃日志全部灌入内存视图。
func (s *Runtime) RecordPreviousCrash(state CrashState, ok bool, crashLogPath string) {
	if !ok {
		return
	}
	message := fmt.Sprintf("检测到上次运行未正常结束：pid=%d phase=%s started=%s updated=%s crashLog=%s",
		state.PID, state.Phase, state.StartedAt, state.UpdatedAt, crashLogPath)
	s.RecordLogWithSeverity("crash", message, "error")
	for _, line := range crash.PreviousLogTail(crashLogPath, previousCrashTailLines, state.UpdatedAt) {
		s.RecordLogWithSeverity("crash", "上次 crash.log："+line, "error")
	}
}

// recordCrashBreadcrumb 写入 crash.log 面包屑；crash scope 自身不会再回写以避免递归噪音。
func (s *Runtime) recordCrashBreadcrumb(scope string, format string, args ...any) {
	if s == nil || s.crashReporter == nil {
		return
	}
	s.crashReporter.Append(scope, format, args...)
}
