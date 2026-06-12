// 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
//
// 这些值同时影响运行时、更新检查、安装包命名和 Windows 桌面生命周期。
// 修改产品名、仓库、版本兜底或窗口类名时，只改 project.metadata.json，再运行同步脚本。
package metadata

import "fmt"

// 常量块声明读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 需要跨函数复用的固定值。
const (
	// 公司名用于构建元信息、安装器注册表项和发布标签。
	CompanyName = "chencn"
	// 应用名是标准产品名和可执行文件名；当前也等于 GitHub 仓库名。
	AppName = "go-desktop"
	// Go 模块路径用于校验绑定导入路径和生成脚本。
	ModulePath = "github.com/chencn/go-desktop"
	// 默认版本只作为开发兜底；正式构建通过 ldflags 覆盖。
	DefaultVersion = "1.0.0"
	// 应用描述会展示在 Wails 运行时信息和前端“关于”页面。
	Description = "Wails3 中文桌面工具"
	// 仓库地址是给用户看的项目地址，不是 GitHub API 端点。
	RepositoryURL = "https://github.com/chencn/go-desktop"
	// 仓库备注用于 Windows 版本资源和安装器元信息。
	RepositoryComment = "github.com/chencn/go-desktop"
	// 版权文本用于 Windows 版本资源和安装器元信息。
	Copyright = "© 2026, chencn"

	// GitHub 仓库归属和仓库名定义更新检查使用的公开 Release 来源。
	GitHubOwner = "chencn"
	GitHubRepo  = "go-desktop"
	// GitHub API 地址和版本必须和测试、检查器默认值、前端预览兜底保持一致。
	GitHubAPIBase    = "https://api.github.com"
	GitHubAPIVersion = "2026-03-10"
	// 请求标识是 GitHub REST API 要求项，Release 列表和资产下载都复用它。
	UserAgent = "go-desktop-updater"

	// 更新源默认值和本地静态升级 manifest 位置。
	DefaultUpdateSource     = "github"
	LocalUpdateBaseURL      = "http://www.xqchen.shop/exe/go-desktop"
	LocalUpdateManifestPath = "releases/latest.json"

	// 默认设置由同一份项目元数据派生，避免 Go、前端和脚本各写一份。
	DefaultGitHubProxyBase          = "https://gh-proxy.com"
	DefaultUpdateCheckIntervalHours = 3
	DefaultMinimizeToTray           = true
	DefaultAlwaysOnTop              = false
	DefaultLogRetentionDays         = 30
	DefaultAutoLaunch               = false
	DefaultCreateDesktopShortcut    = true
	DefaultLaunchHiddenToTray       = false

	// 单实例标识跨版本必须稳定，否则第二次启动无法找到已运行实例。
	WindowsSingleInstanceID = "com.github.chencn.go-desktop"
	// Windows 产品标识用于构建产物和平台元信息。
	WindowsProductID = "com.github.chencn.godesktop"
	// Windows 窗口类名同时被 NSIS 引用，用于安装前定位并关闭正在运行的窗口。
	WindowsWindowClass = "com.github.chencn.go-desktop-window"
	// 当前用户安装目录由 NSIS 使用，避免写 Program Files 和触发管理员权限。
	WindowsInstallDir = "$LOCALAPPDATA\\Programs\\go-desktop"
	// 卸载注册表项名必须稳定，否则升级和卸载记录会分裂。
	WindowsUninstallKeyName = "com.github.chencn.go-desktop"
)

// 返回 Windows 更新优先匹配的 Release 资产名。
func WindowsInstallerAssetName(version string) string {
	return fmt.Sprintf("%s-v%s-windows-amd64.exe", AppName, version)
}

// 返回兼容旧命名的无 v 前缀安装包名。
func WindowsInstallerAssetNameWithoutV(version string) string {
	return fmt.Sprintf("%s-%s-windows-amd64.exe", AppName, version)
}

// 返回允许兜底匹配的简化 setup 安装包名。
func WindowsSetupAssetName(version string) string {
	return fmt.Sprintf("%s-setup-v%s.exe", AppName, version)
}

// 返回兼容旧命名的无 v 前缀 setup 安装包名。
func WindowsSetupAssetNameWithoutV(version string) string {
	return fmt.Sprintf("%s-setup-%s.exe", AppName, version)
}
