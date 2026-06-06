// ============================================================================
// 文件: display_preferences.go
// 描述: 显示偏好配置模块
//
// 功能概述:
// - 提供显示偏好的读取、保存和默认值
// - 通过 SQLite KV 配置项持久化每一个显示偏好
// - 保持前端 typed facade，不把数据库 KV 结构泄漏到 UI 组件
// ============================================================================

package runtime

import (
	"context" // 上下文包，用于 SQLite 操作
	"fmt"     // 格式化日志消息

	"github.com/chencn/go-desktop/internal/desktopapp/display"
)

// DisplayPreferences 是前端显示偏好的 typed 快照
// 每个字段都对应 SQLite config_items 中的一条 display.* KV 配置。
type DisplayPreferences struct {
	UIStyle     string `json:"uiStyle"`     // UIStyle 保存 uiStyle 对应的数据，供当前实体的调用方读取或持久化。
	ThemeMode   string `json:"themeMode"`   // ThemeMode 保存 themeMode 对应的数据，供当前实体的调用方读取或持久化。
	BaseColor   string `json:"baseColor"`   // BaseColor 保存 baseColor 对应的数据，供当前实体的调用方读取或持久化。
	ThemeColor  string `json:"themeColor"`  // ThemeColor 保存 themeColor 对应的数据，供当前实体的调用方读取或持久化。
	AccentColor string `json:"accentColor"` // AccentColor 保存 accentColor 对应的数据，供当前实体的调用方读取或持久化。
	ChartColor  string `json:"chartColor"`  // ChartColor 保存 chartColor 对应的数据，供当前实体的调用方读取或持久化。
	IconTone    string `json:"iconTone"`    // IconTone 保存 iconTone 对应的数据，供当前实体的调用方读取或持久化。
	Menu        string `json:"menu"`        // Menu 保存 menu 对应的数据，供当前实体的调用方读取或持久化。
	MenuAccent  string `json:"menuAccent"`  // MenuAccent 保存 menuAccent 对应的数据，供当前实体的调用方读取或持久化。
	Radius      string `json:"radius"`      // Radius 保存 radius 对应的数据，供当前实体的调用方读取或持久化。
	Density     string `json:"density"`     // Density 保存 density 对应的数据，供当前实体的调用方读取或持久化。
	TextSize    string `json:"textSize"`    // TextSize 保存 textSize 对应的数据，供当前实体的调用方读取或持久化。
	CardBorder  string `json:"cardBorder"`  // CardBorder 保存 cardBorder 对应的数据，供当前实体的调用方读取或持久化。
}

// GetDisplayPreferences API 方法，返回当前显示偏好快照。
func (api *API) GetDisplayPreferences() (preferences DisplayPreferences, err error) {
	defer api.recoverError("读取显示偏好", &err)
	api.runtime.RecordLogWithSeverity("settings-trace", "GetDisplayPreferences：后端收到读取请求", "info")
	preferences = api.runtime.DisplayPreferencesSnapshot()
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("GetDisplayPreferences：后端返回 style=%q theme=%q base=%q themeColor=%q accent=%q chart=%q radius=%q density=%q cardBorder=%q",
		preferences.UIStyle,
		preferences.ThemeMode,
		preferences.BaseColor,
		preferences.ThemeColor,
		preferences.AccentColor,
		preferences.ChartColor,
		preferences.Radius,
		preferences.Density,
		preferences.CardBorder,
	), "info")
	return preferences, nil
}

// SaveDisplayPreferences API 方法，保存显示偏好到 SQLite KV 配置项。
func (api *API) SaveDisplayPreferences(preferences DisplayPreferences) (saved DisplayPreferences, err error) {
	defer api.recoverError("保存显示偏好", &err)
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("SaveDisplayPreferences：后端收到保存请求 style=%q theme=%q base=%q themeColor=%q accent=%q chart=%q radius=%q density=%q cardBorder=%q",
		preferences.UIStyle,
		preferences.ThemeMode,
		preferences.BaseColor,
		preferences.ThemeColor,
		preferences.AccentColor,
		preferences.ChartColor,
		preferences.Radius,
		preferences.Density,
		preferences.CardBorder,
	), "info")
	return api.runtime.SaveDisplayPreferences(preferences)
}

// SaveDisplayPreferences 保存显示偏好并返回标准化后的快照。
func (s *Runtime) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error) {
	preferences = normaliseDisplayPreferences(preferences)
	s.RecordLogWithSeverity("settings", "保存显示偏好：开始写入配置", "debug")

	s.lock.RLock()
	previousPreferences := s.displayPreferences
	s.lock.RUnlock()

	if err := s.saveConfigValues(context.Background(), display.Values(toDomainDisplayPreferences(preferences))); err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("保存显示偏好：写入失败：%s", err), "error")
		return previousPreferences, fmt.Errorf("保存显示偏好失败：%w", err)
	}

	s.lock.Lock()
	s.displayPreferences = preferences
	s.lock.Unlock()

	s.RecordLog("settings", "保存显示偏好：配置已保存")
	return preferences, nil
}

// DisplayPreferencesSnapshot 返回当前显示偏好副本。
func (s *Runtime) DisplayPreferencesSnapshot() DisplayPreferences {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.displayPreferences
}

// loadDisplayPreferences 从 SQLite KV 配置项加载显示偏好。
func (s *Runtime) loadDisplayPreferences() {
	preferences := defaultDisplayPreferences()
	items, err := s.configItemsByKey(context.Background())
	if err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("读取 SQLite 显示偏好失败：%s", err), "warning")
		s.displayPreferences = preferences
		return
	}
	s.displayPreferences = fromDomainDisplayPreferences(display.FromConfigItems(items, toDomainDisplayPreferences(preferences)))
}

// defaultDisplayPreferences 返回显示偏好默认值。
func defaultDisplayPreferences() DisplayPreferences {
	return fromDomainDisplayPreferences(display.Default())
}

// normaliseDisplayPreferences 封装 在后端保存和读取前端显示偏好 中的一段独立逻辑，调用方通过它复用同一业务规则。
func normaliseDisplayPreferences(preferences DisplayPreferences) DisplayPreferences {
	return fromDomainDisplayPreferences(display.Normalize(toDomainDisplayPreferences(preferences)))
}

func toDomainDisplayPreferences(value DisplayPreferences) display.Preferences {
	return display.Preferences{
		UIStyle:     value.UIStyle,
		ThemeMode:   value.ThemeMode,
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

func fromDomainDisplayPreferences(value display.Preferences) DisplayPreferences {
	return DisplayPreferences{
		UIStyle:     value.UIStyle,
		ThemeMode:   value.ThemeMode,
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
