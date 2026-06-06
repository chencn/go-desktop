// ============================================================================
// 文件: app/buildinfo.go
// 描述: 构建信息工具函数，用于读取 Go 模块依赖版本
// ============================================================================

package runtime

import "runtime/debug"

// moduleVersion 从 Go 二进制构建信息中提取指定模块的版本号
//
// 参数:
//   - path: Go 模块路径（如 "github.com/wailsapp/wails/v3"）
//
// 返回:
//   - 模块版本号字符串，如果未找到则返回 "unknown"
//
// 说明:
//   - 优先返回 replace 指令中的版本（用于本地开发替换）
//   - 其次返回依赖列表中的版本号
//   - 用于运行时检测 Wails 框架版本
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
