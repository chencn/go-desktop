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
