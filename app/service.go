// Package app 暴露 main.go 和 Wails 绑定使用的稳定桌面 facade。
// Wails service receiver 和 DTO 必须留在 app 包，避免生成的前端方法 ID
// 与模型 import 依赖 internal 包路径。
package app

import (
	"log/slog"

	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	appruntime "github.com/chencn/go-desktop/internal/desktopapp/runtime"
)

// ServiceOptions 配置 NewRuntime。它只供 main.go 装配应用，不是 Wails DTO，
// 因此可以复用 internal runtime 的结构。
type ServiceOptions = appruntime.ServiceOptions

// Runtime 是 internal runtime 外层的公开应用 facade。
// 嵌入 internal runtime 让 main.go 继续使用生命周期钩子；下方显式方法负责
// 把返回值转回 app 包 DTO，保持 Wails 绑定兼容。
type Runtime struct {
	*appruntime.Runtime
}

// API 是 main.go 注册给 Wails 的 service receiver。
// 不要把它 alias 到 internal API：Wails 会用 receiver 包路径派生前端方法 ID，
// app.API 的身份属于前端绑定契约。
type API struct {
	inner *appruntime.API
}

// CheckResult 是返回给前端的 app 包更新检查 DTO。
// 它镜像 adapter 结果，但不让生成的 Wails bindings 暴露 internal adapter import。
type CheckResult struct {
	Source           string `json:"source,omitempty"`
	Status           string `json:"status"`
	CurrentVersion   string `json:"currentVersion"`
	RequestURL       string `json:"requestUrl,omitempty"`
	HTTPStatus       int    `json:"httpStatus,omitempty"`
	LatestVersion    string `json:"latestVersion,omitempty"`
	TagName          string `json:"tagName,omitempty"`
	ReleaseURL       string `json:"releaseUrl,omitempty"`
	ReleaseNotes     string `json:"releaseNotes,omitempty"`
	AssetName        string `json:"assetName,omitempty"`
	AssetSizeBytes   int64  `json:"assetSizeBytes,omitempty"`
	AssetDownloadURL string `json:"assetDownloadUrl,omitempty"`
	Sha256           string `json:"sha256,omitempty"`
	Sha256Source     string `json:"sha256Source,omitempty"`
	SkipReason       string `json:"skipReason,omitempty"`
	ErrorReason      string `json:"errorReason,omitempty"`
	CheckedAt        string `json:"checkedAt"`
	Message          string `json:"message"`
}

// AppInfo 是前端展示的不可变应用元数据快照。
type AppInfo struct {
	Name        string `json:"name"`        // Name 是启动时配置的显示名称。
	Version     string `json:"version"`     // Version 是更新检查使用的当前 semver。
	Description string `json:"description"` // Description 是来自项目元数据的关于页摘要。
	Repository  string `json:"repository"`  // Repository 是展示给用户的公开项目地址。
	StartedAt   string `json:"startedAt"`   // StartedAt 是 Runtime 创建时间，按 RFC3339 格式化。
}

// EnvironmentInfo 描述首页/诊断视图消费的运行环境信息。
type EnvironmentInfo struct {
	OS              string `json:"os"`              // OS 是当前进程的 runtime.GOOS。
	Arch            string `json:"arch"`            // Arch 是当前进程的 runtime.GOARCH。
	GoVersion       string `json:"goVersion"`       // GoVersion 是编译进二进制的 Go runtime 版本。
	WailsVersion    string `json:"wailsVersion"`    // WailsVersion 可用时来自模块元数据。
	DatabasePath    string `json:"databasePath"`    // DatabasePath 是 SQLite 配置库路径。
	DatabaseReady   bool   `json:"databaseReady"`   // DatabaseReady 表示配置库已打开且默认项已初始化。
	DatabaseStatus  string `json:"databaseStatus"`  // DatabaseStatus 是 ok/error/disabled 等紧凑 UI 状态。
	DatabaseMessage string `json:"databaseMessage"` // DatabaseMessage 是 DatabaseStatus 的本地化说明。
	LogFilePath     string `json:"logFilePath"`     // LogFilePath 是当前每日日志文件路径。
	CachePath       string `json:"cachePath"`       // CachePath 是已校验更新包的缓存目录。
}

