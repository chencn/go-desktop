package app

import (
	"log/slog"

	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	appruntime "github.com/chencn/go-desktop/internal/desktopapp/runtime"
)

type ServiceOptions = appruntime.ServiceOptions

type Runtime struct {
	*appruntime.Runtime
}

type API struct {
	inner *appruntime.API
}

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

type AppInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Repository  string `json:"repository"`
	StartedAt   string `json:"startedAt"`
}

type EnvironmentInfo struct {
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	GoVersion       string `json:"goVersion"`
	WailsVersion    string `json:"wailsVersion"`
	DatabasePath    string `json:"databasePath"`
	DatabaseReady   bool   `json:"databaseReady"`
	DatabaseStatus  string `json:"databaseStatus"`
	DatabaseMessage string `json:"databaseMessage"`
	LogFilePath     string `json:"logFilePath"`
	CachePath       string `json:"cachePath"`
}

type Settings struct {
	UpdateSource             string `json:"updateSource"`
	GitHubOwner              string `json:"githubOwner"`
	GitHubRepo               string `json:"githubRepo"`
	GitHubProxyBase          string `json:"githubProxyBase"`
	UpdateCheckIntervalHours int    `json:"updateCheckIntervalHours"`
	MinimizeToTray           bool   `json:"minimizeToTray"`
	LogRetentionDays         int    `json:"logRetentionDays"`
	LogLevel                 string `json:"logLevel"`
	AutoLaunch               bool   `json:"autoLaunch"`
	CreateDesktopShortcut    bool   `json:"createDesktopShortcut"`
	LaunchHiddenToTray       bool   `json:"launchHiddenToTray"`
}

type DisplayProfile struct {
	UIStyle     string `json:"uiStyle"`
	BaseColor   string `json:"baseColor"`
	ThemeColor  string `json:"themeColor"`
	AccentColor string `json:"accentColor"`
	ChartColor  string `json:"chartColor"`
	IconTone    string `json:"iconTone"`
	Menu        string `json:"menu"`
	MenuAccent  string `json:"menuAccent"`
	Radius      string `json:"radius"`
	Density     string `json:"density"`
	TextSize    string `json:"textSize"`
	CardBorder  string `json:"cardBorder"`
}

type DisplayProfiles struct {
	Shadcn DisplayProfile `json:"shadcn"`
	AntD   DisplayProfile `json:"antd"`
}

type DisplayPreferences struct {
	DisplayScheme string          `json:"displayScheme"`
	UIStyle       string          `json:"uiStyle"`
	ThemeMode     string          `json:"themeMode"`
	BaseColor     string          `json:"baseColor"`
	ThemeColor    string          `json:"themeColor"`
	AccentColor   string          `json:"accentColor"`
	ChartColor    string          `json:"chartColor"`
	IconTone      string          `json:"iconTone"`
	Menu          string          `json:"menu"`
	MenuAccent    string          `json:"menuAccent"`
	Radius        string          `json:"radius"`
	Density       string          `json:"density"`
	TextSize      string          `json:"textSize"`
	CardBorder    string          `json:"cardBorder"`
	Profiles      DisplayProfiles `json:"profiles"`
}

type LogEntry struct {
	Time     string `json:"time"`
	Scope    string `json:"scope"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type LogFileInfo struct {
	Date       string `json:"date"`
	FileName   string `json:"fileName"`
	FilePath   string `json:"filePath"`
	SizeBytes  int64  `json:"sizeBytes"`
	ModifiedAt string `json:"modifiedAt"`
	Current    bool   `json:"current"`
}

type LogQuery struct {
	FileName string `json:"fileName"`
	Scope    string `json:"scope"`
	Severity string `json:"severity"`
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type LogStats struct {
	Total   int `json:"total"`
	Debug   int `json:"debug"`
	Info    int `json:"info"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
}

