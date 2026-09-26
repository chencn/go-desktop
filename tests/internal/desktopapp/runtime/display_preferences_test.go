// 文件职责：验证显示偏好的后端持久化契约，重点是液态玻璃光学三项与主题/尺寸共用同一条 KV 链路。

package runtime_test

import (
	"path/filepath"
	"testing"

	appruntime "github.com/chencn/go-desktop/internal/desktopapp/runtime"
)

// TestDisplayPreferencesNormalizeGlassAxes 锁定后端归一化口径：非法折射风格与越界光强必须回默认档，
// 否则材质层会拿到 0 系数或未知 data-lg-style，渲染出既不通透也无退路的空白晶体。
func TestDisplayPreferencesNormalizeGlassAxes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	saved, err := runtimeService.SaveDisplayPreferences(appruntime.DisplayPreferences{
		ThemeMode:   "dark",
		Size:        "small",
		Backdrop:    true,
		LgStyle:     "crystal",
		LgIntensity: 250,
	})
	if err != nil {
		t.Fatalf("save display preferences: %v", err)
	}
	if saved.LgStyle != "fresnel" {
		t.Fatalf("unknown lg style should fall back to the default refraction, got %q", saved.LgStyle)
	}
	if saved.LgIntensity != 75 {
		t.Fatalf("out-of-range intensity should fall back to the default 75, got %d", saved.LgIntensity)
	}
	if saved.ThemeMode != "dark" || saved.Size != "small" || !saved.Backdrop {
		t.Fatalf("legal axes must be kept as submitted, got %#v", saved)
	}
}

// TestDisplayPreferencesPersistGlassAxesAcrossRestart 验证极光开关、折射风格与光强在重启后仍然生效：
// 这三项以前只落前端 localStorage，用户换设备或清缓存就会丢偏好，现在必须和主题/尺寸一样进 SQLite。
func TestDisplayPreferencesPersistGlassAxesAcrossRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")

	first := appruntime.NewRuntime(appruntime.ServiceOptions{DatabasePath: dbPath})
	if _, err := first.SaveDisplayPreferences(appruntime.DisplayPreferences{
		ThemeMode:   "dark",
		Size:        "large",
		Backdrop:    true,
		LgStyle:     "sheen",
		LgIntensity: 92,
	}); err != nil {
		t.Fatalf("save display preferences: %v", err)
	}
	first.Shutdown()

	reloaded := appruntime.NewRuntime(appruntime.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()

	got := reloaded.DisplayPreferencesSnapshot()
	if got != (appruntime.DisplayPreferences{
		ThemeMode:   "dark",
		Size:        "large",
		Backdrop:    true,
		LgStyle:     "sheen",
		LgIntensity: 92,
	}) {
		t.Fatalf("glass axes should survive restart, got %#v", got)
	}
}