// LicenseStatus 报告可选授权门禁是否启用，以及当前设备是否已授权。
type LicenseStatus struct {
	Enabled    bool   `json:"enabled"`             // Enabled 表示当前构建启用了授权能力。
	Required   bool   `json:"required"`            // Required 表示授权通过前会拦截 Wails API。
	Authorized bool   `json:"authorized"`          // Authorized 表示已保存或刚提交的授权码校验通过。
	DeviceCode string `json:"deviceCode"`          // DeviceCode 是用户申请授权时使用的机器绑定码。
	Message    string `json:"message"`             // Message 是 UI 展示的本地化状态文本。
	ExpiresAt  string `json:"expiresAt,omitempty"` // ExpiresAt 是可选的 RFC3339 授权过期时间。
	LastError  string `json:"lastError,omitempty"` // LastError 携带最近一次校验或配置失败。
}

// Settings 是用户可编辑的后端设置快照。
// SaveSettings 会归一化这些值、持久化到 SQLite，并同步自启动/快捷方式等桌面副作用。
type Settings struct {
	UpdateSource             string `json:"updateSource"`             // UpdateSource 选择 github 或 local 更新源。
	GitHubOwner              string `json:"githubOwner"`              // GitHubOwner 覆盖 Release 检查的 owner。
	GitHubRepo               string `json:"githubRepo"`               // GitHubRepo 覆盖 Release 检查的仓库名。
	GitHubProxyBase          string `json:"githubProxyBase"`          // GitHubProxyBase 是可选 GitHub API 代理。
	UpdateCheckIntervalHours int    `json:"updateCheckIntervalHours"` // UpdateCheckIntervalHours 控制后台检查间隔。
	MinimizeToTray           bool   `json:"minimizeToTray"`           // MinimizeToTray 把关闭窗口转为隐藏到托盘。
	LogRetentionDays         int    `json:"logRetentionDays"`         // LogRetentionDays 控制每日日志清理策略。
	LogLevel                 string `json:"logLevel"`                 // LogLevel 是持久化日志的最低级别。
	AutoLaunch               bool   `json:"autoLaunch"`               // AutoLaunch 注册平台开机自启入口。
	CreateDesktopShortcut    bool   `json:"createDesktopShortcut"`    // CreateDesktopShortcut 管理当前用户桌面快捷方式。
	LaunchHiddenToTray       bool   `json:"launchHiddenToTray"`       // LaunchHiddenToTray 只对自启隐藏启动生效。
}

// DisplayProfile 是某个显示方案的一组可持久化视觉 profile。
type DisplayProfile struct {
	UIStyle     string `json:"uiStyle"`     // UIStyle 选择此 profile 的组件风格。
	BaseColor   string `json:"baseColor"`   // BaseColor 选择中性色/背景色盘。
	ThemeColor  string `json:"themeColor"`  // ThemeColor 选择主品牌色。
	AccentColor string `json:"accentColor"` // AccentColor 选择次级强调色。
	ChartColor  string `json:"chartColor"`  // ChartColor 选择图表色调。
	IconTone    string `json:"iconTone"`    // IconTone 控制图标颜色处理。
	Menu        string `json:"menu"`        // Menu 控制导航区域样式。
	MenuAccent  string `json:"menuAccent"`  // MenuAccent 控制选中菜单强调方式。
	Radius      string `json:"radius"`      // Radius 控制组件圆角。
	Density     string `json:"density"`     // Density 控制间距密度。
	TextSize    string `json:"textSize"`    // TextSize 控制基础 UI 字号。
	CardBorder  string `json:"cardBorder"`  // CardBorder 控制卡片边框强度。
}

