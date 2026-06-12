package settings_test

import (
	"testing"

	appsettings "github.com/chencn/go-desktop/internal/desktopapp/settings"
)

func TestSettingsConfigurationKeepsRepositoryMetadataOutOfUserSettings(t *testing.T) {
	definitions := appsettings.Definitions()
	values := appsettings.Values(appsettings.Default())

	for _, key := range []string{"github.owner", "github.repo"} {
		if _, ok := values[key]; ok {
			t.Fatalf("repository metadata key %q must not be saved as a user setting", key)
		}
		for _, definition := range definitions {
			if definition.Key == key {
				t.Fatalf("repository metadata key %q must not be exposed as a config item", key)
			}
		}
	}

	if _, ok := values["github.proxy_base"]; !ok {
		t.Fatal("github.proxy_base must remain a user setting")
	}
	if values["window.always_on_top"] != "false" {
		t.Fatalf("window.always_on_top should default to false, got %q", values["window.always_on_top"])
	}
	foundAlwaysOnTopDefinition := false
	for _, definition := range definitions {
		if definition.Key == "window.always_on_top" {
			foundAlwaysOnTopDefinition = true
			if definition.Category != "window" || definition.ValueType != "bool" {
				t.Fatalf("window.always_on_top should be a window bool config item, got %#v", definition)
			}
		}
	}
	if !foundAlwaysOnTopDefinition {
		t.Fatal("window.always_on_top must be exposed as a config item")
	}
}
