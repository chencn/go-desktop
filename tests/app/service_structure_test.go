// 文件职责：验证 app facade、main.go 桌面入口和 runtime 存储/日志结构边界。

package app_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chencn/go-desktop/app"
)

// TestAppServiceFileStaysAsThinWailsFacade 验证 app/service.go 只保留 Wails facade 和启动装配，不承载后端实现。
func TestAppServiceFileStaysAsThinWailsFacade(t *testing.T) {
	source := readRootFile(t, "app", "service.go")

	for _, forbidden := range []string{
		"func (s *Runtime) SaveSettings(",
		"func (s *Runtime) DownloadUpdate(",
		"func (s *Runtime) QueryLogs(",
		"func (s *Runtime) ShowMainWindow(",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("app/service.go should stay as a thin Wails facade; move %q into a focused app/*.go file", forbidden)
		}
	}

	for _, required := range []string{
		"appruntime \"github.com/chencn/go-desktop/internal/desktopapp/runtime\"",
		"type ServiceOptions = appruntime.ServiceOptions",
		"type Runtime struct",
		"*appruntime.Runtime",
		"type API struct",
		"inner *appruntime.API",
		"func NewRuntime(",
		"return &Runtime{Runtime: appruntime.NewRuntime(options)}",
		"func (r *Runtime) API() *API",
		"return &API{inner: r.Runtime.API()}",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("app/service.go should keep Wails facade and runtime bootstrap contract %q", required)
		}
	}
}

func TestRuntimeAPIUsesAppPackageForWailsBindingIdentity(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   filepath.Join(t.TempDir(), "logs"),
	})
	defer runtimeService.Shutdown()

	apiType := reflect.TypeOf(runtimeService.API())
	if apiType == nil || apiType.Kind() != reflect.Pointer {
		t.Fatalf("app.Runtime.API() should return a pointer to app.API, got %v", apiType)
	}
	if got, want := apiType.Elem().PkgPath(), "github.com/chencn/go-desktop/app"; got != want {
		t.Fatalf("app.Runtime.API() must keep Wails binding identity stable: package = %q, want %q", got, want)
	}
	if got, want := apiType.Elem().Name(), "API"; got != want {
		t.Fatalf("app.Runtime.API() must return app.API: type = %q, want %q", got, want)
	}
}

func TestAppPackageStaysAsWailsFacade(t *testing.T) {
	appFiles := []string{
		readRootFile(t, "app", "service.go"),
		readRootFile(t, "app", "paths.go"),
	}
	source := strings.Join(appFiles, "\n")
	for _, forbidden := range []string{
		"modernc.org/sqlite",
		"github.com/go-ole/go-ole",
		"database/sql",
		"os/exec",
		"CREATE TABLE IF NOT EXISTS",
		"type dailyLogWriter struct",
		"type processStreamCapture struct",
		"type CrashReporter struct",
		"func settingsConfigDefinitions()",
		"func displayPreferenceConfigDefinitions()",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("app package must stay as Wails facade; move backend implementation %q into internal", forbidden)
		}
	}
}

// TestMainInstallsProcessLoggingAndDefaultLogFilePath 保护桌面入口的日志接线。
// Runtime 里有捕获实现还不够，main.go 必须传默认文件日志路径并安装进程级捕获。
func TestMainInstallsProcessLoggingAndDefaultLogFilePath(t *testing.T) {
	source := readRootFile(t, "main.go")

	for _, required := range []string{
		"LogFilePath:",
		"desktopapp.DefaultLogFilePath(metadata.AppName)",
		"CrashReporter:",
		"appRuntime.InstallProcessLogCapture()",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("main.go should wire process/file logging through desktop runtime: missing %q", required)
		}
	}
}

