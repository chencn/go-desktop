package settings

import (
	"strconv"
	"strings"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

const (
	KeyUpdateSource             = "update.source"
	KeyGitHubProxyBase          = "github.proxy_base"
	KeyUpdateCheckIntervalHours = "update.check_interval_hours"
	KeyWindowMinimizeToTray     = "window.minimize_to_tray"
	KeyWindowAlwaysOnTop        = "window.always_on_top"
	KeyLogRetentionDays         = "log.retention_days"
	KeyLogLevel                 = "log.level"
	KeyStartupAutoLaunch        = "startup.auto_launch"
	KeyStartupDesktopShortcut   = "startup.create_desktop_shortcut"
	KeyStartupHiddenToTray      = "startup.launch_hidden_to_tray"
)

const defaultLogLevel = "info"

// Settings 是业务层设置模型；持久化时会拆成 config_items 字符串 KV。
type Settings struct {
	UpdateSource             string
	GitHubProxyBase          string
	UpdateCheckIntervalHours int
	MinimizeToTray           bool
	AlwaysOnTop              bool
	LogRetentionDays         int
	LogLevel                 string
	AutoLaunch               bool
	CreateDesktopShortcut    bool
	LaunchHiddenToTray       bool
}

// Default 返回由 metadata 生成链和本包本地默认值共同决定的启动默认设置。
func Default() Settings {
	return Settings{
		UpdateSource:             metadata.DefaultUpdateSource,
		GitHubProxyBase:          metadata.DefaultGitHubProxyBase,
		UpdateCheckIntervalHours: metadata.DefaultUpdateCheckIntervalHours,
		MinimizeToTray:           metadata.DefaultMinimizeToTray,
		AlwaysOnTop:              metadata.DefaultAlwaysOnTop,
		LogRetentionDays:         metadata.DefaultLogRetentionDays,
		LogLevel:                 defaultLogLevel,
		AutoLaunch:               metadata.DefaultAutoLaunch,
		CreateDesktopShortcut:    metadata.DefaultCreateDesktopShortcut,
		LaunchHiddenToTray:       metadata.DefaultLaunchHiddenToTray,
	}
}

// Normalize 统一处理前端输入和数据库读取结果中的非法枚举、空仓库信息和特殊保留天数。
// LogRetentionDays=-1 表示永不清理；0 或小于 -1 回退到 metadata 默认值。
func Normalize(value Settings) Settings {
	value.UpdateSource = NormalizeUpdateSource(value.UpdateSource)
	value.UpdateCheckIntervalHours = NormalizeUpdateCheckIntervalHours(value.UpdateCheckIntervalHours)
	if value.LogRetentionDays == 0 || value.LogRetentionDays < -1 {
		value.LogRetentionDays = metadata.DefaultLogRetentionDays
	}
	value.LogLevel = NormalizeLogLevel(value.LogLevel)
	return value
}

// NormalizeUpdateSource 只允许 local，其余输入都回退到 github。
func NormalizeUpdateSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "local":
		return "local"
	default:
		return "github"
	}
}

// NormalizeUpdateCheckIntervalHours 只允许产品约定的自动检查间隔。
func NormalizeUpdateCheckIntervalHours(value int) int {
	switch value {
	case 1, 3, 6, 12:
		return value
	default:
		return metadata.DefaultUpdateCheckIntervalHours
	}
}

// NormalizeLogLevel 标准化最小记录级别；未知值回退到 info。
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

// FromConfigItems 从 SQLite 配置项恢复 typed 设置，缺失或解析失败的项使用 base。
func FromConfigItems(items map[string]configstore.ConfigItem, base Settings) Settings {
	base.UpdateSource = configString(items, KeyUpdateSource, base.UpdateSource)
	base.GitHubProxyBase = configString(items, KeyGitHubProxyBase, base.GitHubProxyBase)
	base.UpdateCheckIntervalHours = configInt(items, KeyUpdateCheckIntervalHours, base.UpdateCheckIntervalHours)
	base.MinimizeToTray = configBool(items, KeyWindowMinimizeToTray, base.MinimizeToTray)
	base.AlwaysOnTop = configBool(items, KeyWindowAlwaysOnTop, base.AlwaysOnTop)
	base.LogRetentionDays = configInt(items, KeyLogRetentionDays, base.LogRetentionDays)
	base.LogLevel = configString(items, KeyLogLevel, base.LogLevel)
	base.AutoLaunch = configBool(items, KeyStartupAutoLaunch, base.AutoLaunch)
	base.CreateDesktopShortcut = configBool(items, KeyStartupDesktopShortcut, base.CreateDesktopShortcut)
	base.LaunchHiddenToTray = configBool(items, KeyStartupHiddenToTray, base.LaunchHiddenToTray)
	return Normalize(base)
}

