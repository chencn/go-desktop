// Package neterr 提供跨模块复用的网络失败分类工具。

package neterr

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
)

// IsOfflineError 判断错误是否属于网络离线、DNS、连接失败或超时场景。
// 该函数用于 UI 降噪和更新源切换提示，不应把业务校验错误归类为离线。
func IsOfflineError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return IsOfflineError(urlErr.Err)
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"no such host",
		"connection refused",
		"connection reset",
		"network is unreachable",
		"i/o timeout",
		"dial tcp",
		"connectex",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}
