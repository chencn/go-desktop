package runtime

import "github.com/chencn/go-desktop/internal/common/timefmt"

// nowRFC3339 返回统一格式的当前时间字符串，用于 runtime 内部日志和状态 DTO。
func nowRFC3339() string {
	return timefmt.NowRFC3339()
}
