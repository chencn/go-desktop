package display

import (
	"encoding/json"
	"strings"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
)

const (
	KeyPreferencesV2 = "display.preferences.v2"

	preferencesVersion = 2
)

type Scheme string

const (
	SchemeShadcn   Scheme = "shadcn"
	SchemeArtistic Scheme = "artistic"
)

// Profile 是单个显示方案的可持久化配置；不同方案只会归一化并使用各自支持字段。
type Profile struct {
	UIStyle     string `json:"uiStyle,omitempty"`
	BaseColor   string `json:"baseColor,omitempty"`
	ThemeColor  string `json:"themeColor,omitempty"`
	AccentColor string `json:"accentColor,omitempty"`
	ChartColor  string `json:"chartColor,omitempty"`
	IconTone    string `json:"iconTone,omitempty"`
	Menu        string `json:"menu,omitempty"`
	MenuAccent  string `json:"menuAccent,omitempty"`
	Radius      string `json:"radius,omitempty"`
	Density     string `json:"density,omitempty"`
	TextSize    string `json:"textSize,omitempty"`
	CardBorder  string `json:"cardBorder,omitempty"`
}

// PreferencesV2 是 SQLite 中 display.preferences.v2 保存的 JSON 结构。
type PreferencesV2 struct {
	Version       int                `json:"version"`
	DisplayScheme string             `json:"displayScheme"`
	ThemeMode     string             `json:"themeMode"`
	Profiles      map[string]Profile `json:"profiles"`
}

// Preferences 是当前显示方案的生效偏好，供 runtime API 以扁平字段返回给前端。
type Preferences struct {
	DisplayScheme string
	UIStyle       string
	ThemeMode     string
	BaseColor     string
	ThemeColor    string
	AccentColor   string
	ChartColor    string
	IconTone      string
	Menu          string
	MenuAccent    string
	Radius        string
	Density       string
	TextSize      string
	CardBorder    string
}

// Default 返回默认方案的生效偏好。
func Default() Preferences {
	return effectiveArtistic(DefaultV2().ThemeMode, defaultArtisticProfile())
}

// DefaultV2 返回完整 V2 默认偏好，包含 shadcn 和 artistic 两套独立 profile。
func DefaultV2() PreferencesV2 {
	return PreferencesV2{
		Version:       preferencesVersion,
		DisplayScheme: string(SchemeArtistic),
		ThemeMode:     "light",
		Profiles: map[string]Profile{
			string(SchemeShadcn):   defaultShadcnProfile(),
			string(SchemeArtistic): defaultArtisticProfile(),
		},
	}
}

// NormalizeV2 固定版本号、修正非法方案/主题，并补齐两套 profile 默认值。
func NormalizeV2(value PreferencesV2) PreferencesV2 {
	defaults := DefaultV2()
	value.Version = preferencesVersion
	value.DisplayScheme = normalizeScheme(value.DisplayScheme, defaults.DisplayScheme)
	value.ThemeMode = allowedOrDefault(value.ThemeMode, allowedThemeModes, defaults.ThemeMode)
	if value.Profiles == nil {
		value.Profiles = map[string]Profile{}
	}
	value.Profiles[string(SchemeShadcn)] = normalizeShadcnProfile(value.Profiles[string(SchemeShadcn)])
	value.Profiles[string(SchemeArtistic)] = normalizeArtisticProfile(value.Profiles[string(SchemeArtistic)])
	return value
}

// Effective 根据 DisplayScheme 从 V2 模型中计算当前方案生效偏好。
func Effective(value PreferencesV2) Preferences {
	value = NormalizeV2(value)
	if value.DisplayScheme == string(SchemeArtistic) {
		return effectiveArtistic(value.ThemeMode, value.Profiles[string(SchemeArtistic)])
	}
	return effectiveShadcn(value.ThemeMode, value.Profiles[string(SchemeShadcn)])
}

// ParseV2 解析 SQLite JSON；空值、非法 JSON 或版本不匹配都会回退到 V2 默认值。
func ParseV2(value string) PreferencesV2 {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultV2()
	}
	var parsed PreferencesV2
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return DefaultV2()
	}
	if parsed.Version != preferencesVersion {
		return DefaultV2()
	}
	return NormalizeV2(parsed)
}

// MarshalV2 输出标准化后的 V2 JSON；编码失败时回退到默认 JSON。
func MarshalV2(value PreferencesV2) string {
	value = NormalizeV2(value)
	data, err := json.Marshal(value)
	if err != nil {
		data, _ = json.Marshal(DefaultV2())
	}
	return string(data)
}

