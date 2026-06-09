// 文件职责：从稳定机器信息生成短设备码。

package machineid

import (
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

func DeviceCode() string {
	return CodeFromParts(readParts())
}

func CodeFromParts(parts []string) string {
	normalized := normalizeParts(parts)
	if len(normalized) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte("go-desktop-device:v1|" + strings.Join(normalized, "|")))
	encoded := strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:10]), "=")
	if len(encoded) < 12 {
		return ""
	}
	return "GD-" + encoded[0:4] + "-" + encoded[4:8] + "-" + encoded[8:12]
}

func normalizeParts(parts []string) []string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.ToUpper(strings.TrimSpace(part))
		switch value {
		case "", "TO BE FILLED BY O.E.M.", "00000000-0000-0000-0000-000000000000":
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}
