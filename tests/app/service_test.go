// ============================================================================
// 文件: tests/app/service_test.go
// 描述: 应用服务测试
//
// 功能概述:
// - 测试设置保存和加载功能
// - 测试设置合并和默认值处理
// ============================================================================

package app_test

import (
	"path/filepath" // 路径处理
	"runtime"       // 运行时信息
	"strings"       // 字符串处理
	"testing"       // 测试框架

	"github.com/chencn/go-desktop/app" // 应用核心包
)

// TestSaveSettingsPersistsAndLoadsFromSQLiteConfig 测试设置持久化。
// 验证业务设置和启动设置都写入 SQLite KV，并能被新 Runtime 重新加载。
func TestSaveSettingsPersistsAndLoadsFromSQLiteConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	if _, err := runtimeService.SaveSettings(app.Settings{
		GitHubOwner:              "example",
		GitHubRepo:               "desktop",
		GitHubProxyBase:          "https://proxy.example",
		UpdateCheckIntervalHours: 6,
		MinimizeToTray:           false,
		LogRetentionDays:         14,
		LogLevel:                 "debug",
		AutoLaunch:               true,
		CreateDesktopShortcut:    false,
		LaunchHiddenToTray:       true,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	reloaded := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	settings := reloaded.SettingsSnapshot()
	if settings.GitHubOwner != "example" || settings.GitHubRepo != "desktop" {
		t.Fatalf("expected github settings to persist, got %#v", settings)
	}
	if settings.GitHubProxyBase != "https://proxy.example" {
		t.Fatalf("expected proxy to persist, got %#v", settings)
	}
	if settings.UpdateCheckIntervalHours != 6 || settings.LogRetentionDays != 14 {
		t.Fatalf("expected numeric settings to persist, got %#v", settings)
	}
	if settings.LogLevel != "debug" {
		t.Fatalf("expected log level to persist, got %#v", settings)
	}
	if settings.MinimizeToTray {
		t.Fatalf("expected minimizeToTray=false to persist, got %#v", settings)
	}
	if !settings.AutoLaunch || settings.CreateDesktopShortcut || !settings.LaunchHiddenToTray {
		t.Fatalf("expected startup settings to persist, got %#v", settings)
	}
}

// TestLoadSettingsWritesSQLiteDefaults 测试数据库没有配置时会写入并读取默认值。
func TestLoadSettingsWritesSQLiteDefaults(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	settings := runtimeService.SettingsSnapshot()
	if !settings.MinimizeToTray {
		t.Fatalf("expected default minimizeToTray=true, got %#v", settings)
	}
	if settings.AutoLaunch || !settings.CreateDesktopShortcut || settings.LaunchHiddenToTray {
		t.Fatalf("expected startup defaults auto=false shortcut=true hidden=false, got %#v", settings)
	}
}

// TestSaveSettingsAllowsNeverCleanLogRetention 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestSaveSettingsAllowsNeverCleanLogRetention(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	if _, err := runtimeService.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 12,
		MinimizeToTray:           true,
		LogRetentionDays:         -1,
		CreateDesktopShortcut:    true,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	reloaded := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	if reloaded.SettingsSnapshot().LogRetentionDays != -1 {
		t.Fatalf("expected logRetentionDays=-1 to persist, got %#v", reloaded.SettingsSnapshot())
	}
}

// TestSaveSettingsNormalisesUnsupportedUpdateInterval 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestSaveSettingsNormalisesUnsupportedUpdateInterval(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	if _, err := runtimeService.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 48,
		MinimizeToTray:           true,
		LogRetentionDays:         30,
		LogLevel:                 "verbose",
		CreateDesktopShortcut:    true,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	if runtimeService.SettingsSnapshot().UpdateCheckIntervalHours != 3 {
		t.Fatalf("expected unsupported update interval to fall back to 3, got %#v", runtimeService.SettingsSnapshot())
	}
	if runtimeService.SettingsSnapshot().LogLevel != "info" {
		t.Fatalf("expected unsupported log level to fall back to info, got %#v", runtimeService.SettingsSnapshot())
	}
}

