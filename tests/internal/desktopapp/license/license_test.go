package license_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/license"
)

func TestIssueAndVerifyDeviceBoundLicense(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := license.Payload{
		Version:    1,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		IssuedAt:   time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}

	code, err := license.Issue(payload, privateKey)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if !strings.HasPrefix(code, "GD1-") {
		t.Fatalf("授权码前缀错误：%q", code)
	}

	status, err := license.Verify(code, license.VerifyOptions{
		PublicKey:  publicKey,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		Now:        time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !status.Authorized || status.Payload.DeviceCode != payload.DeviceCode {
		t.Fatalf("expected authorized license for device, got %#v", status)
	}
}

func TestVerifyRejectsMismatchedDevice(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	code, err := license.Issue(license.Payload{
		Version:    1,
		Product:    "go-desktop",
		DeviceCode: "GD-AAAA-BBBB-CCCC",
		IssuedAt:   time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	_, err = license.Verify(code, license.VerifyOptions{
		PublicKey:  publicKey,
		Product:    "go-desktop",
		DeviceCode: "GD-XXXX-YYYY-ZZZZ",
		Now:        time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err == nil || err.Error() != "授权码不属于当前设备" {
		t.Fatalf("expected device mismatch error, got %v", err)
	}
}

func TestVerifyRejectsTamperedLicense(t *testing.T) {
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
	bodyText, signatureText, ok := strings.Cut(strings.TrimPrefix(code, license.Prefix+"-"), ".")
	if !ok {
		t.Fatalf("generated license should contain signature separator: %q", code)
	}
	body, err := base64.RawURLEncoding.DecodeString(bodyText)
	if err != nil || len(body) == 0 {
		t.Fatalf("generated license body should decode: %v", err)
	}
	body[0] ^= 1
	tampered := license.Prefix + "-" + base64.RawURLEncoding.EncodeToString(body) + "." + signatureText

	_, err = license.Verify(tampered, license.VerifyOptions{
		PublicKey:  publicKey,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		Now:        time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err == nil || err.Error() != "授权码签名无效" {
		t.Fatalf("expected invalid signature error, got %v", err)
	}
}

func TestVerifyAcceptsWrappedLicenseCode(t *testing.T) {
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
	wrapped := strings.Replace(code, ".", "\n.\n", 1)

	status, err := license.Verify(wrapped, license.VerifyOptions{
		PublicKey:  publicKey,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		Now:        time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil || !status.Authorized {
		t.Fatalf("expected wrapped multiline license code to verify, status=%#v err=%v", status, err)
	}
}

func TestVerifyRejectsExpiredLicense(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	code, err := license.Issue(license.Payload{
		Version:    1,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		IssuedAt:   time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
		ExpiresAt:  time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	}, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	_, err = license.Verify(code, license.VerifyOptions{
		PublicKey:  publicKey,
		Product:    "go-desktop",
		DeviceCode: "GD-7K3F-9P2X-MQ8C",
		Now:        time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC),
	})
	if err == nil || err.Error() != "授权码已过期" {
		t.Fatalf("expected expired error, got %v", err)
	}
}
