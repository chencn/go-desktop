//go:build !windows

package machineid

import "os"

// readParts 在非 Windows 平台退回主机名；授权层仍会通过通用过滤处理空值。
func readParts() []string {
	host, _ := os.Hostname()
	return []string{host}
}
