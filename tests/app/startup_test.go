// 文件职责：验证 startup_test.go 覆盖的生产行为、结构约束或构建脚本约束。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package app_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/chencn/go-desktop/app"
)

// TestParseStartupLaunchRecognisesHiddenFlag 验证 startup_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestParseStartupLaunchRecognisesHiddenFlag(t *testing.T) {
	cases := []struct {
		name         string
		args         []string
		wantHidden   bool
		wantShortcut bool
	}{
		{name: "short hidden", args: []string{"-startup-hidden"}, wantHidden: true},
		{name: "long hidden", args: []string{"--startup-hidden"}, wantHidden: true},
		{name: "desktop shortcut", args: []string{"--desktop-shortcut"}, wantShortcut: true},
		{name: "both flags", args: []string{"--desktop-shortcut", "--startup-hidden"}, wantHidden: true, wantShortcut: true},
		{name: "no startup source", args: []string{"--check-update"}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := app.ParseStartupLaunch(tt.args)
			if got.Hidden != tt.wantHidden || got.DesktopShortcut != tt.wantShortcut {
				t.Fatalf("expected hidden=%v desktopShortcut=%v, got %#v", tt.wantHidden, tt.wantShortcut, got)
			}
		})
	}
}

// TestShouldStartHiddenRequiresAutoLaunchSettingAndFlag 验证 startup_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestShouldStartHiddenRequiresAutoLaunchSettingAndFlag(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	if runtimeService.ShouldStartHidden(app.StartupLaunch{Hidden: true}) {
		t.Fatal("expected default autoLaunch=false to ignore startup hidden flag")
	}

	if _, err := runtimeService.SaveSettings(app.Settings{
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 3,
		MinimizeToTray:           true,
		LogRetentionDays:         30,
		AutoLaunch:               true,
		CreateDesktopShortcut:    true,
		LaunchHiddenToTray:       true,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if !runtimeService.ShouldStartHidden(app.StartupLaunch{Hidden: true}) {
		t.Fatal("expected autoLaunch + launchHiddenToTray + startup flag to hide the window")
	}
	if runtimeService.ShouldStartHidden(app.StartupLaunch{}) {
		t.Fatal("expected manual launch without startup flag to show the window")
	}
}

// TestStartupAutostartArgumentsOnlyIncludeHiddenWhenEnabled 验证 startup_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestStartupAutostartArgumentsOnlyIncludeHiddenWhenEnabled(t *testing.T) {
	if args := app.StartupAutostartArguments(app.Settings{AutoLaunch: false, LaunchHiddenToTray: true}); len(args) != 0 {
		t.Fatalf("expected no args when auto launch is disabled, got %#v", args)
	}
	args := app.StartupAutostartArguments(app.Settings{AutoLaunch: true, LaunchHiddenToTray: true})
	if len(args) != 1 || args[0] != "--startup-hidden" {
		t.Fatalf("expected startup hidden arg, got %#v", args)
	}
}

// TestRecordStartupLaunchLogsDesktopShortcutSource 验证桌面快捷图标启动会进入运行日志。
func TestRecordStartupLaunchLogsDesktopShortcutSource(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	runtimeService.RecordStartupLaunch(app.StartupLaunch{DesktopShortcut: true})

	logs := runtimeService.QueryLogs(app.LogQuery{Scope: "startup"})
	if logs.Total != 1 || logs.Logs[0].Message != "桌面快捷图标启动" {
		t.Fatalf("expected desktop shortcut startup log, got %#v", logs.Logs)
	}

	runtimeService.RecordStartupLaunch(app.StartupLaunch{})
	logs = runtimeService.QueryLogs(app.LogQuery{Scope: "startup"})
	if logs.Total != 1 {
		t.Fatalf("expected manual startup source to stay silent, got %#v", logs.Logs)
	}
}

// TestMainDefinesLicenseBuildVariables 验证主入口定义授权构建变量并传给 Runtime。
func TestMainDefinesLicenseBuildVariables(t *testing.T) {
	source := readRootFile(t, "main.go")

	for _, want := range []string{
		`licenseMode`,
		`licensePublicKey`,
		"LicenseMode:      licenseMode",
		"LicensePublicKey: licensePublicKey",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("main.go 必须定义并传递授权构建变量：缺少 %q", want)
		}
	}
}

// TestReadmeDocumentsLicenseUsage 验证 README 有单独授权章节说明本地和 GitHub 用法。
func TestReadmeDocumentsLicenseUsage(t *testing.T) {
	source := readRootFile(t, "README.md")

	for _, want := range []string{
		"## 授权码模式",
		".env.example",
		"GO_DESKTOP_LICENSE_MODE=required",
		"GO_DESKTOP_LICENSE_PUBLIC_KEY",
		"GO_DESKTOP_LICENSE_PRIVATE_KEY",
		"GitHub Repository Variable",
		"go run ./scripts/envrun wails3 task dev",
		"go run ./scripts/envrun wails3 task package",
		"license_issue.go",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("README 授权码模式章节缺少 %q", want)
		}
	}
}