// DisplayProfiles 用稳定 JSON 字段承载所有方案 profile。
// 前端依赖字段名而不是 map，保证生成模型简单且可预测。
type DisplayProfiles struct {
	Shadcn DisplayProfile `json:"shadcn"` // Shadcn 保存 shadcn-vue profile。
	AntD   DisplayProfile `json:"antd"`   // AntD 保存 Ant Design 风格 profile。
}

// DisplayPreferences 是暴露给 UI 的 typed 显示偏好快照。
// 扁平字段是当前 DisplayScheme 的生效值；Profiles 保留所有方案的持久化值。
type DisplayPreferences struct {
	DisplayScheme string          `json:"displayScheme"` // DisplayScheme 选择当前生效显示方案。
	UIStyle       string          `json:"uiStyle"`       // UIStyle 是当前生效组件风格。
	ThemeMode     string          `json:"themeMode"`     // ThemeMode 是当前生效亮暗模式。
	BaseColor     string          `json:"baseColor"`     // BaseColor 是当前生效中性色/背景色盘。
	ThemeColor    string          `json:"themeColor"`    // ThemeColor 是当前生效主色。
	AccentColor   string          `json:"accentColor"`   // AccentColor 是当前生效强调色。
	ChartColor    string          `json:"chartColor"`    // ChartColor 是当前生效图表色调。
	IconTone      string          `json:"iconTone"`      // IconTone 是当前生效图标颜色处理。
	Menu          string          `json:"menu"`          // Menu 是当前生效导航样式。
	MenuAccent    string          `json:"menuAccent"`    // MenuAccent 是当前生效选中菜单强调方式。
	Radius        string          `json:"radius"`        // Radius 是当前生效圆角。
	Density       string          `json:"density"`       // Density 是当前生效间距密度。
	TextSize      string          `json:"textSize"`      // TextSize 是当前生效 UI 字号。
	CardBorder    string          `json:"cardBorder"`    // CardBorder 是当前生效卡片边框强度。
	Profiles      DisplayProfiles `json:"profiles"`      // Profiles 保留所有可编辑方案 profile。
}

// LogEntry 是返回给前端的一条日志。
type LogEntry struct {
	Time     string `json:"time"`     // Time 是 runtime 格式化后的日志时间。
	Scope    string `json:"scope"`    // Scope 按子系统分组，供过滤使用。
	Message  string `json:"message"`  // Message 是本地化日志文本。
	Severity string `json:"severity"` // Severity 是归一化后的 debug/info/warning/error。
}

// LogFileInfo 描述一个前端可选择的每日日志文件。
type LogFileInfo struct {
	Date       string `json:"date"`       // Date 是文件代表的 YYYY-MM-DD 日期。
	FileName   string `json:"fileName"`   // FileName 是 QueryLogs 接受的文件名。
	FilePath   string `json:"filePath"`   // FilePath 是用于诊断展示的完整路径。
	SizeBytes  int64  `json:"sizeBytes"`  // SizeBytes 是当前文件大小。
	ModifiedAt string `json:"modifiedAt"` // ModifiedAt 是文件最后修改时间。
	Current    bool   `json:"current"`    // Current 标记当前正在写入的每日文件。
}

// LogQuery 是前端分页查询日志的输入契约。
type LogQuery struct {
	FileName string `json:"fileName"` // FileName 选择每日日志文件；空值表示当前文件。
	Scope    string `json:"scope"`    // Scope 按子系统过滤；空值表示全部。
	Severity string `json:"severity"` // Severity 按归一化级别过滤；空值表示全部。
	Keyword  string `json:"keyword"`  // Keyword 按日志消息过滤。
	Page     int    `json:"page"`     // Page 从 1 开始，非法值由 runtime 归一化。
	PageSize int    `json:"pageSize"` // PageSize 由 runtime 归一化到允许范围。
}

