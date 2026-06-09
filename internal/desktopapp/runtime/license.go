// 文件职责：管理运行时授权状态、授权码持久化和 Wails API。

package runtime

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/license"
	"github.com/chencn/go-desktop/internal/platform/machineid"
)

const (
	licenseKeyConfig         = "license.key"
	licenseDeviceCodeConfig  = "license.device_code"
	licenseValidatedAtConfig = "license.validated_at"
	licenseLastErrorConfig   = "license.last_error"
)

type LicenseStatus struct {
	Enabled    bool   `json:"enabled"`
	Required   bool   `json:"required"`
	Authorized bool   `json:"authorized"`
	DeviceCode string `json:"deviceCode"`
	Message    string `json:"message"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
	LastError  string `json:"lastError,omitempty"`
}

func (api *API) GetLicenseStatus() (status LicenseStatus, err error) {
	defer api.recoverError("读取授权状态", &err)
	return api.runtime.GetLicenseStatus(), nil
}

func (api *API) ActivateLicense(licenseKey string) (status LicenseStatus, err error) {
	defer api.recoverError("激活授权", &err)
	return api.runtime.ActivateLicense(licenseKey)
}

func (s *Runtime) GetLicenseStatus() LicenseStatus {
	if !s.licenseRequired() {
		return LicenseStatus{Authorized: true, Message: "授权未启用"}
	}
	deviceCode := s.currentLicenseDeviceCode()
	if strings.TrimSpace(deviceCode) == "" {
		return LicenseStatus{Enabled: true, Required: true, Message: "授权配置错误", LastError: "设备码生成失败"}
	}
	if strings.TrimSpace(s.licensePublicKey) == "" {
		return LicenseStatus{Enabled: true, Required: true, DeviceCode: deviceCode, Message: "授权配置错误", LastError: "缺少授权公钥"}
	}
	key := s.storedLicenseKey()
	if strings.TrimSpace(key) == "" {
		return LicenseStatus{Enabled: true, Required: true, DeviceCode: deviceCode, Message: "需要授权"}
	}
	status, err := s.verifyLicenseKey(key, deviceCode)
	if err != nil {
		return LicenseStatus{Enabled: true, Required: true, DeviceCode: deviceCode, Message: "需要授权", LastError: err.Error()}
	}
	status.Enabled = true
	status.Required = true
	status.DeviceCode = deviceCode
	status.Message = "授权已通过"
	return status
}

func (s *Runtime) ActivateLicense(code string) (LicenseStatus, error) {
	if !s.licenseRequired() {
		return s.GetLicenseStatus(), nil
	}
	deviceCode := s.currentLicenseDeviceCode()
	if strings.TrimSpace(deviceCode) == "" {
		err := errors.New("设备码生成失败")
		_ = s.saveConfigValues(context.Background(), map[string]string{licenseLastErrorConfig: err.Error()})
		return s.GetLicenseStatus(), err
	}
	status, err := s.verifyLicenseKey(code, deviceCode)
	if err != nil {
		_ = s.saveConfigValues(context.Background(), map[string]string{licenseLastErrorConfig: err.Error()})
		return s.GetLicenseStatus(), err
	}
	if err := s.saveConfigValues(context.Background(), map[string]string{
		licenseKeyConfig:         normaliseLicenseKey(code),
		licenseDeviceCodeConfig:  deviceCode,
		licenseValidatedAtConfig: time.Now().UTC().Format(time.RFC3339),
		licenseLastErrorConfig:   "",
	}); err != nil {
		return s.GetLicenseStatus(), fmt.Errorf("保存授权码失败：%w", err)
	}
	status.Enabled = true
	status.Required = true
	status.DeviceCode = deviceCode
	status.Message = "授权已通过"
	return status, nil
}

func (s *Runtime) licenseRequired() bool {
	return strings.EqualFold(strings.TrimSpace(s.licenseMode), "required")
}

func (s *Runtime) currentLicenseDeviceCode() string {
	if strings.TrimSpace(s.licenseDeviceCode) != "" {
		return strings.TrimSpace(s.licenseDeviceCode)
	}
	if s.licenseDeviceCodeSource != nil {
		return strings.TrimSpace(s.licenseDeviceCodeSource())
	}
	return machineid.DeviceCode()
}

func (s *Runtime) storedLicenseKey() string {
	items, err := s.configItemsByKey(context.Background())
	if err != nil {
		return ""
	}
	return items[licenseKeyConfig].Value
}

func (s *Runtime) verifyLicenseKey(code string, deviceCode string) (LicenseStatus, error) {
	publicKey, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(s.licensePublicKey))
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return LicenseStatus{}, fmt.Errorf("授权公钥无效")
	}
	verified, err := license.Verify(normaliseLicenseKey(code), license.VerifyOptions{
		PublicKey:  ed25519.PublicKey(publicKey),
		Product:    "go-desktop",
		DeviceCode: deviceCode,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		return LicenseStatus{}, err
	}
	expiresAt := ""
	if !verified.Payload.ExpiresAt.IsZero() {
		expiresAt = verified.Payload.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return LicenseStatus{Authorized: true, ExpiresAt: expiresAt}, nil
}

func (api *API) requireAuthorized() error {
	if api == nil || api.runtime == nil {
		return errors.New("授权状态不可用")
	}
	status := api.runtime.GetLicenseStatus()
	if !status.Required || status.Authorized {
		return nil
	}
	message := strings.TrimSpace(status.LastError)
	if message == "" {
		message = strings.TrimSpace(status.Message)
	}
	if message == "" {
		message = "需要授权"
	}
	return errors.New(message)
}

func normaliseLicenseKey(value string) string {
	return strings.Join(strings.Fields(value), "")
}
