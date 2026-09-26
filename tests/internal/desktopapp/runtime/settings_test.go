package runtime_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	appruntime "github.com/chencn/go-desktop/internal/desktopapp/runtime"
)

func TestSaveSettingsRollsBackWhenStartupIntegrationFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	integrationErr := errors.New("startup integration failed")
	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath: dbPath,
		StartupIntegrationApplier: func(previous appruntime.Settings, next appruntime.Settings) error {
			if previous.AutoLaunch != next.AutoLaunch {
				return integrationErr
			}
			return nil
		},
	})

	previous := runtimeService.SettingsSnapshot()
	_, err := runtimeService.SaveSettings(appruntime.Settings{
		UpdateSource:             "local",
		UpdateCheckIntervalHours: 6,
		MinimizeToTray:           false,
		AlwaysOnTop:              !previous.AlwaysOnTop,
		LogRetentionDays:         60,
		LogLevel:                 "debug",
		AutoLaunch:               !previous.AutoLaunch,
		CreateDesktopShortcut:    false,
		LaunchHiddenToTray:       true,
	})
	if err == nil || !strings.Contains(err.Error(), integrationErr.Error()) {
		t.Fatalf("expected startup integration error, got %v", err)
	}
	if current := runtimeService.SettingsSnapshot(); current != previous {
		t.Fatalf("expected in-memory settings to roll back to %#v, got %#v", previous, current)
	}
	runtimeService.Shutdown()

	reloaded := appruntime.NewRuntime(appruntime.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	if current := reloaded.SettingsSnapshot(); current != previous {
		t.Fatalf("expected SQLite settings to roll back to %#v, got %#v", previous, current)
	}
}

// TestEmptyGitHubProxyBaseSurvivesRestart 锁定「清空 GitHub 代理」是合法的持久化取值：
// GitHubProxyBase 空串表示直连官方 API，重启后不能被 metadata 的默认代理地址覆盖回去。
func TestEmptyGitHubProxyBaseSurvivesRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")

	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath: dbPath,
		// 本用例只验证 KV 往返，系统集成换成空实现，避免测试触达注册表/快捷方式。
		StartupIntegrationApplier: func(previous appruntime.Settings, next appruntime.Settings) error {
			return nil
		},
	})
	saved, err := runtimeService.SaveSettings(appruntime.Settings{
		UpdateSource:             "github",
		GitHubProxyBase:          "",
		UpdateCheckIntervalHours: 3,
		LogRetentionDays:         30,
		LogLevel:                 "info",
	})
	runtimeService.Shutdown()
	if err != nil {
		t.Fatalf("save settings with cleared proxy: %v", err)
	}
	if saved.GitHubProxyBase != "" {
		t.Fatalf("save should keep the cleared proxy choice, got %q", saved.GitHubProxyBase)
	}

	reloaded := appruntime.NewRuntime(appruntime.ServiceOptions{DatabasePath: dbPath})
	defer reloaded.Shutdown()
	if got := reloaded.SettingsSnapshot().GitHubProxyBase; got != "" {
		t.Fatalf("cleared GitHubProxyBase must survive restart, got %q", got)
	}
}