// LogStats 汇总完整过滤结果，而不是只统计当前页。
type LogStats struct {
	Total   int `json:"total"`   // Total 是过滤后的日志总数。
	Debug   int `json:"debug"`   // Debug 是过滤后的 debug 数量。
	Info    int `json:"info"`    // Info 是过滤后的 info 数量。
	Warning int `json:"warning"` // Warning 是过滤后的 warning 数量。
	Error   int `json:"error"`   // Error 是过滤后的 error 数量。
}

// LogResponse 是分页日志查询结果。
type LogResponse struct {
	Logs     []LogEntry `json:"logs"`     // Logs 是过滤和排序后的当前页。
	Source   string     `json:"source"`   // Source 是 file，文件读取降级时为 memory。
	FileName string     `json:"fileName"` // FileName 是实际查询的文件名。
	FilePath string     `json:"filePath"` // FilePath 是实际查询的文件路径。
	Total    int        `json:"total"`    // Total 是过滤后的日志总数。
	Page     int        `json:"page"`     // Page 是归一化后的当前页。
	PageSize int        `json:"pageSize"` // PageSize 是归一化后的页大小。
	HasMore  bool       `json:"hasMore"`  // HasMore 告诉 UI 是否还有下一页。
	Stats    LogStats   `json:"stats"`    // Stats 汇总完整过滤结果。
}

// UpdateStatus 是返回给 UI 的当前更新流程状态。
// runtime 主要按当前进程状态维护它；已校验/待安装包只通过显式状态文件恢复。
type UpdateStatus struct {
	Status          string  `json:"status"`                    // Status 是 idle/downloading/verified/error 等流程状态。
	Message         string  `json:"message"`                   // Message 是给 UI 展示的本地化状态文本。
	Version         string  `json:"version,omitempty"`         // Version 是已知目标版本。
	AssetName       string  `json:"assetName,omitempty"`       // AssetName 是选中的安装器资产名。
	FilePath        string  `json:"filePath,omitempty"`        // FilePath 是已校验安装器路径。
	DownloadedBytes int64   `json:"downloadedBytes,omitempty"` // DownloadedBytes 是当前下载进度。
	TotalBytes      int64   `json:"totalBytes,omitempty"`      // TotalBytes 是预期下载大小。
	ProgressPercent float64 `json:"progressPercent,omitempty"` // ProgressPercent 归一化为 0-100。
	Sha256          string  `json:"sha256,omitempty"`          // Sha256 是预期安装器校验和。
	Verified        bool    `json:"verified"`                  // Verified 表示校验通过后才允许安装。
	ErrorReason     string  `json:"errorReason,omitempty"`     // ErrorReason 是稳定的机器可读失败原因。
	Source          string  `json:"source,omitempty"`          // Source 记录 github 或 local 更新来源。
	UpdatedAt       string  `json:"updatedAt"`                 // UpdatedAt 是 RFC3339 状态更新时间。
}

// SecondInstanceRecord 记录被路由到主进程的第二实例启动请求。
type SecondInstanceRecord struct {
	Args       []string `json:"args"`       // Args 是第二进程启动参数的副本。
	WorkingDir string   `json:"workingDir"` // WorkingDir 是第二进程工作目录。
	ReceivedAt string   `json:"receivedAt"` // ReceivedAt 是主进程处理该请求的时间。
}

// StartupLaunch 是 main.go 使用的启动参数语义别名，不作为 Wails 模型暴露。
type StartupLaunch = appruntime.StartupLaunch

// ExitRequest 是 main.go 在 Wails 事件循环启动前消费的退出参数别名。
type ExitRequest = appruntime.ExitRequest

// CrashState 是早期崩溃 breadcrumb 状态别名。
type CrashState = appruntime.CrashState

