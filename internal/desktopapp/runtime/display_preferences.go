// ============================================================================
// 文件: display_preferences.go
// 描述: 显示偏好配置模块
//
// 功能概述:
// - 提供显示偏好的读取、保存和默认值
// - 通过 SQLite JSON KV 配置项持久化显示偏好 profile
// - 保持前端 typed facade，不把数据库 KV 结构泄漏到 UI 组件
// ============================================================================

package runtime

import (
	"context" // 上下文包，用于 SQLite 操作
	"fmt"     // 格式化日志消息

	"github.com/chencn/go-desktop/internal/desktopapp/display"
)

// DisplayProfile 是单个显示方案的可持久化 profile。
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

// DisplayProfiles 保存所有平级显示方案的独立 profile。
type DisplayProfiles struct {
	Shadcn DisplayProfile `json:"shadcn"`
	AntD   DisplayProfile `json:"antd"`
}

// DisplayPreferences 是前端显示偏好的 typed 快照
// 扁平字段表示当前方案的生效偏好，Profiles 保留所有方案的可持久化 profile。
type DisplayPreferences struct {
	DisplayScheme string          `json:"displayScheme"` // DisplayScheme 保存当前显示方案。
	UIStyle       string          `json:"uiStyle"`       // UIStyle 保存当前生效组件风格。
	ThemeMode     string          `json:"themeMode"`     // ThemeMode 保存全局亮暗模式。
	BaseColor     string          `json:"baseColor"`     // BaseColor 保存当前生效基础色盘。
	ThemeColor    string          `json:"themeColor"`    // ThemeColor 保存当前生效主题色。
	AccentColor   string          `json:"accentColor"`   // AccentColor 保存当前生效强调色。
	ChartColor    string          `json:"chartColor"`    // ChartColor 保存当前生效图表色。
	IconTone      string          `json:"iconTone"`      // IconTone 保存当前生效图标颜色模式。
	Menu          string          `json:"menu"`          // Menu 保存当前生效菜单样式。
	MenuAccent    string          `json:"menuAccent"`    // MenuAccent 保存当前生效菜单强调样式。
	Radius        string          `json:"radius"`        // Radius 保存当前生效圆角。
	Density       string          `json:"density"`       // Density 保存当前生效密度。
	TextSize      string          `json:"textSize"`      // TextSize 保存当前生效字体大小。
	CardBorder    string          `json:"cardBorder"`    // CardBorder 保存当前生效卡片边框强度。
	Profiles      DisplayProfiles `json:"profiles"`      // Profiles 保存所有显示方案的独立偏好。
}

// GetDisplayPreferences API 方法，返回当前显示偏好快照。
func (api *API) GetDisplayPreferences() (preferences DisplayPreferences, err error) {
	defer api.recoverError("读取显示偏好", &err)
	if err := api.requireAuthorized(); err != nil {
		return DisplayPreferences{}, err
	}
	api.runtime.RecordLogWithSeverity("settings-trace", "GetDisplayPreferences：后端收到读取请求", "debug")
	preferences = api.runtime.DisplayPreferencesSnapshot()
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("GetDisplayPreferences：后端返回 scheme=%q style=%q theme=%q base=%q themeColor=%q accent=%q chart=%q radius=%q density=%q cardBorder=%q",
		preferences.DisplayScheme,
		preferences.UIStyle,
		preferences.ThemeMode,
		preferences.BaseColor,
		preferences.ThemeColor,
		preferences.AccentColor,
		preferences.ChartColor,
		preferences.Radius,
		preferences.Density,
		preferences.CardBorder,
	), "debug")
	return preferences, nil
}

// SaveDisplayPreferences API 方法，保存显示偏好到 SQLite JSON KV 配置项。
func (api *API) SaveDisplayPreferences(preferences DisplayPreferences) (saved DisplayPreferences, err error) {
	defer api.recoverError("保存显示偏好", &err)
	if err := api.requireAuthorized(); err != nil {
		return DisplayPreferences{}, err
	}
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("SaveDisplayPreferences：后端收到保存请求 scheme=%q style=%q theme=%q base=%q themeColor=%q accent=%q chart=%q radius=%q density=%q cardBorder=%q",
		preferences.DisplayScheme,
		preferences.UIStyle,
		preferences.ThemeMode,
		preferences.BaseColor,
		preferences.ThemeColor,
		preferences.AccentColor,
		preferences.ChartColor,
		preferences.Radius,
		preferences.Density,
		preferences.CardBorder,
	), "debug")
	return api.runtime.SaveDisplayPreferences(preferences)
}

