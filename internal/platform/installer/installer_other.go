//go:build !windows

package installer

import (
	"context"
	"errors"
)

// RunSilent 在非 Windows 平台固定返回不支持，因为当前更新资产是 Windows 安装器。
func RunSilent(ctx context.Context, installerPath string) error {
	return errors.New("当前平台不支持运行 Windows 安装器")
}
