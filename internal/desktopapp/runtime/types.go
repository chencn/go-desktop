// 文件职责：runtime 的初始化选项、API facade 与全部前后端传输 DTO 定义。
// 拆出独立文件是为了让 service.go 只保留装配与生命周期逻辑。

package runtime

import (
	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	updater "github.com/chencn/go-desktop/internal/desktopapp/update"
)

// ServiceOptions 定义 Runtime 初始化依赖。
// 生产入口只传必要覆盖项；空字段会在 NewRuntime 中补项目默认值或平台默认路径。
type ServiceOptions struct {
	// AppName 应用程序名称
	// 用于窗口标题、单实例标识、默认路径等
	AppName string

	// Version 当前应用版本号
	// 格式: semver (如 "1.0.0")，用于更新检查
	Version string

	// Description 应用描述
	// 显示在关于页面和元数据中
	Description string

	// Repository GitHub 仓库地址
	// 用于更新检查和关于页面链接
	Repository string

	// DatabasePath SQLite 数据库文件路径
	// 只存储 config_items 配置项
	DatabasePath string

	// LogDirPath 文件日志目录
	// 为空时可由 LogFilePath 兼容映射；桌面入口会传入默认路径所在目录
	LogDirPath string

	// LogFilePath 兼容旧调用的文件日志路径
	// 新实现只取其目录，实际日志文件按 appName-YYYY-MM-DD.log 写入
	LogFilePath string

	// CrashReporter 最早期崩溃日志器
	// main.go 在 Runtime 创建前安装，用于捕获 Runtime/Wails 尚未可用时的退出线索
	CrashReporter *CrashReporter

	// CachePath 缓存目录路径
	// 用于存储下载的更新包等临时文件
	CachePath string

	// ReleaseChecker 是默认 GitHub Release 检查器；设置切到 local 或自定义 GitHub 配置时会动态创建新检查器。
	ReleaseChecker *githubrelease.Checker

	// LocalUpdateBaseURL 本地静态升级根地址
	// 为空时使用项目元数据默认值
	LocalUpdateBaseURL string

	// LocalUpdateManifestPath 本地 latest.json 相对路径
	// 为空时使用项目元数据默认值
	LocalUpdateManifestPath string

	// UpdateManager 负责下载、校验和启动安装器；为空时使用 CachePath 创建默认实例。
	UpdateManager *updater.Manager

	// StartupIntegrationApplier 可选的启动集成同步函数，用于替换默认平台集成实现。
	StartupIntegrationApplier func(previous Settings, next Settings) error

	// LicenseMode 授权模式；只有 required 会启用授权检查，空值表示关闭。
	LicenseMode string

	// LicensePublicKey 授权公钥，required 模式下用于验证授权码签名。
	LicensePublicKey string

	// LicenseDeviceCode 测试或特殊场景下覆盖当前设备码；为空时按机器信息生成。
	LicenseDeviceCode string

	// LicenseDeviceCodeSource 测试或特殊场景下覆盖设备码生成函数；为空时按机器信息生成。
	LicenseDeviceCodeSource func() string
}

// API 是 Wails 暴露给前端的服务对象。
// 多数 API 方法会先调用 requireAuthorized；授权状态和激活接口除外。
type API struct {
	// runtime 是内部 Runtime 引用，不直接暴露给前端模型。
	runtime *Runtime
}

// AppInfo 是前端关于页和首页使用的应用基本信息。
type AppInfo struct {
	// Name 应用名称
	Name string `json:"name"`

	// Version 当前版本号
	Version string `json:"version"`

	// Description 应用描述
	Description string `json:"description"`

	// Repository GitHub 仓库地址
	Repository string `json:"repository"`

	// StartedAt 启动时间（ISO 8601 格式）
	StartedAt string `json:"startedAt"`
}

// EnvironmentInfo 是前端展示和诊断使用的运行环境快照。
type EnvironmentInfo struct {
	// OS 操作系统（如 "windows", "darwin", "linux"）
	OS string `json:"os"`

	// Arch CPU 架构（如 "amd64", "arm64"）
	Arch string `json:"arch"`

	// GoVersion Go 运行时版本
	GoVersion string `json:"goVersion"`

	// WailsVersion Wails 框架版本
	WailsVersion string `json:"wailsVersion"`

	// DatabasePath 数据库文件路径
	DatabasePath string `json:"databasePath"`

	// DatabaseReady 表示 SQLite 配置库已打开且默认配置项已初始化。
	DatabaseReady bool `json:"databaseReady"`

	// DatabaseStatus SQLite 配置库状态，供首页监控直接消费。
	DatabaseStatus string `json:"databaseStatus"`

	// DatabaseMessage SQLite 配置库状态说明。
	DatabaseMessage string `json:"databaseMessage"`

	// LogFilePath 文件日志路径
	LogFilePath string `json:"logFilePath"`

	// CachePath 缓存目录路径
	CachePath string `json:"cachePath"`
}

