//go:build !windows

package installer

import (
	"context"
	"errors"
)

func RunSilent(ctx context.Context, installerPath string) error {
	return errors.New("当前平台不支持运行 Windows 安装器")
}
