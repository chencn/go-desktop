package process

import "golang.org/x/sys/windows"

// windowsStillActive 是 GetExitCodeProcess 返回的 STILL_ACTIVE 值。
const windowsStillActive = 259

// Alive 判断 pid 对应的 Windows 进程是否仍处于 STILL_ACTIVE。
// 无权限、pid 无效或查询失败时返回 false，供崩溃哨兵按保守路径处理。
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == windowsStillActive
}