// CrashReporter 是 main.go 在 Runtime 创建前使用的早期崩溃记录器别名。
type CrashReporter = appruntime.CrashReporter

// NewRuntime 构造 internal runtime，并返回稳定的 app facade。
func NewRuntime(options ServiceOptions) *Runtime {
	return &Runtime{Runtime: appruntime.NewRuntime(options)}
}

// API 返回当前 runtime 的 app 包 Wails service facade。
func (r *Runtime) API() *API {
	return &API{inner: r.Runtime.API()}
}

// CheckUpdate 通过 internal runtime 执行更新检查，并转成 app DTO。
func (r *Runtime) CheckUpdate() CheckResult {
	return toCheckResult(r.Runtime.CheckUpdate())
}

// ClearLogs 清空指定 scope 的内存日志视图。
func (r *Runtime) ClearLogs(scope string) bool {
	return r.Runtime.ClearLogs(scope)
}

// DisplayPreferencesSnapshot 返回当前生效显示偏好。
func (r *Runtime) DisplayPreferencesSnapshot() DisplayPreferences {
	return toDisplayPreferences(r.Runtime.DisplayPreferencesSnapshot())
}

// DownloadUpdate 下载并校验最近一次更新检查结果。
func (r *Runtime) DownloadUpdate() UpdateStatus {
	return toUpdateStatus(r.Runtime.DownloadUpdate())
}

// GetAppInfo 返回不可变应用元数据。
func (r *Runtime) GetAppInfo() AppInfo {
	return AppInfo(r.Runtime.GetAppInfo())
}

// GetEnvironmentInfo 返回当前进程运行诊断信息。
func (r *Runtime) GetEnvironmentInfo() EnvironmentInfo {
	return EnvironmentInfo(r.Runtime.GetEnvironmentInfo())
}

// GetLicenseStatus 返回当前授权状态。
func (r *Runtime) GetLicenseStatus() LicenseStatus {
	return LicenseStatus(r.Runtime.GetLicenseStatus())
}

// ActivateLicense 在需要授权时校验并保存授权码。
func (r *Runtime) ActivateLicense(licenseKey string) (LicenseStatus, error) {
	status, err := r.Runtime.ActivateLicense(licenseKey)
	return LicenseStatus(status), err
}

// GetSecondInstanceRecords 返回主进程处理过的第二实例启动请求。
func (r *Runtime) GetSecondInstanceRecords() []SecondInstanceRecord {
	return toSecondInstanceRecords(r.Runtime.GetSecondInstanceRecords())
}

// GetUpdateStatus 返回当前更新流程状态。
func (r *Runtime) GetUpdateStatus() UpdateStatus {
	return toUpdateStatus(r.Runtime.GetUpdateStatus())
}

// InstallDownloadedUpdate 启动已校验更新包的安装器。
func (r *Runtime) InstallDownloadedUpdate() UpdateStatus {
	return toUpdateStatus(r.Runtime.InstallDownloadedUpdate())
}

// ListLogFiles 返回前端可选择的每日日志文件。
func (r *Runtime) ListLogFiles() []LogFileInfo {
	return toLogFileInfos(r.Runtime.ListLogFiles())
}

// ListLogs 返回当前内存日志视图。
func (r *Runtime) ListLogs() []LogEntry {
	return toLogEntries(r.Runtime.ListLogs())
}

// QueryLogs 按过滤和分页读取每日日志文件。
func (r *Runtime) QueryLogs(query LogQuery) LogResponse {
	return toLogResponse(r.Runtime.QueryLogs(appruntime.LogQuery(query)))
}

// SaveDisplayPreferences 持久化显示偏好，并返回归一化后的值。
func (r *Runtime) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error) {
	saved, err := r.Runtime.SaveDisplayPreferences(fromDisplayPreferences(preferences))
	return toDisplayPreferences(saved), err
}