// SaveDisplayPreferences 保存显示偏好并返回标准化后的快照。
func (s *Runtime) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error) {
	s.RecordLogWithSeverity("settings", "保存显示偏好：开始写入配置", "debug")

	s.lock.RLock()
	previousPreferences := s.displayPreferences
	current := s.displayPreferencesV2
	s.lock.RUnlock()

	next := toDomainPreferencesV2(current, preferences)
	effective := fromDomainPreferencesV2(next)
	if err := s.saveConfigValues(context.Background(), map[string]string{display.KeyPreferencesV2: display.MarshalV2(next)}); err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("保存显示偏好：写入失败：%s", err), "error")
		return previousPreferences, fmt.Errorf("保存显示偏好失败：%w", err)
	}

	s.lock.Lock()
	s.displayPreferencesV2 = next
	s.displayPreferences = effective
	s.lock.Unlock()

	s.RecordLog("settings", "保存显示偏好：配置已保存")
	return effective, nil
}

// DisplayPreferencesSnapshot 返回当前显示偏好副本。
func (s *Runtime) DisplayPreferencesSnapshot() DisplayPreferences {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.displayPreferences
}

// loadDisplayPreferences 从 SQLite JSON KV 配置项加载显示偏好。
func (s *Runtime) loadDisplayPreferences() {
	items, err := s.configItemsByKey(context.Background())
	if err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("读取 SQLite 显示偏好失败：%s", err), "warning")
		stored := display.DefaultV2()
		s.displayPreferencesV2 = stored
		s.displayPreferences = fromDomainPreferencesV2(stored)
		return
	}
	raw := ""
	if item, ok := items[display.KeyPreferencesV2]; ok {
		raw = item.Value
	}
	stored := display.ParseV2(raw)
	s.displayPreferencesV2 = stored
	s.displayPreferences = fromDomainPreferencesV2(stored)
}

// defaultDisplayPreferences 返回显示偏好默认值。
func defaultDisplayPreferences() DisplayPreferences {
	return fromDomainPreferencesV2(display.DefaultV2())
}

func toDomainPreferencesV2(current display.PreferencesV2, value DisplayPreferences) display.PreferencesV2 {
	current = display.NormalizeV2(current)
	if !hasDisplayProfiles(value.Profiles) {
		return display.ApplyEffectivePreferences(current, toDomainDisplayPreferences(value))
	}
	return display.NormalizeV2(display.PreferencesV2{
		DisplayScheme: value.DisplayScheme,
		ThemeMode:     value.ThemeMode,
		Profiles: map[string]display.Profile{
			string(display.SchemeShadcn): toDomainDisplayProfile(value.Profiles.Shadcn),
			string(display.SchemeAntD):   toDomainDisplayProfile(value.Profiles.AntD),
		},
	})
}

func toDomainDisplayPreferences(value DisplayPreferences) display.Preferences {
	return display.Preferences{
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
	}
}

func fromDomainDisplayPreferences(value display.Preferences) DisplayPreferences {
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
	}
}

func fromDomainPreferencesV2(value display.PreferencesV2) DisplayPreferences {
	value = display.NormalizeV2(value)
	effective := fromDomainDisplayPreferences(display.Effective(value))
	effective.Profiles = DisplayProfiles{
		Shadcn: fromDomainDisplayProfile(value.Profiles[string(display.SchemeShadcn)]),
		AntD:   fromDomainDisplayProfile(value.Profiles[string(display.SchemeAntD)]),
	}
	return effective
}

func toDomainDisplayProfile(value DisplayProfile) display.Profile {
	return display.Profile{
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

func fromDomainDisplayProfile(value display.Profile) DisplayProfile {
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

func hasDisplayProfiles(value DisplayProfiles) bool {
	return hasDisplayProfile(value.Shadcn) || hasDisplayProfile(value.AntD)
}

func hasDisplayProfile(value DisplayProfile) bool {
	return value.UIStyle != "" ||
		value.BaseColor != "" ||
		value.ThemeColor != "" ||
		value.AccentColor != "" ||
		value.ChartColor != "" ||
		value.IconTone != "" ||
		value.Menu != "" ||
		value.MenuAccent != "" ||
		value.Radius != "" ||
		value.Density != "" ||
		value.TextSize != "" ||
		value.CardBorder != ""
}
