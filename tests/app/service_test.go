// 文件职责：验证 app facade 暴露的设置、显示偏好、路径和窗口生命周期行为。

package app_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chencn/go-desktop/app"
)

// TestSaveSettingsPersistsAndLoadsFromSQLiteConfig 测试设置持久化。
// 验证业务设置和启动设置都写入 SQLite KV，并能被新 Runtime 重新加载。
func TestSaveSettingsPersistsAndLoadsFromSQLiteConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	if _, err := runtimeService.SaveSettings(app.Settings{
		UpdateSource:             "local",
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
	if settings.UpdateSource != "local" {
		t.Fatalf("expected update source to persist, got %#v", settings)
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
	if settings.UpdateSource != "github" {
		t.Fatalf("expected default update source github, got %#v", settings)
	}
	if !settings.MinimizeToTray {
		t.Fatalf("expected default minimizeToTray=true, got %#v", settings)
	}
	if settings.AutoLaunch || !settings.CreateDesktopShortcut || settings.LaunchHiddenToTray {
		t.Fatalf("expected startup defaults auto=false shortcut=true hidden=false, got %#v", settings)
	}
}

// TestSaveSettingsAllowsNeverCleanLogRetention 验证 -1 是“永不清理日志”的显式业务值，保存时不能被默认值覆盖。
func TestSaveSettingsAllowsNeverCleanLogRetention(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	if _, err := runtimeService.SaveSettings(app.Settings{
		UpdateSource:             "ftp",
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

// TestSaveSettingsNormalisesUnsupportedUpdateInterval 验证后端会裁决非法设置值，而不是把前端传入原样落库。
func TestSaveSettingsNormalisesUnsupportedUpdateInterval(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	if _, err := runtimeService.SaveSettings(app.Settings{
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
	if runtimeService.SettingsSnapshot().UpdateSource != "github" {
		t.Fatalf("expected unsupported update source to fall back to github, got %#v", runtimeService.SettingsSnapshot())
	}
	if runtimeService.SettingsSnapshot().LogLevel != "info" {
		t.Fatalf("expected unsupported log level to fall back to info, got %#v", runtimeService.SettingsSnapshot())
	}
}

// TestDisplayPreferencesPersistThroughSQLiteConfig 验证显示偏好走 SQLite 配置项持久化，并在新 Runtime 中恢复。
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
		ChartColor:  "amber",
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
	if saved.ThemeMode != "dark" || saved.BaseColor != "mauve" || saved.IconTone != "colorful" || saved.AccentColor != "blue" {
		t.Fatalf("expected display preferences to save, got %#v", saved)
	}

	reloaded := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	preferences := reloaded.DisplayPreferencesSnapshot()
	if preferences.ThemeMode != "dark" || preferences.BaseColor != "mauve" || preferences.MenuAccent != "bold" || preferences.AccentColor != "blue" {
		t.Fatalf("expected display preferences from sqlite, got %#v", preferences)
	}
}

// TestDisplayPreferencesNormaliseInvalidValues 验证非法显示偏好回退到默认值，同时保留合法输入。
func TestDisplayPreferencesNormaliseInvalidValues(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	saved, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		UIStyle:     "unknown",
		ThemeMode:   "night",
		BaseColor:   "slate",
		ThemeColor:  "blue",
		AccentColor: "emerald",
		ChartColor:  "amber",
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
	if saved.ThemeColor != "blue" || saved.AccentColor != "blue" || saved.MenuAccent != "bold" {
		t.Fatalf("expected artistic accent to follow theme color while valid values survive normalisation, got %#v", saved)
	}
}

// TestDisplayPreferencesJSONPersistsProfilesAcrossRestart 验证显示偏好按方案 profile 写入单个 JSON 配置项，并在重启后保持当前方案。
func TestDisplayPreferencesJSONPersistsProfilesAcrossRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	shutdownRuntime := func() {
		if runtimeService != nil {
			runtimeService.Shutdown()
			runtimeService = nil
		}
	}
	defer shutdownRuntime()

	if _, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "shadcn",
		ThemeMode:     "dark",
	}); err != nil {
		t.Fatalf("切换到 shadcn 显示方案失败：%v", err)
	}

	_, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "shadcn",
		UIStyle:       "nova",
		ThemeMode:     "dark",
		BaseColor:     "mauve",
		ThemeColor:    "rose",
		AccentColor:   "emerald",
		ChartColor:    "amber",
		IconTone:      "colorful",
		Menu:          "inverted",
		MenuAccent:    "bold",
		Radius:        "large",
		Density:       "compact",
		TextSize:      "large",
		CardBorder:    "soft",
	})
	if err != nil {
		t.Fatalf("保存 shadcn 显示偏好失败：%v", err)
	}

	if _, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "artistic",
		ThemeMode:     "dark",
	}); err != nil {
		t.Fatalf("切换到 artistic 显示方案失败：%v", err)
	}

	artistic, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "artistic",
		ThemeMode:     "dark",
		BaseColor:     "stone",
		ThemeColor:    "blue",
		AccentColor:   "cyan",
		ChartColor:    "blue",
		Menu:          "default",
		MenuAccent:    "subtle",
		Radius:        "medium",
		Density:       "comfortable",
		TextSize:      "normal",
		CardBorder:    "visible",
	})
	if err != nil {
		t.Fatalf("保存 artistic 显示偏好失败：%v", err)
	}
	if artistic.DisplayScheme != "artistic" || artistic.ThemeColor != "blue" || artistic.AccentColor != "blue" || artistic.IconTone != "colorful" {
		t.Fatalf("期望 artistic 生效偏好包含方案默认值和显式输入，实际为 %#v", artistic)
	}
	shutdownRuntime()

	reloaded := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	loadedArtistic := reloaded.DisplayPreferencesSnapshot()
	if loadedArtistic.DisplayScheme != "artistic" || loadedArtistic.ThemeMode != "dark" || loadedArtistic.ThemeColor != "blue" || loadedArtistic.Menu != "default" {
		t.Fatalf("期望重启后恢复 artistic 生效偏好，实际为 %#v", loadedArtistic)
	}
	if loadedArtistic.AccentColor != loadedArtistic.ThemeColor {
		t.Fatalf("期望重启后 artistic 品牌辅助色跟随品牌主题色，实际为 %#v", loadedArtistic)
	}

	shadcn, err := reloaded.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "shadcn",
		ThemeMode:     "dark",
	})
	if err != nil {
		t.Fatalf("切回 shadcn 失败：%v", err)
	}
	if shadcn.UIStyle != "nova" || shadcn.BaseColor != "mauve" || shadcn.ThemeColor != "rose" || shadcn.AccentColor != "emerald" || shadcn.IconTone != "colorful" {
		t.Fatalf("期望 shadcn profile 未被 artistic 覆盖，实际为 %#v", shadcn)
	}
}

