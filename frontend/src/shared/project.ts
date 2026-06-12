// 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
// projectMetadata 保存完整项目元数据，前端关于页、默认设置和仓库信息都从这里读取。
export const projectMetadata = {
  "companyName": "chencn",
  "appName": "go-desktop",
  "modulePath": "github.com/chencn/go-desktop",
  "defaultVersion": "1.0.0",
  "description": "Wails3 中文桌面工具",
  "repositoryUrl": "https://github.com/chencn/go-desktop",
  "repositoryComment": "github.com/chencn/go-desktop",
  "copyright": "© 2026, chencn",
  "github": {
    "owner": "chencn",
    "repo": "go-desktop",
    "apiBase": "https://api.github.com",
    "apiVersion": "2026-03-10",
    "userAgent": "go-desktop-updater"
  },
  "update": {
    "defaultSource": "github",
    "localBaseUrl": "http://www.xqchen.shop/exe/go-desktop",
    "localManifestPath": "releases/latest.json"
  },
  "settingsDefaults": {
    "githubProxyBase": "https://gh-proxy.com",
    "updateCheckIntervalHours": 3,
    "minimizeToTray": true,
    "alwaysOnTop": false,
    "logRetentionDays": 30,
    "autoLaunch": false,
    "createDesktopShortcut": true,
    "launchHiddenToTray": false
  },
  "windows": {
    "singleInstanceId": "com.github.chencn.go-desktop",
    "productIdentifier": "com.github.chencn.godesktop",
    "windowClass": "com.github.chencn.go-desktop-window",
    "installDir": "$LOCALAPPDATA\\Programs\\go-desktop",
    "uninstallKeyName": "com.github.chencn.go-desktop"
  }
} as const

// defaultSettings 从完整元数据中提取运行期可编辑设置的默认值。
export const defaultSettings = {
  updateSource: projectMetadata.update.defaultSource,
  githubProxyBase: projectMetadata.settingsDefaults.githubProxyBase,
  updateCheckIntervalHours: projectMetadata.settingsDefaults.updateCheckIntervalHours,
  minimizeToTray: projectMetadata.settingsDefaults.minimizeToTray,
  alwaysOnTop: projectMetadata.settingsDefaults.alwaysOnTop,
  logRetentionDays: projectMetadata.settingsDefaults.logRetentionDays,
  autoLaunch: projectMetadata.settingsDefaults.autoLaunch,
  createDesktopShortcut: projectMetadata.settingsDefaults.createDesktopShortcut,
  launchHiddenToTray: projectMetadata.settingsDefaults.launchHiddenToTray,
} as const
