// Package semver 提供项目内部使用的简化版本号解析和比较。
// 它只接受 v?N、v?N.N 或 v?N.N.N 三种数字版本，不支持 prerelease/build metadata。

package semver

import (
	"strconv"
	"strings"
	"unicode"
)

// Version 是补齐到三段的整数版本，例如 "1.2" 会表示为 [1, 2, 0]。
type Version []int

// Parse 解析 v?N(.N){0,2} 格式版本，并把缺失段补 0。
// 任一段为空、包含非数字字符或超过三段时返回 false。
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

// Compare 比较两个可解析版本。
// 无效版本低于有效版本；两侧都无效时视为相等。
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

// Normalize 返回三段数字版本；无法解析时返回空字符串。
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