// Values 把 typed 设置转换为 SQLite 字符串 KV，调用前后都会保持归一化。
func Values(value Settings) map[string]string {
	value = Normalize(value)
	return map[string]string{
		KeyUpdateSource:             value.UpdateSource,
		KeyGitHubProxyBase:          value.GitHubProxyBase,
		KeyUpdateCheckIntervalHours: strconv.Itoa(value.UpdateCheckIntervalHours),
		KeyWindowMinimizeToTray:     strconv.FormatBool(value.MinimizeToTray),
		KeyWindowAlwaysOnTop:        strconv.FormatBool(value.AlwaysOnTop),
		KeyLogRetentionDays:         strconv.Itoa(value.LogRetentionDays),
		KeyLogLevel:                 value.LogLevel,
		KeyStartupAutoLaunch:        strconv.FormatBool(value.AutoLaunch),
		KeyStartupDesktopShortcut:   strconv.FormatBool(value.CreateDesktopShortcut),
		KeyStartupHiddenToTray:      strconv.FormatBool(value.LaunchHiddenToTray),
	}
}

// Definitions 返回设置配置项元数据；EnsureConfigItems 不会用这些默认值覆盖已有用户 value。
func Definitions() []configstore.ConfigItem {
	defaults := Default()
	return []configstore.ConfigItem{
		{Key: KeyUpdateSource, Category: "update", Title: "更新源", Description: "选择 GitHub Release 或本地静态 manifest 作为唯一更新检查来源。", ValueType: "string", DefaultValue: defaults.UpdateSource, Value: defaults.UpdateSource, SortOrder: 90},
		{Key: KeyGitHubProxyBase, Category: "github", Title: "GitHub API 代理", Description: "为空时直接使用 GitHub 官方 API。", ValueType: "string", DefaultValue: defaults.GitHubProxyBase, Value: defaults.GitHubProxyBase, SortOrder: 30},
		{Key: KeyUpdateCheckIntervalHours, Category: "update", Title: "检查间隔", Description: "自动检查 GitHub Release 的时间间隔。", ValueType: "int", DefaultValue: strconv.Itoa(defaults.UpdateCheckIntervalHours), Value: strconv.Itoa(defaults.UpdateCheckIntervalHours), SortOrder: 100},
		{Key: KeyWindowMinimizeToTray, Category: "window", Title: "关闭到系统托盘", Description: "点击关闭按钮时隐藏窗口到系统托盘。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.MinimizeToTray), Value: strconv.FormatBool(defaults.MinimizeToTray), SortOrder: 200},
		{Key: KeyWindowAlwaysOnTop, Category: "window", Title: "窗口置顶", Description: "窗口显示时保持在其他窗口上方。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.AlwaysOnTop), Value: strconv.FormatBool(defaults.AlwaysOnTop), SortOrder: 210},
		{Key: KeyLogRetentionDays, Category: "log", Title: "日志保留周期", Description: "每日文件日志自动清理周期，-1 表示永不清理。", ValueType: "int", DefaultValue: strconv.Itoa(defaults.LogRetentionDays), Value: strconv.Itoa(defaults.LogRetentionDays), SortOrder: 300},
		{Key: KeyLogLevel, Category: "log", Title: "日志级别", Description: "控制运行日志最小记录级别；debug 用于定位问题，error 和 panic 始终记录。", ValueType: "string", DefaultValue: defaults.LogLevel, Value: defaults.LogLevel, SortOrder: 310},
		{Key: KeyStartupAutoLaunch, Category: "startup", Title: "开机自启", Description: "登录系统后自动启动应用。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.AutoLaunch), Value: strconv.FormatBool(defaults.AutoLaunch), SortOrder: 400},
		{Key: KeyStartupDesktopShortcut, Category: "startup", Title: "创建桌面快捷图标", Description: "在当前用户桌面创建应用快捷方式。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.CreateDesktopShortcut), Value: strconv.FormatBool(defaults.CreateDesktopShortcut), SortOrder: 410},
		{Key: KeyStartupHiddenToTray, Category: "startup", Title: "开机自启时隐藏到托盘", Description: "仅在开机自启启动时隐藏主窗口，手动启动仍显示界面。", ValueType: "bool", DefaultValue: strconv.FormatBool(defaults.LaunchHiddenToTray), Value: strconv.FormatBool(defaults.LaunchHiddenToTray), SortOrder: 420},
	}
}

// configString 读取非空字符串配置，空字符串保留 fallback。
func configString(items map[string]configstore.ConfigItem, key string, fallback string) string {
	if item, ok := items[key]; ok {
		if value := strings.TrimSpace(item.Value); value != "" {
			return value
		}
	}
	return fallback
}

// configBool 读取 bool 配置，解析失败时保留 fallback。
func configBool(items map[string]configstore.ConfigItem, key string, fallback bool) bool {
	if item, ok := items[key]; ok {
		if value, err := strconv.ParseBool(strings.TrimSpace(item.Value)); err == nil {
			return value
		}
	}
	return fallback
}

// configInt 读取 int 配置，解析失败时保留 fallback。
func configInt(items map[string]configstore.ConfigItem, key string, fallback int) int {
	if item, ok := items[key]; ok {
		if value, err := strconv.Atoi(strings.TrimSpace(item.Value)); err == nil {
			return value
		}
	}
	return fallback
}
