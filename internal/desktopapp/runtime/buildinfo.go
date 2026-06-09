// 文件职责：从 Go 构建信息读取运行时依赖版本。

package runtime

import "runtime/debug"

// moduleVersion 从当前二进制构建信息中提取指定 Go 模块版本。
// replace 依赖优先返回 replace 目标版本；未找到或没有版本信息时返回 unknown。
func moduleVersion(path string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path != path {
			continue
		}
		if dep.Replace != nil && dep.Replace.Version != "" {
			return dep.Replace.Version
		}
		if dep.Version != "" {
			return dep.Version
		}
		return "unknown"
	}
	return "unknown"
}