// Settings 是前端可修改并持久化到 SQLite KV 的应用设置。
type Settings struct {
	// UpdateSource 更新源：github 或 local。
	UpdateSource string `json:"updateSource"`

	// GitHubProxyBase 会作为 GitHub API、安装资产和 .sha256 URL 的统一前缀；local 源不使用它。
	GitHubProxyBase string `json:"githubProxyBase"`

	// UpdateCheckIntervalHours 自动更新检查间隔（小时）；0 或负数会回退到 metadata 默认值。
	UpdateCheckIntervalHours int `json:"updateCheckIntervalHours"`

	// MinimizeToTray 关闭窗口时隐藏到系统托盘
	// true: 点击关闭时隐藏窗口到托盘
	// false: 点击关闭时直接关闭窗口
	MinimizeToTray bool `json:"minimizeToTray"`

	// AlwaysOnTop 窗口显示时保持在其他窗口上方
	// 不影响关闭到托盘或自启隐藏策略
	AlwaysOnTop bool `json:"alwaysOnTop"`

	// LogRetentionDays 日志保留天数
	// -1 表示永久保留
	// 0 使用默认值（通常 30 天）
	LogRetentionDays int `json:"logRetentionDays"`

	// LogLevel 最小记录日志级别
	// 支持 debug、info、warning、error；error/panic 永远优先保留。
	LogLevel string `json:"logLevel"`

	// AutoLaunch 开机自启
	// true 时 Runtime 会通过 Wails3 Autostart 注册当前应用
	AutoLaunch bool `json:"autoLaunch"`

	// CreateDesktopShortcut 创建桌面快捷图标
	// true 时 Runtime 会在当前用户桌面创建应用快捷方式
	CreateDesktopShortcut bool `json:"createDesktopShortcut"`

	// LaunchHiddenToTray 开机自启时隐藏到托盘
	// 仅在 AutoLaunch 开启且启动参数来自自启入口时生效
	LaunchHiddenToTray bool `json:"launchHiddenToTray"`
}

// ============================================================================
// 日志结构体
// ============================================================================

// LogEntry 单条日志记录
type LogEntry struct {
	// Time 日志时间（ISO 8601 格式，UTC）
	Time string `json:"time"`

	// Scope 日志作用域（如 "app", "window", "storage"）
	// 用于日志分类和过滤
	Scope string `json:"scope"`

	// Message 日志内容
	Message string `json:"message"`

	// Severity 日志级别（"info", "warning", "error"）
	Severity string `json:"severity"`
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
	Scope string `json:"scope"`

	// Severity 按级别过滤（空表示全部）
	Severity string `json:"severity"`

	// Keyword 按关键词搜索（匹配 Message 字段）
	Keyword string `json:"keyword"`

	// Page 页码（从 1 开始）
	Page int `json:"page"`

	// PageSize 每页条数
	PageSize int `json:"pageSize"`
}

// LogStats 日志统计信息
type LogStats struct {
	// Total 总日志数
	Total int `json:"total"`

	// Debug 调试级别日志数
	Debug int `json:"debug"`

	// Info 信息级别日志数
	Info int `json:"info"`

	// Warning 警告级别日志数
	Warning int `json:"warning"`

	// Error 错误级别日志数
	Error int `json:"error"`
}

// LogResponse 日志查询响应
type LogResponse struct {
	// Logs 当前页的日志列表
	Logs []LogEntry `json:"logs"`

	// Source 日志来源，file 表示每日文件，memory 表示文件不可用时的内存降级。
	Source string `json:"source"`

	// FileName 当前查询的文件名。
	FileName string `json:"fileName"`

	// FilePath 当前查询的文件路径。
	FilePath string `json:"filePath"`

	// Total 符合条件的总日志数
	Total int `json:"total"`

	// Page 当前页码
	Page int `json:"page"`

	// PageSize 每页条数
	PageSize int `json:"pageSize"`

	// HasMore 是否有下一页
	HasMore bool `json:"hasMore"`

	// Stats 日志统计信息
	Stats LogStats `json:"stats"`
}

// ============================================================================
// 更新状态结构体
// ============================================================================

// UpdateStatus 是前端轮询和 update:status:changed 事件共用的更新状态机快照。
type UpdateStatus struct {
	// Status 更新状态：idle、update_available、downloading、verifying、verified、pending_install、installing、install_started、no_update、skipped、error。
	Status string `json:"status"`

	// Message 状态描述（用户可读）
	Message string `json:"message"`

	// Version 是检查到或已下载的目标版本号。
	Version string `json:"version,omitempty"`

	// AssetName 下载的文件名
	AssetName string `json:"assetName,omitempty"`

	// FilePath 是缓存目录内已校验安装包路径。
	FilePath string `json:"filePath,omitempty"`

	// DownloadedBytes 已下载字节数
	DownloadedBytes int64 `json:"downloadedBytes,omitempty"`

	// TotalBytes 总字节数
	TotalBytes int64 `json:"totalBytes,omitempty"`

	// ProgressPercent 下载进度（0-100）
	ProgressPercent float64 `json:"progressPercent,omitempty"`

	// Sha256 文件 SHA256 校验和
	// 用于验证下载文件的完整性
	Sha256 string `json:"sha256,omitempty"`

	// Verified 表示 FilePath 指向的安装包曾通过 SHA256 校验；安装前仍会复验。
	Verified bool `json:"verified"`

	// ErrorReason 错误原因（状态为 error 时）
	ErrorReason string `json:"errorReason,omitempty"`

	// Source 更新源：github 或 local
	Source string `json:"source,omitempty"`

	// UpdatedAt 状态更新时间（ISO 8601 格式）
	UpdatedAt string `json:"updatedAt"`
}

// ============================================================================
// 多实例结构体
// ============================================================================

// SecondInstanceRecord 多实例启动记录
// 当用户第二次启动应用时创建
type SecondInstanceRecord struct {
	// Args 命令行参数
	Args []string `json:"args"`

	// WorkingDir 工作目录
	WorkingDir string `json:"workingDir"`

	// ReceivedAt 接收时间
	ReceivedAt string `json:"receivedAt"`
}

// ============================================================================
// 退出请求结构体
// ============================================================================

// ExitRequest 退出请求信息
// 用于处理应用退出逻辑（正常退出、强制退出、更新退出等）
type ExitRequest struct {
	// Present 是否有退出请求
	Present bool

	// Force 是否强制退出（不保存状态）
	Force bool

	// Source 退出来源（如 "user", "update", "system"）
	Source string
}