// TestDisplayPreferencesSnapshotsIncludeProfilesAcrossRestart 验证 API 快照携带全部 profile，前端重启后切换方案不会用默认值覆盖数据库。
func TestDisplayPreferencesSnapshotsIncludeProfilesAcrossRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})

	saved, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "artistic",
		ThemeMode:     "dark",
		Profiles: app.DisplayProfiles{
			Shadcn: app.DisplayProfile{
				UIStyle:     "nova",
				BaseColor:   "mauve",
				ThemeColor:  "rose",
				AccentColor: "emerald",
				ChartColor:  "amber",
				IconTone:    "colorful",
				Menu:        "inverted-translucent",
				MenuAccent:  "bold",
				Radius:      "large",
				Density:     "compact",
				TextSize:    "large",
				CardBorder:  "soft",
			},
			Artistic: app.DisplayProfile{
				UIStyle:     "vega",
				BaseColor:   "stone",
				ThemeColor:  "orange",
				AccentColor: "cyan",
				ChartColor:  "blue",
				IconTone:    "colorful",
				Menu:        "inverted",
				MenuAccent:  "bold",
				Radius:      "small",
				Density:     "comfortable",
				TextSize:    "medium",
				CardBorder:  "visible",
			},
		},
	})
	if err != nil {
		t.Fatalf("保存带 profiles 的显示偏好失败：%v", err)
	}
	if saved.DisplayScheme != "artistic" || saved.Menu != "inverted" || saved.MenuAccent != "bold" || saved.Radius != "small" {
		t.Fatalf("期望 artistic 生效值来自 artistic profile，实际为 %#v", saved)
	}
	if saved.Profiles.Shadcn.UIStyle != "nova" || saved.Profiles.Shadcn.AccentColor != "emerald" {
		t.Fatalf("期望保存响应带回 shadcn profile，实际为 %#v", saved.Profiles)
	}
	if saved.Profiles.Artistic.AccentColor != saved.Profiles.Artistic.ThemeColor {
		t.Fatalf("期望 artistic profile 的品牌辅助色被归一化为品牌主题色，实际为 %#v", saved.Profiles.Artistic)
	}
	runtimeService.Shutdown()

	reloaded := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	snapshot := reloaded.DisplayPreferencesSnapshot()
	if snapshot.DisplayScheme != "artistic" || snapshot.Profiles.Shadcn.ThemeColor != "rose" || snapshot.Profiles.Artistic.Menu != "inverted" {
		t.Fatalf("期望重启后快照保留两套 profile，实际为 %#v", snapshot)
	}

	switched, err := reloaded.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "shadcn",
		ThemeMode:     snapshot.ThemeMode,
		Profiles:      snapshot.Profiles,
	})
	if err != nil {
		t.Fatalf("前端式切回 shadcn 失败：%v", err)
	}
	if switched.UIStyle != "nova" || switched.ThemeColor != "rose" || switched.AccentColor != "emerald" || switched.Menu != "inverted-translucent" {
		t.Fatalf("期望切回 shadcn 后恢复原 profile，实际为 %#v", switched)
	}
}