// TestMainInstallsEarliestCrashReporter 保护打包二进制在 Runtime/UI 尚未可用时也能落盘排障日志。
func TestMainInstallsEarliestCrashReporter(t *testing.T) {
	source := readRootFile(t, "main.go")
	pathsSource := readRootFile(t, "app", "paths.go")
	platformPathsSource := readRootFile(t, "internal", "platform", "paths", "paths.go")
	appFacadeSource := readRootFile(t, "app", "service.go")
	runtimeCrashSource := readRootFile(t, "internal", "desktopapp", "runtime", "crash_logging.go")
	internalCrashSource := readRootFile(t, "internal", "desktopapp", "crash", "reporter.go")

	for _, required := range []string{
		"desktopapp.StartCrashReporter(crashLogPath, crashStatePath, os.Args)",
		"defer crashReporter.Finish(\"主入口\")",
		"appRuntime.RecordPreviousCrash(previousCrash, hasPreviousCrash, crashLogPath)",
		"crashReporter.TrimLog()",
		"crashReporter.Phase(\"运行 Wails\")",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("main.go should install earliest crash reporter before Wails runtime: missing %q", required)
		}
	}
	importCrashIndex := strings.Index(source, "appRuntime.RecordPreviousCrash(previousCrash, hasPreviousCrash, crashLogPath)")
	trimCrashIndex := strings.Index(source, "crashReporter.TrimLog()")
	if importCrashIndex < 0 || trimCrashIndex < 0 || importCrashIndex > trimCrashIndex {
		t.Fatalf("main.go should import previous crash log before trimming crash.log: import=%d trim=%d", importCrashIndex, trimCrashIndex)
	}
	for _, required := range []string{
		"func DefaultCrashLogPath(appName string) string",
		"func DefaultCrashStatePath(appName string) string",
		"appruntime.DefaultCrashLogPath(appName)",
		"appruntime.DefaultCrashStatePath(appName)",
	} {
		if !strings.Contains(pathsSource, required) {
			t.Fatalf("app/paths.go should expose path facade wrappers: missing %q", required)
		}
	}
	for _, required := range []string{
		"func DefaultCrashLogPath(appName string) string",
		"func DefaultCrashStatePath(appName string) string",
		"DefaultLogDirPath(appName)",
	} {
		if !strings.Contains(platformPathsSource, required) {
			t.Fatalf("internal/platform/paths should own writable crash log paths: missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"func DefaultSettingsPath(appName string) string",
		"SettingsPath string",
		"settingsPath",
		"settings.json",
	} {
		if strings.Contains(source, forbidden) || strings.Contains(pathsSource, forbidden) || strings.Contains(platformPathsSource, forbidden) || strings.Contains(appFacadeSource, forbidden) {
			t.Fatalf("runtime environment info must not expose legacy settings file path: found %q", forbidden)
		}
	}
	for _, required := range []string{
		"type CrashState = appruntime.CrashState",
		"type CrashReporter = appruntime.CrashReporter",
		"func StartCrashReporter(",
		"return appruntime.StartCrashReporter(logPath, statePath, args)",
	} {
		if !strings.Contains(appFacadeSource, required) {
			t.Fatalf("app/service.go should expose crash facade wrappers: missing %q", required)
		}
	}
	for _, required := range []string{
		"type CrashState = crash.State",
		"type CrashReporter = crash.Reporter",
		"func StartCrashReporter(",
		"crash.StartReporter(logPath, statePath, args)",
		"func (s *Runtime) RecordPreviousCrash(",
	} {
		if !strings.Contains(runtimeCrashSource, required) {
			t.Fatalf("internal/desktopapp/runtime/crash_logging.go should import crash diagnostics into runtime logs: missing %q", required)
		}
	}
	for _, required := range []string{
		"type Reporter struct",
		"func StartReporter(",
		"func (r *Reporter) InstallRuntimeCrashOutput()",
		"func (r *Reporter) TrimLog()",
		"func TrimLogFile(",
		"debug.SetCrashOutput",
		"debug.SetTraceback(\"all\")",
		"func (r *Reporter) Finish(operation string)",
		"debug.Stack()",
		"panic(recovered)",
		"func PreviousLogTail(",
	} {
		if !strings.Contains(internalCrashSource, required) {
			t.Fatalf("internal/desktopapp/crash should persist crash diagnostics: missing %q", required)
		}
	}
}

// TestMainInstallsPanicHandler 保护 Wails panic 不走默认 fatal/os.Exit。
func TestMainInstallsPanicHandler(t *testing.T) {
	source := readRootFile(t, "main.go")

	for _, required := range []string{
		"PanicHandler: func(details *application.PanicDetails)",
		"appRuntime.RecordLogWithSeverity(\"panic\"",
		"defer appRuntime.RecoverPanic(\"Wails panic handler\")",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("main.go should install Wails panic handler: missing %q", required)
		}
	}
}

