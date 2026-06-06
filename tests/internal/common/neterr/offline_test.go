// 文件职责：验证网络错误分类工具能被多个业务模块复用。

package neterr_test

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/chencn/go-desktop/internal/common/neterr"
)

// TestIsOfflineErrorRecognisesNetworkFailures 验证常见网络失败会被归类为离线错误。
func TestIsOfflineErrorRecognisesNetworkFailures(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "deadline exceeded", err: context.DeadlineExceeded},
		{name: "dns error", err: &net.DNSError{Err: "no such host", Name: "example.invalid"}},
		{name: "wrapped url error", err: &url.Error{Op: "Get", URL: "https://example.invalid", Err: errors.New("dial tcp: connection refused")}},
		{name: "message fallback", err: errors.New("network is unreachable")},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if !neterr.IsOfflineError(tt.err) {
				t.Fatalf("expected %v to be classified as offline", tt.err)
			}
		})
	}
}

// TestIsOfflineErrorIgnoresNonNetworkErrors 验证普通业务错误不会被误判为离线。
func TestIsOfflineErrorIgnoresNonNetworkErrors(t *testing.T) {
	if neterr.IsOfflineError(errors.New("sha256 mismatch")) {
		t.Fatal("expected non-network error to stay non-offline")
	}
}
