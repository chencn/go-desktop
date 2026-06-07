// ============================================================================
// 文件: app/service.go
// 描述: 应用核心运行时服务，管理应用程序的生命周期和状态
//
// 功能概述:
// - 运行时初始化和配置管理
// - 应用信息与环境信息获取
// - 设置持久化和加载
// - 日志记录和当前更新状态跟踪
// - 更新状态管理
// - 多实例处理
// ============================================================================

package runtime

import (
	"context"  // 上下文接口，用于停止后台任务
	"log/slog" // 结构化日志框架
	"path/filepath"
	goruntime "runtime" // 运行时信息，获取操作系统和架构
	"strings"
	"sync" // 同步原语，保护并发访问
	"time" // 时间包，用于时间戳

	// 项目内部包
	"github.com/chencn/go-desktop/internal/adapters/configstore"      // 存储层（SQLite）
	"github.com/chencn/go-desktop/internal/adapters/filelog"          // 文件日志适配层
	"github.com/chencn/go-desktop/internal/adapters/githubrelease"    // GitHub 版本检查模块
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"       // 项目元数据常量
	updater "github.com/chencn/go-desktop/internal/desktopapp/update" // 更新管理器
	"github.com/chencn/go-desktop/internal/platform/paths"
	processutil "github.com/chencn/go-desktop/internal/platform/process"

	// Wails v3 框架
	"github.com/wailsapp/wails/v3/pkg/application" // Wails 应用主包
)

// ============================================================================
// 配置结构体
// ============================================================================

// ServiceOptions 定义 Runtime 的初始化配置
// 所有字段都是可选的，空值会使用默认值填充
// ServiceOptions 是运行时服务的配置选项
type ServiceOptions struct {
	// AppName 应用程序名称
	// 用于窗口标题、单实例标识、默认路径等
	AppName string // AppName 保存 AppName 对应的数据，供当前实体的调用方读取或持久化。

	// Version 当前应用版本号
	// 格式: semver (如 "1.0.0")，用于更新检查
	Version string // Version 保存 Version 对应的数据，供当前实体的调用方读取或持久化。

	// Description 应用描述
	// 显示在关于页面和元数据中
	Description string // Description 保存 Description 对应的数据，供当前实体的调用方读取或持久化。

	// Repository GitHub 仓库地址
	// 用于更新检查和关于页面链接
	Repository string // Repository 保存 Repository 对应的数据，供当前实体的调用方读取或持久化。

	// DatabasePath SQLite 数据库文件路径
	// 只存储 config_items 配置项
	DatabasePath string // DatabasePath 保存 DatabasePath 对应的数据，供当前实体的调用方读取或持久化。

	// LogDirPath 文件日志目录
	// 为空时可由 LogFilePath 兼容映射；桌面入口会传入默认路径所在目录
	LogDirPath string // LogDirPath 保存 LogDirPath 对应的数据，供当前实体的调用方读取或持久化。

	// LogFilePath 兼容旧调用的文件日志路径
	// 新实现只取其目录，实际日志文件按 appName-YYYY-MM-DD.log 写入
	LogFilePath string // LogFilePath 保存 LogFilePath 对应的数据，供当前实体的调用方读取或持久化。

	// CrashReporter 最早期崩溃日志器
	// main.go 在 Runtime 创建前安装，用于捕获 Runtime/Wails 尚未可用时的退出线索
	CrashReporter *CrashReporter // CrashReporter 保存 CrashReporter 对应的数据，供当前实体的调用方读取或持久化。

	// CachePath 缓存目录路径
	// 用于存储下载的更新包等临时文件
	CachePath string // CachePath 保存 CachePath 对应的数据，供当前实体的调用方读取或持久化。

	// ReleaseChecker GitHub 版本检查器实例
	// 如果为空，会自动创建默认实例
	ReleaseChecker *githubrelease.Checker // ReleaseChecker 保存 ReleaseChecker 对应的数据，供当前实体的调用方读取或持久化。

	// LocalUpdateBaseURL 本地静态升级根地址
	// 为空时使用项目元数据默认值
	LocalUpdateBaseURL string

	// LocalUpdateManifestPath 本地 latest.json 相对路径
	// 为空时使用项目元数据默认值
	LocalUpdateManifestPath string

	// UpdateManager 更新管理器实例
	// 如果为空，会自动创建默认实例
	UpdateManager *updater.Manager // UpdateManager 保存 UpdateManager 对应的数据，供当前实体的调用方读取或持久化。

	// StartupIntegrationApplier 可选的启动集成同步函数，用于替换默认平台集成实现。
	StartupIntegrationApplier func(previous Settings, next Settings) error
}