// TestMainRuntimeCallbacksRecoverPanic 保护 Wails 回调、托盘回调和窗口钩子内的 panic 只进日志，不扩散成进程退出。
func TestMainRuntimeCallbacksRecoverPanic(t *testing.T) {
	source := readRootFile(t, "main.go")

	for _, required := range []string{
		"defer appRuntime.RecoverPanic(\"第二实例启动回调\")",
		"defer appRuntime.RecoverPanic(\"窗口关闭钩子\")",
		"defer appRuntime.RecoverPanic(\"托盘显示菜单\")",
		"defer appRuntime.RecoverPanic(\"托盘退出菜单\")",
		"defer appRuntime.RecoverPanic(\"托盘点击\")",
		"appRuntime.RecordLogWithSeverity(\"app\", fmt.Sprintf(\"Wails 运行失败",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("main.go should recover runtime callback panic and log run errors: missing %q", required)
		}
	}

	if strings.Contains(source, "log.Fatal(err)") {
		t.Fatal("main.go should not use log.Fatal because it bypasses deferred Shutdown and file-log flush")
	}
}

// TestSettingsAPIsExposeErrorChannelAndRecoverPanic 保护设置 API 把服务端异常返回给前端，而不是让 Wails 默认 fatal。
func TestSettingsAPIsExposeErrorChannelAndRecoverPanic(t *testing.T) {
	settingsSource := readRootFile(t, "internal", "desktopapp", "runtime", "settings.go")
	displaySource := readRootFile(t, "internal", "desktopapp", "runtime", "display_preferences.go")
	apiSafetySource := readRootFile(t, "internal", "desktopapp", "runtime", "api_safety.go")

	for _, required := range []string{
		"func (api *API) SaveSettings(settings Settings) (saved Settings, err error)",
		"defer api.recoverError(\"保存设置\", &err)",
		"func (s *Runtime) SaveSettings(settings Settings) (Settings, error)",
	} {
		if !strings.Contains(settingsSource, required) {
			t.Fatalf("settings API should expose error channel and panic guard: missing %q", required)
		}
	}
	for _, required := range []string{
		"func (api *API) SaveDisplayPreferences(preferences DisplayPreferences) (saved DisplayPreferences, err error)",
		"defer api.recoverError(\"保存显示偏好\", &err)",
		"func (s *Runtime) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error)",
	} {
		if !strings.Contains(displaySource, required) {
			t.Fatalf("display preferences API should expose error channel and panic guard: missing %q", required)
		}
	}
	for _, required := range []string{
		"func (api *API) recoverError(operation string, errp *error)",
		"func (s *Runtime) RecoverPanic(operation string)",
		"debug.Stack()",
		"RecordLogWithSeverity(\"panic\"",
	} {
		if !strings.Contains(apiSafetySource, required) {
			t.Fatalf("api_safety.go should convert API panic to error: missing %q", required)
		}
	}
}

// TestLicenseAPIsStayBehindAppFacade 验证授权 API 通过 app facade 暴露，不泄漏 internal 类型。
func TestLicenseAPIsStayBehindAppFacade(t *testing.T) {
	source := readRootFile(t, "app", "service.go")

	for _, want := range []string{
		"type LicenseStatus struct",
		"func (r *Runtime) GetLicenseStatus() LicenseStatus",
		"func (r *Runtime) ActivateLicense(licenseKey string) (LicenseStatus, error)",
		"func (api *API) GetLicenseStatus() (LicenseStatus, error)",
		"func (api *API) ActivateLicense(licenseKey string) (LicenseStatus, error)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("app facade 必须暴露授权 API 且不泄漏 internal 类型：缺少 %q", want)
		}
	}
}

// TestWindowLifecycleRecordsTroubleshootingLogs 保护窗口行为日志覆盖。
// 托盘隐藏和恢复显示都是用户排查窗口状态时需要看到的关键行为。
func TestWindowLifecycleRecordsTroubleshootingLogs(t *testing.T) {
	mainSource := readRootFile(t, "main.go")
	windowSource := readRootFile(t, "internal", "desktopapp", "runtime", "window.go")

	for _, required := range []string{
		`appRuntime.RecordLog("window", "窗口已隐藏到托盘")`,
		`appRuntime.ShowMainWindow()`,
		`RecordLog("window", "窗口已显示")`,
	} {
		if !strings.Contains(mainSource+windowSource, required) {
			t.Fatalf("window lifecycle should record troubleshooting log %q", required)
		}
	}

	if strings.Contains(mainSource, "func showMainWindow(") {
		t.Fatal("main.go should route window display through Runtime.ShowMainWindow so display logs are not bypassed")
	}
}

