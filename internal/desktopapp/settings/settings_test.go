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
}
