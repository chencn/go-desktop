// ============================================================================
// 文件: app/window.go
// 描述: 窗口管理模块
//
// 功能概述:
// - 主窗口显示和隐藏
// - 应用退出逻辑
// - 单实例处理
// - 关闭到托盘策略
// ============================================================================

package runtime

import (
	goruntime "runtime" // 运行时信息，获取操作系统类型
	"strings"           // 字符串处理
	"time"              // 时间包
)

// StartupLaunch 描述本次启动是否来自需要特殊窗口行为的启动入口。
type StartupLaunch struct {
	// Hidden 表示启动入口要求默认隐藏主窗口。
	Hidden bool // Hidden 保存 Hidden 对应的数据，供当前实体的调用方读取或持久化。

	// DesktopShortcut 表示本次启动来自应用创建的桌面快捷方式。
	DesktopShortcut bool // DesktopShortcut 保存 DesktopShortcut 对应的数据，供当前实体的调用方读取或持久化。
}

// ShowMainWindow API 方法，显示主窗口
func (api *API) ShowMainWindow() (err error) {
	defer api.recoverError("显示主窗口", &err)
	api.runtime.ShowMainWindow()
	return nil
}

// ShowMainWindow 显示主窗口
// 如果窗口最小化则恢复，如果未最大化则最大化
// Windows 平台会先设置置顶再取消，确保窗口可见
func (s *Runtime) ShowMainWindow() {
	s.lock.RLock()
	window := s.mainWindow
	s.lock.RUnlock()
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
	if goruntime.GOOS == "windows" {
		window.SetAlwaysOnTop(true)
		window.Focus()
		s.RecordLog("window", "窗口已显示")
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
		return
	}
	window.Focus()
	s.RecordLog("window", "窗口已显示")
}

// QuitApp API 方法，退出应用
func (api *API) QuitApp() (err error) {
	defer api.recoverError("退出应用", &err)
	api.runtime.QuitApp()
	return nil
}

// QuitApp 退出应用
// 设置强制退出标志，记录日志，然后调用 Wails 退出
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

// ShouldHideOnClose 判断关闭窗口时是否应隐藏到托盘
// 当设置为关闭到托盘且非强制退出时返回 true
// 返回:
//   - bool: 是否应隐藏
func (s *Runtime) ShouldHideOnClose() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.settings.MinimizeToTray && !s.forceQuit
}

// RecordSecondInstance 记录第二实例启动请求
// 当用户再次启动应用时，会记录启动参数和工作目录
// 最多保留 20 条记录
// 参数:
//   - args: 启动参数
//   - workingDir: 工作目录
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

// GetSecondInstanceRecords API 方法，获取第二实例记录
func (api *API) GetSecondInstanceRecords() (records []SecondInstanceRecord, err error) {
	defer api.recoverError("读取第二实例记录", &err)
	if err := api.requireAuthorized(); err != nil {
		return nil, err
	}
	return api.runtime.GetSecondInstanceRecords(), nil
}

// GetSecondInstanceRecords 读取、解析或归一化 管理主窗口、托盘窗口状态和桌面端生命周期 需要的数据，并把结果返回给调用方。
func (s *Runtime) GetSecondInstanceRecords() []SecondInstanceRecord {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return append([]SecondInstanceRecord(nil), s.secondStart...)
}

// ParseExitRequest 解析命令行退出请求
// 支持以下参数:
//   - "-exit": 正常退出
//   - "-force-exit" 或 "-installer-exit": 强制退出（安装器使用）
//
// 参数:
//   - args: 命令行参数
//
// 返回:
//   - ExitRequest: 退出请求信息
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
// 目前只识别 --startup-hidden，用于 Wails3 Autostart 启动时隐藏主窗口。
func ParseStartupLaunch(args []string) StartupLaunch {
	var launch StartupLaunch
	for _, arg := range args {
		switch normaliseArg(arg) {
		case "-startup-hidden":
			launch.Hidden = true
		case "-desktop-shortcut":
			launch.DesktopShortcut = true
		}
	}
	return launch
}

// ShouldStartHidden 判断本次启动是否应默认隐藏主窗口。
// 只有设置开启开机自启、开启自启隐藏，并且本次启动参数包含 --startup-hidden 时才生效。
func (s *Runtime) ShouldStartHidden(startup StartupLaunch) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.settings.AutoLaunch && s.settings.LaunchHiddenToTray && startup.Hidden
}

// normaliseArg 标准化命令行参数
// 将 "--xxx" 转换为 "-xxx"
// 参数:
//   - arg: 原始参数
//
// 返回:
//   - string: 标准化后的参数
func normaliseArg(arg string) string {
	arg = strings.TrimSpace(arg)
	if strings.HasPrefix(arg, "--") {
		return "-" + strings.TrimLeft(arg, "-")
	}
	return arg
}
