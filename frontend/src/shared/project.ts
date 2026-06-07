// ============================================================================
// 文件: shared/project.ts
// 描述: 项目元数据配置
//
// 功能概述:
// - 由 scripts/sync_project_metadata.go 根据 project.metadata.json 自动生成
// - 包含应用名称、版本、仓库地址、默认设置等常量
// - 此文件不要手工修改，运行 task sync 即可更新
// ============================================================================

// 项目元数据常量对象
// 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
export const projectMetadata = {
  // 公司名称
  "companyName": "chencn",
  // 应用名称
  "appName": "go-desktop",
  // Go 模块路径
  "modulePath": "github.com/chencn/go-desktop",
  // 默认版本号
  "defaultVersion": "1.0.0",
  // 应用描述
  "description": "Wails3 中文桌面工具",
  // 仓库地址
  "repositoryUrl": "https://github.com/chencn/go-desktop",
  // 仓库备注
  "repositoryComment": "github.com/chencn/go-desktop",
  // 版权信息
  "copyright": "© 2026, chencn",
  // GitHub 配置
  "github": {
    // GitHub 用户名/组织名
    "owner": "chencn",
    // GitHub 仓库名
    "repo": "go-desktop",
    // GitHub API 基础地址
    "apiBase": "https://api.github.com",
    // GitHub API 版本
    "apiVersion": "2026-03-10",
    // 请求头 User-Agent
    "userAgent": "go-desktop-updater"
  },
  // 更新配置
  "update": {
    // 默认更新源
    "defaultSource": "github",
    // 本地静态升级根地址
    "localBaseUrl": "http://www.xqchen.shop/exe/go-desktop",
    // 本地 manifest 相对路径
    "localManifestPath": "releases/latest.json"
  },
  // 默认设置值
  "settingsDefaults": {
    // GitHub 代理地址（空表示不使用代理）
    "githubProxyBase": "",
    // 更新检查间隔（小时）
    "updateCheckIntervalHours": 3,
    // 是否关闭到系统托盘
    "minimizeToTray": true,
    // 日志保留天数
    "logRetentionDays": 30,
    // 默认不开启开机自启
    "autoLaunch": false,
    // 默认创建桌面快捷图标
    "createDesktopShortcut": true,
    // 默认开机自启时不隐藏到托盘
    "launchHiddenToTray": false
  },
  // Windows 平台配置
  "windows": {
    // 单实例唯一标识
    "singleInstanceId": "com.github.chencn.go-desktop",
    // 产品标识符
    "productIdentifier": "com.github.chencn.godesktop",
    // 窗口类名
    "windowClass": "com.github.chencn.go-desktop-window",
    // 安装目录
    "installDir": "$LOCALAPPDATA\\Programs\\go-desktop",
    // 卸载注册表键名
    "uninstallKeyName": "com.github.chencn.go-desktop"
  }
} as const

// 默认设置对象
// 从 projectMetadata 中提取默认设置值
export const defaultSettings = {
  // 更新源
  updateSource: projectMetadata.update.defaultSource,
  // GitHub 仓库所有者
  githubOwner: projectMetadata.github.owner,
  // GitHub 仓库名称
  githubRepo: projectMetadata.github.repo,
  // GitHub 代理地址
  githubProxyBase: projectMetadata.settingsDefaults.githubProxyBase,
  // 更新检查间隔（小时）
  updateCheckIntervalHours: projectMetadata.settingsDefaults.updateCheckIntervalHours,
  // 是否关闭到系统托盘
  minimizeToTray: projectMetadata.settingsDefaults.minimizeToTray,
  // 日志保留天数
  logRetentionDays: projectMetadata.settingsDefaults.logRetentionDays,
  // 是否开机自启
  autoLaunch: projectMetadata.settingsDefaults.autoLaunch,
  // 是否创建桌面快捷图标
  createDesktopShortcut: projectMetadata.settingsDefaults.createDesktopShortcut,
  // 开机自启时是否隐藏到托盘
  launchHiddenToTray: projectMetadata.settingsDefaults.launchHiddenToTray,
} as const
