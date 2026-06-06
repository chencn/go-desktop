// ============================================================================
// 文件: internal/desktopapp/metadata/metadata.go
// 描述: 项目元数据常量
//
// 功能概述:
// - 由 scripts/sync_project_metadata.go 根据 project.metadata.json 自动生成
// - 这些值同时影响运行时、更新检查、安装包命名和 Windows 桌面生命周期
// - 修改产品名、仓库、版本兜底或窗口类名时，只改 project.metadata.json，再运行同步脚本
//
// 警告: 此文件不要手工修改！运行 task sync 即可更新。
// ============================================================================

package metadata

import "fmt"

// ============================================================================
// 基础信息常量
// ============================================================================

const (
	// CompanyName 公司名用于构建元信息、安装器注册表项和发布标签
	CompanyName = "chencn"
	// AppName 应用名是标准产品名和可执行文件名；当前也等于 GitHub 仓库名
	AppName = "go-desktop"
	// ModulePath Go 模块路径用于校验绑定导入路径和生成脚本
	ModulePath = "github.com/chencn/go-desktop"
	// DefaultVersion 默认版本只作为开发兜底；正式构建通过 ldflags 覆盖
	DefaultVersion = "0.0.1"
	// Description 应用描述会展示在 Wails 运行时信息和前端"关于"页面
	Description = "Wails3 中文桌面工具"
	// RepositoryURL 仓库地址是给用户看的项目地址，不是 GitHub API 端点
	RepositoryURL = "https://github.com/chencn/go-desktop"
	// RepositoryComment 仓库备注用于 Windows 版本资源和安装器元信息
	RepositoryComment = "github.com/chencn/go-desktop"
	// Copyright 版权文本用于 Windows 版本资源和安装器元信息
	Copyright = "© 2026, chencn"
)

// ============================================================================
// GitHub 配置常量
// ============================================================================

const (
	// GitHubOwner GitHub 仓库归属和仓库名定义更新检查使用的公开 Release 来源
	GitHubOwner = "chencn"
	// GitHubRepo GitHub 仓库名
	GitHubRepo = "go-desktop"
	// GitHubAPIBase GitHub API 地址和版本必须和测试、检查器默认值、前端预览兜底保持一致
	GitHubAPIBase = "https://api.github.com"
	// GitHubAPIVersion GitHub API 版本
	GitHubAPIVersion = "2026-03-10"
	// UserAgent 请求标识是 GitHub REST API 要求项，Release 列表和资产下载都复用它
	UserAgent = "go-desktop-updater"
)

// ============================================================================
// 默认设置常量
// ============================================================================

const (
	// DefaultGitHubProxyBase 默认 GitHub 代理地址（空表示不使用代理）
	DefaultGitHubProxyBase = ""
	// DefaultUpdateCheckIntervalHours 默认更新检查间隔（小时）
	DefaultUpdateCheckIntervalHours = 3
	// DefaultMinimizeToTray 默认是否关闭到系统托盘
	DefaultMinimizeToTray = true
	// DefaultLogRetentionDays 默认日志保留天数
	DefaultLogRetentionDays = 30
	// DefaultAutoLaunch 默认不开启开机自启
	DefaultAutoLaunch = false
	// DefaultCreateDesktopShortcut 默认创建桌面快捷图标
	DefaultCreateDesktopShortcut = true
	// DefaultLaunchHiddenToTray 默认开机自启时不隐藏到托盘
	DefaultLaunchHiddenToTray = false
)

// ============================================================================
// Windows 平台配置常量
// ============================================================================

const (
	// WindowsSingleInstanceID 单实例标识跨版本必须稳定，否则第二次启动无法找到已运行实例
	WindowsSingleInstanceID = "com.chencn.go-desktop"
	// WindowsProductID Windows 产品标识用于构建产物和平台元信息
	WindowsProductID = "com.chencn.godesktop"
	// WindowsWindowClass Windows 窗口类名同时被 NSIS 引用，用于安装前定位并关闭正在运行的窗口
	WindowsWindowClass = "GoDesktopWailsWindow"
	// WindowsInstallDir 当前用户安装目录由 NSIS 使用，避免写 Program Files 和触发管理员权限
	WindowsInstallDir = "$LOCALAPPDATA\\Programs\\go-desktop"
	// WindowsUninstallKeyName 卸载注册表项名必须稳定，否则升级和卸载记录会分裂
	WindowsUninstallKeyName = "chencn-go-desktop"
)

// ============================================================================
// 工具函数
// ============================================================================

// WindowsInstallerAssetName 返回 Windows 更新优先匹配的 Release 资产名
// 参数:
//   - version: 版本号
//
// 返回:
//   - string: 安装包文件名，如 "go-desktop-v1.0.0-windows-amd64.exe"
func WindowsInstallerAssetName(version string) string {
	return fmt.Sprintf("%s-v%s-windows-amd64.exe", AppName, version)
}

// WindowsInstallerAssetNameWithoutV 返回兼容旧命名的无 v 前缀安装包名
// 参数:
//   - version: 版本号
//
// 返回:
//   - string: 安装包文件名，如 "go-desktop-1.0.0-windows-amd64.exe"
func WindowsInstallerAssetNameWithoutV(version string) string {
	return fmt.Sprintf("%s-%s-windows-amd64.exe", AppName, version)
}

// WindowsSetupAssetName 返回允许兜底匹配的简化 setup 安装包名
// 参数:
//   - version: 版本号
//
// 返回:
//   - string: 安装包文件名，如 "go-desktop-setup-v1.0.0.exe"
func WindowsSetupAssetName(version string) string {
	return fmt.Sprintf("%s-setup-v%s.exe", AppName, version)
}

// WindowsSetupAssetNameWithoutV 返回兼容旧命名的无 v 前缀 setup 安装包名
// 参数:
//   - version: 版本号
//
// 返回:
//   - string: 安装包文件名，如 "go-desktop-setup-1.0.0.exe"
func WindowsSetupAssetNameWithoutV(version string) string {
	return fmt.Sprintf("%s-setup-%s.exe", AppName, version)
}