// TestStartupWindowLifecycleKeepsMainWindowHiddenUntilFrontendReady 保护启动窗口顺序：
// 主窗口必须先 Hidden 创建，不能被 Windows StartState 提前显示；Wails runtime ready 也不能直接切主窗口。
func TestStartupWindowLifecycleKeepsMainWindowHiddenUntilFrontendReady(t *testing.T) {
	source := readRootFile(t, "main.go")

	start := strings.Index(source, `Name:            "main"`)
	end := strings.Index(source, `appRuntime.SetMainWindow(mainWindow)`)
	if start < 0 || end < 0 || end <= start {
		t.Fatal("main.go 缺少可检查的主窗口创建结构")
	}
	mainWindowBlock := source[start:end]
	for _, required := range []string{
		"Hidden:          startLoadingHidden || startHidden || splashWindow != nil",
		"InitialPosition: application.WindowCentered",
	} {
		if !strings.Contains(mainWindowBlock, required) {
			t.Fatalf("主窗口启动可见性合同缺少 %q", required)
		}
	}
	if strings.Contains(mainWindowBlock, "StartState") {
		t.Fatal("主窗口启动时不能设置 StartState；Windows SW_MAXIMIZE 会覆盖 Hidden:true 并提前露出白屏")
	}

	readyStart := strings.Index(source, "events.Common.WindowRuntimeReady")
	if readyStart < 0 {
		t.Fatal("main.go 缺少 WindowRuntimeReady 事件注册")
	}
	readyBlock := source[readyStart:]
	if closingHook := strings.Index(readyBlock, "WindowClosing"); closingHook > 0 {
		readyBlock = readyBlock[:closingHook]
	}
	for _, forbidden := range []string{
		"appRuntime.ShowMainWindow()",
		"splashWindow.Close()",
	} {
		if strings.Contains(readyBlock, forbidden) {
			t.Fatalf("WindowRuntimeReady 不能执行 %q；它只表示 Wails bridge 就绪，不代表前端首帧已渲染", forbidden)
		}
	}
}

// TestMainWindowKeepsWindowsFramelessDecorations 验证主窗口保留 Windows 原生 DWM 外框、阴影和圆角。
func TestMainWindowKeepsWindowsFramelessDecorations(t *testing.T) {
	source := readRootFile(t, "main.go")

	start := strings.Index(source, `Name:            "main"`)
	end := strings.Index(source, `appRuntime.SetMainWindow(mainWindow)`)
	if start < 0 || end < 0 || end <= start {
		t.Fatal("main.go 缺少可检查的主窗口创建结构")
	}
	mainWindowBlock := source[start:end]
	for _, required := range []string{
		"Frameless:       true",
		"DisableFramelessWindowDecorations: false",
		"CustomTheme:                       mainWindowWindowsTheme()",
		"BorderColour: application.NewRGBPtr(100, 116, 139)",
		"BorderColour: application.NewRGBPtr(203, 213, 225)",
		"Windows 原生 DWM 外框色",
	} {
		if !strings.Contains(mainWindowBlock+source, required) {
			t.Fatalf("主窗口 Windows 原生边界合同缺少 %q", required)
		}
	}
}

// TestRuntimeShowMainWindowOwnsSplashHandoff 保护 splash 到主窗口的交接点集中在 Runtime.ShowMainWindow。
func TestRuntimeShowMainWindowOwnsSplashHandoff(t *testing.T) {
	mainSource := readRootFile(t, "main.go")
	serviceSource := readRootFile(t, "internal", "desktopapp", "runtime", "service.go")
	windowSource := readRootFile(t, "internal", "desktopapp", "runtime", "window.go")

	mainWindowIdx := strings.Index(mainSource, "appRuntime.SetMainWindow(mainWindow)")
	splashWindowIdx := strings.Index(mainSource, "appRuntime.SetSplashWindow(splashWindow)")
	if mainWindowIdx < 0 || splashWindowIdx < 0 {
		t.Fatal("main.go 必须同时注册 mainWindow 和 splashWindow")
	}
	if splashWindowIdx < mainWindowIdx {
		t.Fatal("appRuntime.SetSplashWindow 必须在 appRuntime.SetMainWindow 之后调用")
	}

	for _, required := range []string{
		"splashWindow *application.WebviewWindow",
		"func (s *Runtime) SetSplashWindow(window *application.WebviewWindow)",
		"s.splashWindow = window",
	} {
		if !strings.Contains(serviceSource, required) {
			t.Fatalf("Runtime 必须保存 splashWindow 引用：缺少 %q", required)
		}
	}

	start := strings.Index(windowSource, "func (s *Runtime) ShowMainWindow()")
	if start < 0 {
		t.Fatal("window.go 缺少 Runtime.ShowMainWindow")
	}
	methodBody := windowSource[start:]
	if nextFunc := strings.Index(methodBody[1:], "\nfunc "); nextFunc > 0 {
		methodBody = methodBody[:nextFunc+1]
	}
	for _, required := range []string{
		"s.lock.Lock()",
		"splash := s.splashWindow",
		"s.splashWindow = nil",
		"window.Show()",
		"splash.Close()",
	} {
		if !strings.Contains(methodBody, required) {
			t.Fatalf("Runtime.ShowMainWindow 必须原子接管 splash 交接：缺少 %q", required)
		}
	}
	if strings.Contains(methodBody, "s.lock.RLock()") {
		t.Fatal("Runtime.ShowMainWindow 需要写锁清空 splashWindow，不能使用 RLock")
	}
	if strings.Index(methodBody, "splash.Close()") < strings.Index(methodBody, "window.Show()") {
		t.Fatal("splash.Close() 必须在 window.Show() 之后，避免 loading 到主窗口之间出现空白")
	}
}

