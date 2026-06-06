// 文件职责：提供跨模块复用的网络错误分类工具。

package neterr

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
)

// IsOfflineError 判断错误是否属于网络离线或连接不可用场景。
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
