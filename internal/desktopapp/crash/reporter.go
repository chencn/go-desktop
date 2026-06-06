package crash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/chencn/go-desktop/internal/platform/process"
)

// State 记录一次桌面进程启动的关键阶段。
type State struct {
	PID       int      `json:"pid"`
	Args      []string `json:"args"`
	StartedAt string   `json:"startedAt"`
	UpdatedAt string   `json:"updatedAt"`
	Phase     string   `json:"phase"`
}

// Reporter 是 main 最早安装的落盘日志，不能依赖 Runtime、SQLite 或 Wails。
type Reporter struct {
	logPath     string
	statePath   string
	state       State
	clean       bool
	cleanReason string
	lock        sync.Mutex
}

// NewReporter 创建最早期 crash 日志器。
func NewReporter(logPath string, statePath string) *Reporter {
	return &Reporter{
		logPath:   strings.TrimSpace(logPath),
		statePath: strings.TrimSpace(statePath),
	}
}

// StartReporter 读取上次异常状态并安装本次最早期 crash 输出。
func StartReporter(logPath string, statePath string, args []string) (*Reporter, State, bool) {
	previousCrash, hasPreviousCrash := ReadPreviousState(statePath)
	reporter := NewReporter(logPath, statePath)
	reporter.Start(args)
	reporter.InstallRuntimeCrashOutput()
	return reporter, previousCrash, hasPreviousCrash
}

// ReadPreviousState 读取上次未清理的运行状态。
func ReadPreviousState(path string) (State, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}, false
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, false
	}
	if strings.TrimSpace(state.StartedAt) == "" {
		return State{}, false
	}
	if crashStateMarkedClean(state) || crashStateBelongsToLiveProcess(state) {
		return State{}, false
	}
	return state, true
}

// Start 写入本次启动状态，后续如果进程直接消失，下次启动能看到最后阶段。
func (r *Reporter) Start(args []string) {
	if r == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	r.lock.Lock()
	r.state = State{
		PID:       os.Getpid(),
		Args:      append([]string(nil), args...),
		StartedAt: now,
		UpdatedAt: now,
		Phase:     "启动",
	}
	r.lock.Unlock()
	r.writeState()
	r.Append("crash", "进程启动：pid=%d args=%q", os.Getpid(), strings.Join(args, " "))
}

// InstallRuntimeCrashOutput 把 Go runtime 的未处理 panic/fatal error 直接写入 crash.log。
func (r *Reporter) InstallRuntimeCrashOutput() {
	if r == nil || r.logPath == "" {
		return
	}
	debug.SetTraceback("all")
	_ = os.MkdirAll(filepath.Dir(r.logPath), 0o755)
	file, err := os.OpenFile(r.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		r.Append("crash", "安装 Go crash output 失败：%s", err)
		return
	}
	defer file.Close()
	if err := debug.SetCrashOutput(file, debug.CrashOptions{}); err != nil {
		r.Append("crash", "安装 Go crash output 失败：%s", err)
		return
	}
	r.Append("crash", "Go crash output 已安装")
}

// Phase 更新启动阶段并写入 crash.log。
func (r *Reporter) Phase(phase string) {
	if r == nil {
		return
	}
	phase = strings.TrimSpace(phase)
	if phase == "" {
		return
	}
	r.lock.Lock()
	r.state.Phase = phase
	r.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	r.lock.Unlock()
	r.writeState()
	r.Append("crash", "启动阶段：%s", phase)
}

// MarkClean 标记当前进程退出是业务预期路径；未标记退出会在下次启动导入日志页。
func (r *Reporter) MarkClean(reason string) {
	if r == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "正常退出"
	}
	r.lock.Lock()
	r.clean = true
	r.cleanReason = reason
	r.state.Phase = "正常退出：" + reason
	r.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	r.lock.Unlock()
	r.writeState()
	r.Append("crash", "标记正常退出：%s", reason)
}

// Append 直接追加 crash 文件日志。
func (r *Reporter) Append(scope string, format string, args ...any) {
	if r == nil || r.logPath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(r.logPath), 0o755)
	file, err := os.OpenFile(r.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	message := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(file, "%s\t%s\terror\t%s\n", time.Now().UTC().Format(time.RFC3339Nano), scope, fileLogMessage(message))
}

// fileLogMessage 将 crash 文本日志内容压成单行，避免堆栈里的换行破坏一行一条。
func fileLogMessage(message string) string {
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\t", " ")
	return strings.TrimSpace(message)
}

// Finish 清理状态文件；如果主入口 panic，先写 stack 再继续抛出，保留异常退出语义。
func (r *Reporter) Finish(operation string) {
	if r == nil {
		return
	}
	if recovered := recover(); recovered != nil {
		r.markPhase("panic")
		r.Append("panic", "%s panic：%v\n%s", strings.TrimSpace(operation), recovered, string(debug.Stack()))
		panic(recovered)
	}
	clean, reason := r.cleanSnapshot()
	if clean {
		r.Append("crash", "进程正常结束：%s", reason)
		if r.statePath != "" {
			_ = os.Remove(r.statePath)
		}
		return
	}
	r.markPhase("未标记正常退出：" + strings.TrimSpace(operation))
	r.Append("crash", "进程结束但未标记正常退出：%s", strings.TrimSpace(operation))
}

func (r *Reporter) cleanSnapshot() (bool, string) {
	if r == nil {
		return false, ""
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.clean, r.cleanReason
}

func (r *Reporter) markPhase(phase string) {
	phase = strings.TrimSpace(phase)
	if r == nil || phase == "" {
		return
	}
	r.lock.Lock()
	r.state.Phase = phase
	r.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	r.lock.Unlock()
	r.writeState()
}

func (r *Reporter) writeState() {
	if r == nil || r.statePath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(r.statePath), 0o755)
	data, err := json.MarshalIndent(r.state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(r.statePath, data, 0o644)
}

func PreviousLogTail(path string, limit int, notAfter string) []string {
	path = strings.TrimSpace(path)
	if path == "" || limit <= 0 {
		return nil
	}
	cutoff, hasCutoff := parseCrashLogTime(notAfter)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	rawLines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	lines := make([]string, 0, limit)
	for i := len(rawLines) - 1; i >= 0 && len(lines) < limit; i-- {
		line := strings.TrimSpace(rawLines[i])
		if line == "" {
			continue
		}
		if hasCutoff {
			if lineTime, ok := parseCrashLogLineTime(line); ok && lineTime.After(cutoff) {
				continue
			}
		}
		lines = append(lines, line)
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines
}

func parseCrashLogLineTime(line string) (time.Time, bool) {
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) == 0 {
		return time.Time{}, false
	}
	return parseCrashLogTime(parts[0])
}

func parseCrashLogTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func crashStateMarkedClean(state State) bool {
	return strings.HasPrefix(strings.TrimSpace(state.Phase), "正常退出")
}

func crashStateBelongsToLiveProcess(state State) bool {
	if state.PID <= 0 || state.PID == os.Getpid() {
		return state.PID == os.Getpid()
	}
	return process.Alive(state.PID)
}
