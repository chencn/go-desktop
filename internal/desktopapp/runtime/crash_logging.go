package runtime

import (
	"fmt"

	"github.com/chencn/go-desktop/internal/desktopapp/crash"
)

const previousCrashTailLines = 40

type CrashState = crash.State
type CrashReporter = crash.Reporter

func NewCrashReporter(logPath string, statePath string) *CrashReporter {
	return crash.NewReporter(logPath, statePath)
}

func StartCrashReporter(logPath string, statePath string, args []string) (*CrashReporter, CrashState, bool) {
	return crash.StartReporter(logPath, statePath, args)
}

func ReadPreviousCrashState(path string) (CrashState, bool) {
	return crash.ReadPreviousState(path)
}

// RecordPreviousCrash 把上次未正常结束导入应用日志页。
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

func (s *Runtime) recordCrashBreadcrumb(scope string, format string, args ...any) {
	if s == nil || s.crashReporter == nil {
		return
	}
	s.crashReporter.Append(scope, format, args...)
}
