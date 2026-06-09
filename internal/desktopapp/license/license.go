// 文件职责：提供离线授权码载荷、签发和验签能力。

package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Prefix 是当前离线授权码格式版本前缀。
const Prefix = "GD1"

// Payload 是授权码签名覆盖的 JSON 载荷。
type Payload struct {
	Version    int       `json:"v"`
	Product    string    `json:"product"`
	DeviceCode string    `json:"deviceCode"`
	IssuedAt   time.Time `json:"issuedAt"`
	ExpiresAt  time.Time `json:"expiresAt,omitempty"`
}

// VerifyOptions 是验签所需的本地上下文。
type VerifyOptions struct {
	PublicKey  ed25519.PublicKey
	Product    string
	DeviceCode string
	Now        time.Time
}

// Status 是验签成功后的授权状态和原始载荷。
type Status struct {
	Authorized bool
	Payload    Payload
}

// Issue 使用 Ed25519 私钥签发离线授权码，格式为 GD1-base64url(payload).base64url(signature)。
func Issue(payload Payload, privateKey ed25519.PrivateKey) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", errors.New("授权私钥无效")
	}
	if payload.Version == 0 {
		payload.Version = 1
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	signature := ed25519.Sign(privateKey, body)
	return Prefix + "-" + base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// Verify 验证授权码签名、产品、设备码和过期时间。
func Verify(code string, options VerifyOptions) (Status, error) {
	body, signature, err := decode(code)
	if err != nil {
		return Status{}, err
	}
	if len(options.PublicKey) != ed25519.PublicKeySize || !ed25519.Verify(options.PublicKey, body, signature) {
		return Status{}, errors.New("授权码签名无效")
	}

	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return Status{}, errors.New("授权码载荷无效")
	}
	if payload.Version != 1 || payload.Product != options.Product {
		return Status{}, errors.New("授权码产品不匹配")
	}
	if payload.DeviceCode != options.DeviceCode {
		return Status{}, errors.New("授权码不属于当前设备")
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !payload.ExpiresAt.IsZero() && !payload.ExpiresAt.After(now) {
		return Status{}, errors.New("授权码已过期")
	}
	return Status{Authorized: true, Payload: payload}, nil
}

// decode 解析授权码格式并解码签名覆盖的 payload 和 signature。
func decode(code string) ([]byte, []byte, error) {
	trimmed := strings.Join(strings.Fields(code), "")
	if !strings.HasPrefix(trimmed, Prefix+"-") {
		return nil, nil, errors.New("授权码格式无效")
	}
	rest := strings.TrimPrefix(trimmed, Prefix+"-")
	bodyText, signatureText, ok := strings.Cut(rest, ".")
	if !ok || bodyText == "" || signatureText == "" {
		return nil, nil, errors.New("授权码格式无效")
	}
	body, err := base64.RawURLEncoding.DecodeString(bodyText)
	if err != nil {
		return nil, nil, errors.New("授权码格式无效")
	}
	signature, err := base64.RawURLEncoding.DecodeString(signatureText)
	if err != nil {
		return nil, nil, errors.New("授权码格式无效")
	}
	return body, signature, nil
}
