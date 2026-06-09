//go:build windows

package machineid

import "golang.org/x/sys/windows/registry"

// readParts 在 Windows 上读取 HKLM MachineGuid，失败时返回空片段交给通用层过滤。
func readParts() []string {
	return []string{readMachineGuid()}
}

// readMachineGuid 读取 64 位注册表视图中的 MachineGuid。
// 无权限、键缺失或值类型不匹配时返回空字符串。
func readMachineGuid() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer key.Close()
	value, _, err := key.GetStringValue("MachineGuid")
	if err != nil {
		return ""
	}
	return value
}