// SaveSettings 持久化归一化设置，并同步桌面集成。
func (r *Runtime) SaveSettings(settings Settings) (Settings, error) {
	saved, err := r.Runtime.SaveSettings(appruntime.Settings(settings))
	return Settings(saved), err
}

// ScheduleDownloadedUpdateOnStartup 标记已校验更新包在下次启动时安装。
func (r *Runtime) ScheduleDownloadedUpdateOnStartup() UpdateStatus {
	return toUpdateStatus(r.Runtime.ScheduleDownloadedUpdateOnStartup())
}

// SettingsSnapshot 返回当前后端设置快照。
func (r *Runtime) SettingsSnapshot() Settings {
	return Settings(r.Runtime.SettingsSnapshot())
}

// CheckUpdate 是暴露给 Wails 的更新检查入口。
func (api *API) CheckUpdate() (CheckResult, error) {
	result, err := api.inner.CheckUpdate()
	return toCheckResult(result), err
}

// ClearLogs 是暴露给 Wails 的日志视图清空入口。
func (api *API) ClearLogs(scope string) (bool, error) {
	return api.inner.ClearLogs(scope)
}

// DownloadUpdate 是暴露给 Wails 的更新下载入口。
func (api *API) DownloadUpdate() (UpdateStatus, error) {
	status, err := api.inner.DownloadUpdate()
	return toUpdateStatus(status), err
}

// GetAppInfo 是暴露给 Wails 的应用元数据入口。
func (api *API) GetAppInfo() (AppInfo, error) {
	info, err := api.inner.GetAppInfo()
	return AppInfo(info), err
}

// GetDisplayPreferences 是暴露给 Wails 的显示偏好读取入口。
func (api *API) GetDisplayPreferences() (DisplayPreferences, error) {
	preferences, err := api.inner.GetDisplayPreferences()
	return toDisplayPreferences(preferences), err
}

// GetEnvironmentInfo 是暴露给 Wails 的环境诊断入口。
func (api *API) GetEnvironmentInfo() (EnvironmentInfo, error) {
	info, err := api.inner.GetEnvironmentInfo()
	return EnvironmentInfo(info), err
}

// GetLicenseStatus 是暴露给 Wails 的授权状态入口。
func (api *API) GetLicenseStatus() (LicenseStatus, error) {
	status, err := api.inner.GetLicenseStatus()
	return LicenseStatus(status), err
}

// ActivateLicense 是暴露给 Wails 的授权激活入口。
func (api *API) ActivateLicense(licenseKey string) (LicenseStatus, error) {
	status, err := api.inner.ActivateLicense(licenseKey)
	return LicenseStatus(status), err
}

// GetSecondInstanceRecords 是暴露给 Wails 的第二实例记录入口。
func (api *API) GetSecondInstanceRecords() ([]SecondInstanceRecord, error) {
	records, err := api.inner.GetSecondInstanceRecords()
	return toSecondInstanceRecords(records), err
}

// GetSettings 是暴露给 Wails 的后端设置读取入口。
func (api *API) GetSettings() (Settings, error) {
	settings, err := api.inner.GetSettings()
	return Settings(settings), err
}

// GetUpdateStatus 是暴露给 Wails 的更新状态入口。
func (api *API) GetUpdateStatus() (UpdateStatus, error) {
	status, err := api.inner.GetUpdateStatus()
	return toUpdateStatus(status), err
}

// InstallDownloadedUpdate 是暴露给 Wails 的立即安装入口。
func (api *API) InstallDownloadedUpdate() (UpdateStatus, error) {
	status, err := api.inner.InstallDownloadedUpdate()
	return toUpdateStatus(status), err
}

// ListLogFiles 是暴露给 Wails 的每日日志文件列表入口。
func (api *API) ListLogFiles() ([]LogFileInfo, error) {
	files, err := api.inner.ListLogFiles()
	return toLogFileInfos(files), err
}

