// ============================================================================
// 文件: settings.go
// 描述: 设置管理模块
//
// 功能概述:
// - 提供设置的读取、保存、加载功能
// - 设置以 SQLite KV 配置项持久化
// - 支持默认值和参数校验
// - 自动清理过期日志
// ============================================================================

package runtime

import (
	"context" // 上下文包，用于数据库操作
	"fmt"     // 格式化字符串

	appsettings "github.com/chencn/go-desktop/internal/desktopapp/settings"
)

// GetSettings API 方法，返回当前设置快照
// 返回的是副本，修改不会影响内部状态
func (api *API) GetSettings() (settings Settings, err error) {
	defer api.recoverError("读取设置", &err)
	api.runtime.RecordLogWithSeverity("settings-trace", "GetSettings：后端收到读取请求", "info")
	settings = api.runtime.SettingsSnapshot()
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("GetSettings：后端返回 owner=%q repo=%q autoLaunch=%t shortcut=%t tray=%t interval=%d retention=%d logLevel=%q",
		settings.GitHubOwner,
		settings.GitHubRepo,
		settings.AutoLaunch,
		settings.CreateDesktopShortcut,
		settings.MinimizeToTray,
		settings.UpdateCheckIntervalHours,
		settings.LogRetentionDays,
		settings.LogLevel,
	), "info")
	return settings, nil
}

// SaveSettings API 方法，保存设置到内存和磁盘
// 参数:
//   - settings: 要保存的设置
//
// 返回:
//   - Settings: 保存后的设置（经过标准化处理）
func (api *API) SaveSettings(settings Settings) (saved Settings, err error) {
	defer api.recoverError("保存设置", &err)
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("SaveSettings：后端收到保存请求 owner=%q repo=%q autoLaunch=%t shortcut=%t tray=%t interval=%d retention=%d logLevel=%q",
		settings.GitHubOwner,
		settings.GitHubRepo,
		settings.AutoLaunch,
		settings.CreateDesktopShortcut,
		settings.MinimizeToTray,
		settings.UpdateCheckIntervalHours,
		settings.LogRetentionDays,
		settings.LogLevel,
	), "info")
	return api.runtime.SaveSettings(settings)
}

// SaveSettings 修改 读写用户设置并同步自启动、快捷方式等桌面副作用 管理的状态、文件或外部副作用，并把失败原因向上返回。
func (s *Runtime) SaveSettings(settings Settings) (Settings, error) {
	settings = normaliseSettings(settings)
	s.RecordLogWithSeverity("settings", "保存设置：开始写入配置", "debug")

	s.lock.RLock()
	previousSettings := s.settings
	s.lock.RUnlock()

	if err := s.saveConfigValues(context.Background(), appsettings.Values(toDomainSettings(settings))); err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("保存设置失败：%s", err), "error")
		return previousSettings, fmt.Errorf("保存设置失败：%w", err)
	}

	s.lock.Lock()
	s.settings = settings
	s.lock.Unlock()
	if s.logLevel != nil {
		s.logLevel.Set(SlogLevelFromLogLevel(settings.LogLevel))
	}

	if previousSettings.LogRetentionDays != settings.LogRetentionDays {
		go s.cleanupExpiredLogFiles(context.Background(), settings.LogRetentionDays)
	}
	if err := s.applyChangedStartupIntegrations(previousSettings, settings); err != nil {
		return settings, fmt.Errorf("同步设置系统集成失败：%w", err)
	}
	s.RecordLogWithSeverity("settings", "保存设置：系统集成同步完成", "debug")
	s.RecordLog("settings", "保存设置：配置已保存")
	return settings, nil
}

// loadSettings 读取、解析或归一化 读写用户设置并同步自启动、快捷方式等桌面副作用 需要的数据，并把结果返回给调用方。
func (s *Runtime) loadSettings() {
	settings := defaultSettings()
	items, err := s.configItemsByKey(context.Background())
	if err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("读取 SQLite 设置失败：%s", err), "warning")
		s.settings = settings
		return
	}
	s.settings = fromDomainSettings(appsettings.FromConfigItems(items, toDomainSettings(settings)))
}

// defaultSettings 读取、解析或归一化 读写用户设置并同步自启动、快捷方式等桌面副作用 需要的数据，并把结果返回给调用方。
func defaultSettings() Settings {
	return fromDomainSettings(appsettings.Default())
}

// normaliseSettings 封装 读写用户设置并同步自启动、快捷方式等桌面副作用 中的一段独立逻辑，调用方通过它复用同一业务规则。
func normaliseSettings(settings Settings) Settings {
	return fromDomainSettings(appsettings.Normalize(toDomainSettings(settings)))
}

// normaliseUpdateCheckIntervalHours 封装 读写用户设置并同步自启动、快捷方式等桌面副作用 中的一段独立逻辑，调用方通过它复用同一业务规则。
func normaliseUpdateCheckIntervalHours(value int) int {
	return appsettings.NormalizeUpdateCheckIntervalHours(value)
}

// SettingsSnapshot 修改 读写用户设置并同步自启动、快捷方式等桌面副作用 管理的状态、文件或外部副作用，并把失败原因向上返回。
func (s *Runtime) SettingsSnapshot() Settings {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.settings
}

// settingsFromConfigItems 修改 读写用户设置并同步自启动、快捷方式等桌面副作用 管理的状态、文件或外部副作用，并把失败原因向上返回。
func toDomainSettings(value Settings) appsettings.Settings {
	return appsettings.Settings{
		GitHubOwner:              value.GitHubOwner,
		GitHubRepo:               value.GitHubRepo,
		GitHubProxyBase:          value.GitHubProxyBase,
		UpdateCheckIntervalHours: value.UpdateCheckIntervalHours,
		MinimizeToTray:           value.MinimizeToTray,
		LogRetentionDays:         value.LogRetentionDays,
		LogLevel:                 value.LogLevel,
		AutoLaunch:               value.AutoLaunch,
		CreateDesktopShortcut:    value.CreateDesktopShortcut,
		LaunchHiddenToTray:       value.LaunchHiddenToTray,
	}
}

func fromDomainSettings(value appsettings.Settings) Settings {
	return Settings{
		GitHubOwner:              value.GitHubOwner,
		GitHubRepo:               value.GitHubRepo,
		GitHubProxyBase:          value.GitHubProxyBase,
		UpdateCheckIntervalHours: value.UpdateCheckIntervalHours,
		MinimizeToTray:           value.MinimizeToTray,
		LogRetentionDays:         value.LogRetentionDays,
		LogLevel:                 value.LogLevel,
		AutoLaunch:               value.AutoLaunch,
		CreateDesktopShortcut:    value.CreateDesktopShortcut,
		LaunchHiddenToTray:       value.LaunchHiddenToTray,
	}
}