// TestDisplayPreferencesJSONDefaultsWhenDatabaseIsEmpty 验证空数据库只需要 JSON 默认项即可得到 artistic 默认生效偏好。
func TestDisplayPreferencesJSONDefaultsWhenDatabaseIsEmpty(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	preferences := runtimeService.DisplayPreferencesSnapshot()
	if preferences.DisplayScheme != "artistic" {
		t.Fatalf("期望空数据库默认显示方案为 artistic，实际为 %#v", preferences)
	}
	if preferences.Menu != "default" {
		t.Fatalf("期望空数据库默认菜单值可被设置页直接展示，实际为 %#v", preferences)
	}
	if preferences.BaseColor != "neutral" || preferences.ChartColor != "apple-blue" || preferences.Radius != "medium" || preferences.CardBorder != "visible" {
		t.Fatalf("期望空数据库默认值与外观设置页默认预设一致，实际为 %#v", preferences)
	}
}

// TestDisplayPreferencesArtisticNormalisesValues 验证后端裁决 artistic 方案非法值，并把辅助色托管为主题色。
func TestDisplayPreferencesArtisticNormalisesValues(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	if _, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "artistic",
		ThemeMode:     "dark",
	}); err != nil {
		t.Fatalf("切换到 artistic 显示方案失败：%v", err)
	}

	saved, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		DisplayScheme: "artistic",
		UIStyle:       "nova",
		ThemeMode:     "dark",
		BaseColor:     "mauve",
		ThemeColor:    "rose",
		AccentColor:   "emerald",
		ChartColor:    "amber",
		IconTone:      "colorful",
		Menu:          "default-translucent",
		MenuAccent:    "bold",
		Radius:        "none",
		Density:       "compact",
		TextSize:      "large",
		CardBorder:    "soft",
	})
	if err != nil {
		t.Fatalf("保存 artistic 显示偏好失败：%v", err)
	}
	if saved.DisplayScheme != "artistic" || saved.UIStyle != "nova" || saved.BaseColor != "mauve" || saved.ThemeColor != "rose" || saved.AccentColor != "rose" {
		t.Fatalf("期望 artistic 合法输入被保留，实际为 %#v", saved)
	}
	if saved.Menu != "default-translucent" || saved.Radius != "none" || saved.ChartColor != "amber" || saved.IconTone != "colorful" || saved.MenuAccent != "bold" || saved.Density != "compact" || saved.TextSize != "large" || saved.CardBorder != "soft" {
		t.Fatalf("期望 artistic 可编辑项保留合法输入，实际为 %#v", saved)
	}
}