// TestRuntimeDoesNotUseSQLiteForLogsOrUpdateBusinessData 验证日志和更新业务数据不再写入 SQLite。
func TestRuntimeDoesNotUseSQLiteForLogsOrUpdateBusinessData(t *testing.T) {
	runtimeSources := strings.Join([]string{
		readRootFile(t, "internal", "desktopapp", "runtime", "logs.go"),
		readRootFile(t, "internal", "desktopapp", "runtime", "logging.go"),
		readRootFile(t, "internal", "desktopapp", "runtime", "process_logging.go"),
		readRootFile(t, "internal", "desktopapp", "runtime", "storage.go"),
		readRootFile(t, "internal", "desktopapp", "runtime", "update.go"),
	}, "\n")
	storageSources := strings.Join([]string{
		readRootFile(t, "internal", "adapters", "configstore", "migrations.go"),
		readRootFile(t, "internal", "adapters", "configstore", "models.go"),
		readRootFile(t, "internal", "adapters", "configstore", "sqlite.go"),
	}, "\n")

	for _, forbidden := range []string{
		"CREATE TABLE IF NOT EXISTS " + "logs",
		"CREATE TABLE IF NOT EXISTS " + "release_" + "audits",
		"CREATE TABLE IF NOT EXISTS " + "update_" + "events",
		"type Log" + "Entry struct",
		"type Update" + "Event struct",
		"func (s *Store) Insert" + "Log",
		"func (s *Store) Query" + "Logs",
		"func (s *Store) Clear" + "Logs",
		"func (s *Store) Prune" + "Logs",
		"func (s *Store) Insert" + "Audit",
		"func (s *Store) List" + "Audits",
		"func (s *Store) Insert" + "Update" + "Event",
		"func (s *Store) List" + "Update" + "Events",
		"func (s *Store) Latest" + "PendingInstall",
	} {
		if strings.Contains(storageSources, forbidden) {
			t.Fatalf("SQLite must stay config-only; found forbidden storage usage %q", forbidden)
		}
	}

	for _, forbidden := range []string{
		"store.Insert" + "Log(",
		"store.Query" + "Logs(",
		"store.Clear" + "Logs(",
		"store.Prune" + "Logs(",
		"store.Insert" + "Audit(",
		"store.List" + "Audits(",
		"store.Insert" + "Update" + "Event(",
		"store.List" + "Update" + "Events(",
		"store.Latest" + "PendingInstall(",
	} {
		if strings.Contains(runtimeSources, forbidden) {
			t.Fatalf("runtime must not call SQLite for logs/update business data: found %q", forbidden)
		}
	}

	for _, forbidden := range []string{
		"append(s.file" + "LogEntries(), memoryLogs...)",
		"file" + "LogEntries 读取所有每日 JSONL 日志文件",
		"内存、SQLite 和文件日志",
	} {
		if strings.Contains(runtimeSources, forbidden) {
			t.Fatalf("runtime logs must not merge file logs with memory logs or mention SQLite logging: found %q", forbidden)
		}
	}
}

func readRootFile(t *testing.T, parts ...string) string {
	t.Helper()
	path := rootPath(parts...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// rootPath 封装 tests/app 外部测试模块访问仓库根目录的路径规则，避免各测试重复拼接相对层级。
func rootPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}
