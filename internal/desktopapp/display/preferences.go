package display

import "github.com/chencn/go-desktop/internal/adapters/configstore"

const (
	KeyUIStyle     = "display.ui_style"
	KeyThemeMode   = "display.theme_mode"
	KeyBaseColor   = "display.base_color"
	KeyThemeColor  = "display.theme_color"
	KeyAccentColor = "display.accent_color"
	KeyChartColor  = "display.chart_color"
	KeyIconTone    = "display.icon_tone"
	KeyMenu        = "display.menu"
	KeyMenuAccent  = "display.menu_accent"
	KeyRadius      = "display.radius"
	KeyDensity     = "display.density"
	KeyTextSize    = "display.text_size"
	KeyCardBorder  = "display.card_border"
)

type Preferences struct {
	UIStyle     string
	ThemeMode   string
	BaseColor   string
	ThemeColor  string
	AccentColor string
	ChartColor  string
	IconTone    string
	Menu        string
	MenuAccent  string
	Radius      string
	Density     string
	TextSize    string
	CardBorder  string
}

func Default() Preferences {
	return Preferences{
		UIStyle:     "vega",
		ThemeMode:   "light",
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

func Normalize(preferences Preferences) Preferences {
	defaults := Default()
	preferences.UIStyle = allowedOrDefault(preferences.UIStyle, allowedUIStyles, defaults.UIStyle)
	preferences.ThemeMode = allowedOrDefault(preferences.ThemeMode, allowedThemeModes, defaults.ThemeMode)
	preferences.BaseColor = allowedOrDefault(preferences.BaseColor, allowedBaseColors, defaults.BaseColor)
	preferences.ThemeColor = allowedOrDefault(preferences.ThemeColor, allowedAccentColors, defaults.ThemeColor)
	preferences.AccentColor = allowedOrDefault(preferences.AccentColor, allowedAccentColors, defaults.AccentColor)
	preferences.ChartColor = allowedOrDefault(preferences.ChartColor, allowedAccentColors, defaults.ChartColor)
	preferences.IconTone = allowedOrDefault(preferences.IconTone, allowedIconTones, defaults.IconTone)
	preferences.Menu = allowedOrDefault(preferences.Menu, allowedMenus, defaults.Menu)
	preferences.MenuAccent = allowedOrDefault(preferences.MenuAccent, allowedMenuAccents, defaults.MenuAccent)
	preferences.Radius = allowedOrDefault(preferences.Radius, allowedRadii, defaults.Radius)
	preferences.Density = allowedOrDefault(preferences.Density, allowedDensities, defaults.Density)
	preferences.TextSize = allowedOrDefault(preferences.TextSize, allowedTextSizes, defaults.TextSize)
	preferences.CardBorder = allowedOrDefault(preferences.CardBorder, allowedCardBorders, defaults.CardBorder)
	return preferences
}

func FromConfigItems(items map[string]configstore.ConfigItem, base Preferences) Preferences {
	base.UIStyle = configString(items, KeyUIStyle, base.UIStyle)
	base.ThemeMode = configString(items, KeyThemeMode, base.ThemeMode)
	base.BaseColor = configString(items, KeyBaseColor, base.BaseColor)
	base.ThemeColor = configString(items, KeyThemeColor, base.ThemeColor)
	base.AccentColor = configString(items, KeyAccentColor, base.AccentColor)
	base.ChartColor = configString(items, KeyChartColor, base.ChartColor)
	base.IconTone = configString(items, KeyIconTone, base.IconTone)
	base.Menu = configString(items, KeyMenu, base.Menu)
	base.MenuAccent = configString(items, KeyMenuAccent, base.MenuAccent)
	base.Radius = configString(items, KeyRadius, base.Radius)
	base.Density = configString(items, KeyDensity, base.Density)
	base.TextSize = configString(items, KeyTextSize, base.TextSize)
	base.CardBorder = configString(items, KeyCardBorder, base.CardBorder)
	return Normalize(base)
}

func Values(preferences Preferences) map[string]string {
	preferences = Normalize(preferences)
	return map[string]string{
		KeyUIStyle:     preferences.UIStyle,
		KeyThemeMode:   preferences.ThemeMode,
		KeyBaseColor:   preferences.BaseColor,
		KeyThemeColor:  preferences.ThemeColor,
		KeyAccentColor: preferences.AccentColor,
		KeyChartColor:  preferences.ChartColor,
		KeyIconTone:    preferences.IconTone,
		KeyMenu:        preferences.Menu,
		KeyMenuAccent:  preferences.MenuAccent,
		KeyRadius:      preferences.Radius,
		KeyDensity:     preferences.Density,
		KeyTextSize:    preferences.TextSize,
		KeyCardBorder:  preferences.CardBorder,
	}
}

func Definitions() []configstore.ConfigItem {
	defaults := Default()
	return []configstore.ConfigItem{
		{Key: KeyUIStyle, Category: "display", Title: "组件风格", Description: "对应 shadcn-vue create 的 style。", ValueType: "string", DefaultValue: defaults.UIStyle, Value: defaults.UIStyle, SortOrder: 500},
		{Key: KeyThemeMode, Category: "display", Title: "主题模式", Description: "控制亮色或暗色主题。", ValueType: "string", DefaultValue: defaults.ThemeMode, Value: defaults.ThemeMode, SortOrder: 510},
		{Key: KeyBaseColor, Category: "display", Title: "基础色盘", Description: "影响亮色和暗色中性色 token。", ValueType: "string", DefaultValue: defaults.BaseColor, Value: defaults.BaseColor, SortOrder: 520},
		{Key: KeyThemeColor, Category: "display", Title: "主题", Description: "控制主按钮、焦点环、选中态和高强调 token。", ValueType: "string", DefaultValue: defaults.ThemeColor, Value: defaults.ThemeColor, SortOrder: 530},
		{Key: KeyAccentColor, Category: "display", Title: "强调色", Description: "用于主按钮、选中导航、更新图标、进度、开关和焦点环。", ValueType: "string", DefaultValue: defaults.AccentColor, Value: defaults.AccentColor, SortOrder: 540},
		{Key: KeyChartColor, Category: "display", Title: "图表色", Description: "用于统计和可视化色板。", ValueType: "string", DefaultValue: defaults.ChartColor, Value: defaults.ChartColor, SortOrder: 550},
		{Key: KeyIconTone, Category: "display", Title: "图标颜色", Description: "控制语义图标是否按含义着色。", ValueType: "string", DefaultValue: defaults.IconTone, Value: defaults.IconTone, SortOrder: 580},
		{Key: KeyMenu, Category: "display", Title: "菜单", Description: "控制侧栏菜单外观基线。", ValueType: "string", DefaultValue: defaults.Menu, Value: defaults.Menu, SortOrder: 590},
		{Key: KeyMenuAccent, Category: "display", Title: "菜单强调", Description: "控制菜单 hover 和 active 背景强度。", ValueType: "string", DefaultValue: defaults.MenuAccent, Value: defaults.MenuAccent, SortOrder: 600},
		{Key: KeyRadius, Category: "display", Title: "圆角", Description: "派生到卡片、按钮、输入框、弹窗和列表。", ValueType: "string", DefaultValue: defaults.Radius, Value: defaults.Radius, SortOrder: 610},
		{Key: KeyDensity, Category: "display", Title: "密度", Description: "影响页面留白、工具栏高度和控件最小高度。", ValueType: "string", DefaultValue: defaults.Density, Value: defaults.Density, SortOrder: 620},
		{Key: KeyTextSize, Category: "display", Title: "字体大小", Description: "影响标题、正文、按钮和日志字号。", ValueType: "string", DefaultValue: defaults.TextSize, Value: defaults.TextSize, SortOrder: 630},
		{Key: KeyCardBorder, Category: "display", Title: "卡片边框", Description: "统一控制 Card、列表、日志表和弹窗边框强度。", ValueType: "string", DefaultValue: defaults.CardBorder, Value: defaults.CardBorder, SortOrder: 640},
	}
}

func allowedOrDefault(value string, allowed map[string]struct{}, fallback string) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return fallback
}

func configString(items map[string]configstore.ConfigItem, key string, fallback string) string {
	if item, ok := items[key]; ok && item.Value != "" {
		return item.Value
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
	allowedMenuAccents  = stringSet("subtle", "bold")
	allowedRadii        = stringSet("default", "none", "small", "medium", "large")
	allowedDensities    = stringSet("compact", "comfortable")
	allowedTextSizes    = stringSet("small", "normal", "medium", "large")
	allowedCardBorders  = stringSet("visible", "soft", "hidden")
)