type LogResponse struct {
	Logs     []LogEntry `json:"logs"`
	Source   string     `json:"source"`
	FileName string     `json:"fileName"`
	FilePath string     `json:"filePath"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
	HasMore  bool       `json:"hasMore"`
	Stats    LogStats   `json:"stats"`
}

type UpdateStatus struct {
	Status          string  `json:"status"`
	Message         string  `json:"message"`
	Version         string  `json:"version,omitempty"`
	AssetName       string  `json:"assetName,omitempty"`
	FilePath        string  `json:"filePath,omitempty"`
	DownloadedBytes int64   `json:"downloadedBytes,omitempty"`
	TotalBytes      int64   `json:"totalBytes,omitempty"`
	ProgressPercent float64 `json:"progressPercent,omitempty"`
	Sha256          string  `json:"sha256,omitempty"`
	Verified        bool    `json:"verified"`
	ErrorReason     string  `json:"errorReason,omitempty"`
	Source          string  `json:"source,omitempty"`
	UpdatedAt       string  `json:"updatedAt"`
}

type SecondInstanceRecord struct {
	Args       []string `json:"args"`
	WorkingDir string   `json:"workingDir"`
	ReceivedAt string   `json:"receivedAt"`
}

type StartupLaunch = appruntime.StartupLaunch
type ExitRequest = appruntime.ExitRequest
type CrashState = appruntime.CrashState
type CrashReporter = appruntime.CrashReporter

func NewRuntime(options ServiceOptions) *Runtime {
	return &Runtime{Runtime: appruntime.NewRuntime(options)}
}

func (r *Runtime) API() *API {
	return &API{inner: r.Runtime.API()}
}

func (r *Runtime) CheckUpdate() CheckResult {
	return toCheckResult(r.Runtime.CheckUpdate())
}

func (r *Runtime) ClearLogs(scope string) bool {
	return r.Runtime.ClearLogs(scope)
}

func (r *Runtime) DisplayPreferencesSnapshot() DisplayPreferences {
	return toDisplayPreferences(r.Runtime.DisplayPreferencesSnapshot())
}

func (r *Runtime) DownloadUpdate() UpdateStatus {
	return toUpdateStatus(r.Runtime.DownloadUpdate())
}

func (r *Runtime) GetAppInfo() AppInfo {
	return AppInfo(r.Runtime.GetAppInfo())
}

func (r *Runtime) GetEnvironmentInfo() EnvironmentInfo {
	return EnvironmentInfo(r.Runtime.GetEnvironmentInfo())
}

func (r *Runtime) GetSecondInstanceRecords() []SecondInstanceRecord {
	return toSecondInstanceRecords(r.Runtime.GetSecondInstanceRecords())
}

func (r *Runtime) GetUpdateStatus() UpdateStatus {
	return toUpdateStatus(r.Runtime.GetUpdateStatus())
}

func (r *Runtime) InstallDownloadedUpdate() UpdateStatus {
	return toUpdateStatus(r.Runtime.InstallDownloadedUpdate())
}

func (r *Runtime) ListLogFiles() []LogFileInfo {
	return toLogFileInfos(r.Runtime.ListLogFiles())
}

func (r *Runtime) ListLogs() []LogEntry {
	return toLogEntries(r.Runtime.ListLogs())
}

func (r *Runtime) QueryLogs(query LogQuery) LogResponse {
	return toLogResponse(r.Runtime.QueryLogs(appruntime.LogQuery(query)))
}

func (r *Runtime) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error) {
	saved, err := r.Runtime.SaveDisplayPreferences(fromDisplayPreferences(preferences))
	return toDisplayPreferences(saved), err
}

func (r *Runtime) SaveSettings(settings Settings) (Settings, error) {
	saved, err := r.Runtime.SaveSettings(appruntime.Settings(settings))
	return Settings(saved), err
}

func (r *Runtime) ScheduleDownloadedUpdateOnStartup() UpdateStatus {
	return toUpdateStatus(r.Runtime.ScheduleDownloadedUpdateOnStartup())
}

func (r *Runtime) SettingsSnapshot() Settings {
	return Settings(r.Runtime.SettingsSnapshot())
}

func (api *API) CheckUpdate() (CheckResult, error) {
	result, err := api.inner.CheckUpdate()
	return toCheckResult(result), err
}

func (api *API) ClearLogs(scope string) (bool, error) {
	return api.inner.ClearLogs(scope)
}

func (api *API) DownloadUpdate() (UpdateStatus, error) {
	status, err := api.inner.DownloadUpdate()
	return toUpdateStatus(status), err
}

func (api *API) GetAppInfo() (AppInfo, error) {
	info, err := api.inner.GetAppInfo()
	return AppInfo(info), err
}

func (api *API) GetDisplayPreferences() (DisplayPreferences, error) {
	preferences, err := api.inner.GetDisplayPreferences()
	return toDisplayPreferences(preferences), err
}

func (api *API) GetEnvironmentInfo() (EnvironmentInfo, error) {
	info, err := api.inner.GetEnvironmentInfo()
	return EnvironmentInfo(info), err
}

func (api *API) GetSecondInstanceRecords() ([]SecondInstanceRecord, error) {
	records, err := api.inner.GetSecondInstanceRecords()
	return toSecondInstanceRecords(records), err
}

func (api *API) GetSettings() (Settings, error) {
	settings, err := api.inner.GetSettings()
	return Settings(settings), err
}

func (api *API) GetUpdateStatus() (UpdateStatus, error) {
	status, err := api.inner.GetUpdateStatus()
	return toUpdateStatus(status), err
}

func (api *API) InstallDownloadedUpdate() (UpdateStatus, error) {
	status, err := api.inner.InstallDownloadedUpdate()
	return toUpdateStatus(status), err
}

func (api *API) ListLogFiles() ([]LogFileInfo, error) {
	files, err := api.inner.ListLogFiles()
	return toLogFileInfos(files), err
}

func (api *API) ListLogs() ([]LogEntry, error) {
	logs, err := api.inner.ListLogs()
	return toLogEntries(logs), err
}

func (api *API) QueryLogs(query LogQuery) (LogResponse, error) {
	response, err := api.inner.QueryLogs(appruntime.LogQuery(query))
	return toLogResponse(response), err
}

func (api *API) QuitApp() error {
	return api.inner.QuitApp()
}

func (api *API) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error) {
	saved, err := api.inner.SaveDisplayPreferences(fromDisplayPreferences(preferences))
	return toDisplayPreferences(saved), err
}

func (api *API) SaveSettings(settings Settings) (Settings, error) {
	saved, err := api.inner.SaveSettings(appruntime.Settings(settings))
	return Settings(saved), err
}

func (api *API) ScheduleDownloadedUpdateOnStartup() (UpdateStatus, error) {
	status, err := api.inner.ScheduleDownloadedUpdateOnStartup()
	return toUpdateStatus(status), err
}

func (api *API) ShowMainWindow() error {
	return api.inner.ShowMainWindow()
}

func SlogLevelFromLogLevel(level string) slog.Level {
	return appruntime.SlogLevelFromLogLevel(level)
}

func ParseExitRequest(args []string) ExitRequest {
	return appruntime.ParseExitRequest(args)
}

func ParseStartupLaunch(args []string) StartupLaunch {
	return appruntime.ParseStartupLaunch(args)
}

func StartupAutostartArguments(settings Settings) []string {
	return appruntime.StartupAutostartArguments(appruntime.Settings(settings))
}

func NewCrashReporter(logPath string, statePath string) *CrashReporter {
	return appruntime.NewCrashReporter(logPath, statePath)
}

func StartCrashReporter(logPath string, statePath string, args []string) (*CrashReporter, CrashState, bool) {
	return appruntime.StartCrashReporter(logPath, statePath, args)
}

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
