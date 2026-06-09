//go:build windows

package installer

import (
	"context"
	"os/exec"
)

// RunSilent 在 Windows 上以 NSIS /S 参数启动安装器。
// 函数只等待进程成功启动，不等待安装过程结束。
func RunSilent(ctx context.Context, installerPath string) error {
	cmd := exec.CommandContext(ctx, installerPath, "/S")
	return cmd.Start()
}