// TestDisplayPreferencesPersistThroughSQLiteConfig 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDisplayPreferencesPersistThroughSQLiteConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	saved, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		UIStyle:     "nova",
		ThemeMode:   "dark",
		BaseColor:   "mauve",
		ThemeColor:  "blue",
		AccentColor: "emerald",
		ChartColor:  "yellow",
		IconTone:    "colorful",
		Menu:        "inverted",
		MenuAccent:  "bold",
		Radius:      "large",
		Density:     "compact",
		TextSize:    "large",
		CardBorder:  "soft",
	})
	if err != nil {
		t.Fatalf("save display preferences: %v", err)
	}
	if saved.ThemeMode != "dark" || saved.BaseColor != "mauve" || saved.IconTone != "colorful" {
		t.Fatalf("expected display preferences to save, got %#v", saved)
	}

	reloaded := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	preferences := reloaded.DisplayPreferencesSnapshot()
	if preferences.ThemeMode != "dark" || preferences.BaseColor != "mauve" || preferences.MenuAccent != "bold" {
		t.Fatalf("expected display preferences from sqlite, got %#v", preferences)
	}
}

// TestDisplayPreferencesNormaliseInvalidValues 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDisplayPreferencesNormaliseInvalidValues(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	saved, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		UIStyle:     "unknown",
		ThemeMode:   "night",
		BaseColor:   "slate",
		ThemeColor:  "blue",
		AccentColor: "emerald",
		ChartColor:  "yellow",
		IconTone:    "rainbow",
		Menu:        "default",
		MenuAccent:  "bold",
		Radius:      "large",
		Density:     "compact",
		TextSize:    "large",
		CardBorder:  "soft",
	})
	if err != nil {
		t.Fatalf("save display preferences: %v", err)
	}
	if saved.UIStyle != "vega" || saved.ThemeMode != "light" || saved.BaseColor != "neutral" {
		t.Fatalf("expected invalid display values to fall back to defaults, got %#v", saved)
	}
	if saved.ThemeColor != "blue" || saved.AccentColor != "emerald" || saved.MenuAccent != "bold" {
		t.Fatalf("expected valid display values to survive normalisation, got %#v", saved)
	}
}

// TestDefaultRuntimePathsUseExecutableDataDirectory 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDefaultRuntimePathsUseExecutableDataDirectory(t *testing.T) {
	databasePath := filepath.ToSlash(app.DefaultDatabasePath("go-desktop"))
	logFilePath := filepath.ToSlash(app.DefaultLogFilePath("go-desktop"))
	cachePath := filepath.ToSlash(app.DefaultCachePath("go-desktop"))

	if !strings.Contains(databasePath, "/data/") || !strings.HasSuffix(databasePath, "/go-desktop.db") {
		t.Fatalf("expected database path under data directory, got %q", databasePath)
	}
	if !strings.Contains(logFilePath, "/data/logs/") || !strings.Contains(logFilePath, "/go-desktop-") || !strings.HasSuffix(logFilePath, ".log") {
		t.Fatalf("expected daily log path under data/logs directory, got %q", logFilePath)
	}
	if !strings.HasSuffix(cachePath, "/data/updates") {
		t.Fatalf("expected cache path to use data/updates directory, got %q", cachePath)
	}
}

// TestGetEnvironmentInfoReturnsRuntimeAndStoragePaths 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestGetEnvironmentInfoReturnsRuntimeAndStoragePaths(t *testing.T) {
	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.json")
	databasePath := filepath.Join(tempDir, "go-desktop.db")
	logFilePath := filepath.Join(tempDir, "go-desktop.log")
	cachePath := filepath.Join(tempDir, "cache")

	runtimeService := app.NewRuntime(app.ServiceOptions{
		SettingsPath: settingsPath,
		DatabasePath: databasePath,
		LogFilePath:  logFilePath,
		CachePath:    cachePath,
	})
	defer runtimeService.Shutdown()

	info := runtimeService.GetEnvironmentInfo()
	if info.OS != runtime.GOOS {
		t.Fatalf("expected OS %q, got %#v", runtime.GOOS, info)
	}
	if info.Arch != runtime.GOARCH {
		t.Fatalf("expected Arch %q, got %#v", runtime.GOARCH, info)
	}
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Fatalf("expected Go version, got %#v", info)
	}
	if info.SettingsPath != settingsPath || info.DatabasePath != databasePath || !strings.HasPrefix(info.LogFilePath, tempDir) || !strings.Contains(info.LogFilePath, "go-desktop-") || info.CachePath != cachePath {
		t.Fatalf("expected configured paths in environment info, got %#v", info)
	}
	if strings.TrimSpace(info.WailsVersion) == "" {
		t.Fatalf("expected Wails module version, got %#v", info)
	}
}

