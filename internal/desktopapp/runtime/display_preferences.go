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
	"context"
	"fmt"

	"github.com/chencn/go-desktop/internal/desktopapp/display"
)

// DisplayProfile 是单个显示方案的可持久化 profile；不同方案只使用各自支持的字段。
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

// DisplayProfiles 保存所有平级显示方案的独立 profile，用于切换方案时保留各自偏好。
type DisplayProfiles struct {
	Shadcn DisplayProfile `json:"shadcn"`
	AntD   DisplayProfile `json:"antd"`
}

// DisplayPreferences 是前端显示偏好的 typed 快照。
// 扁平字段表示当前方案的生效偏好；Profiles 是完整 V2 JSON 持久化模型的前端可见副本。
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

// SaveDisplayPreferences 保存 V2 JSON 配置项并返回当前方案的标准化生效快照。
// 没带 Profiles 的旧前端请求只更新当前方案；带 Profiles 时按完整 V2 模型覆盖两个方案。
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

// DisplayPreferencesSnapshot 返回当前显示偏好副本；它来自内存中的 V2 配置生效结果。
func (s *Runtime) DisplayPreferencesSnapshot() DisplayPreferences {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.displayPreferences
}

// loadDisplayPreferences 从 SQLite JSON KV 配置项加载显示偏好；缺失或非法 JSON 会回到 V2 默认值。
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

// defaultDisplayPreferences 返回当前默认方案的生效偏好，不暴露数据库 KV 结构。
func defaultDisplayPreferences() DisplayPreferences {
	return fromDomainPreferencesV2(display.DefaultV2())
}

// toDomainPreferencesV2 合并前端请求和当前 V2 状态，兼容旧的扁平字段保存协议。
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

// toDomainDisplayPreferences 只转换当前方案的扁平生效字段。
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

// fromDomainDisplayPreferences 只转换当前方案的扁平生效字段。
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

// fromDomainPreferencesV2 同时返回当前方案生效值和所有方案 profile。
func fromDomainPreferencesV2(value display.PreferencesV2) DisplayPreferences {
	value = display.NormalizeV2(value)
	effective := fromDomainDisplayPreferences(display.Effective(value))
	effective.Profiles = DisplayProfiles{
		Shadcn: fromDomainDisplayProfile(value.Profiles[string(display.SchemeShadcn)]),
		AntD:   fromDomainDisplayProfile(value.Profiles[string(display.SchemeAntD)]),
	}
	return effective
}

// toDomainDisplayProfile 转换单个方案 profile；字段合法性由 display.NormalizeV2 处理。
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

// fromDomainDisplayProfile 转换单个方案 profile。
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

// hasDisplayProfiles 判断请求是否携带完整 profile 模型，用于区分旧扁平保存协议。
func hasDisplayProfiles(value DisplayProfiles) bool {
	return hasDisplayProfile(value.Shadcn) || hasDisplayProfile(value.AntD)
}

// hasDisplayProfile 判断单个 profile 是否包含任意可保存字段。
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
