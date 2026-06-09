//go:build !windows

package process

import (
	"os"
	"syscall"
)

// Alive 通过 signal 0 判断非 Windows 进程是否仍可访问。
// pid 无效、查找失败或 signal 失败时返回 false。
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
