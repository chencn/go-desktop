// Package machineid 从平台机器信息生成授权用短设备码。

package machineid

import (
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

// DeviceCode 读取当前平台可用机器信息并生成 GD-XXXX-XXXX-XXXX 格式设备码。
// 如果平台信息为空或只包含占位值，返回空字符串，由授权流程报告缺少设备码。
func DeviceCode() string {
	return CodeFromParts(readParts())
}

// CodeFromParts 根据已采集的机器信息生成稳定短码。
// 输入会去空白、转大写并过滤常见 OEM/空 UUID 占位值；输出不包含原始机器标识。
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

// normalizeParts 过滤不稳定或无意义的机器信息，避免把 OEM 默认值写入授权指纹。
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
