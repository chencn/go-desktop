package build_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLicenseScriptsUsePrivateKeyOnlyForIssuing(t *testing.T) {
	keygen := readRootFile(t, "scripts", "license_keygen.go")
	issuer := readRootFile(t, "scripts", "license_issue.go")

	for _, want := range []string{"ed25519.GenerateKey", "publicKey=", "privateKey="} {
		if !strings.Contains(keygen, want) {
			t.Fatalf("license_keygen.go 必须生成密钥对字符串：缺少 %q", want)
		}
	}
	for _, want := range []string{"GO_DESKTOP_LICENSE_PRIVATE_KEY", "license.Issue", "-device", "readLicensePrivateKey", ".env", `strings.Cut(line, "=")`} {
		if !strings.Contains(issuer, want) {
			t.Fatalf("license_issue.go 必须能从环境变量或本地 .env 读取私钥并通过设备码生成授权码：缺少 %q", want)
		}
	}
	if strings.Contains(issuer, "GO_DESKTOP_LICENSE_PUBLIC_KEY") {
		t.Fatal("license_issue.go 生成授权码时不应需要公钥")
	}
}

func TestDotEnvExampleIncludesLocalPrivateKeyForIssuingOnly(t *testing.T) {
	source := readRootFile(t, ".env.example")

	for _, want := range []string{
		"GO_DESKTOP_LICENSE_PRIVATE_KEY=",
		"只给本地授权码签发脚本读取",
		"不要提交真实私钥",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf(".env.example 必须说明本地私钥签发配置：缺少 %q", want)
		}
	}
}

func TestLicenseIssueReadsPrivateKeyFromLocalDotEnv(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	workDir := rootPath(".tmp", "license-issue-test")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("create temp work dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(workDir, ".env"))
		_ = os.Remove(workDir)
	})
	privateKeyText := base64.RawURLEncoding.EncodeToString(privateKey)
	if err := os.WriteFile(filepath.Join(workDir, ".env"), []byte("GO_DESKTOP_LICENSE_PRIVATE_KEY="+privateKeyText+"\n"), 0o600); err != nil {
		t.Fatalf("write local .env: %v", err)
	}

	cmd := exec.Command("go", "run", filepath.Join("..", "..", "scripts", "license_issue.go"), "-device", "GD-7K3F-9P2X-MQ8C")
	cmd.Dir = workDir
	cmd.Env = withoutEnv(os.Environ(), "GO_DESKTOP_LICENSE_PRIVATE_KEY")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("license_issue.go should read private key from local .env: %v\n%s", err, stderr.String())
	}
	if got := strings.TrimSpace(string(output)); !strings.HasPrefix(got, "GD1-") {
		t.Fatalf("expected license code from local .env private key, got %q", got)
	}
}

func withoutEnv(env []string, name string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, name) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
