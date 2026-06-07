package settings

import (
	"strconv"
	"strings"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

const (
	KeyUpdateSource             = "update.source"
	KeyGitHubOwner              = "github.owner"
	KeyGitHubRepo               = "github.repo"
	KeyGitHubProxyBase          = "github.proxy_base"
	KeyUpdateCheckIntervalHours = "update.check_interval_hours"
	KeyWindowMinimizeToTray     = "window.minimize_to_tray"
	KeyLogRetentionDays         = "log.retention_days"
	KeyLogLevel                 = "log.level"
	KeyStartupAutoLaunch        = "startup.auto_launch"
	KeyStartupDesktopShortcut   = "startup.create_desktop_shortcut"
	KeyStartupHiddenToTray      = "startup.launch_hidden_to_tray"
)

const defaultLogLevel = "info"

type Settings struct {
	UpdateSource             string
	GitHubOwner              string
	GitHubRepo               string
	GitHubProxyBase          string
	UpdateCheckIntervalHours int
	MinimizeToTray           bool
	LogRetentionDays         int
	LogLevel                 string
	AutoLaunch               bool
	CreateDesktopShortcut    bool
	LaunchHiddenToTray       bool
}

func Default() Settings {
	return Settings{
		UpdateSource:             metadata.DefaultUpdateSource,
		GitHubOwner:              metadata.GitHubOwner,
		GitHubRepo:               metadata.GitHubRepo,
		GitHubProxyBase:          metadata.DefaultGitHubProxyBase,
		UpdateCheckIntervalHours: metadata.DefaultUpdateCheckIntervalHours,
		MinimizeToTray:           metadata.DefaultMinimizeToTray,
		LogRetentionDays:         metadata.DefaultLogRetentionDays,
		LogLevel:                 defaultLogLevel,
		AutoLaunch:               metadata.DefaultAutoLaunch,
		CreateDesktopShortcut:    metadata.DefaultCreateDesktopShortcut,
		LaunchHiddenToTray:       metadata.DefaultLaunchHiddenToTray,
	}
}

func Normalize(value Settings) Settings {
	value.UpdateSource = NormalizeUpdateSource(value.UpdateSource)
	if value.GitHubOwner == "" {
		value.GitHubOwner = metadata.GitHubOwner
	}
	if value.GitHubRepo == "" {
		value.GitHubRepo = metadata.GitHubRepo
	}
	value.UpdateCheckIntervalHours = NormalizeUpdateCheckIntervalHours(value.UpdateCheckIntervalHours)
	if value.LogRetentionDays == 0 || value.LogRetentionDays < -1 {
		value.LogRetentionDays = metadata.DefaultLogRetentionDays
	}
	value.LogLevel = NormalizeLogLevel(value.LogLevel)
	return value
}

func NormalizeUpdateSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "local":
		return "local"
	default:
		return "github"
	}
}

func NormalizeUpdateCheckIntervalHours(value int) int {
	switch value {
	case 1, 3, 6, 12:
		return value
	default:
		return metadata.DefaultUpdateCheckIntervalHours
	}
}

func NormalizeLogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return "debug"
	case "warning", "warn":
		return "warning"
	case "error":
		return "error"
	default:
		return defaultLogLevel
	}
}

func FromConfigItems(items map[string]configstore.ConfigItem, base Settings) Settings {
	base.UpdateSource = configString(items, KeyUpdateSource, base.UpdateSource)
	base.GitHubOwner = configString(items, KeyGitHubOwner, base.GitHubOwner)
	base.GitHubRepo = configString(items, KeyGitHubRepo, base.GitHubRepo)
	base.GitHubProxyBase = configString(items, KeyGitHubProxyBase, base.GitHubProxyBase)
	base.UpdateCheckIntervalHours = configInt(items, KeyUpdateCheckIntervalHours, base.UpdateCheckIntervalHours)
	base.MinimizeToTray = configBool(items, KeyWindowMinimizeToTray, base.MinimizeToTray)
	base.LogRetentionDays = configInt(items, KeyLogRetentionDays, base.LogRetentionDays)
	base.LogLevel = configString(items, KeyLogLevel, base.LogLevel)
	base.AutoLaunch = configBool(items, KeyStartupAutoLaunch, base.AutoLaunch)
	base.CreateDesktopShortcut = configBool(items, KeyStartupDesktopShortcut, base.CreateDesktopShortcut)
	base.LaunchHiddenToTray = configBool(items, KeyStartupHiddenToTray, base.LaunchHiddenToTray)
	return Normalize(base)
}

