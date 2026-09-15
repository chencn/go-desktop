package runtime

import "time"

// nowRFC3339 返回统一格式的当前时间字符串，用于 runtime 内部日志和状态 DTO。
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