// ListLogs 是暴露给 Wails 的内存日志列表入口。
func (api *API) ListLogs() ([]LogEntry, error) {
	logs, err := api.inner.ListLogs()
	return toLogEntries(logs), err
}

// QueryLogs 是暴露给 Wails 的分页日志查询入口。
func (api *API) QueryLogs(query LogQuery) (LogResponse, error) {
	response, err := api.inner.QueryLogs(appruntime.LogQuery(query))
	return toLogResponse(response), err
}

// QuitApp 是暴露给 Wails 的应用退出入口。
func (api *API) QuitApp() error {
	return api.inner.QuitApp()
}

// SaveDisplayPreferences 是暴露给 Wails 的显示偏好保存入口。
func (api *API) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error) {
	saved, err := api.inner.SaveDisplayPreferences(fromDisplayPreferences(preferences))
	return toDisplayPreferences(saved), err
}

// SaveSettings 是暴露给 Wails 的后端设置保存入口。
func (api *API) SaveSettings(settings Settings) (Settings, error) {
	saved, err := api.inner.SaveSettings(appruntime.Settings(settings))
	return Settings(saved), err
}

// ScheduleDownloadedUpdateOnStartup 是暴露给 Wails 的下次启动安装入口。
func (api *API) ScheduleDownloadedUpdateOnStartup() (UpdateStatus, error) {
	status, err := api.inner.ScheduleDownloadedUpdateOnStartup()
	return toUpdateStatus(status), err
}

// ShowMainWindow 是暴露给 Wails 的恢复并聚焦主窗口入口。
func (api *API) ShowMainWindow() error {
	return api.inner.ShowMainWindow()
}

// SlogLevelFromLogLevel 将持久化日志级别字符串转换为 slog 级别。
func SlogLevelFromLogLevel(level string) slog.Level {
	return appruntime.SlogLevelFromLogLevel(level)
}

// ParseExitRequest 在 Wails 启动前解析安装器或用户退出参数。
func ParseExitRequest(args []string) ExitRequest {
	return appruntime.ParseExitRequest(args)
}

// ParseStartupLaunch 解析影响初始窗口可见性的启动参数。
func ParseStartupLaunch(args []string) StartupLaunch {
	return appruntime.ParseStartupLaunch(args)
}

// StartupAutostartArguments 按设置返回平台注册自启时使用的参数。
func StartupAutostartArguments(settings Settings) []string {
	return appruntime.StartupAutostartArguments(appruntime.Settings(settings))
}

// NewCrashReporter 创建 Runtime 存在前使用的早期崩溃记录器。
func NewCrashReporter(logPath string, statePath string) *CrashReporter {
	return appruntime.NewCrashReporter(logPath, statePath)
}

// StartCrashReporter 启动早期崩溃追踪，并返回上次崩溃状态。
func StartCrashReporter(logPath string, statePath string, args []string) (*CrashReporter, CrashState, bool) {
	return appruntime.StartCrashReporter(logPath, statePath, args)
}

// ReadPreviousCrashState 读取上次持久化的崩溃 breadcrumb 状态。
func ReadPreviousCrashState(path string) (CrashState, bool) {
	return appruntime.ReadPreviousCrashState(path)
}

func toCheckResult(value githubrelease.CheckResult) CheckResult {
	return CheckResult(value)
}

func toUpdateStatus(value appruntime.UpdateStatus) UpdateStatus {
	return UpdateStatus(value)
}

func toLogEntries(values []appruntime.LogEntry) []LogEntry {
	if values == nil {
		return nil
	}
	result := make([]LogEntry, len(values))
	for index, value := range values {
		result[index] = LogEntry(value)
	}
	return result
}

func toLogFileInfos(values []appruntime.LogFileInfo) []LogFileInfo {
	if values == nil {
		return nil
	}
	result := make([]LogFileInfo, len(values))
	for index, value := range values {
		result[index] = LogFileInfo(value)
	}
	return result
}