// TestDefaultRuntimePathsUseExecutableDataDirectory 验证默认数据库、日志、崩溃状态和更新缓存都落在可写 data 目录。
func TestDefaultRuntimePathsUseExecutableDataDirectory(t *testing.T) {
	databasePath := filepath.ToSlash(app.DefaultDatabasePath("go-desktop"))
	logFilePath := filepath.ToSlash(app.DefaultLogFilePath("go-desktop"))
	crashLogPath := filepath.ToSlash(app.DefaultCrashLogPath("go-desktop"))
	crashStatePath := filepath.ToSlash(app.DefaultCrashStatePath("go-desktop"))
	cachePath := filepath.ToSlash(app.DefaultCachePath("go-desktop"))

	if !strings.Contains(databasePath, "/data/") || !strings.HasSuffix(databasePath, "/go-desktop.db") {
		t.Fatalf("expected database path under data directory, got %q", databasePath)
	}
	if !strings.Contains(logFilePath, "/data/logs/") || !strings.Contains(logFilePath, "/go-desktop-") || !strings.HasSuffix(logFilePath, ".log") {
		t.Fatalf("expected daily log path under data/logs directory, got %q", logFilePath)
	}
	if !strings.Contains(crashLogPath, "/data/logs/") || !strings.HasSuffix(crashLogPath, "/crash.log") {
		t.Fatalf("expected crash log path under data/logs directory, got %q", crashLogPath)
	}
	if !strings.Contains(crashStatePath, "/data/logs/") || !strings.HasSuffix(crashStatePath, "/crash-state.json") {
		t.Fatalf("expected crash state path under data/logs directory, got %q", crashStatePath)
	}
	if !strings.HasSuffix(cachePath, "/data/updates") {
		t.Fatalf("expected cache path to use data/updates directory, got %q", cachePath)
	}
}

// TestGetEnvironmentInfoReturnsRuntimeAndStoragePaths 验证环境诊断返回运行时信息和当前配置的存储路径。
func TestGetEnvironmentInfoReturnsRuntimeAndStoragePaths(t *testing.T) {
	tempDir := t.TempDir()
	databasePath := filepath.Join(tempDir, "go-desktop.db")
	logFilePath := filepath.Join(tempDir, "go-desktop.log")
	cachePath := filepath.Join(tempDir, "cache")

	runtimeService := app.NewRuntime(app.ServiceOptions{
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
	if info.DatabasePath != databasePath || !strings.HasPrefix(info.LogFilePath, tempDir) || !strings.Contains(info.LogFilePath, "go-desktop-") || info.CachePath != cachePath {
		t.Fatalf("expected configured paths in environment info, got %#v", info)
	}
	if strings.TrimSpace(info.WailsVersion) == "" {
		t.Fatalf("expected Wails module version, got %#v", info)
	}
}

// TestParseExitRequestRecognisesInstallerAndForceExitArgs 验证安装器/用户退出参数在 Wails 启动前就能被识别。
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

// TestRuntimeHideOnCloseHonoursSettingAndQuitState 验证关闭到托盘只受设置和显式退出状态控制，不拦截真正退出。
func TestRuntimeHideOnCloseHonoursSettingAndQuitState(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()
	if !runtimeService.ShouldHideOnClose() {
		t.Fatal("expected default settings to hide on close")
	}

	if _, err := runtimeService.SaveSettings(app.Settings{
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
		UpdateCheckIntervalHours: 3,
		MinimizeToTray:           true,
		LogRetentionDays:         30,
		CreateDesktopShortcut:    true,
	})
	if err == nil || !strings.Contains(err.Error(), "配置存储不可用") {
		t.Fatalf("expected config store unavailable error, got %v", err)
	}
}

// TestRecordSecondInstanceStoresCopyAndCapsHistory 验证第二实例参数会复制保存，并限制历史数量避免无界增长。
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
