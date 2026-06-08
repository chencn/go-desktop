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
	SchemeShadcn Scheme = "shadcn"
	SchemeAntD   Scheme = "antd"
)

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

type PreferencesV2 struct {
	Version       int                `json:"version"`
	DisplayScheme string             `json:"displayScheme"`
	ThemeMode     string             `json:"themeMode"`
	Profiles      map[string]Profile `json:"profiles"`
}

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

func Default() Preferences {
	return effectiveShadcn(DefaultV2().ThemeMode, defaultShadcnProfile())
}

func DefaultV2() PreferencesV2 {
	return PreferencesV2{
		Version:       preferencesVersion,
		DisplayScheme: string(SchemeShadcn),
		ThemeMode:     "light",
		Profiles: map[string]Profile{
			string(SchemeShadcn): defaultShadcnProfile(),
			string(SchemeAntD):   defaultAntDProfile(),
		},
	}
}

func NormalizeV2(value PreferencesV2) PreferencesV2 {
	defaults := DefaultV2()
	value.Version = preferencesVersion
	value.DisplayScheme = normalizeScheme(value.DisplayScheme, defaults.DisplayScheme)
	value.ThemeMode = allowedOrDefault(value.ThemeMode, allowedThemeModes, defaults.ThemeMode)
	if value.Profiles == nil {
		value.Profiles = map[string]Profile{}
	}
	value.Profiles[string(SchemeShadcn)] = normalizeShadcnProfile(value.Profiles[string(SchemeShadcn)])
	value.Profiles[string(SchemeAntD)] = normalizeAntDProfile(value.Profiles[string(SchemeAntD)])
	return value
}

func Effective(value PreferencesV2) Preferences {
	value = NormalizeV2(value)
	if value.DisplayScheme == string(SchemeAntD) {
		return effectiveAntD(value.ThemeMode, value.Profiles[string(SchemeAntD)])
	}
	return effectiveShadcn(value.ThemeMode, value.Profiles[string(SchemeShadcn)])
}

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

func MarshalV2(value PreferencesV2) string {
	value = NormalizeV2(value)
	data, err := json.Marshal(value)
	if err != nil {
		data, _ = json.Marshal(DefaultV2())
	}
	return string(data)
}

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
	if targetScheme == string(SchemeAntD) {
		next.Profiles[string(SchemeAntD)] = normalizeAntDProfile(Profile{
			ChartColor: incoming.ChartColor,
			IconTone:   incoming.IconTone,
			Menu:       incoming.Menu,
			MenuAccent: incoming.MenuAccent,
			Radius:     incoming.Radius,
			Density:    incoming.Density,
			TextSize:   incoming.TextSize,
			CardBorder: incoming.CardBorder,
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

func Definitions() []configstore.ConfigItem {
	defaultValue := MarshalV2(DefaultV2())
	return []configstore.ConfigItem{
		{Key: KeyPreferencesV2, Category: "display", Title: "显示偏好", Description: "显示偏好 JSON。", ValueType: "string", DefaultValue: defaultValue, Value: defaultValue, SortOrder: 490},
	}
}

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

func defaultAntDProfile() Profile {
	return Profile{
		ChartColor: "blue",
		IconTone:   "default",
		Menu:       "default",
		MenuAccent: "subtle",
		Radius:     "medium",
		Density:    "comfortable",
		TextSize:   "normal",
		CardBorder: "visible",
	}
}

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

func effectiveAntD(themeMode string, profile Profile) Preferences {
	profile = normalizeAntDProfile(profile)
	return Preferences{
		DisplayScheme: string(SchemeAntD),
		UIStyle:       "vega",
		ThemeMode:     allowedOrDefault(themeMode, allowedThemeModes, "light"),
		BaseColor:     "neutral",
		ThemeColor:    "blue",
		AccentColor:   "blue",
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

func normalizeAntDProfile(profile Profile) Profile {
	defaults := defaultAntDProfile()
	return Profile{
		ChartColor: allowedOrDefault(profile.ChartColor, allowedAccentColors, defaults.ChartColor),
		IconTone:   allowedOrDefault(profile.IconTone, allowedIconTones, defaults.IconTone),
		Menu:       allowedOrDefault(profile.Menu, allowedAntDMenus, defaults.Menu),
		MenuAccent: allowedOrDefault(profile.MenuAccent, allowedMenuAccents, defaults.MenuAccent),
		Radius:     allowedOrDefault(profile.Radius, allowedAntDRadii, defaults.Radius),
		Density:    allowedOrDefault(profile.Density, allowedDensities, defaults.Density),
		TextSize:   allowedOrDefault(profile.TextSize, allowedTextSizes, defaults.TextSize),
		CardBorder: allowedOrDefault(profile.CardBorder, allowedCardBorders, defaults.CardBorder),
	}
}

func normalizeScheme(value string, fallback string) string {
	if value == string(SchemeShadcn) || value == string(SchemeAntD) {
		return value
	}
	if fallback == string(SchemeAntD) {
		return fallback
	}
	return string(SchemeShadcn)
}

func allowedOrDefault(value string, allowed map[string]struct{}, fallback string) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return fallback
}

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
	allowedAntDMenus    = stringSet("default", "inverted")
	allowedMenuAccents  = stringSet("subtle", "bold")
	allowedRadii        = stringSet("default", "none", "small", "medium", "large")
	allowedAntDRadii    = stringSet("default", "small", "medium", "large")
	allowedDensities    = stringSet("compact", "comfortable")
	allowedTextSizes    = stringSet("small", "normal", "medium", "large")
	allowedCardBorders  = stringSet("visible", "soft", "hidden")
)
