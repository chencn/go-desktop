//go:build windows

package installer

import (
	"context"
	"os/exec"
)

func RunSilent(ctx context.Context, installerPath string) error {
	cmd := exec.CommandContext(ctx, installerPath, "/S")
	return cmd.Start()
}