// ============================================================================
// 核心运行时结构体
// ============================================================================

// Runtime 是应用的核心运行时，管理所有应用状态和服务
// 线程安全：所有公开方法都使用锁保护并发访问
type Runtime struct {
	// options 初始化配置（只读，初始化后不变）
	options ServiceOptions // options 保存 options 对应的数据，供当前实体的调用方读取或持久化。

	// releaseChecker GitHub 版本检查器
	// 用于检查是否有新版本发布
	releaseChecker *githubrelease.Checker // releaseChecker 保存 releaseChecker 对应的数据，供当前实体的调用方读取或持久化。

	// startedAt 应用启动时间（UTC）
	// 用于计算运行时长和日志时间戳
	startedAt time.Time // startedAt 保存 startedAt 对应的数据，供当前实体的调用方读取或持久化。

	// databasePath 数据库文件路径
	databasePath string // databasePath 保存 databasePath 对应的数据，供当前实体的调用方读取或持久化。

	// logger 是应用统一结构化日志入口。
	logger *slog.Logger // logger 保存 logger 对应的数据，供当前实体的调用方读取或持久化。

	// shuttingDown 表示运行时正在释放资源，禁止重新打开日志文件。
	shuttingDown bool

	// logLevel 允许设置变更时动态调整 slog 过滤级别。
	logLevel *slog.LevelVar // logLevel 保存 logLevel 对应的数据，供当前实体的调用方读取或持久化。

	// logWriter 是每日文件 writer，只负责按日期切文件。
	logWriter *filelog.DailyWriter // logWriter 保存 logWriter 对应的数据，供当前实体的调用方读取或持久化。

	// logDirPath 文件日志目录。
	logDirPath string // logDirPath 保存 logDirPath 对应的数据，供当前实体的调用方读取或持久化。

	// logFilePattern 描述每日文件命名规则，便于环境信息展示和排查。
	logFilePattern string // logFilePattern 保存 logFilePattern 对应的数据，供当前实体的调用方读取或持久化。

	// logCleanupStop 停止启动期日志保留清理任务。
	logCleanupStop context.CancelFunc // logCleanupStop 保存 logCleanupStop 对应的数据，供当前实体的调用方读取或持久化。

	// crashReporter 最早期崩溃日志器
	// error/panic 和 Wails 生命周期异常会同步写入，避免 Runtime 日志不可用时丢线索
	crashReporter *CrashReporter // crashReporter 保存 crashReporter 对应的数据，供当前实体的调用方读取或持久化。

	// processCapture 当前 stdout/stderr 捕获器
	// 用于把非 log/slog 的进程标准流输出也接入日志页
	processCapture *processutil.StreamCapture // processCapture 保存 processCapture 对应的数据，供当前实体的调用方读取或持久化。

	// processLogRestore 记录安装进程日志捕获前的全局 log/slog 状态
	// Shutdown 会用它恢复标准库日志和结构化日志，避免留下跨运行时副作用
	processLogRestore *processLogRestore // processLogRestore 保存 processLogRestore 对应的数据，供当前实体的调用方读取或持久化。

	// cachePath 缓存目录路径
	cachePath string // cachePath 保存 cachePath 对应的数据，供当前实体的调用方读取或持久化。

	// store SQLite 数据库实例
	// 只用于持久化配置项
	store *configstore.Store // store 保存 store 对应的数据，供当前实体的调用方读取或持久化。

	// configDefaultsEnsured 标记当前 store 是否已经写入过配置默认项元数据
	configDefaultsEnsured bool // configDefaultsEnsured 避免每次设置保存都重复刷新全部默认配置项。

	// updateManager 更新管理器
	// 处理更新下载、安装等操作
	updateManager *updater.Manager // updateManager 保存 updateManager 对应的数据，供当前实体的调用方读取或持久化。

	// updateOperationLock 串行化下载、安排和安装，避免同一安装包临时文件并发写入。
	updateOperationLock sync.Mutex

	// lock 读写锁，保护并发访问
	// 读操作使用 RLock，写操作使用 Lock
	lock sync.RWMutex // lock 保存 lock 对应的数据，供当前实体的调用方读取或持久化。

	// wailsApp Wails 应用实例
	// 用于访问窗口、托盘等 GUI 功能
	wailsApp *application.App // wailsApp 保存 wailsApp 对应的数据，供当前实体的调用方读取或持久化。

	// mainWindow 主窗口实例
	// 用于窗口操作（显示、隐藏、事件发送等）
	mainWindow *application.WebviewWindow // mainWindow 保存 mainWindow 对应的数据，供当前实体的调用方读取或持久化。

	// settings 当前应用设置
	// 包含 GitHub 配置、更新间隔、日志保留等
	settings Settings // settings 保存 settings 对应的数据，供当前实体的调用方读取或持久化。

	// updateSchedulerStop 停止后台更新检查任务。
	updateSchedulerStop context.CancelFunc

	// displayPreferences 当前显示偏好
	// 由 SQLite KV 配置项加载，前端只通过 typed facade 读取和保存
	displayPreferences DisplayPreferences // displayPreferences 保存 displayPreferences 对应的数据，供当前实体的调用方读取或持久化。

	// logs 内存中的日志条目
	// 用于当前前端视图和文件不可读时兜底
	logs []LogEntry // logs 保存 logs 对应的数据，供当前实体的调用方读取或持久化。

	// logViewClearedAt 记录当前前端视图的清空时间，不删除文件日志。
	logViewClearedAt map[string]time.Time // logViewClearedAt 保存 logViewClearedAt 对应的数据，供当前实体的调用方读取或持久化。

	// latestUpdateCheck 当前进程最近一次更新检查结果。
	latestUpdateCheck githubrelease.CheckResult // latestUpdateCheck 保存 latestUpdateCheck 对应的数据，供当前实体的调用方读取或持久化。

	// hasUpdateCheck 标记 latestUpdateCheck 是否可用；重启后不从数据库恢复。
	hasUpdateCheck bool // hasUpdateCheck 保存 hasUpdateCheck 对应的数据，供当前实体的调用方读取或持久化。

	// updateState 当前更新状态
	// 反映更新下载/安装的进度和结果
	updateState UpdateStatus // updateState 保存 updateState 对应的数据，供当前实体的调用方读取或持久化。

	// forceQuit 强制退出标志
	// 为 true 时，关闭窗口直接退出而不是隐藏到托盘
	forceQuit bool // forceQuit 保存 forceQuit 对应的数据，供当前实体的调用方读取或持久化。

	// secondStart 多实例启动记录
	// 当用户第二次启动应用时记录参数
	secondStart []SecondInstanceRecord // secondStart 保存 secondStart 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// API 结构体
// ============================================================================

// API 是前端调用的服务接口
// 所有方法都会被 Wails 自动暴露给前端
type API struct {
	// runtime 运行时实例引用
	runtime *Runtime // runtime 保存 runtime 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 应用信息结构体
// ============================================================================

// AppInfo 应用基本信息
// 前端调用 GetAppInfo() 获取
type AppInfo struct {
	// Name 应用名称
	Name string `json:"name"` // Name 保存 name 对应的数据，供当前实体的调用方读取或持久化。

	// Version 当前版本号
	Version string `json:"version"` // Version 保存 version 对应的数据，供当前实体的调用方读取或持久化。

	// Description 应用描述
	Description string `json:"description"` // Description 保存 description 对应的数据，供当前实体的调用方读取或持久化。

	// Repository GitHub 仓库地址
	Repository string `json:"repository"` // Repository 保存 repository 对应的数据，供当前实体的调用方读取或持久化。

	// StartedAt 启动时间（ISO 8601 格式）
	StartedAt string `json:"startedAt"` // StartedAt 保存 startedAt 对应的数据，供当前实体的调用方读取或持久化。
}

// EnvironmentInfo 环境信息
// 前端调用 GetEnvironmentInfo() 获取
type EnvironmentInfo struct {
	// OS 操作系统（如 "windows", "darwin", "linux"）
	OS string `json:"os"` // OS 保存 os 对应的数据，供当前实体的调用方读取或持久化。

	// Arch CPU 架构（如 "amd64", "arm64"）
	Arch string `json:"arch"` // Arch 保存 arch 对应的数据，供当前实体的调用方读取或持久化。

	// GoVersion Go 运行时版本
	GoVersion string `json:"goVersion"` // GoVersion 保存 goVersion 对应的数据，供当前实体的调用方读取或持久化。

	// WailsVersion Wails 框架版本
	WailsVersion string `json:"wailsVersion"` // WailsVersion 保存 wailsVersion 对应的数据，供当前实体的调用方读取或持久化。

	// DatabasePath 数据库文件路径
	DatabasePath string `json:"databasePath"` // DatabasePath 保存 databasePath 对应的数据，供当前实体的调用方读取或持久化。

	// DatabaseReady SQLite 配置库是否已打开并完成默认配置初始化
	DatabaseReady bool `json:"databaseReady"` // DatabaseReady 保存 databaseReady 对应的数据，供当前实体的调用方读取或持久化。

	// DatabaseStatus SQLite 配置库状态，供首页监控直接消费。
	DatabaseStatus string `json:"databaseStatus"` // DatabaseStatus 保存 databaseStatus 对应的数据，供当前实体的调用方读取或持久化。

	// DatabaseMessage SQLite 配置库状态说明。
	DatabaseMessage string `json:"databaseMessage"` // DatabaseMessage 保存 databaseMessage 对应的数据，供当前实体的调用方读取或持久化。

	// LogFilePath 文件日志路径
	LogFilePath string `json:"logFilePath"` // LogFilePath 保存 logFilePath 对应的数据，供当前实体的调用方读取或持久化。

	// CachePath 缓存目录路径
	CachePath string `json:"cachePath"` // CachePath 保存 cachePath 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 设置结构体
// ============================================================================

// Settings 应用设置
// 所有字段都可通过前端修改并持久化
type Settings struct {
	// UpdateSource 更新源：github 或 local
	// 决定检查更新时只访问 GitHub Release 还是本地静态 manifest
	UpdateSource string `json:"updateSource"` // UpdateSource 保存 updateSource 对应的数据，供当前实体的调用方读取或持久化。

	// GitHubOwner GitHub 仓库所有者（用户名或组织名）
	// 用于检查更新时的 API 请求
	GitHubOwner string `json:"githubOwner"` // GitHubOwner 保存 githubOwner 对应的数据，供当前实体的调用方读取或持久化。

	// GitHubRepo GitHub 仓库名称
	GitHubRepo string `json:"githubRepo"` // GitHubRepo 保存 githubRepo 对应的数据，供当前实体的调用方读取或持久化。

	// GitHubProxyBase GitHub API 代理地址
	// 用于国内加速，为空则使用官方 API
	GitHubProxyBase string `json:"githubProxyBase"` // GitHubProxyBase 保存 githubProxyBase 对应的数据，供当前实体的调用方读取或持久化。

	// UpdateCheckIntervalHours 自动更新检查间隔（小时）
	// 0 或负数表示禁用自动检查
	UpdateCheckIntervalHours int `json:"updateCheckIntervalHours"` // UpdateCheckIntervalHours 保存 updateCheckIntervalHours 对应的数据，供当前实体的调用方读取或持久化。

	// MinimizeToTray 关闭窗口时隐藏到系统托盘
	// true: 点击关闭时隐藏窗口到托盘
	// false: 点击关闭时直接关闭窗口
	MinimizeToTray bool `json:"minimizeToTray"` // MinimizeToTray 保存 minimizeToTray 对应的数据，供当前实体的调用方读取或持久化。

	// LogRetentionDays 日志保留天数
	// -1 表示永久保留
	// 0 使用默认值（通常 30 天）
	LogRetentionDays int `json:"logRetentionDays"` // LogRetentionDays 保存 logRetentionDays 对应的数据，供当前实体的调用方读取或持久化。

	// LogLevel 最小记录日志级别
	// 支持 debug、info、warning、error；error/panic 永远优先保留。
	LogLevel string `json:"logLevel"` // LogLevel 保存 logLevel 对应的数据，供当前实体的调用方读取或持久化。

	// AutoLaunch 开机自启
	// true 时 Runtime 会通过 Wails3 Autostart 注册当前应用
	AutoLaunch bool `json:"autoLaunch"` // AutoLaunch 保存 autoLaunch 对应的数据，供当前实体的调用方读取或持久化。

	// CreateDesktopShortcut 创建桌面快捷图标
	// true 时 Runtime 会在当前用户桌面创建应用快捷方式
	CreateDesktopShortcut bool `json:"createDesktopShortcut"` // CreateDesktopShortcut 保存 createDesktopShortcut 对应的数据，供当前实体的调用方读取或持久化。

	// LaunchHiddenToTray 开机自启时隐藏到托盘
	// 仅在 AutoLaunch 开启且启动参数来自自启入口时生效
	LaunchHiddenToTray bool `json:"launchHiddenToTray"` // LaunchHiddenToTray 保存 launchHiddenToTray 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 日志结构体
// ============================================================================

// LogEntry 单条日志记录
type LogEntry struct {
	// Time 日志时间（ISO 8601 格式，UTC）
	Time string `json:"time"` // Time 保存 time 对应的数据，供当前实体的调用方读取或持久化。

	// Scope 日志作用域（如 "app", "window", "storage"）
	// 用于日志分类和过滤
	Scope string `json:"scope"` // Scope 保存 scope 对应的数据，供当前实体的调用方读取或持久化。

	// Message 日志内容
	Message string `json:"message"` // Message 保存 message 对应的数据，供当前实体的调用方读取或持久化。

	// Severity 日志级别（"info", "warning", "error"）
	Severity string `json:"severity"` // Severity 保存 severity 对应的数据，供当前实体的调用方读取或持久化。
}

// LogFileInfo 描述一个可在前端选择的每日文件日志。
type LogFileInfo struct {
	// Date 日志文件日期，格式为 YYYY-MM-DD。
	Date string `json:"date"`

	// FileName 日志文件名，不包含目录。
	FileName string `json:"fileName"`

	// FilePath 日志文件完整路径。
	FilePath string `json:"filePath"`

	// SizeBytes 文件大小。
	SizeBytes int64 `json:"sizeBytes"`

	// ModifiedAt 文件最后修改时间。
	ModifiedAt string `json:"modifiedAt"`

	// Current 表示该文件是否为当前写入的每日文件。
	Current bool `json:"current"`
}

// LogQuery 日志查询参数
type LogQuery struct {
	// FileName 指定要读取的每日日志文件；为空时读取当前每日文件。
	FileName string `json:"fileName"`

	// Scope 按作用域过滤（空表示全部）
	Scope string `json:"scope"` // Scope 保存 scope 对应的数据，供当前实体的调用方读取或持久化。

	// Severity 按级别过滤（空表示全部）
	Severity string `json:"severity"` // Severity 保存 severity 对应的数据，供当前实体的调用方读取或持久化。

	// Keyword 按关键词搜索（匹配 Message 字段）
	Keyword string `json:"keyword"` // Keyword 保存 keyword 对应的数据，供当前实体的调用方读取或持久化。

	// Page 页码（从 1 开始）
	Page int `json:"page"` // Page 保存 page 对应的数据，供当前实体的调用方读取或持久化。

	// PageSize 每页条数
	PageSize int `json:"pageSize"` // PageSize 保存 pageSize 对应的数据，供当前实体的调用方读取或持久化。
}

// LogStats 日志统计信息
type LogStats struct {
	// Total 总日志数
	Total int `json:"total"` // Total 保存 total 对应的数据，供当前实体的调用方读取或持久化。

	// Debug 调试级别日志数
	Debug int `json:"debug"` // Debug 保存 debug 对应的数据，供当前实体的调用方读取或持久化。

	// Info 信息级别日志数
	Info int `json:"info"` // Info 保存 info 对应的数据，供当前实体的调用方读取或持久化。

	// Warning 警告级别日志数
	Warning int `json:"warning"` // Warning 保存 warning 对应的数据，供当前实体的调用方读取或持久化。

	// Error 错误级别日志数
	Error int `json:"error"` // Error 保存 error 对应的数据，供当前实体的调用方读取或持久化。
}

// LogResponse 日志查询响应
type LogResponse struct {
	// Logs 当前页的日志列表
	Logs []LogEntry `json:"logs"` // Logs 保存 logs 对应的数据，供当前实体的调用方读取或持久化。

	// Source 日志来源，file 表示每日文件，memory 表示文件不可用时的内存降级。
	Source string `json:"source"`

	// FileName 当前查询的文件名。
	FileName string `json:"fileName"`

	// FilePath 当前查询的文件路径。
	FilePath string `json:"filePath"`

	// Total 符合条件的总日志数
	Total int `json:"total"` // Total 保存 total 对应的数据，供当前实体的调用方读取或持久化。

	// Page 当前页码
	Page int `json:"page"` // Page 保存 page 对应的数据，供当前实体的调用方读取或持久化。

	// PageSize 每页条数
	PageSize int `json:"pageSize"` // PageSize 保存 pageSize 对应的数据，供当前实体的调用方读取或持久化。

	// HasMore 是否有下一页
	HasMore bool `json:"hasMore"` // HasMore 保存 hasMore 对应的数据，供当前实体的调用方读取或持久化。

	// Stats 日志统计信息
	Stats LogStats `json:"stats"` // Stats 保存 stats 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 更新状态结构体
// ============================================================================

// UpdateStatus 更新状态信息
// 前端通过轮询或事件获取此状态
type UpdateStatus struct {
	// Status 更新状态（"idle", "checking", "downloading", "installing", "completed", "error"）
	Status string `json:"status"` // Status 保存 status 对应的数据，供当前实体的调用方读取或持久化。

	// Message 状态描述（用户可读）
	Message string `json:"message"` // Message 保存 message 对应的数据，供当前实体的调用方读取或持久化。

	// Version 目标版本号（检查更新时）
	Version string `json:"version,omitempty"` // Version 保存 version 对应的数据，供当前实体的调用方读取或持久化。

	// AssetName 下载的文件名
	AssetName string `json:"assetName,omitempty"` // AssetName 保存 assetName 对应的数据，供当前实体的调用方读取或持久化。

	// FilePath 下载文件的本地路径
	FilePath string `json:"filePath,omitempty"` // FilePath 保存 filePath 对应的数据，供当前实体的调用方读取或持久化。

	// DownloadedBytes 已下载字节数
	DownloadedBytes int64 `json:"downloadedBytes,omitempty"` // DownloadedBytes 保存 downloadedBytes 对应的数据，供当前实体的调用方读取或持久化。

	// TotalBytes 总字节数
	TotalBytes int64 `json:"totalBytes,omitempty"` // TotalBytes 保存 totalBytes 对应的数据，供当前实体的调用方读取或持久化。

	// ProgressPercent 下载进度（0-100）
	ProgressPercent float64 `json:"progressPercent,omitempty"` // ProgressPercent 保存 progressPercent 对应的数据，供当前实体的调用方读取或持久化。

	// Sha256 文件 SHA256 校验和
	// 用于验证下载文件的完整性
	Sha256 string `json:"sha256,omitempty"` // Sha256 保存 sha256 对应的数据，供当前实体的调用方读取或持久化。

	// Verified 是否通过完整性验证
	Verified bool `json:"verified"` // Verified 保存 verified 对应的数据，供当前实体的调用方读取或持久化。

	// ErrorReason 错误原因（状态为 error 时）
	ErrorReason string `json:"errorReason,omitempty"` // ErrorReason 保存 errorReason 对应的数据，供当前实体的调用方读取或持久化。

	// Source 更新源：github 或 local
	Source string `json:"source,omitempty"` // Source 保存 source 对应的数据，供当前实体的调用方读取或持久化。

	// UpdatedAt 状态更新时间（ISO 8601 格式）
	UpdatedAt string `json:"updatedAt"` // UpdatedAt 保存 updatedAt 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 多实例结构体
// ============================================================================

// SecondInstanceRecord 多实例启动记录
// 当用户第二次启动应用时创建
type SecondInstanceRecord struct {
	// Args 命令行参数
	Args []string `json:"args"` // Args 保存 args 对应的数据，供当前实体的调用方读取或持久化。

	// WorkingDir 工作目录
	WorkingDir string `json:"workingDir"` // WorkingDir 保存 workingDir 对应的数据，供当前实体的调用方读取或持久化。

	// ReceivedAt 接收时间
	ReceivedAt string `json:"receivedAt"` // ReceivedAt 保存 receivedAt 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 退出请求结构体
// ============================================================================

// ExitRequest 退出请求信息
// 用于处理应用退出逻辑（正常退出、强制退出、更新退出等）
type ExitRequest struct {
	// Present 是否有退出请求
	Present bool // Present 保存 Present 对应的数据，供当前实体的调用方读取或持久化。

	// Force 是否强制退出（不保存状态）
	Force bool // Force 保存 Force 对应的数据，供当前实体的调用方读取或持久化。

	// Source 退出来源（如 "user", "update", "system"）
	Source string // Source 保存 Source 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 构造函数
// ============================================================================

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
		options:            options,
		releaseChecker:     options.ReleaseChecker,
		startedAt:          time.Now().UTC(),
		databasePath:       options.DatabasePath,
		logDirPath:         logDirPath,
		logFilePattern:     logFilePattern,
		crashReporter:      options.CrashReporter,
		cachePath:          options.CachePath,
		settings:           defaultSettings(),
		displayPreferences: defaultDisplayPreferences(),
		logViewClearedAt:   map[string]time.Time{},
		updateState:        idleUpdateStatus(),
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

// ============================================================================
// API 方法（前端可调用）
// ============================================================================

// GetAppInfo 获取应用信息
// 前端调用: wails.App.GetAppInfo()
func (api *API) GetAppInfo() (info AppInfo, err error) {
	defer api.recoverError("读取应用信息", &err)
	api.runtime.RecordLogWithSeverity("api-trace", "GetAppInfo：后端收到请求", "info")
	info = api.runtime.GetAppInfo()
	api.runtime.RecordLogWithSeverity("api-trace", "GetAppInfo：后端返回成功", "info")
	return info, nil
}

// GetAppInfo 获取应用信息（内部实现）
func (s *Runtime) GetAppInfo() AppInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return AppInfo{
		Name:        s.options.AppName,
		Version:     s.options.Version,
		Description: s.options.Description,
		Repository:  s.options.Repository,
		StartedAt:   s.startedAt.Format(time.RFC3339),
	}
}

// GetEnvironmentInfo 获取环境信息
// 前端调用: wails.App.GetEnvironmentInfo()
func (api *API) GetEnvironmentInfo() (info EnvironmentInfo, err error) {
	defer api.recoverError("读取运行环境", &err)
	api.runtime.RecordLogWithSeverity("api-trace", "GetEnvironmentInfo：后端收到请求", "info")
	info = api.runtime.GetEnvironmentInfo()
	api.runtime.RecordLogWithSeverity("api-trace", "GetEnvironmentInfo：后端返回成功", "info")
	return info, nil
}

// GetEnvironmentInfo 获取环境信息（内部实现）
func (s *Runtime) GetEnvironmentInfo() EnvironmentInfo {
	s.lock.RLock()
	storeReady := s.store != nil && s.configDefaultsEnsured
	info := EnvironmentInfo{
		OS:              goruntime.GOOS,
		Arch:            goruntime.GOARCH,
		GoVersion:       goruntime.Version(),
		WailsVersion:    moduleVersion("github.com/wailsapp/wails/v3"),
		DatabasePath:    s.databasePath,
		DatabaseReady:   storeReady,
		DatabaseStatus:  databaseStatus(storeReady, s.databasePath),
		DatabaseMessage: databaseStatusMessage(storeReady, s.databasePath),
		CachePath:       s.cachePath,
	}
	s.lock.RUnlock()
	info.LogFilePath = s.currentLogFilePath()
	return info
}

func databaseStatus(ready bool, path string) string {
	if path == "" {
		return "disabled"
	}
	if ready {
		return "ok"
	}
	return "error"
}

func databaseStatusMessage(ready bool, path string) string {
	if path == "" {
		return "未配置 SQLite 数据库路径。"
	}
	if ready {
		return "SQLite 配置库已打开，默认配置已初始化。"
	}
	return "SQLite 配置库未打开或默认配置未完成。"
}
