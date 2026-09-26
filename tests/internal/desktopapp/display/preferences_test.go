// 文件职责：验证显示偏好 V3 的解析与归一化口径，重点是「老数据/脏数据不得牵连合法轴」。

package display_test

import (
	"encoding/json"
	"testing"

	"github.com/chencn/go-desktop/internal/desktopapp/display"
)

// TestDisplayPreferencesParseFallbacks 覆盖三条读库回退路径：空值、坏 JSON、版本不匹配。
// 升级事故的真实形态就是老库里留着 v1/v2 行——它们必须整体回默认，而不是被当成 v3 局部解析。
func TestDisplayPreferencesParseFallbacks(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{name: "空值", raw: ""},
		{name: "空白", raw: "   "},
		{name: "坏 JSON", raw: "{\"version\":3,\"size\":"},
		{name: "v2 旧版本", raw: "{\"version\":2,\"themeMode\":\"dark\",\"size\":\"large\"}"},
		{name: "缺版本号", raw: "{\"themeMode\":\"dark\",\"size\":\"large\"}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := display.ParseV3(tc.raw); got != display.DefaultV3() {
				t.Fatalf("ParseV3(%q) should fall back to defaults, got %#v", tc.raw, got)
			}
		})
	}
}

// TestDisplayPreferencesNormalizeIsPerAxis 验证每个轴独立回默认：
// 光学项脏不能把用户选好用的主题与尺寸一起抹掉，反之亦然。
func TestDisplayPreferencesNormalizeIsPerAxis(t *testing.T) {
	got := display.NormalizeV3(display.PreferencesV3{
		Version:     99,
		ThemeMode:   "night",
		Size:        "huge",
		Backdrop:    true,
		LgStyle:     "crystal",
		LgIntensity: 250,
	})
	want := display.PreferencesV3{
		Version:     3,
		ThemeMode:   "light",
		Size:        "default",
		Backdrop:    true,
		LgStyle:     display.DefaultGlassStyle,
		LgIntensity: display.DefaultGlassIntensity,
	}
	if got != want {
		t.Fatalf("illegal axes should default independently (Backdrop=true is legal and kept), got %#v", got)
	}
}

// TestDisplayPreferencesGlassIntensityBounds 锁定光强闭区间 30–100。
// 0 必须视为「老数据没有该字段」回默认档，而不是当成合法的最低强度（材质层会拿到 0 系数变成全透明）。
func TestDisplayPreferencesGlassIntensityBounds(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{in: display.GlassIntensityMin, want: display.GlassIntensityMin},
		{in: display.GlassIntensityMax, want: display.GlassIntensityMax},
		{in: display.GlassIntensityMin - 1, want: display.DefaultGlassIntensity},
		{in: display.GlassIntensityMax + 1, want: display.DefaultGlassIntensity},
		{in: 0, want: display.DefaultGlassIntensity},
		{in: -50, want: display.DefaultGlassIntensity},
	}
	for _, tc := range cases {
		got := display.NormalizeV3(display.PreferencesV3{LgIntensity: tc.in})
		if got.LgIntensity != tc.want {
			t.Fatalf("intensity %d should normalize to %d, got %d", tc.in, tc.want, got.LgIntensity)
		}
	}
}

// TestDisplayPreferencesPartialV3JSONKeepsLegalAxes 模拟旧 SQLite 行缺玻璃三项：
// 主题与尺寸照常恢复，玻璃三项补默认，且整份 JSON 重新序列化后必须自带全部五轴。
func TestDisplayPreferencesPartialV3JSONKeepsLegalAxes(t *testing.T) {
	got := display.ParseV3(`{"version":3,"themeMode":"dark","size":"large"}`)
	if got.ThemeMode != "dark" || got.Size != "large" {
		t.Fatalf("legal axes should be restored from partial data, got %#v", got)
	}
	if got.Backdrop || got.LgStyle != "fresnel" || got.LgIntensity != 75 {
		t.Fatalf("missing glass fields should be backfilled with defaults, got %#v", got)
	}

	var round map[string]any
	if err := json.Unmarshal([]byte(display.MarshalV3(got)), &round); err != nil {
		t.Fatalf("marshal normalized preferences: %v", err)
	}
	for _, key := range []string{"themeMode", "size", "backdrop", "lgStyle", "lgIntensity"} {
		if _, ok := round[key]; !ok {
			t.Fatalf("persisted v3 JSON should always carry %q, got %s", key, display.MarshalV3(got))
		}
	}
}

// TestDisplayPreferencesAllowedGlassStylesMatchMaterialLayer 防止后端枚举与材质层脱节：
// 只接受 html[data-lg-style] 真正实现了的三种风格，放进第四个值就会渲染出无样式的默认晶体。
func TestDisplayPreferencesAllowedGlassStylesMatchMaterialLayer(t *testing.T) {
	for _, style := range []string{"fresnel", "frosted", "sheen"} {
		if got := display.NormalizeV3(display.PreferencesV3{LgStyle: style}); got.LgStyle != style {
			t.Fatalf("style %q is implemented by the material layer and must survive normalization, got %q", style, got.LgStyle)
		}
	}
	for _, style := range []string{"", "glass", "Fresnel", "frosted "} {
		if got := display.NormalizeV3(display.PreferencesV3{LgStyle: style}); got.LgStyle != display.DefaultGlassStyle {
			t.Fatalf("style %q is not implemented and must fall back, got %q", style, got.LgStyle)
		}
	}
}
