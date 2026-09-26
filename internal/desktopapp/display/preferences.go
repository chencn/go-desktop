// ============================================================================
// 文件: preferences.go
// 描述: 显示偏好领域模型（Element Plus 主题模型 V3）
//
// 功能概述:
// - 提供亮暗模式 / 全局尺寸 / 液态玻璃光学（极光舞台、折射风格、光强）五项偏好的默认值、归一化和序列化
// - 通过 SQLite JSON KV 配置项持久化，旧版本 key（v1/v2）不再读取
// - 玻璃三项是 V3 结构的追加字段：老数据缺字段时按默认值补齐，不回退主题与尺寸
// ============================================================================

package display

import (
	"encoding/json"
	"strings"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
)

const (
	KeyPreferencesV3 = "display.preferences.v3"

	preferencesVersion = 3

	// DefaultGlassStyle 是设计稿默认档：自然高透（Fresnel）。
	DefaultGlassStyle = "fresnel"
	// GlassIntensityMin / GlassIntensityMax 是设计稿光强滑块的合法区间（百分比）。
	GlassIntensityMin = 30
	GlassIntensityMax = 100
	// DefaultGlassIntensity 同时是「旧数据没有该字段」时的补齐值，因此不能用区间下界。
	DefaultGlassIntensity = 75
)

// PreferencesV3 是 SQLite 中 display.preferences.v3 保存的 JSON 结构。
type PreferencesV3 struct {
	Version   int    `json:"version"`
	ThemeMode string `json:"themeMode"`
	Size      string `json:"size"`
	// Backdrop 开启极光折射流光背景（<html class="has-backdrop">）。
	Backdrop bool `json:"backdrop"`
	// LgStyle 是液态玻璃折射风格：fresnel / frosted / sheen，写进 <html data-lg-style>。
	LgStyle string `json:"lgStyle"`
	// LgIntensity 是折射光强百分比，材质层把它换算成 --lg-intensity-factor 系数。
	LgIntensity int `json:"lgIntensity"`
}

// DefaultV3 返回 V3 默认偏好；主色固定 apple-blue，由 CSS 定义。
func DefaultV3() PreferencesV3 {
	return PreferencesV3{
		Version:     preferencesVersion,
		ThemeMode:   "light",
		Size:        "default",
		Backdrop:    false,
		LgStyle:     DefaultGlassStyle,
		LgIntensity: DefaultGlassIntensity,
	}
}

// NormalizeV3 固定版本号，并把非法值回退到默认。
func NormalizeV3(value PreferencesV3) PreferencesV3 {
	defaults := DefaultV3()
	value.Version = preferencesVersion
	value.ThemeMode = allowedOrDefault(value.ThemeMode, allowedThemeModes, defaults.ThemeMode)
	value.Size = allowedOrDefault(value.Size, allowedSizes, defaults.Size)
	value.LgStyle = allowedOrDefault(value.LgStyle, allowedGlassStyles, defaults.LgStyle)
	// 光强只接受闭区间内的值：0（老数据缺字段）和越界值都回到默认档，避免材质层拿到 0 系数变成全透明晶体。
	if value.LgIntensity < GlassIntensityMin || value.LgIntensity > GlassIntensityMax {
		value.LgIntensity = defaults.LgIntensity
	}
	return value
}

// ParseV3 解析 SQLite JSON；空值、非法 JSON 或版本不匹配都会回退到 V3 默认值。
func ParseV3(value string) PreferencesV3 {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultV3()
	}
	var parsed PreferencesV3
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return DefaultV3()
	}
	if parsed.Version != preferencesVersion {
		return DefaultV3()
	}
	return NormalizeV3(parsed)
}

// MarshalV3 输出标准化后的 V3 JSON；编码失败时回退到默认 JSON。
func MarshalV3(value PreferencesV3) string {
	value = NormalizeV3(value)
	data, err := json.Marshal(value)
	if err != nil {
		data, _ = json.Marshal(DefaultV3())
	}
	return string(data)
}

// Definitions 返回显示偏好配置项元数据；value 是完整 V3 JSON 字符串。
func Definitions() []configstore.ConfigItem {
	defaultValue := MarshalV3(DefaultV3())
	return []configstore.ConfigItem{
		{Key: KeyPreferencesV3, Category: "display", Title: "显示偏好", Description: "显示偏好 JSON：亮暗模式、控件尺寸与液态玻璃光学（极光舞台 / 折射风格 / 光强）。", ValueType: "string", DefaultValue: defaultValue, Value: defaultValue, SortOrder: 490},
	}
}

// allowedOrDefault 校验枚举值，非法值回退到 fallback。
func allowedOrDefault(value string, allowed map[string]struct{}, fallback string) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return fallback
}

// stringSet 构造枚举校验表。
func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

var (
	allowedThemeModes = stringSet("light", "dark")
	allowedSizes      = stringSet("large", "default", "small")
	// allowedGlassStyles 对应材质层的三种折射风格（styles/liquid-glass.css 的 html[data-lg-style]）。
	allowedGlassStyles = stringSet("fresnel", "frosted", "sheen")
)
