//go:build ignore

package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/desktopapp/license"
)

func main() {
	device := flag.String("device", "", "device code")
	expires := flag.String("expires", "", "expiry date YYYY-MM-DD, optional")
	flag.Parse()

	if *device == "" {
		exitf("缺少 -device")
	}
	privateKeyText := readLicensePrivateKey()
	privateKey, err := base64.RawURLEncoding.DecodeString(privateKeyText)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		exitf("GO_DESKTOP_LICENSE_PRIVATE_KEY 无效")
	}

	payload := license.Payload{
		Version:    1,
		Product:    "go-desktop",
		DeviceCode: *device,
		IssuedAt:   time.Now().UTC(),
	}
	if *expires != "" {
		expiresAt, err := time.Parse("2006-01-02", *expires)
		if err != nil {
			exitf("-expires 必须是 YYYY-MM-DD")
		}
		payload.ExpiresAt = expiresAt
	}

	code, err := license.Issue(payload, ed25519.PrivateKey(privateKey))
	if err != nil {
		exitf("生成授权码失败：%v", err)
	}
	fmt.Println(code)
}

func readLicensePrivateKey() string {
	if value := strings.TrimSpace(os.Getenv("GO_DESKTOP_LICENSE_PRIVATE_KEY")); value != "" {
		return value
	}
	return readDotEnvValue(findDotEnv(), "GO_DESKTOP_LICENSE_PRIVATE_KEY")
}

func findDotEnv() string {
	dir, err := os.Getwd()
	if err != nil {
		return ".env"
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join(dir, ".env")
		}
		dir = parent
	}
}

func readDotEnvValue(path string, name string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != name {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ""
}

func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
