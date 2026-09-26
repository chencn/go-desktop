// ============================================================================
// 文件: display_preferences.go
// 描述: 显示偏好配置模块（Element Plus 主题模型 V3）
//
// 功能概述:
// - 提供亮暗模式 / 全局尺寸 / 液态玻璃光学（极光舞台、折射风格、光强）的读取、保存和默认值
// - 通过 SQLite JSON KV 配置项持久化显示偏好
// - 保持前端 typed facade，不把数据库 KV 结构泄漏到 UI 组件
// ============================================================================

package runtime

import (
	"context"
	"fmt"

	"github.com/chencn/go-desktop/internal/desktopapp/display"
)

// DisplayPreferences 是前端显示偏好的 typed 快照（含液态玻璃光学三项）。
type DisplayPreferences struct {
	ThemeMode string `json:"themeMode"` // ThemeMode 是当前亮暗模式。
	Size      string `json:"size"`      // Size 是 Element Plus 全局组件尺寸。
	// Backdrop 是极光折射流光背景开关，前端写成 <html class="has-backdrop">。
	Backdrop bool `json:"backdrop"`
	// LgStyle 是液态玻璃折射风格：fresnel / frosted / sheen。
	LgStyle string `json:"lgStyle"`
	// LgIntensity 是 30-100 的折射光强百分比，前端换算成 --lg-intensity-factor。
	LgIntensity int `json:"lgIntensity"`
}

// GetDisplayPreferences API 方法，返回当前显示偏好快照。
func (api *API) GetDisplayPreferences() (preferences DisplayPreferences, err error) {
	defer api.recoverError("读取显示偏好", &err)
	if err := api.requireAuthorized(); err != nil {
		return DisplayPreferences{}, err
	}
	api.runtime.RecordLogWithSeverity("settings-trace", "GetDisplayPreferences：后端收到读取请求", "debug")
	preferences = api.runtime.DisplayPreferencesSnapshot()
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("GetDisplayPreferences：后端返回 theme=%q size=%q backdrop=%t lgStyle=%q lgIntensity=%d",
		preferences.ThemeMode,
		preferences.Size,
		preferences.Backdrop,
		preferences.LgStyle,
		preferences.LgIntensity,
	), "debug")
	return preferences, nil
}

// SaveDisplayPreferences API 方法，保存显示偏好到 SQLite JSON KV 配置项。
func (api *API) SaveDisplayPreferences(preferences DisplayPreferences) (saved DisplayPreferences, err error) {
	defer api.recoverError("保存显示偏好", &err)
	if err := api.requireAuthorized(); err != nil {
		return DisplayPreferences{}, err
	}
	api.runtime.RecordLogWithSeverity("settings-trace", fmt.Sprintf("SaveDisplayPreferences：后端收到保存请求 theme=%q size=%q backdrop=%t lgStyle=%q lgIntensity=%d",
		preferences.ThemeMode,
		preferences.Size,
		preferences.Backdrop,
		preferences.LgStyle,
		preferences.LgIntensity,
	), "debug")
	return api.runtime.SaveDisplayPreferences(preferences)
}

// SaveDisplayPreferences 保存 V3 JSON 配置项并返回标准化后的快照。
func (s *Runtime) SaveDisplayPreferences(preferences DisplayPreferences) (DisplayPreferences, error) {
	s.RecordLogWithSeverity("settings", "保存显示偏好：开始写入配置", "debug")

	s.lock.RLock()
	previousPreferences := s.displayPreferences
	s.lock.RUnlock()

	next := display.NormalizeV3(display.PreferencesV3{
		ThemeMode:   preferences.ThemeMode,
		Size:        preferences.Size,
		Backdrop:    preferences.Backdrop,
		LgStyle:     preferences.LgStyle,
		LgIntensity: preferences.LgIntensity,
	})
	effective := fromDomainPreferencesV3(next)
	if err := s.saveConfigValues(context.Background(), map[string]string{display.KeyPreferencesV3: display.MarshalV3(next)}); err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("保存显示偏好：写入失败：%s", err), "error")
		return previousPreferences, fmt.Errorf("保存显示偏好失败：%w", err)
	}

	s.lock.Lock()
	s.displayPreferences = effective
	s.lock.Unlock()

	s.RecordLog("settings", "保存显示偏好：配置已保存")
	return effective, nil
}

// DisplayPreferencesSnapshot 返回当前显示偏好副本；它来自内存中的 V3 配置。
func (s *Runtime) DisplayPreferencesSnapshot() DisplayPreferences {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.displayPreferences
}

// loadDisplayPreferences 从 SQLite JSON KV 配置项加载显示偏好；缺失或非法 JSON 会回到 V3 默认值。
func (s *Runtime) loadDisplayPreferences() {
	items, err := s.configItemsByKey(context.Background())
	if err != nil {
		s.RecordLogWithSeverity("settings", fmt.Sprintf("读取 SQLite 显示偏好失败：%s", err), "warning")
		s.displayPreferences = fromDomainPreferencesV3(display.DefaultV3())
		return
	}
	raw := ""
	if item, ok := items[display.KeyPreferencesV3]; ok {
		raw = item.Value
	}
	s.displayPreferences = fromDomainPreferencesV3(display.ParseV3(raw))
}

// defaultDisplayPreferences 返回默认显示偏好，不暴露数据库 KV 结构。
func defaultDisplayPreferences() DisplayPreferences {
	return fromDomainPreferencesV3(display.DefaultV3())
}

// fromDomainPreferencesV3 转换领域模型为前端快照。
func fromDomainPreferencesV3(value display.PreferencesV3) DisplayPreferences {
	return DisplayPreferences{
		ThemeMode:   value.ThemeMode,
		Size:        value.Size,
		Backdrop:    value.Backdrop,
		LgStyle:     value.LgStyle,
		LgIntensity: value.LgIntensity,
	}
}
