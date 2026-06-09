package timefmt

import "time"

// NowRFC3339 返回当前 UTC 时间的 RFC3339 字符串，用于跨模块统一日志/状态时间格式。
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