func Values(value Settings) map[string]string {
	value = Normalize(value)
	return map[string]string{
		KeyUpdateSource:             value.UpdateSource,
		KeyGitHubOwner:              value.GitHubOwner,
		KeyGitHubRepo:               value.GitHubRepo,
		KeyGitHubProxyBase:          value.GitHubProxyBase,
		KeyUpdateCheckIntervalHours: strconv.Itoa(value.UpdateCheckIntervalHours),
		KeyWindowMinimizeToTray:     strconv.FormatBool(value.MinimizeToTray),
		KeyLogRetentionDays:         strconv.Itoa(value.LogRetentionDays),
		KeyLogLevel:                 value.LogLevel,
		KeyStartupAutoLaunch:        strconv.FormatBool(value.AutoLaunch),
		KeyStartupDesktopShortcut:   strconv.FormatBool(value.CreateDesktopShortcut),
		KeyStartupHiddenToTray:      strconv.FormatBool(value.LaunchHiddenToTray),
	}
}

func Definitions() []configstore.ConfigItem {
	defaults := Default()
	return []configstore.ConfigItem{
		{Key: KeyUpdateSource, Category: "update", Title: "更新源", Description: "选择 GitHub Release 或本地静态 manifest 作为唯一更新检查来源。", ValueType: "string", DefaultValue: defaults.UpdateSource, Value: defaults.UpdateSource, SortOrder: 90},
		{Key: KeyGitHubOwner, Category: "github", Title: "GitHub Owner", Description: "用于 Release 更新检查的仓库所有者。", ValueType: "string", DefaultValue: defaults.GitHubOwner, Value: defaults.GitHubOwner, SortOrder: 10},
		{Key: KeyGitHubRepo, Category: "github", Title: "GitHub Repo", Description: "用于 Release 更新检查的仓库名称。", ValueType: "string", DefaultValue: defaults.GitHubRepo, Value: defaults.GitHubRepo, SortOrder: 20},
		{Key: KeyGitHubProxyBase, Category: "github", Title: "GitHub API 代理", Description: "为空时直接使用 GitHub 官方 API。", ValueType: "string", DefaultValue: defaults.GitHubProxyBase, Value: defaults.GitHubProxyBase, SortOrder: 30},
		{Key: KeyUpdateCheckIntervalHours, Category: "update", Title: "检查间隔", Description: "自动检查 GitHub Release 的时间间隔。", ValueType: "int", DefaultValue: strconv.Itoa(defaults.UpdateCheckIntervalHours), Value: strconv.Itoa(defaults.UpdateCheckIntervalHours), SortOrder: 100},
		{Key: KeyWindowMinimizeToTray, Category: "window", Title: "关闭到系统托盘", Description: "点击关闭按钮时隐藏窗口到系统托盘。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.MinimizeToTray), Value: strconv.FormatBool(defaults.MinimizeToTray), SortOrder: 200},
		{Key: KeyLogRetentionDays, Category: "log", Title: "日志保留周期", Description: "每日文件日志自动清理周期，-1 表示永不清理。", ValueType: "int", DefaultValue: strconv.Itoa(defaults.LogRetentionDays), Value: strconv.Itoa(defaults.LogRetentionDays), SortOrder: 300},
		{Key: KeyLogLevel, Category: "log", Title: "日志级别", Description: "控制运行日志最小记录级别；debug 用于定位问题，error 和 panic 始终记录。", ValueType: "string", DefaultValue: defaults.LogLevel, Value: defaults.LogLevel, SortOrder: 310},
		{Key: KeyStartupAutoLaunch, Category: "startup", Title: "开机自启", Description: "登录系统后自动启动应用。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.AutoLaunch), Value: strconv.FormatBool(defaults.AutoLaunch), SortOrder: 400},
		{Key: KeyStartupDesktopShortcut, Category: "startup", Title: "创建桌面快捷图标", Description: "在当前用户桌面创建应用快捷方式。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.CreateDesktopShortcut), Value: strconv.FormatBool(defaults.CreateDesktopShortcut), SortOrder: 410},
		{Key: KeyStartupHiddenToTray, Category: "startup", Title: "开机自启时隐藏到托盘", Description: "仅在开机自启启动时隐藏主窗口，手动启动仍显示界面。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.LaunchHiddenToTray), Value: strconv.FormatBool(defaults.LaunchHiddenToTray), SortOrder: 420},
	}
}

func configString(items map[string]configstore.ConfigItem, key string, fallback string) string {
	if item, ok := items[key]; ok {
		if value := strings.TrimSpace(item.Value); value != "" {
			return value
		}
	}
	return fallback
}

func configBool(items map[string]configstore.ConfigItem, key string, fallback bool) bool {
	if item, ok := items[key]; ok {
		if value, err := strconv.ParseBool(strings.TrimSpace(item.Value)); err == nil {
			return value
		}
	}
	return fallback
}

func configInt(items map[string]configstore.ConfigItem, key string, fallback int) int {
	if item, ok := items[key]; ok {
		if value, err := strconv.Atoi(strings.TrimSpace(item.Value)); err == nil {
			return value
		}
	}
	return fallback
}
