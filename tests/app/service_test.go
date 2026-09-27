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
		AlwaysOnTop:              true,
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
	if !settings.AlwaysOnTop {
		t.Fatalf("expected alwaysOnTop=true to persist, got %#v", settings)
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
	if settings.AlwaysOnTop {
		t.Fatalf("expected default alwaysOnTop=false, got %#v", settings)
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
		ThemeMode: "dark",
		Size:      "small",
	})
	if err != nil {
		t.Fatalf("save display preferences: %v", err)
	}
	if saved.ThemeMode != "dark" || saved.Size != "small" {
		t.Fatalf("expected display preferences to save, got %#v", saved)
	}

	reloaded := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	preferences := reloaded.DisplayPreferencesSnapshot()
	if preferences.ThemeMode != "dark" || preferences.Size != "small" {
		t.Fatalf("expected display preferences from sqlite, got %#v", preferences)
	}
}

// TestDisplayPreferencesNormaliseInvalidValues 验证非法显示偏好回退到默认值，同时保留合法输入。
func TestDisplayPreferencesNormaliseInvalidValues(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	saved, err := runtimeService.SaveDisplayPreferences(app.DisplayPreferences{
		ThemeMode: "night",
		Size:      "huge",
	})
	if err != nil {
		t.Fatalf("save display preferences: %v", err)
	}
	if saved.ThemeMode != "light" || saved.Size != "default" {
		t.Fatalf("expected invalid display values to fall back to defaults, got %#v", saved)
	}
}

// TestDisplayPreferencesJSONDefaultsWhenDatabaseIsEmpty 验证空数据库返回 V3 默认偏好。
func TestDisplayPreferencesJSONDefaultsWhenDatabaseIsEmpty(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	defer runtimeService.Shutdown()

	preferences := runtimeService.DisplayPreferencesSnapshot()
	if preferences.ThemeMode != "light" || preferences.Size != "default" {
		t.Fatalf("期望空数据库返回默认亮暗模式和全局尺寸，实际为 %#v", preferences)
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
