// ============================================================================
// 文件: main.go
// 描述: Go Desktop 应用程序的主入口文件
//
// 功能概述:
// - 初始化 Wails v3 桌面应用程序
// - 配置系统托盘、窗口管理、单实例控制
// - 集成自动更新检查、日志记录、设置持久化
// - 处理多平台特定行为（Windows/macOS/Linux）
// ============================================================================

package main

import (
	"embed"   // Go 1.16+ 嵌入文件到二进制
	_ "embed" // 空白导入，用于 embed 指令
	"fmt"     // 格式化 panic 日志
	"log"     // 标准日志库，用于运行失败输出
	"os"      // 操作系统接口，读取命令行参数

	// 项目内部包
	desktopapp "github.com/chencn/go-desktop/app"                  // 应用核心逻辑包
	"github.com/chencn/go-desktop/internal/adapters/githubrelease" // GitHub 版本检查模块
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"    // 项目元数据常量

	// Wails v3 框架
	"github.com/wailsapp/wails/v3/pkg/application" // Wails 应用主包
	"github.com/wailsapp/wails/v3/pkg/events"      // Wails 事件系统
)

// ============================================================================
// 嵌入资源
// ============================================================================

// assets 嵌入前端构建产物（HTML/CSS/JS）
// 开发模式下不使用嵌入，Wails 会启动独立的前端开发服务器
// 生产模式下，所有前端资源会被编译进 Go 二进制文件
//
//go:embed all:frontend/dist
var assets embed.FS

// appIcon 应用程序图标的 PNG 字节数据
// 用于系统托盘图标显示
//
//go:embed build/appicon.png
var appIcon []byte

// appVersion 从项目元数据读取的当前版本号
// 格式: semver (如 "1.0.0")
var (
	appVersion       = metadata.DefaultVersion
	licenseMode      = ""
	licensePublicKey = ""
)

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
	appRuntime.RecordPreviousCrash(previousCrash, hasPreviousCrash, crashLogPath)
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
	startHidden := appRuntime.ShouldStartHidden(startupLaunch)
	var splashWindow *application.WebviewWindow
	if !startHidden {
		crashReporter.Phase("创建启动加载窗口")
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
			InitialPosition:  application.WindowCentered,
			BackgroundType:   application.BackgroundTypeTransparent,
			BackgroundColour: application.NewRGBA(0, 0, 0, 0),
			HTML:             splashHTML(metadata.AppName),
			Windows: application.WindowsWindow{
				DisableIcon:                       true,
				DisableFramelessWindowDecorations: true,
				HiddenOnTaskbar:                   true,
			},
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
		StartState:      application.WindowStateMaximised,
		InitialPosition: application.WindowCentered,
		Hidden:          startHidden || splashWindow != nil,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: startHidden,
		},
		BackgroundColour: application.NewRGB(246, 248, 252),
		URL:              "/",
	})
	appRuntime.SetMainWindow(mainWindow)

	mainWindow.OnWindowEvent(events.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
		defer appRuntime.RecoverPanic("窗口运行时就绪钩子")
		if startHidden {
			appRuntime.RecordLog("window", "窗口内容已加载，按启动策略保持隐藏")
			return
		}
		appRuntime.ShowMainWindow()
		if splashWindow != nil {
			splashWindow.Close()
			splashWindow = nil
		}
		appRuntime.RecordLog("window", "主窗口内容已加载，启动加载窗口已关闭")
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
}

func splashHTML(_ string) string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <style>
      html,
      body {
        width: 100%;
        height: 100%;
        margin: 0;
        overflow: hidden;
        background: transparent;
        font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        -webkit-font-smoothing: antialiased;
        text-rendering: optimizeLegibility;
      }

      body {
        display: grid;
        place-items: center;
      }

      .splash {
        display: inline-flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 8px;
        width: 120px;
        height: 72px;
        color: #1890ff;
      }

      .ant-spin-dot {
        position: relative;
        display: inline-block;
        width: 32px;
        height: 32px;
        animation: ant-spin-rotate 1.2s linear infinite;
      }

      .ant-spin-dot-item {
        position: absolute;
        display: block;
        width: 14px;
        height: 14px;
        border-radius: 100%;
        background-color: currentColor;
        opacity: 0.3;
        animation: ant-spin-move 1s linear infinite alternate;
      }

      .ant-spin-dot-item:nth-child(1) {
        top: 0;
        left: 0;
      }

      .ant-spin-dot-item:nth-child(2) {
        top: 0;
        right: 0;
        animation-delay: 0.4s;
      }

      .ant-spin-dot-item:nth-child(3) {
        right: 0;
        bottom: 0;
        animation-delay: 0.8s;
      }

      .ant-spin-dot-item:nth-child(4) {
        bottom: 0;
        left: 0;
        animation-delay: 1.2s;
      }

      .ant-spin-text {
        color: rgba(0, 0, 0, 0.65);
        font-size: 14px;
        line-height: 1.5715;
        letter-spacing: 0;
        white-space: nowrap;
      }

      @keyframes ant-spin-rotate {
        to {
          transform: rotate(360deg);
        }
      }

      @keyframes ant-spin-move {
        to {
          opacity: 1;
          transform: scale(1);
        }
      }
    </style>
  </head>
  <body>
    <main class="splash" role="status" aria-live="polite">
      <span class="ant-spin-dot" aria-hidden="true">
        <i class="ant-spin-dot-item"></i>
        <i class="ant-spin-dot-item"></i>
        <i class="ant-spin-dot-item"></i>
        <i class="ant-spin-dot-item"></i>
      </span>
      <span class="ant-spin-text">正在加载</span>
    </main>
  </body>
</html>`
}

func releaseAssetNames(version string) []string {
	return []string{
		metadata.WindowsInstallerAssetName(version),
		metadata.WindowsInstallerAssetNameWithoutV(version),
		metadata.WindowsSetupAssetName(version),
		metadata.WindowsSetupAssetNameWithoutV(version),
	}
}
