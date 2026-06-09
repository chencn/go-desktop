package runtime_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/license"
	appruntime "github.com/chencn/go-desktop/internal/desktopapp/runtime"
)

func TestLicenseDisabledWhenModeMissing(t *testing.T) {
	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db")})
	defer runtimeService.Shutdown()

	status := runtimeService.GetLicenseStatus()
	if status.Enabled || status.Required || !status.Authorized {
		t.Fatalf("missing license mode should disable authorization and allow app, got %#v", status)
	}
}

func TestRequiredLicenseNeedsActivation(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath:      filepath.Join(t.TempDir(), "go-desktop.db"),
		LicenseMode:       "required",
		LicensePublicKey:  encodePublicKeyForTest(publicKey),
		LicenseDeviceCode: "GD-7K3F-9P2X-MQ8C",
	})
	defer runtimeService.Shutdown()

	status := runtimeService.GetLicenseStatus()
	if !status.Enabled || !status.Required || status.Authorized {
		t.Fatalf("required mode without license should require activation, got %#v", status)
	}
	if status.DeviceCode != "GD-7K3F-9P2X-MQ8C" {
		t.Fatalf("expected configured device code, got %q", status.DeviceCode)
	}
}

func TestRequiredLicenseReportsMissingPublicKey(t *testing.T) {
	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath:      filepath.Join(t.TempDir(), "go-desktop.db"),
		LicenseMode:       "required",
		LicenseDeviceCode: "GD-7K3F-9P2X-MQ8C",
	})
	defer runtimeService.Shutdown()

	status := runtimeService.GetLicenseStatus()
	if status.Authorized || !strings.Contains(status.LastError, "缺少授权公钥") {
		t.Fatalf("expected missing public key configuration error, got %#v", status)
	}
}

func TestRequiredLicenseReportsMissingDeviceCode(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath:            filepath.Join(t.TempDir(), "go-desktop.db"),
		LicenseMode:             "required",
		LicensePublicKey:        encodePublicKeyForTest(publicKey),
		LicenseDeviceCodeSource: func() string { return "" },
	})
	defer runtimeService.Shutdown()

	status := runtimeService.GetLicenseStatus()
	if status.Authorized || !strings.Contains(status.LastError, "设备码生成失败") {
		t.Fatalf("expected missing device code error, got %#v", status)
	}
}

func TestRequiredLicenseBlocksBusinessAPIsUntilActivated(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	runtimeService := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath:      filepath.Join(t.TempDir(), "go-desktop.db"),
		LicenseMode:       "required",
		LicensePublicKey:  encodePublicKeyForTest(publicKey),
		LicenseDeviceCode: "GD-7K3F-9P2X-MQ8C",
	})
	defer runtimeService.Shutdown()

	api := runtimeService.API()
	if _, err := api.GetLicenseStatus(); err != nil {
		t.Fatalf("GetLicenseStatus should remain available before activation: %v", err)
	}
	if _, err := api.GetSettings(); err == nil || !strings.Contains(err.Error(), "需要授权") {
		t.Fatalf("GetSettings should be blocked before activation, got %v", err)
	}

	code, err := license.Issue(license.Payload{
		Version:    1,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		IssuedAt:   time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.ActivateLicense(code); err != nil {
		t.Fatalf("ActivateLicense should remain available before activation: %v", err)
	}
	if _, err := api.GetSettings(); err != nil {
		t.Fatalf("GetSettings should be available after activation: %v", err)
	}
}

func TestActivateLicensePersistsAndRestores(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	code, err := license.Issue(license.Payload{
		Version:    1,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		IssuedAt:   time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	first := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath:      dbPath,
		LicenseMode:       "required",
		LicensePublicKey:  encodePublicKeyForTest(publicKey),
		LicenseDeviceCode: "GD-7K3F-9P2X-MQ8C",
	})

	status, err := first.ActivateLicense(code)
	first.Shutdown()
	if err != nil || !status.Authorized {
		t.Fatalf("ActivateLicense() status=%#v err=%v", status, err)
	}

	restarted := appruntime.NewRuntime(appruntime.ServiceOptions{
		DatabasePath:      dbPath,
		LicenseMode:       "required",
		LicensePublicKey:  encodePublicKeyForTest(publicKey),
		LicenseDeviceCode: "GD-7K3F-9P2X-MQ8C",
	})
	defer restarted.Shutdown()
	if status := restarted.GetLicenseStatus(); !status.Authorized {
		t.Fatalf("expected persisted license to authorize after restart, got %#v", status)
	}
}

func encodePublicKeyForTest(key ed25519.PublicKey) string {
	return base64.RawURLEncoding.EncodeToString(key)
}
