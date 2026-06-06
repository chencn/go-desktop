// 文件职责：提供存储层内部通用归一化 helper。

package configstore

import (
	"strings"
	"time"
)

func defaultTime(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return time.Now().UTC().Format(time.RFC3339)
}
