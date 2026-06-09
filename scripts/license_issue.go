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

// main 根据设备码签发授权码。
// 私钥从 GO_DESKTOP_LICENSE_PRIVATE_KEY 或向上查找的 .env 读取；-expires 为空时生成不过期授权。
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

// readLicensePrivateKey 优先读取进程环境变量，再回退到仓库或上级目录中的 .env。
func readLicensePrivateKey() string {
	if value := strings.TrimSpace(os.Getenv("GO_DESKTOP_LICENSE_PRIVATE_KEY")); value != "" {
		return value
	}
	return readDotEnvValue(findDotEnv(), "GO_DESKTOP_LICENSE_PRIVATE_KEY")
}

// findDotEnv 从当前目录向上查找 .env，允许在 scripts、frontend 或构建子目录执行。
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

// readDotEnvValue 读取简单 KEY=VALUE 行；注释、空行和不匹配的键会被忽略。
// 该解析器不展开变量，只去掉最外层单双引号。
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

// exitf 输出失败原因并以非零状态退出，避免生成无效授权码后被继续使用。
func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
