// ============================================================================
// 文件: internal/common/semver/semver.go
// 描述: 语义化版本解析和比较工具
//
// 功能概述:
// - 解析语义化版本字符串（如 "v1.2.3"）
// - 比较两个版本号的大小
// - 标准化版本号（移除 "v" 前缀）
// ============================================================================

package semver

import (
	"strconv" // 字符串转整数
	"strings" // 字符串处理
	"unicode" // Unicode 字符判断
)

// semanticVersion 是语义化版本的整数切片表示
// 例如 "1.2.3" 表示为 [1, 2, 3]
type Version []int

// parseVersion 解析语义化版本字符串
// 支持 "v" 或 "V" 前缀（如 "v1.2.3"、"V1.2.3"）
// 参数:
//   - value: 版本字符串
//
// 返回:
//   - semanticVersion: 解析后的版本号
//   - bool: 解析是否成功
func Parse(value string) (Version, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.TrimPrefix(value, "v"), "V")
	if value == "" {
		return nil, false
	}

	parts := strings.Split(value, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return nil, false
	}

	result := make([]int, 0, 3)
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
		for _, char := range part {
			if !unicode.IsDigit(char) {
				return nil, false
			}
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil, false
		}
		result = append(result, n)
	}
	for len(result) < 3 {
		result = append(result, 0)
	}
	return result, true
}

// compareVersions 比较两个版本号
// 参数:
//   - left: 左侧版本号
//   - right: 右侧版本号
//
// 返回:
//   - int: 1 表示 left > right，-1 表示 left < right，0 表示相等
func Compare(left, right string) int {
	lv, lok := Parse(left)
	rv, rok := Parse(right)
	if !lok && !rok {
		return 0
	}
	if !lok {
		return -1
	}
	if !rok {
		return 1
	}

	maxLen := len(lv)
	if len(rv) > maxLen {
		maxLen = len(rv)
	}
	for i := 0; i < maxLen; i++ {
		var l, r int
		if i < len(lv) {
			l = lv[i]
		}
		if i < len(rv) {
			r = rv[i]
		}
		if l > r {
			return 1
		}
		if l < r {
			return -1
		}
	}
	return 0
}

// normalisedVersion 标准化版本号（移除 "v" 前缀）
// 参数:
//   - value: 版本字符串
//
// 返回:
//   - string: 标准化后的版本号
func Normalize(value string) string {
	version, ok := Parse(value)
	if !ok {
		return ""
	}
	parts := make([]string, 0, 3)
	for _, part := range version {
		parts = append(parts, strconv.Itoa(part))
	}
	return strings.Join(parts, ".")
}