func toSecondInstanceRecords(values []appruntime.SecondInstanceRecord) []SecondInstanceRecord {
	if values == nil {
		return nil
	}
	result := make([]SecondInstanceRecord, len(values))
	for index, value := range values {
		result[index] = SecondInstanceRecord(value)
	}
	return result
}

func toLogStats(value appruntime.LogStats) LogStats {
	return LogStats(value)
}

func toLogResponse(value appruntime.LogResponse) LogResponse {
	return LogResponse{
		Logs:     toLogEntries(value.Logs),
		Source:   value.Source,
		FileName: value.FileName,
		FilePath: value.FilePath,
		Total:    value.Total,
		Page:     value.Page,
		PageSize: value.PageSize,
		HasMore:  value.HasMore,
		Stats:    toLogStats(value.Stats),
	}
}

func toDisplayPreferences(value appruntime.DisplayPreferences) DisplayPreferences {
	return DisplayPreferences{
		DisplayScheme: value.DisplayScheme,
		UIStyle:       value.UIStyle,
		ThemeMode:     value.ThemeMode,
		BaseColor:     value.BaseColor,
		ThemeColor:    value.ThemeColor,
		AccentColor:   value.AccentColor,
		ChartColor:    value.ChartColor,
		IconTone:      value.IconTone,
		Menu:          value.Menu,
		MenuAccent:    value.MenuAccent,
		Radius:        value.Radius,
		Density:       value.Density,
		TextSize:      value.TextSize,
		CardBorder:    value.CardBorder,
		Profiles: DisplayProfiles{
			Shadcn: toDisplayProfile(value.Profiles.Shadcn),
			AntD:   toDisplayProfile(value.Profiles.AntD),
		},
	}
}

func fromDisplayPreferences(value DisplayPreferences) appruntime.DisplayPreferences {
	return appruntime.DisplayPreferences{
		DisplayScheme: value.DisplayScheme,
		UIStyle:       value.UIStyle,
		ThemeMode:     value.ThemeMode,
		BaseColor:     value.BaseColor,
		ThemeColor:    value.ThemeColor,
		AccentColor:   value.AccentColor,
		ChartColor:    value.ChartColor,
		IconTone:      value.IconTone,
		Menu:          value.Menu,
		MenuAccent:    value.MenuAccent,
		Radius:        value.Radius,
		Density:       value.Density,
		TextSize:      value.TextSize,
		CardBorder:    value.CardBorder,
		Profiles: appruntime.DisplayProfiles{
			Shadcn: fromDisplayProfile(value.Profiles.Shadcn),
			AntD:   fromDisplayProfile(value.Profiles.AntD),
		},
	}
}

func toDisplayProfile(value appruntime.DisplayProfile) DisplayProfile {
	return DisplayProfile{
		UIStyle:     value.UIStyle,
		BaseColor:   value.BaseColor,
		ThemeColor:  value.ThemeColor,
		AccentColor: value.AccentColor,
		ChartColor:  value.ChartColor,
		IconTone:    value.IconTone,
		Menu:        value.Menu,
		MenuAccent:  value.MenuAccent,
		Radius:      value.Radius,
		Density:     value.Density,
		TextSize:    value.TextSize,
		CardBorder:  value.CardBorder,
	}
}

func fromDisplayProfile(value DisplayProfile) appruntime.DisplayProfile {
	return appruntime.DisplayProfile{
		UIStyle:     value.UIStyle,
		BaseColor:   value.BaseColor,
		ThemeColor:  value.ThemeColor,
		AccentColor: value.AccentColor,
		ChartColor:  value.ChartColor,
		IconTone:    value.IconTone,
		Menu:        value.Menu,
		MenuAccent:  value.MenuAccent,
		Radius:      value.Radius,
		Density:     value.Density,
		TextSize:    value.TextSize,
		CardBorder:  value.CardBorder,
	}
}
