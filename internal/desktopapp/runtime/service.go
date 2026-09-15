// 文件职责：Runtime 的装配与生命周期。NewRuntime 打开数据库、加载设置并启动后台任务；
// Shutdown 按依赖顺序释放资源。状态字段定义见 runtime.go，DTO 见 types.go。

package runtime

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/display"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
	updater "github.com/chencn/go-desktop/internal/desktopapp/update"
	"github.com/chencn/go-desktop/internal/platform/paths"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewRuntime 创建新的运行时实例
// 参数:
//   - options: 初始化配置，空字段会使用默认值
//
// 返回:
//   - 初始化完成的 Runtime 实例
//
// 注意:
//   - 会自动打开数据库、加载设置
//   - 不会启动窗口，需要后续调用 SetMainWindow
func NewRuntime(options ServiceOptions) *Runtime {
	// 填充默认值
	if options.AppName == "" {
		options.AppName = metadata.AppName
	}
	if options.Version == "" {
		options.Version = metadata.DefaultVersion
	}
	if options.Repository == "" {
		options.Repository = metadata.RepositoryURL
	}
	if strings.TrimSpace(options.LocalUpdateBaseURL) == "" {
		options.LocalUpdateBaseURL = metadata.LocalUpdateBaseURL
	}
	if strings.TrimSpace(options.LocalUpdateManifestPath) == "" {
		options.LocalUpdateManifestPath = metadata.LocalUpdateManifestPath
	}
	logDirPath := strings.TrimSpace(options.LogDirPath)
	if logDirPath == "" && strings.TrimSpace(options.LogFilePath) != "" {
		logDirPath = filepath.Dir(options.LogFilePath)
	}
	logFilePattern := ""
	if logDirPath != "" {
		logFilePattern = filepath.Join(logDirPath, options.AppName+"-%Y-%m-%d.log")
	}

	// 创建运行时实例
	runtime := &Runtime{
		options:                 options,
		releaseChecker:          options.ReleaseChecker,
		startedAt:               time.Now().UTC(),
		databasePath:            options.DatabasePath,
		logDirPath:              logDirPath,
		logFilePattern:          logFilePattern,
		crashReporter:           options.CrashReporter,
		cachePath:               options.CachePath,
		settings:                defaultSettings(),
		displayPreferencesV2:    display.DefaultV2(),
		displayPreferences:      defaultDisplayPreferences(),
		logViewClearedAt:        map[string]time.Time{},
		updateState:             idleUpdateStatus(),
		licenseMode:             strings.TrimSpace(options.LicenseMode),
		licensePublicKey:        strings.TrimSpace(options.LicensePublicKey),
		licenseDeviceCode:       strings.TrimSpace(options.LicenseDeviceCode),
		licenseDeviceCodeSource: options.LicenseDeviceCodeSource,
	}

	// 设置默认缓存路径
	if runtime.cachePath == "" {
		runtime.cachePath = paths.DefaultCachePath(options.AppName)
	}

	// 创建默认更新管理器
	if options.UpdateManager != nil {
		runtime.updateManager = options.UpdateManager
	} else {
		runtime.updateManager = updater.NewManager(updater.Config{CacheDir: runtime.cachePath})
	}

	// 先初始化文件日志，再打开数据库和加载配置；早期错误也必须落到同一条日志管线。
	runtime.initRuntimeLogger()
	runtime.openStore()
	runtime.loadSettings()
	if runtime.logLevel != nil {
		runtime.logLevel.Set(SlogLevelFromLogLevel(runtime.SettingsSnapshot().LogLevel))
	}
	runtime.startLogRetentionCleanup()
	runtime.cleanupInstalledUpdateCache()
	runtime.loadDisplayPreferences()

	return runtime
}

// ============================================================================
// 公开方法
// ============================================================================

// API 获取 API 服务实例
// 返回的 API 实例会被 Wails 注册为前端可调用的服务
func (r *Runtime) API() *API {
	return &API{runtime: r}
}

// SetApplication 设置 Wails 应用实例
// 由 main.go 在创建应用后调用
// 参数:
//   - app: Wails 应用实例
func (s *Runtime) SetApplication(app *application.App) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.wailsApp = app
}

// SetMainWindow 设置主窗口实例
// 由 main.go 在创建窗口后调用
// 参数:
//   - window: 主窗口实例
func (s *Runtime) SetMainWindow(window *application.WebviewWindow) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.mainWindow = window
}

// SetSplashWindow 设置启动加载窗口实例。
// ShowMainWindow 首次调用时会自动关闭该窗口。
func (s *Runtime) SetSplashWindow(window *application.WebviewWindow) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.splashWindow = window
}

// Shutdown 关闭运行时，释放资源
// 会在应用退出时自动调用（通过 defer）
// 注意:
//   - 会关闭数据库连接
//   - 不会关闭窗口（由 Wails 处理）
func (s *Runtime) Shutdown() {
	s.lock.Lock()
	s.shuttingDown = true
	stopCleanup := s.logCleanupStop
	s.logCleanupStop = nil
	stopUpdateScheduler := s.updateSchedulerStop
	s.updateSchedulerStop = nil
	s.lock.Unlock()
	if stopCleanup != nil {
		stopCleanup()
	}
	if stopUpdateScheduler != nil {
		stopUpdateScheduler()
	}

	s.closeProcessLogCapture()

	s.lock.Lock()
	store := s.store
	s.store = nil
	s.lock.Unlock()
	if store != nil {
		_ = store.Close()
	}
	s.closeRuntimeLogger()
}