// TestParseExitRequestRecognisesInstallerAndForceExitArgs 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestParseExitRequestRecognisesInstallerAndForceExitArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want app.ExitRequest
	}{
		{
			name: "short exit",
			args: []string{"-exit"},
			want: app.ExitRequest{Present: true, Source: "-exit"},
		},
		{
			name: "long exit",
			args: []string{"--exit"},
			want: app.ExitRequest{Present: true, Source: "-exit"},
		},
		{
			name: "force exit",
			args: []string{"--force-exit"},
			want: app.ExitRequest{Present: true, Force: true, Source: "-force-exit"},
		},
		{
			name: "installer exit",
			args: []string{"--installer-exit"},
			want: app.ExitRequest{Present: true, Force: true, Source: "-installer-exit"},
		},
		{
			name: "mixed args",
			args: []string{"C:\\Program Files\\go-desktop\\go-desktop.exe", "--installer-exit"},
			want: app.ExitRequest{Present: true, Force: true, Source: "-installer-exit"},
		},
		{
			name: "no exit",
			args: []string{"--check-update"},
			want: app.ExitRequest{},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := app.ParseExitRequest(tt.args)
			if got != tt.want {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

// TestRuntimeHideOnCloseHonoursSettingAndQuitState 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestRuntimeHideOnCloseHonoursSettingAndQuitState(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()
	if !runtimeService.ShouldHideOnClose() {
		t.Fatal("expected default settings to hide on close")
	}

	if _, err := runtimeService.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 12,
		MinimizeToTray:           false,
		LogRetentionDays:         30,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if runtimeService.ShouldHideOnClose() {
		t.Fatal("expected disabled minimizeToTray to close the window")
	}

	if _, err := runtimeService.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 12,
		MinimizeToTray:           true,
		LogRetentionDays:         30,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	runtimeService.QuitApp()
	if runtimeService.ShouldHideOnClose() {
		t.Fatal("expected explicit quit to bypass close-to-tray")
	}
}

// TestSaveSettingsReturnsErrorWhenConfigStoreUnavailable 验证配置读取可降级，但写配置必须向调用方返回失败。
func TestSaveSettingsReturnsErrorWhenConfigStoreUnavailable(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{})
	defer runtimeService.Shutdown()

	_, err := runtimeService.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 3,
		MinimizeToTray:           true,
		LogRetentionDays:         30,
		CreateDesktopShortcut:    true,
	})
	if err == nil || !strings.Contains(err.Error(), "配置存储不可用") {
		t.Fatalf("expected config store unavailable error, got %v", err)
	}
}

// TestRecordSecondInstanceStoresCopyAndCapsHistory 验证 service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestRecordSecondInstanceStoresCopyAndCapsHistory(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{})
	args := []string{"go-desktop.exe", "--installer-exit"}
	runtimeService.RecordSecondInstance(args, `C:\work`)
	args[1] = "--mutated"

	records := runtimeService.GetSecondInstanceRecords()
	if len(records) != 1 {
		t.Fatalf("expected one second instance record, got %#v", records)
	}
	if records[0].Args[1] != "--installer-exit" || records[0].WorkingDir != `C:\work` {
		t.Fatalf("expected copied second instance data, got %#v", records[0])
	}

	for i := 0; i < 25; i++ {
		runtimeService.RecordSecondInstance([]string{"go-desktop.exe"}, "")
	}
	records = runtimeService.GetSecondInstanceRecords()
	if len(records) != 20 {
		t.Fatalf("expected second instance history to be capped at 20, got %d", len(records))
	}
}