// ApplyEffectivePreferences 兼容旧扁平保存协议：只更新当前目标方案的 profile。
// 如果请求只是切换 DisplayScheme，则保留两个方案已有 profile。
func ApplyEffectivePreferences(current PreferencesV2, incoming Preferences) PreferencesV2 {
	current = NormalizeV2(current)
	targetScheme := normalizeScheme(incoming.DisplayScheme, current.DisplayScheme)
	next := current
	next.DisplayScheme = targetScheme
	next.ThemeMode = allowedOrDefault(incoming.ThemeMode, allowedThemeModes, current.ThemeMode)

	if targetScheme != current.DisplayScheme {
		return NormalizeV2(next)
	}
	if next.Profiles == nil {
		next.Profiles = map[string]Profile{}
	}
	if targetScheme == string(SchemeArtistic) {
		next.Profiles[string(SchemeArtistic)] = normalizeArtisticProfile(Profile{
			UIStyle:     incoming.UIStyle,
			BaseColor:   incoming.BaseColor,
			ThemeColor:  incoming.ThemeColor,
			AccentColor: incoming.AccentColor,
			ChartColor:  incoming.ChartColor,
			IconTone:    incoming.IconTone,
			Menu:        incoming.Menu,
			MenuAccent:  incoming.MenuAccent,
			Radius:      incoming.Radius,
			Density:     incoming.Density,
			TextSize:    incoming.TextSize,
			CardBorder:  incoming.CardBorder,
		})
		return NormalizeV2(next)
	}
	next.Profiles[string(SchemeShadcn)] = normalizeShadcnProfile(Profile{
		UIStyle:     incoming.UIStyle,
		BaseColor:   incoming.BaseColor,
		ThemeColor:  incoming.ThemeColor,
		AccentColor: incoming.AccentColor,
		ChartColor:  incoming.ChartColor,
		IconTone:    incoming.IconTone,
		Menu:        incoming.Menu,
		MenuAccent:  incoming.MenuAccent,
		Radius:      incoming.Radius,
		Density:     incoming.Density,
		TextSize:    incoming.TextSize,
		CardBorder:  incoming.CardBorder,
	})
	return NormalizeV2(next)
}

// Definitions 返回显示偏好配置项元数据；value 是完整 V2 JSON 字符串。
func Definitions() []configstore.ConfigItem {
	defaultValue := MarshalV2(DefaultV2())
	return []configstore.ConfigItem{
		{Key: KeyPreferencesV2, Category: "display", Title: "显示偏好", Description: "显示偏好 JSON。", ValueType: "string", DefaultValue: defaultValue, Value: defaultValue, SortOrder: 490},
	}
}

// defaultShadcnProfile 返回 shadcn 方案默认 profile。
func defaultShadcnProfile() Profile {
	return Profile{
		UIStyle:     "vega",
		BaseColor:   "neutral",
		ThemeColor:  "neutral",
		AccentColor: "neutral",
		ChartColor:  "neutral",
		IconTone:    "default",
		Menu:        "default",
		MenuAccent:  "subtle",
		Radius:      "default",
		Density:     "comfortable",
		TextSize:    "normal",
		CardBorder:  "visible",
	}
}

// defaultArtisticProfile 返回 artistic 方案默认 profile。
func defaultArtisticProfile() Profile {
	return Profile{
		UIStyle:     "vega",
		BaseColor:   "stone",
		ThemeColor:  "orange",
		AccentColor: "orange",
		ChartColor:  "emerald",
		IconTone:    "colorful",
		Menu:        "default",
		MenuAccent:  "bold",
		Radius:      "large",
		Density:     "comfortable",
		TextSize:    "normal",
		CardBorder:  "soft",
	}
}

// effectiveShadcn 计算 shadcn 方案生效偏好。
func effectiveShadcn(themeMode string, profile Profile) Preferences {
	profile = normalizeShadcnProfile(profile)
	return Preferences{
		DisplayScheme: string(SchemeShadcn),
		UIStyle:       profile.UIStyle,
		ThemeMode:     allowedOrDefault(themeMode, allowedThemeModes, "light"),
		BaseColor:     profile.BaseColor,
		ThemeColor:    profile.ThemeColor,
		AccentColor:   profile.AccentColor,
		ChartColor:    profile.ChartColor,
		IconTone:      profile.IconTone,
		Menu:          profile.Menu,
		MenuAccent:    profile.MenuAccent,
		Radius:        profile.Radius,
		Density:       profile.Density,
		TextSize:      profile.TextSize,
		CardBorder:    profile.CardBorder,
	}
}

// effectiveArtistic 计算 artistic 方案生效偏好。
func effectiveArtistic(themeMode string, profile Profile) Preferences {
	profile = normalizeArtisticProfile(profile)
	return Preferences{
		DisplayScheme: string(SchemeArtistic),
		UIStyle:       profile.UIStyle,
		ThemeMode:     allowedOrDefault(themeMode, allowedThemeModes, "light"),
		BaseColor:     profile.BaseColor,
		ThemeColor:    profile.ThemeColor,
		AccentColor:   profile.AccentColor,
		ChartColor:    profile.ChartColor,
		IconTone:      profile.IconTone,
		Menu:          profile.Menu,
		MenuAccent:    profile.MenuAccent,
		Radius:        profile.Radius,
		Density:       profile.Density,
		TextSize:      profile.TextSize,
		CardBorder:    profile.CardBorder,
	}
}

