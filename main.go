// main.go 是桌面进程入口，负责装配 app facade、Wails 应用、窗口生命周期
// 和启动期更新/崩溃恢复流程。业务状态留在 app.Runtime，入口只编排依赖。

package main

import (
	"embed"
	_ "embed"
	"fmt"
	"log"
	"os"

	desktopapp "github.com/chencn/go-desktop/app"
	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// assets 是生产包使用的前端构建产物；开发模式由 Wails dev server 接管。
//
//go:embed all:frontend/dist
var assets embed.FS

// appIcon 是托盘图标资源，随二进制一起嵌入。
//
//go:embed build/appicon.png
var appIcon []byte

var (
	// appVersion 是发布链在构建期注入或从元数据读取的当前版本号。
	appVersion = metadata.DefaultVersion
	// licenseMode 和 licensePublicKey 由授权构建注入；空值表示授权关闭。
	licenseMode      = ""
	licensePublicKey = ""
)

// mainWindowWindowsTheme 定义 Windows 原生 DWM 外框色；这是操作系统窗口装饰，不受前端 CSS token 控制。
func mainWindowWindowsTheme() application.ThemeSettings {
	return application.ThemeSettings{
		DarkModeActive: &application.WindowTheme{
			BorderColour: application.NewRGBPtr(100, 116, 139),
		},
		DarkModeInactive: &application.WindowTheme{
			BorderColour: application.NewRGBPtr(71, 85, 105),
		},
		LightModeActive: &application.WindowTheme{
			BorderColour: application.NewRGBPtr(203, 213, 225),
		},
		LightModeInactive: &application.WindowTheme{
			BorderColour: application.NewRGBPtr(226, 232, 240),
		},
	}
}

// main 是命令入口，负责解析启动上下文、装配依赖并启动核心流程。
func main() {
	crashLogPath := desktopapp.DefaultCrashLogPath(metadata.AppName)
	crashStatePath := desktopapp.DefaultCrashStatePath(metadata.AppName)
	crashReporter, previousCrash, hasPreviousCrash := desktopapp.StartCrashReporter(crashLogPath, crashStatePath, os.Args)
	defer crashReporter.Finish("主入口")

	crashReporter.Phase("解析启动参数")
	initialExitRequest := desktopapp.ParseExitRequest(os.Args[1:])
	startupLaunch := desktopapp.ParseStartupLaunch(os.Args[1:])
	var mainWindow *application.WebviewWindow

	crashReporter.Phase("创建 Runtime")
	appRuntime := desktopapp.NewRuntime(desktopapp.ServiceOptions{
		AppName:          metadata.AppName,
		Version:          appVersion,
		Description:      metadata.Description,
		Repository:       metadata.RepositoryURL,
		DatabasePath:     desktopapp.DefaultDatabasePath(metadata.AppName),
		LogFilePath:      desktopapp.DefaultLogFilePath(metadata.AppName),
		CrashReporter:    crashReporter,
		CachePath:        desktopapp.DefaultCachePath(metadata.AppName),
		LicenseMode:      licenseMode,
		LicensePublicKey: licensePublicKey,
		ReleaseChecker: githubrelease.NewChecker(githubrelease.Config{
			Owner:          metadata.GitHubOwner,
			Repo:           metadata.GitHubRepo,
			CurrentVersion: appVersion,
			UserAgent:      metadata.UserAgent,
			APIVersion:     metadata.GitHubAPIVersion,
			AssetNames:     releaseAssetNames,
		}),
	})
	// crash.log 保留为早期崩溃兜底；启动时先把上次异常退出尾部导入正常日志，再裁剪 crash.log。
	appRuntime.RecordPreviousCrash(previousCrash, hasPreviousCrash, crashLogPath)
	crashReporter.TrimLog()
	appRuntime.RecordStartupLaunch(startupLaunch)

	crashReporter.Phase("安装进程日志捕获")
	appRuntime.InstallProcessLogCapture()
	defer appRuntime.Shutdown()

	crashReporter.Phase("创建 Wails 应用")
	wailsApp := application.New(application.Options{
		Name:        metadata.AppName,
		Description: metadata.Description,
		LogLevel:    desktopapp.SlogLevelFromLogLevel(appRuntime.SettingsSnapshot().LogLevel),
		OnShutdown: func() {
			crashReporter.Append("app", "Wails OnShutdown")
			appRuntime.RecordLogWithSeverity("app", "Wails OnShutdown", "warning")
		},
		PanicHandler: func(details *application.PanicDetails) {
			defer appRuntime.RecoverPanic("Wails panic handler")
			if details == nil {
				appRuntime.RecordLogWithSeverity("panic", "Wails panic：未提供 panic 详情", "error")
				return
			}
			appRuntime.RecordLogWithSeverity("panic", fmt.Sprintf("Wails panic：%s\n%s", details.Error, details.StackTrace), "error")
		},
		Services: []application.Service{
			// 只注册 app.API facade，避免 Wails 绑定暴露 internal 包路径。
			application.NewService(appRuntime.API()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: metadata.WindowsSingleInstanceID,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				defer appRuntime.RecoverPanic("第二实例启动回调")
				appRuntime.RecordSecondInstance(data.Args, data.WorkingDir)
				if desktopapp.ParseExitRequest(data.Args).Present {
					appRuntime.QuitApp()
					return
				}
				appRuntime.ShowMainWindow()
				if mainWindow != nil {
					mainWindow.EmitEvent("desktop:second-instance", data)
				}
			},
		},
		Windows: application.WindowsOptions{
			WndClass: metadata.WindowsWindowClass,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	appRuntime.SetApplication(wailsApp)

	if initialExitRequest.Present {
		crashReporter.Append("app", "启动参数请求退出：source=%s force=%t", initialExitRequest.Source, initialExitRequest.Force)
		appRuntime.RecordLogWithSeverity("app", fmt.Sprintf("启动参数请求退出：source=%s force=%t", initialExitRequest.Source, initialExitRequest.Force), "warning")
		crashReporter.MarkClean("启动参数请求退出")
		return
	}
	if status := appRuntime.InstallPendingUpdateOnStartup(); status.Status == "install_started" {
		crashReporter.Append("update", "启动期自动安装已启动，退出当前应用")
		crashReporter.MarkClean("启动期自动安装更新")
		return
	}

	crashReporter.Phase("启动期系统集成")
	appRuntime.ApplyStartupIntegrations()
	startLoadingHidden := appRuntime.ShouldHideDuringStartupLoading(startupLaunch)
	startHidden := appRuntime.ShouldStartHidden(startupLaunch)
	var splashWindow *application.WebviewWindow
	if !startLoadingHidden {
		crashReporter.Phase("创建启动加载窗口")
		// 主窗口保持 Hidden 到 runtime ready；splash 自身等 WebView 导航完成后再显示，避免先露出白底窗口。
		splashWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
			Name:             "splash",
			Title:            "",
			Width:            120,
			Height:           72,
			MinWidth:         120,
			MinHeight:        72,
			MaxWidth:         120,
			MaxHeight:        72,
			AlwaysOnTop:      true,
			DisableResize:    true,
			Frameless:        true,
			Hidden:           true,
			InitialPosition:  application.WindowCentered,
			BackgroundType:   application.BackgroundTypeTransparent,
			BackgroundColour: application.NewRGBA(0, 0, 0, 0),
			HTML:             appRuntime.SplashHTML(),
			Windows: application.WindowsWindow{
				DisableIcon:                       true,
				DisableFramelessWindowDecorations: true,
				HiddenOnTaskbar:                   true,
			},
		})
		splashWindow.OnWindowEvent(events.Windows.WebViewNavigationCompleted, func(_ *application.WindowEvent) {
			defer appRuntime.RecoverPanic("启动加载窗口导航完成钩子")
			if splashWindow != nil && !startHidden {
				splashWindow.Show()
			}
		})
	}
	crashReporter.Phase("创建主窗口")
	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "main",
		Title:           metadata.AppName,
		Width:           1024,
		Height:          768,
		MinWidth:        1024,
		MinHeight:       768,
		Frameless:       true,
		InitialPosition: application.WindowCentered,
		Hidden:          startLoadingHidden || startHidden || splashWindow != nil,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: false,
			CustomTheme:                       mainWindowWindowsTheme(),
			HiddenOnTaskbar:                   startHidden,
		},
		BackgroundColour: application.NewRGB(246, 248, 252),
		URL:              "/",
	})
	appRuntime.SetMainWindow(mainWindow)
	appRuntime.SetSplashWindow(splashWindow)

	mainWindow.OnWindowEvent(events.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
		defer appRuntime.RecoverPanic("窗口运行时就绪钩子")
		if startHidden {
			appRuntime.RecordLog("window", "窗口内容已加载，按自启隐藏策略保持托盘隐藏")
			return
		}
		// 不在此处显示主窗口：WindowRuntimeReady 只表示 Wails JS bridge 就绪，
		// 前端框架和启动数据可能尚未加载完成，提前显示会导致白屏。
		// 主窗口由前端 initialise 完成后调用 ShowMainWindow API 触发显示。
		appRuntime.RecordLog("window", "Wails 运行时就绪，等待前端初始化完成后显示主窗口")
	})

	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		defer appRuntime.RecoverPanic("窗口关闭钩子")
		if e == nil {
			appRuntime.RecordLogWithSeverity("window", "窗口关闭事件为空", "warning")
			return
		}
		if appRuntime.ShouldHideOnClose() {
			mainWindow.Hide()
			appRuntime.RecordLog("window", "窗口已隐藏到托盘")
			e.Cancel()
			return
		}
		crashReporter.MarkClean("窗口关闭")
	})

	systemTray := wailsApp.SystemTray.New()
	systemTray.SetIcon(appIcon)
	systemTray.SetTooltip(metadata.AppName)
	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("显示").OnClick(func(_ *application.Context) {
		defer appRuntime.RecoverPanic("托盘显示菜单")
		appRuntime.ShowMainWindow()
	})
	trayMenu.Add("退出").OnClick(func(_ *application.Context) {
		defer appRuntime.RecoverPanic("托盘退出菜单")
		appRuntime.QuitApp()
	})
	systemTray.SetMenu(trayMenu)
	systemTray.OnClick(func() {
		defer appRuntime.RecoverPanic("托盘点击")
		appRuntime.ShowMainWindow()
	})

	appRuntime.RecordLog("app", "应用启动")
	appRuntime.StartUpdateBackgroundTasks()

	crashReporter.Phase("运行 Wails")
	if err := wailsApp.Run(); err != nil {
		crashReporter.Append("app", "Wails 运行失败：%s", err)
		appRuntime.RecordLogWithSeverity("app", fmt.Sprintf("Wails 运行失败：%s", err), "error")
		log.Printf("Wails run failed: %v", err)
		return
	}
	crashReporter.Append("app", "Wails 主循环已返回")
	appRuntime.RecordLogWithSeverity("app", "Wails 主循环已返回", "warning")
	// Wails 主循环正常返回说明进程走完了退出流程；这里补标记，
	// 让 Finish 删除状态文件，避免下次启动把本次正常退出误报为上次异常结束。
	crashReporter.MarkClean("Wails 主循环返回")
}

// releaseAssetNames 保持 main.go 注入的更新检查器与 runtime 默认匹配同一组资产名。
func releaseAssetNames(version string) []string {
	return []string{
		metadata.WindowsInstallerAssetName(version),
		metadata.WindowsInstallerAssetNameWithoutV(version),
		metadata.WindowsSetupAssetName(version),
		metadata.WindowsSetupAssetNameWithoutV(version),
	}
}
