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
	"context"
	"errors"
	"fmt"

	appsettings "github.com/chencn/go-desktop/internal/desktopapp/settings"
)

// GetSettings API 方法，返回当前设置快照
// 返回的是副本，修改不会影响内部状态
func (api *API) GetSettings() (settings Settings, err error) {
	defer api.recoverError("读取设置", &err)
	if err := api.requireAuthorized(); err != nil {
		return Settings{}, err
	}
	api.runtime.RecordLogWithSeverity("settings-trace", "GetSettings：后端收到读取请求", "debug")
	settings = api.runtime.SettingsSnapshot()
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("GetSettings：后端返回 source=%q proxy=%q autoLaunch=%t shortcut=%t tray=%t top=%t interval=%d retention=%d logLevel=%q",
		settings.UpdateSource,
		settings.GitHubProxyBase,
		settings.AutoLaunch,
		settings.CreateDesktopShortcut,
		settings.MinimizeToTray,
		settings.AlwaysOnTop,
		settings.UpdateCheckIntervalHours,
		settings.LogRetentionDays,
		settings.LogLevel,
	), "debug")
	return settings, nil
}

// SaveSettings API 方法，保存标准化后的设置，并同步相关桌面集成。
func (api *API) SaveSettings(settings Settings) (saved Settings, err error) {
	defer api.recoverError("保存设置", &err)
	if err := api.requireAuthorized(); err != nil {
		return Settings{}, err
	}
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("SaveSettings：后端收到保存请求 source=%q proxy=%q autoLaunch=%t shortcut=%t tray=%t top=%t interval=%d retention=%d logLevel=%q",
		settings.UpdateSource,
		settings.GitHubProxyBase,
		settings.AutoLaunch,
		settings.CreateDesktopShortcut,
		settings.MinimizeToTray,
		settings.AlwaysOnTop,
		settings.UpdateCheckIntervalHours,
		settings.LogRetentionDays,
		settings.LogLevel,
	), "debug")
	return api.runtime.SaveSettings(settings)
}

// SaveSettings 先写 SQLite KV，再更新内存快照和日志级别，最后同步自启动/桌面快捷方式。
// 系统集成失败时会回滚 SQLite、内存快照和日志级别，避免前端看到已保存但外部状态未同步的配置。
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
	if previousSettings.AlwaysOnTop != settings.AlwaysOnTop {
		s.applyMainWindowAlwaysOnTop(settings.AlwaysOnTop)
	}

	if err := s.applyChangedStartupIntegrations(previousSettings, settings); err != nil {
		rollbackErr := s.rollbackSettingsAfterIntegrationFailure(previousSettings)
		return previousSettings, fmt.Errorf("同步设置系统集成失败：%w", errors.Join(err, rollbackErr))
	}
	if previousSettings.LogRetentionDays != settings.LogRetentionDays {
		go s.cleanupExpiredLogFiles(context.Background(), settings.LogRetentionDays)
	}
	s.RecordLogWithSeverity("settings", "保存设置：系统集成同步完成", "debug")
	s.RecordLog("settings", "保存设置：配置已保存")
	return settings, nil
}

func (s *Runtime) rollbackSettingsAfterIntegrationFailure(previous Settings) error {
	var rollbackErr error
	if err := s.saveConfigValues(context.Background(), appsettings.Values(toDomainSettings(previous))); err != nil {
		rollbackErr = fmt.Errorf("回滚设置持久化失败：%w", err)
		s.RecordLogWithSeverity("settings", rollbackErr.Error(), "error")
	}
	s.lock.Lock()
	s.settings = previous
	s.lock.Unlock()
	if s.logLevel != nil {
		s.logLevel.Set(SlogLevelFromLogLevel(previous.LogLevel))
	}
	s.applyMainWindowAlwaysOnTop(previous.AlwaysOnTop)
	s.RecordLogWithSeverity("settings", "保存设置失败：已回滚内存设置和 SQLite 配置", "warning")
	return rollbackErr
}

// loadSettings 从 SQLite 配置项加载设置；读取失败时保留默认值并允许 Runtime 继续启动。
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

// defaultSettings 返回 metadata 和 settings domain 层定义的默认设置。
func defaultSettings() Settings {
	return fromDomainSettings(appsettings.Default())
}

// normaliseSettings 复用 settings domain 层的枚举、默认值和区间归一化规则。
func normaliseSettings(settings Settings) Settings {
	return fromDomainSettings(appsettings.Normalize(toDomainSettings(settings)))
}

// normaliseUpdateCheckIntervalHours 只接受 settings domain 层声明的自动检查间隔。
func normaliseUpdateCheckIntervalHours(value int) int {
	return appsettings.NormalizeUpdateCheckIntervalHours(value)
}

// SettingsSnapshot 返回当前内存设置副本；调用方不能通过返回值修改 Runtime 状态。
func (s *Runtime) SettingsSnapshot() Settings {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.settings
}

// toDomainSettings 把 runtime API DTO 转成 settings domain 模型，持久化前仍会由 domain 层归一化。
func toDomainSettings(value Settings) appsettings.Settings {
	return appsettings.Settings{
		UpdateSource:             value.UpdateSource,
		GitHubProxyBase:          value.GitHubProxyBase,
		UpdateCheckIntervalHours: value.UpdateCheckIntervalHours,
		MinimizeToTray:           value.MinimizeToTray,
		AlwaysOnTop:              value.AlwaysOnTop,
		LogRetentionDays:         value.LogRetentionDays,
		LogLevel:                 value.LogLevel,
		AutoLaunch:               value.AutoLaunch,
		CreateDesktopShortcut:    value.CreateDesktopShortcut,
		LaunchHiddenToTray:       value.LaunchHiddenToTray,
	}
}

// fromDomainSettings 把 settings domain 模型转回 Wails API 暴露的 DTO。
func fromDomainSettings(value appsettings.Settings) Settings {
	return Settings{
		UpdateSource:             value.UpdateSource,
		GitHubProxyBase:          value.GitHubProxyBase,
		UpdateCheckIntervalHours: value.UpdateCheckIntervalHours,
		MinimizeToTray:           value.MinimizeToTray,
		AlwaysOnTop:              value.AlwaysOnTop,
		LogRetentionDays:         value.LogRetentionDays,
		LogLevel:                 value.LogLevel,
		AutoLaunch:               value.AutoLaunch,
		CreateDesktopShortcut:    value.CreateDesktopShortcut,
		LaunchHiddenToTray:       value.LaunchHiddenToTray,
	}
}
