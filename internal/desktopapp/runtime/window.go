// 文件职责：管理主窗口显示、退出、关闭到托盘和第二实例启动记录。

package runtime

import (
	goruntime "runtime"
	"strings"
	"time"
)

// StartupLaunch 描述本次启动是否来自需要特殊窗口行为的启动入口。
type StartupLaunch struct {
	// Autostart 表示本次启动来自开机自启入口。
	Autostart bool

	// Hidden 表示启动入口要求默认隐藏主窗口。
	Hidden bool

	// DesktopShortcut 表示本次启动来自应用创建的桌面快捷方式。
	DesktopShortcut bool
}

// ShowMainWindow API 方法，显示主窗口。
func (api *API) ShowMainWindow() (err error) {
	defer api.recoverError("显示主窗口", &err)
	api.runtime.ShowMainWindow()
	return nil
}

// ShowMainWindow 显示主窗口。
// 如果存在启动加载窗口（splash），会在显示主窗口后自动关闭它。
// Windows 平台默认短暂置顶再取消；用户开启窗口置顶时会保持置顶。
func (s *Runtime) ShowMainWindow() {
	s.lock.Lock()
	window := s.mainWindow
	splash := s.splashWindow
	alwaysOnTop := s.settings.AlwaysOnTop
	s.splashWindow = nil
	s.lock.Unlock()
	if window == nil {
		return
	}
	if window.IsMinimised() {
		window.UnMinimise()
	}
	if !window.IsMaximised() {
		window.Maximise()
	}
	window.Show()
	if splash != nil {
		splash.Close()
		s.RecordLog("window", "主窗口内容已就绪，启动加载窗口已关闭")
	}
	if goruntime.GOOS == "windows" {
		window.SetAlwaysOnTop(true)
		window.Focus()
		s.RecordLog("window", "窗口已显示")
		if !alwaysOnTop {
			go func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						s.RecordLogWithSeverity("panic", "恢复窗口置顶状态异常", "error")
					}
				}()
				time.Sleep(150 * time.Millisecond)
				if window != nil {
					window.SetAlwaysOnTop(false)
				}
			}()
		}
		return
	}
	window.SetAlwaysOnTop(alwaysOnTop)
	window.Focus()
	s.RecordLog("window", "窗口已显示")
}

func (s *Runtime) applyMainWindowAlwaysOnTop(alwaysOnTop bool) {
	s.lock.RLock()
	window := s.mainWindow
	s.lock.RUnlock()
	if window == nil {
		return
	}
	window.SetAlwaysOnTop(alwaysOnTop)
}

// QuitApp API 方法，退出应用。
func (api *API) QuitApp() (err error) {
	defer api.recoverError("退出应用", &err)
	api.runtime.QuitApp()
	return nil
}

// QuitApp 走显式退出路径：标记 forceQuit、记录 clean crash 状态，然后请求 Wails 退出。
func (s *Runtime) QuitApp() {
	s.lock.Lock()
	s.forceQuit = true
	app := s.wailsApp
	crashReporter := s.crashReporter
	s.lock.Unlock()
	if crashReporter != nil {
		crashReporter.MarkClean("Runtime.QuitApp")
	}
	s.RecordLog("app", "应用退出")
	if app != nil {
		app.Quit()
	}
}

// ShouldHideOnClose 判断窗口关闭事件是否应转为隐藏到托盘。
// 显式退出路径会先设置 forceQuit，因此不会被托盘策略拦截。
func (s *Runtime) ShouldHideOnClose() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.settings.MinimizeToTray && !s.forceQuit
}

// RecordSecondInstance 记录第二实例启动请求，最多保留最近 20 条。
// 第二实例来自桌面快捷方式时，也会写入启动来源日志。
func (s *Runtime) RecordSecondInstance(args []string, workingDir string) {
	record := SecondInstanceRecord{
		Args:       append([]string(nil), args...),
		WorkingDir: workingDir,
		ReceivedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.lock.Lock()
	s.secondStart = append([]SecondInstanceRecord{record}, s.secondStart...)
	if len(s.secondStart) > 20 {
		s.secondStart = s.secondStart[:20]
	}
	s.lock.Unlock()
	s.RecordLog("single-instance", "收到第二实例启动请求")
	s.recordStartupLaunch(ParseStartupLaunch(args), "：第二实例")
}

// RecordStartupLaunch 记录本次启动来源。
func (s *Runtime) RecordStartupLaunch(launch StartupLaunch) {
	s.recordStartupLaunch(launch, "")
}

func (s *Runtime) recordStartupLaunch(launch StartupLaunch, suffix string) {
	if !launch.DesktopShortcut {
		return
	}
	message := "桌面快捷图标启动" + suffix
	s.recordCrashBreadcrumb("startup", "%s", message)
	s.RecordLog("startup", message)
}

// GetSecondInstanceRecords API 方法，获取第二实例记录。
func (api *API) GetSecondInstanceRecords() (records []SecondInstanceRecord, err error) {
	defer api.recoverError("读取第二实例记录", &err)
	if err := api.requireAuthorized(); err != nil {
		return nil, err
	}
	return api.runtime.GetSecondInstanceRecords(), nil
}

// GetSecondInstanceRecords 返回第二实例记录快照，调用方修改返回切片不会影响 Runtime 内部状态。
func (s *Runtime) GetSecondInstanceRecords() []SecondInstanceRecord {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return append([]SecondInstanceRecord(nil), s.secondStart...)
}

// ParseExitRequest 解析命令行退出请求。
// 支持 -exit、-force-exit 和安装器使用的 -installer-exit。
func ParseExitRequest(args []string) ExitRequest {
	for _, arg := range args {
		normalised := normaliseArg(arg)
		switch normalised {
		case "-exit":
			return ExitRequest{Present: true, Source: normalised}
		case "-force-exit", "-installer-exit":
			return ExitRequest{Present: true, Force: true, Source: normalised}
		}
	}
	return ExitRequest{}
}

// ParseStartupLaunch 解析启动行为参数。
// 识别 --startup、--startup-hidden 和 --desktop-shortcut；normaliseArg 会兼容单横线形式。
func ParseStartupLaunch(args []string) StartupLaunch {
	var launch StartupLaunch
	for _, arg := range args {
		switch normaliseArg(arg) {
		case "-startup":
			launch.Autostart = true
		case "-startup-hidden":
			launch.Autostart = true
			launch.Hidden = true
		case "-desktop-shortcut":
			launch.DesktopShortcut = true
		}
	}
	return launch
}

// ShouldHideDuringStartupLoading 判断自启加载期是否应临时隐藏主窗口。
// 它不受 LaunchHiddenToTray 影响；加载完成后是否继续隐藏由 ShouldStartHidden 决定。
func (s *Runtime) ShouldHideDuringStartupLoading(startup StartupLaunch) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.settings.AutoLaunch && startup.Autostart
}

// ShouldStartHidden 判断本次启动是否应默认隐藏主窗口。
// 只有设置开启开机自启、开启自启隐藏，并且本次启动参数包含 --startup-hidden 时才生效。
func (s *Runtime) ShouldStartHidden(startup StartupLaunch) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.settings.AutoLaunch && s.settings.LaunchHiddenToTray && startup.Hidden
}

// normaliseArg 把 "--xxx" 归一化为 "-xxx"，便于兼容 Wails 自启和手动传参。
func normaliseArg(arg string) string {
	arg = strings.TrimSpace(arg)
	if strings.HasPrefix(arg, "--") {
		return "-" + strings.TrimLeft(arg, "-")
	}
	return arg
}