// normalizeShadcnProfile 只接受 shadcn 方案允许的 token 值。
func normalizeShadcnProfile(profile Profile) Profile {
	defaults := defaultShadcnProfile()
	return Profile{
		UIStyle:     allowedOrDefault(profile.UIStyle, allowedUIStyles, defaults.UIStyle),
		BaseColor:   allowedOrDefault(profile.BaseColor, allowedBaseColors, defaults.BaseColor),
		ThemeColor:  allowedOrDefault(profile.ThemeColor, allowedAccentColors, defaults.ThemeColor),
		AccentColor: allowedOrDefault(profile.AccentColor, allowedAccentColors, defaults.AccentColor),
		ChartColor:  allowedOrDefault(profile.ChartColor, allowedAccentColors, defaults.ChartColor),
		IconTone:    allowedOrDefault(profile.IconTone, allowedIconTones, defaults.IconTone),
		Menu:        allowedOrDefault(profile.Menu, allowedMenus, defaults.Menu),
		MenuAccent:  allowedOrDefault(profile.MenuAccent, allowedMenuAccents, defaults.MenuAccent),
		Radius:      allowedOrDefault(profile.Radius, allowedRadii, defaults.Radius),
		Density:     allowedOrDefault(profile.Density, allowedDensities, defaults.Density),
		TextSize:    allowedOrDefault(profile.TextSize, allowedTextSizes, defaults.TextSize),
		CardBorder:  allowedOrDefault(profile.CardBorder, allowedCardBorders, defaults.CardBorder),
	}
}

// normalizeArtisticProfile 只接受 artistic 方案允许的 token 值。
func normalizeArtisticProfile(profile Profile) Profile {
	defaults := defaultArtisticProfile()
	themeColor := allowedOrDefault(profile.ThemeColor, allowedAccentColors, defaults.ThemeColor)
	return Profile{
		UIStyle:     allowedOrDefault(profile.UIStyle, allowedUIStyles, defaults.UIStyle),
		BaseColor:   allowedOrDefault(profile.BaseColor, allowedBaseColors, defaults.BaseColor),
		ThemeColor:  themeColor,
		AccentColor: themeColor,
		ChartColor:  allowedOrDefault(profile.ChartColor, allowedAccentColors, defaults.ChartColor),
		IconTone:    allowedOrDefault(profile.IconTone, allowedIconTones, defaults.IconTone),
		Menu:        allowedOrDefault(profile.Menu, allowedMenus, defaults.Menu),
		MenuAccent:  allowedOrDefault(profile.MenuAccent, allowedMenuAccents, defaults.MenuAccent),
		Radius:      allowedOrDefault(profile.Radius, allowedRadii, defaults.Radius),
		Density:     allowedOrDefault(profile.Density, allowedDensities, defaults.Density),
		TextSize:    allowedOrDefault(profile.TextSize, allowedTextSizes, defaults.TextSize),
		CardBorder:  allowedOrDefault(profile.CardBorder, allowedCardBorders, defaults.CardBorder),
	}
}

// normalizeScheme 只允许已注册显示方案，非法值回退到 fallback 或 shadcn。
func normalizeScheme(value string, fallback string) string {
	if value == string(SchemeShadcn) || value == string(SchemeArtistic) {
		return value
	}
	if fallback == string(SchemeArtistic) {
		return fallback
	}
	return string(SchemeShadcn)
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
	allowedUIStyles     = stringSet("reka", "vega", "nova", "maia", "lyra", "mira", "luma", "sera")
	allowedThemeModes   = stringSet("light", "dark")
	allowedBaseColors   = stringSet("neutral", "stone", "zinc", "mauve", "olive", "mist", "taupe")
	allowedAccentColors = stringSet("neutral", "stone", "zinc", "mauve", "olive", "mist", "taupe", "amber", "blue", "cyan", "emerald", "fuchsia", "green", "indigo", "lime", "orange", "pink", "purple", "red", "rose", "sky", "teal", "violet", "yellow")
	allowedIconTones    = stringSet("default", "colorful")
	allowedMenus        = stringSet("default", "inverted", "default-translucent", "inverted-translucent")
	allowedMenuAccents  = stringSet("subtle", "bold")
	allowedRadii        = stringSet("default", "none", "small", "medium", "large")
	allowedDensities    = stringSet("compact", "comfortable")
	allowedTextSizes    = stringSet("small", "normal", "medium", "large")
	allowedCardBorders  = stringSet("visible", "soft", "hidden")
)
