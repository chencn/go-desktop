// ============================================================================
// 文件: scripts/sync_project_metadata.go
// 描述: 项目元数据同步脚本
//
// 功能概述:
// - 从 project.metadata.json 读取项目元数据
// - 自动生成各平台的配置文件（Go、TypeScript、NSIS、Android、iOS、macOS、Linux）
// - 支持 Taskfile.yml、Info.plist、build.gradle 等文件的生成
//
// 使用方法:
//   go run scripts/sync_project_metadata.go -sync    # 同步所有派生文件
//   go run scripts/sync_project_metadata.go -print appName  # 打印指定元数据值
// ============================================================================

package main

import (
	"encoding/json" // JSON 解析
	"flag"          // 命令行参数解析
	"fmt"           // 格式化输出
	"go/format"     // Go 代码格式化
	"os"            // 文件操作
	"path/filepath" // 路径处理
	"strings"       // 字符串处理
)

// metadata 定义读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 使用的数据实体，字段会直接参与校验、渲染或平台适配。
type metadata struct {
	CompanyName       string         `json:"companyName"`       // CompanyName 保存 companyName 对应的数据，供当前文件的生成或运行流程读取。
	AppName           string         `json:"appName"`           // AppName 保存 appName 对应的数据，供当前文件的生成或运行流程读取。
	ModulePath        string         `json:"modulePath"`        // ModulePath 保存 modulePath 对应的数据，供当前文件的生成或运行流程读取。
	DefaultVersion    string         `json:"defaultVersion"`    // DefaultVersion 保存 defaultVersion 对应的数据，供当前文件的生成或运行流程读取。
	Description       string         `json:"description"`       // Description 保存 description 对应的数据，供当前文件的生成或运行流程读取。
	RepositoryURL     string         `json:"repositoryUrl"`     // RepositoryURL 保存 repositoryUrl 对应的数据，供当前文件的生成或运行流程读取。
	RepositoryComment string         `json:"repositoryComment"` // RepositoryComment 保存 repositoryComment 对应的数据，供当前文件的生成或运行流程读取。
	Copyright         string         `json:"copyright"`         // Copyright 保存 copyright 对应的数据，供当前文件的生成或运行流程读取。
	GitHub            githubMetadata `json:"github"`            // GitHub 保存 github 对应的数据，供当前文件的生成或运行流程读取。
	Update            updateMetadata `json:"update"`            // Update 保存 update 对应的数据，供当前文件的生成或运行流程读取。
	SettingsDefaults  settingsMeta   `json:"settingsDefaults"`  // SettingsDefaults 保存 settingsDefaults 对应的数据，供当前文件的生成或运行流程读取。
	Windows           windowsMeta    `json:"windows"`           // Windows 保存 windows 对应的数据，供当前文件的生成或运行流程读取。
}

// githubMetadata 定义读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 使用的数据实体，字段会直接参与校验、渲染或平台适配。
type githubMetadata struct {
	Owner      string `json:"owner"`      // Owner 保存 owner 对应的数据，供当前文件的生成或运行流程读取。
	Repo       string `json:"repo"`       // Repo 保存 repo 对应的数据，供当前文件的生成或运行流程读取。
	APIBase    string `json:"apiBase"`    // APIBase 保存 apiBase 对应的数据，供当前文件的生成或运行流程读取。
	APIVersion string `json:"apiVersion"` // APIVersion 保存 apiVersion 对应的数据，供当前文件的生成或运行流程读取。
	UserAgent  string `json:"userAgent"`  // UserAgent 保存 userAgent 对应的数据，供当前文件的生成或运行流程读取。
}

// updateMetadata 定义更新源默认值和本地静态 manifest 位置。
type updateMetadata struct {
	DefaultSource     string `json:"defaultSource"`     // DefaultSource 保存默认更新源，允许 github 或 local。
	LocalBaseURL      string `json:"localBaseUrl"`      // LocalBaseURL 保存本地静态升级根地址。
	LocalManifestPath string `json:"localManifestPath"` // LocalManifestPath 保存本地 latest.json 相对路径。
}

// settingsMeta 定义读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 使用的数据实体，字段会直接参与校验、渲染或平台适配。
type settingsMeta struct {
	GitHubProxyBase          string `json:"githubProxyBase"`          // GitHubProxyBase 保存 githubProxyBase 对应的数据，供当前文件的生成或运行流程读取。
	UpdateCheckIntervalHours int    `json:"updateCheckIntervalHours"` // UpdateCheckIntervalHours 保存 updateCheckIntervalHours 对应的数据，供当前文件的生成或运行流程读取。
	MinimizeToTray           bool   `json:"minimizeToTray"`           // MinimizeToTray 保存 minimizeToTray 对应的数据，供当前文件的生成或运行流程读取。
	LogRetentionDays         int    `json:"logRetentionDays"`         // LogRetentionDays 保存 logRetentionDays 对应的数据，供当前文件的生成或运行流程读取。
	AutoLaunch               bool   `json:"autoLaunch"`               // AutoLaunch 保存 autoLaunch 对应的数据，供当前文件的生成或运行流程读取。
	CreateDesktopShortcut    bool   `json:"createDesktopShortcut"`    // CreateDesktopShortcut 保存 createDesktopShortcut 对应的数据，供当前文件的生成或运行流程读取。
	LaunchHiddenToTray       bool   `json:"launchHiddenToTray"`       // LaunchHiddenToTray 保存 launchHiddenToTray 对应的数据，供当前文件的生成或运行流程读取。
}

// windowsMeta 定义读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 使用的数据实体，字段会直接参与校验、渲染或平台适配。
type windowsMeta struct {
	SingleInstanceID  string `json:"singleInstanceId"`  // SingleInstanceID 保存 singleInstanceId 对应的数据，供当前文件的生成或运行流程读取。
	ProductIdentifier string `json:"productIdentifier"` // ProductIdentifier 保存 productIdentifier 对应的数据，供当前文件的生成或运行流程读取。
	WindowClass       string `json:"windowClass"`       // WindowClass 保存 windowClass 对应的数据，供当前文件的生成或运行流程读取。
	InstallDir        string `json:"installDir"`        // InstallDir 保存 installDir 对应的数据，供当前文件的生成或运行流程读取。
	UninstallKeyName  string `json:"uninstallKeyName"`  // UninstallKeyName 保存 uninstallKeyName 对应的数据，供当前文件的生成或运行流程读取。
}

// main 是命令入口，负责解析参数、读取元数据并按开关调度同步或打印流程。
func main() {
	syncFiles := flag.Bool("sync", false, "同步派生文件")
	printKey := flag.String("print", "", "打印单个元数据值")
	flag.Parse()

	meta := mustReadMetadata()

	if *printKey != "" {
		printValue(meta, *printKey)
		return
	}
	if !*syncFiles {
		exitf("必须传入 -sync 或 -print")
	}

	wailsVersion, err := wailsVersionFromGoModFile("go.mod")
	if err != nil {
		exitf("读取 Wails 版本失败：%v", err)
	}

	mustWrite("Taskfile.yml", renderRootTaskfile(meta))
	mustWrite("frontend/index.html", renderFrontendIndex(meta))
	mustWrite("internal/desktopapp/metadata/metadata.go", renderGo(meta))
	mustWrite("frontend/src/shared/project.ts", renderTypeScript(meta))
	mustWrite("build/windows/nsis/project_metadata.nsh", renderNSIS(meta))
	mustWrite("build/windows/nsis/project.nsi", renderNSISProject(meta))
	mustWrite("build/config.yml", renderBuildConfig(meta))
	mustWrite("build/windows/info.json", renderWindowsInfo(meta))
	mustWrite("build/windows/wails.exe.manifest", renderWindowsManifest(meta))
	mustWrite("build/windows/msix/template.xml", renderWindowsMSIXTemplate(meta))
	mustWrite("build/windows/msix/app_manifest.xml", renderWindowsMSIXManifest(meta))
	mustWrite("build/linux/desktop", renderLinuxDesktop(meta))
	mustWrite("build/linux/nfpm/nfpm.yaml", renderLinuxNfpm(meta))
	mustWrite("build/darwin/Info.plist", renderDarwinInfoPlist(meta, false))
	mustWrite("build/darwin/Info.dev.plist", renderDarwinInfoPlist(meta, true))
	mustWrite("build/ios/Info.plist", renderIOSInfoPlist(meta, false))
	mustWrite("build/ios/Info.dev.plist", renderIOSInfoPlist(meta, true))
	mustWrite("build/ios/build.sh", renderIOSBuildScript(meta))
	mustWrite("build/android/app/build.gradle", renderAndroidBuildGradle(meta))
	mustWrite("build/android/settings.gradle", renderAndroidSettingsGradle(meta))
	mustWrite("build/android/app/src/main/res/values/strings.xml", renderAndroidStrings(meta))
	mustWrite("build/android/Taskfile.yml", renderAndroidTaskfile(meta))
	mustWrite("build/ios/Taskfile.yml", renderIOSTaskfile(meta))
	mustWrite("build/ios/LaunchScreen.storyboard", renderIOSLaunchScreen(meta))
	mustWrite("build/ios/project.pbxproj", renderIOSProjectPBXProj(meta))
	mustWrite(".github/workflows/release.yml", renderReleaseWorkflow(meta, wailsVersion))
}

// mustReadMetadata 执行必须成功的辅助步骤，失败时直接终止当前命令以避免继续写入错误结果。
func mustReadMetadata() metadata {
	data, err := os.ReadFile("project.metadata.json")
	if err != nil {
		exitf("读取 project.metadata.json 失败：%v", err)
	}
	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		exitf("解析 project.metadata.json 失败：%v", err)
	}
	validate(meta)
	return meta
}

// wailsVersionFromGoModFile 提取 go.mod 中声明的 Wails v3 版本，用于生成发布工作流的工具链约束。
func wailsVersionFromGoModFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return wailsVersionFromGoMod(string(data))
}

// wailsVersionFromGoMod 提取 go.mod 中声明的 Wails v3 版本，用于生成发布工作流的工具链约束。
func wailsVersionFromGoMod(source string) (string, error) {
	for _, rawLine := range strings.Split(source, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "github.com/wailsapp/wails/v3" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("go.mod 未声明 github.com/wailsapp/wails/v3")
}

// validate 校验项目元数据的必填项和默认值约束，保证所有派生文件来自完整可信的配置。
func validate(meta metadata) {
	required := map[string]string{
		"companyName":               meta.CompanyName,
		"appName":                   meta.AppName,
		"modulePath":                meta.ModulePath,
		"defaultVersion":            meta.DefaultVersion,
		"description":               meta.Description,
		"repositoryUrl":             meta.RepositoryURL,
		"repositoryComment":         meta.RepositoryComment,
		"copyright":                 meta.Copyright,
		"github.owner":              meta.GitHub.Owner,
		"github.repo":               meta.GitHub.Repo,
		"github.apiBase":            meta.GitHub.APIBase,
		"github.apiVersion":         meta.GitHub.APIVersion,
		"github.userAgent":          meta.GitHub.UserAgent,
		"update.defaultSource":      meta.Update.DefaultSource,
		"update.localBaseUrl":       meta.Update.LocalBaseURL,
		"update.localManifestPath":  meta.Update.LocalManifestPath,
		"windows.singleInstanceId":  meta.Windows.SingleInstanceID,
		"windows.productIdentifier": meta.Windows.ProductIdentifier,
		"windows.windowClass":       meta.Windows.WindowClass,
		"windows.installDir":        meta.Windows.InstallDir,
		"windows.uninstallKeyName":  meta.Windows.UninstallKeyName,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			exitf("project.metadata.json 缺少必填项：%s", key)
		}
	}
	if meta.SettingsDefaults.UpdateCheckIntervalHours <= 0 {
		exitf("project.metadata.json 的 settingsDefaults.updateCheckIntervalHours 必须大于 0")
	}
	switch strings.ToLower(strings.TrimSpace(meta.Update.DefaultSource)) {
	case "github", "local":
	default:
		exitf("project.metadata.json 的 update.defaultSource 必须为 github 或 local")
	}
	if !meta.SettingsDefaults.MinimizeToTray {
		exitf("project.metadata.json 的 settingsDefaults.minimizeToTray 当前必须为 true，避免缺字段时静默关闭托盘策略")
	}
	if meta.SettingsDefaults.LogRetentionDays == 0 || meta.SettingsDefaults.LogRetentionDays < -1 {
		exitf("project.metadata.json 的 settingsDefaults.logRetentionDays 必须为 -1 或大于 0")
	}
	if meta.SettingsDefaults.AutoLaunch {
		exitf("project.metadata.json 的 settingsDefaults.autoLaunch 必须默认为 false")
	}
	if !meta.SettingsDefaults.CreateDesktopShortcut {
		exitf("project.metadata.json 的 settingsDefaults.createDesktopShortcut 必须默认为 true")
	}
	if meta.SettingsDefaults.LaunchHiddenToTray {
		exitf("project.metadata.json 的 settingsDefaults.launchHiddenToTray 必须默认为 false")
	}
}

// printValue 按命令行 key 输出单个元数据值，供 Taskfile 和脚本复用。
func printValue(meta metadata, key string) {
	values := map[string]string{
		"companyName":                    meta.CompanyName,
		"appName":                        meta.AppName,
		"modulePath":                     meta.ModulePath,
		"defaultVersion":                 meta.DefaultVersion,
		"description":                    meta.Description,
		"repositoryUrl":                  meta.RepositoryURL,
		"repositoryComment":              meta.RepositoryComment,
		"copyright":                      meta.Copyright,
		"github.owner":                   meta.GitHub.Owner,
		"github.repo":                    meta.GitHub.Repo,
		"github.apiBase":                 meta.GitHub.APIBase,
		"github.apiVersion":              meta.GitHub.APIVersion,
		"github.userAgent":               meta.GitHub.UserAgent,
		"update.defaultSource":           meta.Update.DefaultSource,
		"update.localBaseUrl":            meta.Update.LocalBaseURL,
		"update.localManifestPath":       meta.Update.LocalManifestPath,
		"settings.githubProxyBase":       meta.SettingsDefaults.GitHubProxyBase,
		"settings.updateInterval":        fmt.Sprintf("%d", meta.SettingsDefaults.UpdateCheckIntervalHours),
		"settings.minimizeToTray":        fmt.Sprintf("%t", meta.SettingsDefaults.MinimizeToTray),
		"settings.logRetentionDays":      fmt.Sprintf("%d", meta.SettingsDefaults.LogRetentionDays),
		"settings.autoLaunch":            fmt.Sprintf("%t", meta.SettingsDefaults.AutoLaunch),
		"settings.createDesktopShortcut": fmt.Sprintf("%t", meta.SettingsDefaults.CreateDesktopShortcut),
		"settings.launchHiddenToTray":    fmt.Sprintf("%t", meta.SettingsDefaults.LaunchHiddenToTray),
		"windows.singleInstanceId":       meta.Windows.SingleInstanceID,
		"windows.productIdentifier":      meta.Windows.ProductIdentifier,
		"windows.windowClass":            meta.Windows.WindowClass,
		"windows.installDir":             meta.Windows.InstallDir,
		"windows.uninstallKeyName":       meta.Windows.UninstallKeyName,
	}
	value, ok := values[key]
	if !ok {
		exitf("未知元数据键：%s", key)
	}
	fmt.Print(value)
}

// renderRootTaskfile 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderRootTaskfile(meta metadata) string {
	return fmt.Sprintf(`# 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
version: '3'

includes:
  common: ./build/Taskfile.yml
  windows: ./build/windows/Taskfile.yml
  darwin: ./build/darwin/Taskfile.yml
  linux: ./build/linux/Taskfile.yml
  ios: ./build/ios/Taskfile.yml
  android: ./build/android/Taskfile.yml

vars:
  APP_NAME: %s
  APP_VERSION:
    sh: go run ./scripts/resolve_app_version.go -mode '{{env "APP_VERSION_MODE" | default "local"}}' -config build/config.yml -version '{{env "APP_VERSION"}}' -tag '{{env "GITHUB_REF_NAME"}}'
  BIN_DIR: "bin"
  VITE_PORT: '{{.WAILS_VITE_PORT | default 9245}}'

tasks:
  build:
    summary: Builds the application
    cmds:
      - go run ./scripts/envrun wails3 task {{OS}}:build

  package:
    summary: Packages Windows installer and stages local static update files
    cmds:
      - go run ./scripts/envrun wails3 task windows:package
      - go build -o ".tmp/local-release-stage.exe" ./scripts/stage_local_update.go
      - '.tmp/local-release-stage.exe -version "{{.APP_VERSION}}" -installer "{{.BIN_DIR}}/{{.APP_NAME}}-v{{.APP_VERSION}}-windows-amd64.exe" -out "{{.BIN_DIR}}/{{.APP_NAME}}" -arch amd64'
      - powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Test-Path -LiteralPath '{{.BIN_DIR}}/{{.APP_NAME}}.exe') { Remove-Item -LiteralPath '{{.BIN_DIR}}/{{.APP_NAME}}.exe' -Force }; if (Test-Path -LiteralPath '{{.BIN_DIR}}/{{.APP_NAME}}-v{{.APP_VERSION}}-windows-amd64.exe') { Remove-Item -LiteralPath '{{.BIN_DIR}}/{{.APP_NAME}}-v{{.APP_VERSION}}-windows-amd64.exe' -Force }; if (Test-Path -LiteralPath '{{.BIN_DIR}}/local-release-stage.exe') { Remove-Item -LiteralPath '{{.BIN_DIR}}/local-release-stage.exe' -Force }; if (Test-Path -LiteralPath '.tmp/local-release-stage.exe') { Remove-Item -LiteralPath '.tmp/local-release-stage.exe' -Force }"

  package:github:
    summary: Packages a production build for GitHub Release
    cmds:
      - go run ./scripts/envrun wails3 task windows:package

  run:
    summary: Runs the application
    cmds:
      - task: "{{OS}}:run"

  dev:
    summary: Runs the application in development mode
    env:
      # Some Windows automation shells strip drive-level variables.
      # Keep caches out of frontend/%%SystemDrive%% by resolving ProgramData explicitly.
      SystemDrive: '{{if eq OS "windows"}}{{env "SystemDrive" | default "C:"}}{{end}}'
      ProgramData: '{{if eq OS "windows"}}{{env "ProgramData" | default (printf "%%s\\ProgramData" (env "SystemDrive" | default "C:"))}}{{end}}'
      SystemRoot: '{{if eq OS "windows"}}{{env "SystemRoot" | default "C:\\WINDOWS"}}{{end}}'
      WINDIR: '{{if eq OS "windows"}}{{env "WINDIR" | default "C:\\WINDOWS"}}{{end}}'
      ComSpec: '{{if eq OS "windows"}}{{env "ComSpec" | default "C:\\WINDOWS\\System32\\cmd.exe"}}{{end}}'
      LOCALAPPDATA: '{{if eq OS "windows"}}{{env "LOCALAPPDATA" | default (printf "%%s\\AppData\\Local" (env "USERPROFILE"))}}{{end}}'
      APPDATA: '{{if eq OS "windows"}}{{env "APPDATA" | default (printf "%%s\\AppData\\Roaming" (env "USERPROFILE"))}}{{end}}'
      GOCACHE: '{{if eq OS "windows"}}{{env "GOCACHE" | default (printf "%%s\\AppData\\Local\\go-build" (env "USERPROFILE"))}}{{end}}'
    cmds:
      - go run ./scripts/envrun wails3 dev -config ./build/config.yml -port {{.VITE_PORT}}

  test:
    summary: Runs the isolated Go and frontend test modules
    cmds:
      - cd tests && go test ./...
      - cd frontend && go run ../scripts/envrun npm test

  setup:docker:
    summary: Builds Docker image for cross-compilation (~800MB download)
    cmds:
      - task: common:setup:docker

  build:server:
    summary: Builds the application in server mode (no GUI, HTTP server only)
    cmds:
      - task: common:build:server

  run:server:
    summary: Runs the application in server mode
    cmds:
      - task: common:run:server

  build:docker:
    summary: Builds a Docker image for server mode deployment
    cmds:
      - task: common:build:docker

  run:docker:
    summary: Builds and runs the Docker image
    cmds:
      - task: common:run:docker
`, yamlString(meta.AppName))
}

// renderFrontendIndex 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderFrontendIndex(meta metadata) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/wails.png" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>%s</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
`, htmlText(meta.AppName))
}

// renderGo 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderGo(meta metadata) string {
	return goSource(fmt.Sprintf(`// 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
//
// 这些值同时影响运行时、更新检查、安装包命名和 Windows 桌面生命周期。
// 修改产品名、仓库、版本兜底或窗口类名时，只改 project.metadata.json，再运行同步脚本。
package metadata

import "fmt"

// 常量块声明读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 需要跨函数复用的固定值。
const (
	// 公司名用于构建元信息、安装器注册表项和发布标签。
	CompanyName       = %s
	// 应用名是标准产品名和可执行文件名；当前也等于 GitHub 仓库名。
	AppName           = %s
	// Go 模块路径用于校验绑定导入路径和生成脚本。
	ModulePath        = %s
	// 默认版本只作为开发兜底；正式构建通过 ldflags 覆盖。
	DefaultVersion    = %s
	// 应用描述会展示在 Wails 运行时信息和前端“关于”页面。
	Description       = %s
	// 仓库地址是给用户看的项目地址，不是 GitHub API 端点。
	RepositoryURL     = %s
	// 仓库备注用于 Windows 版本资源和安装器元信息。
	RepositoryComment = %s
	// 版权文本用于 Windows 版本资源和安装器元信息。
	Copyright         = %s

	// GitHub 仓库归属和仓库名定义更新检查使用的公开 Release 来源。
	GitHubOwner      = %s
	GitHubRepo       = %s
	// GitHub API 地址和版本必须和测试、检查器默认值、前端预览兜底保持一致。
	GitHubAPIBase    = %s
	GitHubAPIVersion = %s
	// 请求标识是 GitHub REST API 要求项，Release 列表和资产下载都复用它。
	UserAgent        = %s

	// 更新源默认值和本地静态升级 manifest 位置。
	DefaultUpdateSource     = %s
	LocalUpdateBaseURL      = %s
	LocalUpdateManifestPath = %s

	// 默认设置由同一份项目元数据派生，避免 Go、前端和脚本各写一份。
	DefaultGitHubProxyBase          = %s
	DefaultUpdateCheckIntervalHours = %d
	DefaultMinimizeToTray           = %t
	DefaultLogRetentionDays         = %d
	DefaultAutoLaunch               = %t
	DefaultCreateDesktopShortcut    = %t
	DefaultLaunchHiddenToTray       = %t

	// 单实例标识跨版本必须稳定，否则第二次启动无法找到已运行实例。
	WindowsSingleInstanceID = %s
	// Windows 产品标识用于构建产物和平台元信息。
	WindowsProductID        = %s
	// Windows 窗口类名同时被 NSIS 引用，用于安装前定位并关闭正在运行的窗口。
	WindowsWindowClass      = %s
	// 当前用户安装目录由 NSIS 使用，避免写 Program Files 和触发管理员权限。
	WindowsInstallDir       = %s
	// 卸载注册表项名必须稳定，否则升级和卸载记录会分裂。
	WindowsUninstallKeyName = %s
)

// 返回 Windows 更新优先匹配的 Release 资产名。
func WindowsInstallerAssetName(version string) string {
	return fmt.Sprintf("%%s-v%%s-windows-amd64.exe", AppName, version)
}

// 返回兼容旧命名的无 v 前缀安装包名。
func WindowsInstallerAssetNameWithoutV(version string) string {
	return fmt.Sprintf("%%s-%%s-windows-amd64.exe", AppName, version)
}

// 返回允许兜底匹配的简化 setup 安装包名。
func WindowsSetupAssetName(version string) string {
	return fmt.Sprintf("%%s-setup-v%%s.exe", AppName, version)
}

// 返回兼容旧命名的无 v 前缀 setup 安装包名。
func WindowsSetupAssetNameWithoutV(version string) string {
	return fmt.Sprintf("%%s-setup-%%s.exe", AppName, version)
}
`,
		goString(meta.CompanyName),
		goString(meta.AppName),
		goString(meta.ModulePath),
		goString(meta.DefaultVersion),
		goString(meta.Description),
		goString(meta.RepositoryURL),
		goString(meta.RepositoryComment),
		goString(meta.Copyright),
		goString(meta.GitHub.Owner),
		goString(meta.GitHub.Repo),
		goString(meta.GitHub.APIBase),
		goString(meta.GitHub.APIVersion),
		goString(meta.GitHub.UserAgent),
		goString(meta.Update.DefaultSource),
		goString(meta.Update.LocalBaseURL),
		goString(meta.Update.LocalManifestPath),
		goString(meta.SettingsDefaults.GitHubProxyBase),
		meta.SettingsDefaults.UpdateCheckIntervalHours,
		meta.SettingsDefaults.MinimizeToTray,
		meta.SettingsDefaults.LogRetentionDays,
		meta.SettingsDefaults.AutoLaunch,
		meta.SettingsDefaults.CreateDesktopShortcut,
		meta.SettingsDefaults.LaunchHiddenToTray,
		goString(meta.Windows.SingleInstanceID),
		goString(meta.Windows.ProductIdentifier),
		goString(meta.Windows.WindowClass),
		goString(meta.Windows.InstallDir),
		goString(meta.Windows.UninstallKeyName),
	))
}

// renderTypeScript 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderTypeScript(meta metadata) string {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		exitf("生成前端元数据失败：%v", err)
	}
	return fmt.Sprintf(`// 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
// projectMetadata 保存完整项目元数据，前端关于页、默认设置和仓库信息都从这里读取。
export const projectMetadata = %s as const

// defaultSettings 从完整元数据中提取运行期可编辑设置的默认值。
export const defaultSettings = {
  updateSource: projectMetadata.update.defaultSource,
  githubOwner: projectMetadata.github.owner,
  githubRepo: projectMetadata.github.repo,
  githubProxyBase: projectMetadata.settingsDefaults.githubProxyBase,
  updateCheckIntervalHours: projectMetadata.settingsDefaults.updateCheckIntervalHours,
  minimizeToTray: projectMetadata.settingsDefaults.minimizeToTray,
  logRetentionDays: projectMetadata.settingsDefaults.logRetentionDays,
  autoLaunch: projectMetadata.settingsDefaults.autoLaunch,
  createDesktopShortcut: projectMetadata.settingsDefaults.createDesktopShortcut,
  launchHiddenToTray: projectMetadata.settingsDefaults.launchHiddenToTray,
} as const
`, string(data))
}

// renderNSIS 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderNSIS(meta metadata) string {
	return fmt.Sprintf(`## 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
!define INFO_PROJECTNAME    %s
!define INFO_COMPANYNAME    %s
!define INFO_PRODUCTNAME    %s
!define INFO_COPYRIGHT      %s
!define PRODUCT_EXECUTABLE  %s
!define UNINST_KEY_NAME     %s
!define APP_WINDOW_CLASS    %s
!define APP_WINDOW_TITLE    %s
!define APP_INSTALL_DIR     %s
`,
		nsisString(meta.AppName),
		nsisString(meta.CompanyName),
		nsisString(meta.AppName),
		nsisString(meta.Copyright),
		nsisString(meta.AppName+".exe"),
		nsisString(meta.Windows.UninstallKeyName),
		nsisString(meta.Windows.WindowClass),
		nsisString(meta.AppName),
		nsisString(meta.Windows.InstallDir),
	)
}

// renderNSISProject 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderNSISProject(meta metadata) string {
	return strings.ReplaceAll(`Unicode true

####
## 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
## 注意：这个文件里不能直接使用 Wails 模板替换，只能使用下面这些默认 define。
## 如果某个值没有在这里定义，wails_tools.nsh 会补默认值。
## 如果这里已经定义，wails_tools.nsh 不会覆盖，方便脱离 Wails 单独调试安装器。
## 
## 开发调试时先运行 Wails3 Windows 打包任务生成 wails_tools.nsh：
## > wails3 task windows:package
## 然后可以手动传入二进制路径调用 makensis。
## 仅 AMD64 安装器：
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## 仅 ARM64 安装器：
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## 同时包含两种架构的安装器：
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## 产品元数据由 project.metadata.json 生成，避免安装器和 Go/前端各写一遍。
####
!include "project_metadata.nsh"
!ifdef ARG_PRODUCT_VERSION
!define INFO_PRODUCTVERSION "${ARG_PRODUCT_VERSION}"
!else
!define INFO_PRODUCTVERSION {{DEFAULT_VERSION}}
!endif
###
####
!define REQUEST_EXECUTION_LEVEL "user"
!define UNINST_KEY_CURRENT_USER "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}"
####
## 引入 Wails 安装器辅助宏。
####
!include "wails_tools.nsh"

# Windows 版本资源必须是四段数字，这里给产品版本补最后一段。
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# 启用 HiDPI 支持。参考：https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" # 调试向导页时可加左侧图片，尺寸必须是 164x314。
!define MUI_FINISHPAGE_NOAUTOCLOSE # 只在关闭静默模式调试时可见。
!define MUI_ABORTWARNING # 只在关闭静默模式调试时可见。

SilentInstall silent
SilentUnInstall silent
AutoCloseWindow true
ShowInstDetails nevershow

!insertmacro MUI_PAGE_WELCOME # 安装器欢迎页。
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # 需要许可协议页时再启用。
!insertmacro MUI_PAGE_DIRECTORY # 安装目录页。
!insertmacro MUI_PAGE_INSTFILES # 安装进度页。
!insertmacro MUI_PAGE_FINISH # 安装完成页。

!insertmacro MUI_UNPAGE_INSTFILES # 卸载进度页。

!insertmacro MUI_LANGUAGE "SimpChinese" # 默认安装器语言。
!insertmacro MUI_LANGUAGE "English" # 兜底语言。

## 下面两行用于签名安装器和卸载器，%1 是待签名二进制路径。
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-v${INFO_PRODUCTVERSION}-windows-${ARCH}.exe" # 安装器输出文件名。
InstallDir "${APP_INSTALL_DIR}"

## .onInit 安装器初始化时按架构检查、请求旧实例退出并清理仍存活的窗口/进程。
Function .onInit
   !insertmacro wails.checkArchitecture
   Call RequestRunningApplicationExit
   Call CloseRunningApplicationWindow
   Call ForceTerminateRunningApplication
FunctionEnd

## .onInstSuccess 安装成功后启动已安装的主程序，保持更新安装后的用户连续性。
Function .onInstSuccess
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 done
        DetailPrint "正在启动 ${INFO_PRODUCTNAME}..."
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
    done:
FunctionEnd

## RequestRunningApplicationExit 通过 --installer-exit 请求已运行应用主动退出。
Function RequestRunningApplicationExit
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 done
        DetailPrint "正在请求已运行的 ${INFO_PRODUCTNAME} 退出..."
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}" --installer-exit'
        Sleep 1500
    done:
FunctionEnd

## CloseRunningApplicationWindow 通过窗口类名和标题定位主窗口并发送关闭消息。
Function CloseRunningApplicationWindow
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        DetailPrint "正在关闭已运行的 ${INFO_PRODUCTNAME}..."
        SendMessage $0 ${WM_CLOSE} 0 0 /TIMEOUT=5000
        Sleep 1500
    ${EndIf}
FunctionEnd

## ForceTerminateRunningApplication 在正常关闭失败后按窗口进程号强制结束旧实例。
Function ForceTerminateRunningApplication
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        System::Call 'user32::GetWindowThreadProcessId(p r0, *i .r1) i .r2'
        ${If} $1 != 0
            DetailPrint "正在确保已运行的 ${INFO_PRODUCTNAME} 退出..."
            nsExec::ExecToLog 'taskkill /PID $1 /T /F'
            Pop $2
            Sleep 500
        ${EndIf}
    ${EndIf}
FunctionEnd

## WriteCurrentUserUninstaller 写入当前用户范围卸载器和注册表卸载信息。
Function WriteCurrentUserUninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    SetRegView 64
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"

    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKCU "${UNINST_KEY_CURRENT_USER}" "EstimatedSize" "$0"
FunctionEnd

## un.RequestRunningApplicationExit 卸载前请求已运行应用主动退出，避免删除占用文件。
Function un.RequestRunningApplicationExit
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 done
        DetailPrint "正在请求已运行的 ${INFO_PRODUCTNAME} 退出..."
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}" --installer-exit'
        Sleep 1500
    done:
FunctionEnd

## un.CloseRunningApplicationWindow 卸载阶段通过窗口句柄关闭仍在运行的主窗口。
Function un.CloseRunningApplicationWindow
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        DetailPrint "正在关闭已运行的 ${INFO_PRODUCTNAME}..."
        SendMessage $0 ${WM_CLOSE} 0 0 /TIMEOUT=5000
        Sleep 1500
    ${EndIf}
FunctionEnd

## un.ForceTerminateRunningApplication 卸载阶段兜底结束仍占用安装目录的旧进程。
Function un.ForceTerminateRunningApplication
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        System::Call 'user32::GetWindowThreadProcessId(p r0, *i .r1) i .r2'
        ${If} $1 != 0
            DetailPrint "正在确保已运行的 ${INFO_PRODUCTNAME} 退出..."
            nsExec::ExecToLog 'taskkill /PID $1 /T /F'
            Pop $2
            Sleep 500
        ${EndIf}
    ${EndIf}
FunctionEnd

## un.DeleteCurrentUserUninstaller 删除当前用户范围卸载器和对应注册表项。
Function un.DeleteCurrentUserUninstaller
    Delete "$INSTDIR\uninstall.exe"

    SetRegView 64
    DeleteRegKey HKCU "${UNINST_KEY_CURRENT_USER}"
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    
    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    Call WriteCurrentUserUninstaller
SectionEnd

Section "uninstall" 
    Call un.RequestRunningApplicationExit
    Call un.CloseRunningApplicationWindow
    Call un.ForceTerminateRunningApplication
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # 删除 WebView2 数据目录。

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    Call un.DeleteCurrentUserUninstaller
SectionEnd
`, "{{DEFAULT_VERSION}}", nsisString(meta.DefaultVersion))
}

// renderBuildConfig 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderBuildConfig(meta metadata) string {
	return fmt.Sprintf(`# 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
# 修改产品名、仓库、版本兜底或 Windows 标识时，只改 project.metadata.json，再运行同步脚本。
version: '3'

info:
  companyName: %s
  productName: %s
  productIdentifier: %s
  description: %s
  copyright: %s
  comments: %s
  version: %s

dev_mode:
  root_path: .
  log_level: warn
  debounce: 1000
  ignore:
    dir:
      - .git
      - node_modules
      - frontend
      - bin
    file:
      - .DS_Store
      - .gitignore
      - .gitkeep
    watched_extension:
      - "*.go"
      - "*.js"
      - "*.ts"
    git_ignore: true
  executes:
    - cmd: go run ./scripts/envrun wails3 task windows:build DEV=true
      type: blocking
    - cmd: wails3 task common:dev:frontend
      type: background
    - cmd: wails3 task run
      type: primary

fileAssociations:

other:
  - name: My Other Data
`,
		yamlString(meta.CompanyName),
		yamlString(meta.AppName),
		yamlString(meta.Windows.ProductIdentifier),
		yamlString(meta.Description),
		yamlString(meta.Copyright),
		yamlString(meta.RepositoryComment),
		yamlString(meta.DefaultVersion),
	)
}

// renderWindowsInfo 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderWindowsInfo(meta metadata) string {
	document := map[string]any{
		"fixed": map[string]any{
			"file_version": meta.DefaultVersion,
		},
		"info": map[string]any{
			"0000": map[string]any{
				"ProductVersion":  meta.DefaultVersion,
				"FileVersion":     meta.DefaultVersion,
				"CompanyName":     meta.CompanyName,
				"FileDescription": meta.AppName,
				"LegalCopyright":  meta.Copyright,
				"ProductName":     meta.AppName,
				"Comments":        meta.RepositoryComment,
			},
		},
	}
	rendered, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		exitf("生成 Windows 版本资源失败：%v", err)
	}
	return string(rendered) + "\n"
}

// renderWindowsManifest 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderWindowsManifest(meta metadata) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly manifestVersion="1.0" xmlns="urn:schemas-microsoft-com:asm.v1" xmlns:asmv3="urn:schemas-microsoft-com:asm.v3">
    <assemblyIdentity type="win32" name="%s" version="%s" processorArchitecture="*"/>
    <dependency>
        <dependentAssembly>
            <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
        </dependentAssembly>
    </dependency>
    <asmv3:application>
        <asmv3:windowsSettings>
            <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
            <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">permonitorv2,permonitor</dpiAwareness>
        </asmv3:windowsSettings>
    </asmv3:application>
    <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
        <security>
            <requestedPrivileges>
                <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
            </requestedPrivileges>
        </security>
    </trustInfo>
</assembly>
`, xmlAttr(meta.Windows.ProductIdentifier), xmlAttr(fourPartVersion(meta.DefaultVersion)))
}

// renderWindowsMSIXTemplate 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderWindowsMSIXTemplate(meta metadata) string {
	executable := meta.AppName + ".exe"
	version := fourPartVersion(meta.DefaultVersion)
	publisher := "CN=" + meta.CompanyName
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<MsixPackagingToolTemplate
    xmlns="http://schemas.microsoft.com/msix/packaging/msixpackagingtool/template/2022">
    <Settings
        AllowTelemetry="false"
        ApplyACLsToPackageFiles="true"
        GenerateCommandLineFile="true"
        AllowPromptForPassword="false">
    </Settings>
    <Installer
        Path="%s"
        Arguments=""
        InstallLocation="%s">
    </Installer>
    <PackageInformation
        PackageName="%s"
        PackageDisplayName="%s"
        PublisherName="%s"
        PublisherDisplayName="%s"
        Version="%s"
        PackageDescription="%s">
        <Capabilities>
            <Capability Name="runFullTrust" />
            
        </Capabilities>
        <Applications>
            <Application
                Id="%s"
                Description="%s"
                DisplayName="%s"
                ExecutableName="%s"
                EntryPoint="Windows.FullTrustApplication">
                
            </Application>
        </Applications>
        <Resources>
            <Resource Language="zh-cn" />
        </Resources>
        <Dependencies>
            <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.17763.0" MaxVersionTested="10.0.19041.0" />
        </Dependencies>
        <Properties>
            <Framework>false</Framework>
            <DisplayName>%s</DisplayName>
            <PublisherDisplayName>%s</PublisherDisplayName>
            <Description>%s</Description>
            <Logo>Assets\AppIcon.png</Logo>
        </Properties>
    </PackageInformation>
    <SaveLocation PackagePath="%s.msix" />
    <PackageIntegrity>
        <CertificatePath></CertificatePath>
    </PackageIntegrity>
</MsixPackagingToolTemplate>
`,
		xmlAttr(executable),
		xmlAttr(windowsEnvInstallDir(meta.Windows.InstallDir)),
		xmlAttr(meta.Windows.ProductIdentifier),
		xmlAttr(meta.AppName),
		xmlAttr(publisher),
		xmlAttr(meta.CompanyName),
		xmlAttr(version),
		xmlAttr(meta.Description),
		xmlAttr(meta.Windows.ProductIdentifier),
		xmlAttr(meta.Description),
		xmlAttr(meta.AppName),
		xmlAttr(executable),
		htmlText(meta.AppName),
		htmlText(meta.CompanyName),
		htmlText(meta.Description),
		xmlAttr(meta.AppName),
	)
}

// renderWindowsMSIXManifest 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderWindowsMSIXManifest(meta metadata) string {
	executable := meta.AppName + ".exe"
	version := fourPartVersion(meta.DefaultVersion)
	publisher := "CN=" + meta.CompanyName
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<Package
  xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:uap3="http://schemas.microsoft.com/appx/manifest/uap/windows10/3"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
  xmlns:desktop="http://schemas.microsoft.com/appx/manifest/desktop/windows10"
  IgnorableNamespaces="uap3">

  <Identity
    Name="%s"
    Publisher="%s"
    Version="%s"
    ProcessorArchitecture="x64" />

  <Properties>
    <DisplayName>%s</DisplayName>
    <PublisherDisplayName>%s</PublisherDisplayName>
    <Description>%s</Description>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>

  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.17763.0" MaxVersionTested="10.0.19041.0" />
  </Dependencies>

  <Resources>
    <Resource Language="zh-cn" />
  </Resources>

  <Applications>
    <Application Id="%s" Executable="%s" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements
        DisplayName="%s"
        Description="%s"
        BackgroundColor="transparent"
        Square150x150Logo="Assets\Square150x150Logo.png"
        Square44x44Logo="Assets\Square44x44Logo.png">
        <uap:DefaultTile Wide310x150Logo="Assets\Wide310x150Logo.png" />
        <uap:SplashScreen Image="Assets\SplashScreen.png" />
      </uap:VisualElements>
      
      <Extensions>
        <desktop:Extension Category="windows.fullTrustProcess" Executable="%s" />
        
        
      </Extensions>
    </Application>
  </Applications>
  
  <Capabilities>
    <rescap:Capability Name="runFullTrust" />
    
  </Capabilities>
</Package>
`,
		xmlAttr(meta.Windows.ProductIdentifier),
		xmlAttr(publisher),
		xmlAttr(version),
		htmlText(meta.AppName),
		htmlText(meta.CompanyName),
		htmlText(meta.Description),
		xmlAttr(meta.Windows.ProductIdentifier),
		xmlAttr(executable),
		xmlAttr(meta.AppName),
		xmlAttr(meta.Description),
		xmlAttr(executable),
	)
}

// renderLinuxDesktop 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderLinuxDesktop(meta metadata) string {
	appName := linuxText(meta.AppName)
	return fmt.Sprintf(`[Desktop Entry]
Version=1.0
Name=%s
Comment=%s
Exec=/usr/local/bin/%s %%u
Terminal=false
Type=Application
Icon=%s
Categories=Utility;
StartupWMClass=%s
`, appName, linuxText(meta.Description), appName, appName, appName)
}

// renderLinuxNfpm 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderLinuxNfpm(meta metadata) string {
	appName := meta.AppName
	return fmt.Sprintf(`# 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
name: %s
arch: ${GOARCH}
platform: "linux"
version: %s
section: "default"
priority: "extra"
maintainer: ${GIT_COMMITTER_NAME} <${GIT_COMMITTER_EMAIL}>
description: %s
vendor: %s
homepage: %s
license: "MIT"
release: "1"

contents:
  - src: %s
    dst: %s
  - src: "./build/appicon.png"
    dst: %s
  - src: %s
    dst: %s

# Default dependencies for Debian 12/Ubuntu 22.04+ with WebKit 4.1
depends:
  - libgtk-3-0
  - libwebkit2gtk-4.1-0

# Distribution-specific overrides for different package formats and WebKit versions
overrides:
  # RPM packages for RHEL/CentOS/AlmaLinux/Rocky Linux (WebKit 4.0)
  rpm:
    depends:
      - gtk3
      - webkit2gtk4.1
  
  # Arch Linux packages (WebKit 4.1)  
  archlinux:
    depends:
      - gtk3
      - webkit2gtk-4.1

# scripts section to ensure desktop database is updated after install
scripts:
  postinstall: "./build/linux/nfpm/scripts/postinstall.sh"
  # You can also add preremove, postremove if needed
  # preremove: "./build/linux/nfpm/scripts/preremove.sh"
  # postremove: "./build/linux/nfpm/scripts/postremove.sh"

# replaces:
#   - foobar
# provides:
#   - bar
# depends:
#   - gtk3
#   - libwebkit2gtk
# recommends:
#   - whatever
# suggests:
#   - something-else
# conflicts:
#   - not-foo
#   - not-bar
# changelog: "changelog.yaml"
`,
		yamlString(appName),
		yamlString(meta.DefaultVersion),
		yamlString(meta.Description),
		yamlString(meta.CompanyName),
		yamlString(meta.RepositoryURL),
		yamlString("./bin/"+appName),
		yamlString("/usr/local/bin/"+appName),
		yamlString("/usr/share/icons/hicolor/128x128/apps/"+appName+".png"),
		yamlString("./build/linux/"+appName+".desktop"),
		yamlString("/usr/share/applications/"+appName+".desktop"),
	)
}

// renderDarwinInfoPlist 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderDarwinInfoPlist(meta metadata, dev bool) string {
	displayName := meta.AppName
	identifier := meta.Windows.ProductIdentifier
	shortVersion := meta.DefaultVersion
	if dev {
		displayName += " (Dev)"
		identifier += ".dev"
		shortVersion += "-dev"
	}
	return fmt.Sprintf(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>CFBundlePackageType</key>
            <string>APPL</string>
        <key>CFBundleName</key>
            <string>%s</string>
        <key>CFBundleExecutable</key>
            <string>%s</string>
        <key>CFBundleIdentifier</key>
            <string>%s</string>
        <key>CFBundleVersion</key>
            <string>%s</string>
        <key>CFBundleGetInfoString</key>
            <string>%s</string>
        <key>CFBundleShortVersionString</key>
            <string>%s</string>
        <key>CFBundleIconFile</key>
            <string>icons</string>
        <key>CFBundleIconName</key>
            <string>appicon</string>
        <key>LSMinimumSystemVersion</key>
            <string>10.15.0</string>
        <key>NSHighResolutionCapable</key>
            <string>true</string>
        <key>NSHumanReadableCopyright</key>
            <string>%s</string>
        <key>NSAppTransportSecurity</key>
        <dict>
            <key>NSAllowsLocalNetworking</key>
            <true/>
        </dict>
    </dict>
</plist>
`, htmlText(displayName), htmlText(meta.AppName), htmlText(identifier), htmlText(meta.DefaultVersion), htmlText(meta.RepositoryComment), htmlText(shortVersion), htmlText(meta.Copyright))
}

// renderIOSInfoPlist 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderIOSInfoPlist(meta metadata, dev bool) string {
	displayName := meta.AppName
	identifier := meta.Windows.ProductIdentifier
	shortVersion := meta.DefaultVersion
	allowsArbitraryLoads := "false"
	developmentMode := ""
	if dev {
		displayName += " (Dev)"
		identifier += ".dev"
		shortVersion += "-dev"
		allowsArbitraryLoads = "true"
		developmentMode = `
    <!-- Development mode enabled -->
    <key>WailsDevelopmentMode</key>
    <true/>
`
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>%s</string>
    <key>CFBundleIdentifier</key>
    <string>%s</string>
    <key>CFBundleName</key>
    <string>%s</string>
    <key>CFBundleDisplayName</key>
    <string>%s</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>%s</string>
    <key>CFBundleVersion</key>
    <string>%s</string>
    <key>LSRequiresIPhoneOS</key>
    <true/>
    <key>MinimumOSVersion</key>
    <string>15.0</string>
    <key>UILaunchStoryboardName</key>
    <string>LaunchScreen</string>
    <key>UIRequiredDeviceCapabilities</key>
    <array>
        <string>armv7</string>
        <string>arm64</string>
    </array>
    <key>UISupportedInterfaceOrientations</key>
    <array>
        <string>UIInterfaceOrientationPortrait</string>
        <string>UIInterfaceOrientationLandscapeLeft</string>
        <string>UIInterfaceOrientationLandscapeRight</string>
    </array>
    <key>UISupportedInterfaceOrientations~ipad</key>
    <array>
        <string>UIInterfaceOrientationPortrait</string>
        <string>UIInterfaceOrientationPortraitUpsideDown</string>
        <string>UIInterfaceOrientationLandscapeLeft</string>
        <string>UIInterfaceOrientationLandscapeRight</string>
    </array>
    <key>NSAppTransportSecurity</key>
    <dict>
        <key>NSAllowsArbitraryLoads</key>
        <%s/>
        <key>NSAllowsLocalNetworking</key>
        <true/>
    </dict>%s
    <key>NSHumanReadableCopyright</key>
    <string>%s</string>
    <key>CFBundleGetInfoString</key>
    <string>%s</string>
</dict>
</plist>
`, htmlText(meta.AppName), htmlText(identifier), htmlText(displayName), htmlText(displayName), htmlText(shortVersion), htmlText(meta.DefaultVersion), allowsArbitraryLoads, developmentMode, htmlText(meta.Copyright), htmlText(meta.RepositoryComment))
}

// renderIOSBuildScript 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderIOSBuildScript(meta metadata) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

# 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
APP_NAME=%s
BUNDLE_ID=%s
VERSION=%s
BUILD_NUMBER=%s
BUILD_DIR="build/ios"
TARGET="simulator"

echo "Building iOS app: $APP_NAME"
echo "Bundle ID: $BUNDLE_ID"
echo "Version: $VERSION ($BUILD_NUMBER)"
echo "Target: $TARGET"

mkdir -p "$BUILD_DIR"

if [ "$TARGET" = "simulator" ]; then
    SDK="iphonesimulator"
    ARCH="arm64-apple-ios15.0-simulator"
elif [ "$TARGET" = "device" ]; then
    SDK="iphoneos"
    ARCH="arm64-apple-ios15.0"
else
    echo "Unknown target: $TARGET"
    exit 1
fi

SDK_PATH=$(xcrun --sdk $SDK --show-sdk-path)

echo "Compiling with SDK: $SDK"
xcrun -sdk $SDK clang \
    -target $ARCH \
    -isysroot "$SDK_PATH" \
    -framework Foundation \
    -framework UIKit \
    -framework WebKit \
    -framework CoreGraphics \
    -o "$BUILD_DIR/$APP_NAME" \
    "$BUILD_DIR/main.m"

echo "Creating app bundle..."
APP_BUNDLE="$BUILD_DIR/$APP_NAME.app"
rm -rf "$APP_BUNDLE"
mkdir -p "$APP_BUNDLE"

mv "$BUILD_DIR/$APP_NAME" "$APP_BUNDLE/"
cp "$BUILD_DIR/Info.plist" "$APP_BUNDLE/"

echo "Signing app..."
codesign --force --sign - "$APP_BUNDLE"

echo "Build complete: $APP_BUNDLE"

if [ "$TARGET" = "simulator" ]; then
    echo "Deploying to simulator..."
    xcrun simctl terminate booted "$BUNDLE_ID" 2>/dev/null || true
    xcrun simctl install booted "$APP_BUNDLE"
    xcrun simctl launch booted "$BUNDLE_ID"
    echo "App launched on simulator"
fi
`, shellString(meta.AppName), shellString(meta.Windows.ProductIdentifier), shellString(meta.DefaultVersion), shellString(meta.DefaultVersion))
}

// renderAndroidBuildGradle 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderAndroidBuildGradle(meta metadata) string {
	return fmt.Sprintf(`plugins {
    id 'com.android.application'
}

android {
    namespace 'com.wails.app'
    compileSdk 34

    buildFeatures {
        buildConfig = true
    }

    defaultConfig {
        applicationId %s
        minSdk 21
        targetSdk 34
        versionCode 1
        versionName %s

        // Configure supported ABIs
        ndk {
            abiFilters 'arm64-v8a', 'x86_64'
        }
    }

    buildTypes {
        release {
            minifyEnabled false
            proguardFiles getDefaultProguardFile('proguard-android-optimize.txt'), 'proguard-rules.pro'
        }
        debug {
            debuggable true
        }
    }

    compileOptions {
        sourceCompatibility JavaVersion.VERSION_11
        targetCompatibility JavaVersion.VERSION_11
    }

    // Source sets configuration
    sourceSets {
        main {
            // JNI libraries are in jniLibs folder
            jniLibs.srcDirs = ['src/main/jniLibs']
            // Assets for the WebView
            assets.srcDirs = ['src/main/assets']
        }
    }

    // Packaging options
    packagingOptions {
        // Don't strip Go symbols in debug builds
        doNotStrip '*/arm64-v8a/libwails.so'
        doNotStrip '*/x86_64/libwails.so'
    }
}

dependencies {
    implementation 'androidx.appcompat:appcompat:1.6.1'
    implementation 'androidx.webkit:webkit:1.9.0'
    implementation 'com.google.android.material:material:1.11.0'
}
`, groovyString(meta.Windows.ProductIdentifier), groovyString(meta.DefaultVersion))
}

// renderAndroidSettingsGradle 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderAndroidSettingsGradle(meta metadata) string {
	return fmt.Sprintf(`pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = %s
include ':app'
`, groovyString(meta.AppName))
}

// renderAndroidStrings 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderAndroidStrings(meta metadata) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<resources>
    <string name="app_name">%s</string>
</resources>
`, xmlText(meta.AppName))
}

// renderAndroidTaskfile 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderAndroidTaskfile(meta metadata) string {
	return fmt.Sprintf(`version: '3'

includes:
  common: ../Taskfile.yml

vars:
  APP_ID: '{{.APP_ID | default %s}}'
  MIN_SDK: '21'
  TARGET_SDK: '34'
  NDK_VERSION: 'r26d'

tasks:
  install:deps:
    summary: Check and install Android development dependencies
    cmds:
      - go run build/android/scripts/deps/install_deps.go
    env:
      TASK_FORCE_YES: '{{if .YES}}true{{else}}false{{end}}'
    prompt: This will check and install Android development dependencies. Continue?

  # Note: Bindings generation may show CGO warnings for Android C imports.
  # These warnings are harmless and don't affect the generated bindings,
  # as the generator only needs to parse Go types, not C implementations.
  build:
    summary: Creates a build of the application for Android
    deps:
      - task: common:go:mod:tidy
      - task: generate:android:bindings
        vars:
          BUILD_FLAGS:
            ref: .BUILD_FLAGS
      - task: common:build:frontend
        vars:
          BUILD_FLAGS:
            ref: .BUILD_FLAGS
          PRODUCTION:
            ref: .PRODUCTION
      - task: common:generate:icons
    cmds:
      - echo "Building Android app {{.APP_NAME}}..."
      - task: compile:go:shared
        vars:
          ARCH: '{{.ARCH | default "arm64"}}'
    vars:
      BUILD_FLAGS: '{{if eq .PRODUCTION "true"}}-tags production,android -trimpath -buildvcs=false -ldflags="-w -s"{{else}}-tags android,debug -buildvcs=false -gcflags=all="-l"{{end}}'
    env:
      PRODUCTION: '{{.PRODUCTION | default "false"}}'

  compile:go:shared:
    summary: Compile Go code to shared library (.so)
    cmds:
      - |
        NDK_ROOT="${ANDROID_NDK_HOME:-$ANDROID_HOME/ndk/{{.NDK_VERSION}}}"
        if [ ! -d "$NDK_ROOT" ]; then
          echo "Error: Android NDK not found at $NDK_ROOT"
          echo "Please set ANDROID_NDK_HOME or install NDK {{.NDK_VERSION}} via Android Studio"
          exit 1
        fi

        # Determine toolchain based on host OS
        case "$(uname -s)" in
          Darwin) HOST_TAG="darwin-x86_64" ;;
          Linux)  HOST_TAG="linux-x86_64" ;;
          *)      echo "Unsupported host OS"; exit 1 ;;
        esac

        TOOLCHAIN="$NDK_ROOT/toolchains/llvm/prebuilt/$HOST_TAG"

        # Set compiler based on architecture
        case "{{.ARCH}}" in
          arm64)
            export CC="$TOOLCHAIN/bin/aarch64-linux-android{{.MIN_SDK}}-clang"
            export CXX="$TOOLCHAIN/bin/aarch64-linux-android{{.MIN_SDK}}-clang++"
            export GOARCH=arm64
            JNI_DIR="arm64-v8a"
            ;;
          amd64|x86_64)
            export CC="$TOOLCHAIN/bin/x86_64-linux-android{{.MIN_SDK}}-clang"
            export CXX="$TOOLCHAIN/bin/x86_64-linux-android{{.MIN_SDK}}-clang++"
            export GOARCH=amd64
            JNI_DIR="x86_64"
            ;;
          *)
            echo "Unsupported architecture: {{.ARCH}}"
            exit 1
            ;;
        esac

        export CGO_ENABLED=1
        export GOOS=android

        mkdir -p {{.BIN_DIR}}
        mkdir -p build/android/app/src/main/jniLibs/$JNI_DIR

        go build -buildmode=c-shared {{.BUILD_FLAGS}} \
          -o build/android/app/src/main/jniLibs/$JNI_DIR/libwails.so
    vars:
      BUILD_FLAGS: '{{if eq .PRODUCTION "true"}}-tags production,android -trimpath -buildvcs=false -ldflags="-w -s"{{else}}-tags android,debug -buildvcs=false -gcflags=all="-l"{{end}}'

  compile:go:all-archs:
    summary: Compile Go code for all Android architectures (fat APK)
    cmds:
      - task: compile:go:shared
        vars:
          ARCH: arm64
      - task: compile:go:shared
        vars:
          ARCH: amd64

  package:
    summary: Packages a production build of the application into an APK
    deps:
      - task: build
        vars:
          PRODUCTION: "true"
    cmds:
      - task: assemble:apk

  package:fat:
    summary: Packages a production build for all architectures (fat APK)
    cmds:
      - task: compile:go:all-archs
      - task: assemble:apk

  assemble:apk:
    summary: Assembles the APK using Gradle
    cmds:
      - |
        cd build/android
        ./gradlew assembleDebug
        cp app/build/outputs/apk/debug/app-debug.apk "../../{{.BIN_DIR}}/{{.APP_NAME}}.apk"
        echo "APK created: {{.BIN_DIR}}/{{.APP_NAME}}.apk"

  assemble:apk:release:
    summary: Assembles a release APK using Gradle
    cmds:
      - |
        cd build/android
        ./gradlew assembleRelease
        cp app/build/outputs/apk/release/app-release-unsigned.apk "../../{{.BIN_DIR}}/{{.APP_NAME}}-release.apk"
        echo "Release APK created: {{.BIN_DIR}}/{{.APP_NAME}}-release.apk"

  generate:android:bindings:
    internal: true
    summary: Generates bindings for Android
    sources:
      - "**/*.go"
      - go.mod
      - go.sum
    generates:
      - frontend/bindings/**/*
    cmds:
      - wails3 generate bindings -f '{{.BUILD_FLAGS}}' -clean=false
    env:
      GOOS: android
      CGO_ENABLED: 1
      GOARCH: '{{.ARCH | default "arm64"}}'

  ensure-emulator:
    internal: true
    summary: Ensure Android Emulator is running
    silent: true
    cmds:
      - |
        # Check if an emulator is already running
        if adb devices | grep -q "emulator"; then
          echo "Emulator already running"
          exit 0
        fi

        # Get first available AVD
        AVD_NAME=$(emulator -list-avds | head -1)
        if [ -z "$AVD_NAME" ]; then
          echo "No Android Virtual Devices found."
          echo "Create one using: Android Studio > Tools > Device Manager"
          exit 1
        fi

        echo "Starting emulator: $AVD_NAME"
        emulator -avd "$AVD_NAME" -no-snapshot-load &

        # Wait for emulator to boot (max 60 seconds)
        echo "Waiting for emulator to boot..."
        adb wait-for-device

        for i in {1..60}; do
          BOOT_COMPLETED=$(adb shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')
          if [ "$BOOT_COMPLETED" = "1" ]; then
            echo "Emulator booted successfully"
            exit 0
          fi
          sleep 1
        done

        echo "Emulator boot timeout"
        exit 1
    preconditions:
      - sh: command -v adb
        msg: "adb not found. Please install Android SDK and add platform-tools to PATH"
      - sh: command -v emulator
        msg: "emulator not found. Please install Android SDK and add emulator to PATH"

  deploy-emulator:
    summary: Deploy to Android Emulator
    deps: [package]
    cmds:
      - adb uninstall {{.APP_ID}} 2>/dev/null || true
      - adb install "{{.BIN_DIR}}/{{.APP_NAME}}.apk"
      - adb shell am start -n {{.APP_ID}}/.MainActivity

  run:
    summary: Run the application in Android Emulator
    deps:
      - task: ensure-emulator
      - task: build
        vars:
          ARCH: x86_64
    cmds:
      - task: assemble:apk
      - adb uninstall {{.APP_ID}} 2>/dev/null || true
      - adb install "{{.BIN_DIR}}/{{.APP_NAME}}.apk"
      - adb shell am start -n {{.APP_ID}}/.MainActivity

  logs:
    summary: Stream Android logcat filtered to this app
    cmds:
      - adb logcat -v time | grep -E "(Wails|{{.APP_NAME}})"

  logs:all:
    summary: Stream all Android logcat (verbose)
    cmds:
      - adb logcat -v time

  clean:
    summary: Clean build artifacts
    cmds:
      - rm -rf {{.BIN_DIR}}
      - rm -rf build/android/app/build
      - rm -rf build/android/app/src/main/jniLibs/*/libwails.so
      - rm -rf build/android/.gradle
`, yamlString(meta.Windows.ProductIdentifier))
}

// renderIOSTaskfile 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderIOSTaskfile(meta metadata) string {
	return fmt.Sprintf(`version: '3'

includes:
  common: ../Taskfile.yml

vars:
  BUNDLE_ID: '{{.BUNDLE_ID | default %s}}'
  # SDK_PATH is computed lazily at task-level to avoid errors on non-macOS systems
  # Each task that needs it defines SDK_PATH in its own vars section

tasks:
  install:deps:
    summary: Check and install iOS development dependencies
    cmds:
      - go run build/ios/scripts/deps/install_deps.go
    env:
      TASK_FORCE_YES: '{{if .YES}}true{{else}}false{{end}}'
    prompt: This will check and install iOS development dependencies. Continue?

  # Note: Bindings generation may show CGO warnings for iOS C imports.
  # These warnings are harmless and don't affect the generated bindings,
  # as the generator only needs to parse Go types, not C implementations.
  build:
    summary: Creates a build of the application for iOS
    deps:
      - task: generate:ios:overlay
      - task: generate:ios:xcode
      - task: common:go:mod:tidy
      - task: generate:ios:bindings
        vars:
          BUILD_FLAGS:
            ref: .BUILD_FLAGS
      - task: common:build:frontend
        vars:
          BUILD_FLAGS:
            ref: .BUILD_FLAGS
          PRODUCTION:
            ref: .PRODUCTION
      - task: common:generate:icons
    cmds:
      - echo "Building iOS app {{.APP_NAME}}..."
      - go build -buildmode=c-archive -overlay build/ios/xcode/overlay.json {{.BUILD_FLAGS}} -o {{.OUTPUT}}.a
    vars:
      BUILD_FLAGS: '{{if eq .PRODUCTION "true"}}-tags production,ios -trimpath -buildvcs=false -ldflags="-w -s"{{else}}-tags ios,debug -buildvcs=false -gcflags=all="-l"{{end}}'
      DEFAULT_OUTPUT: '{{.BIN_DIR}}/{{.APP_NAME}}'
      OUTPUT: '{{ .OUTPUT | default .DEFAULT_OUTPUT }}'
      SDK_PATH:
        sh: xcrun --sdk iphonesimulator --show-sdk-path
    env:
      GOOS: ios
      CGO_ENABLED: 1
      GOARCH: '{{.ARCH | default "arm64"}}'
      PRODUCTION: '{{.PRODUCTION | default "false"}}'
      CGO_CFLAGS: '-isysroot {{.SDK_PATH}} -target arm64-apple-ios15.0-simulator -mios-simulator-version-min=15.0'
      CGO_LDFLAGS: '-isysroot {{.SDK_PATH}} -target arm64-apple-ios15.0-simulator'

  compile:objc:
    summary: Compile Objective-C iOS wrapper
    vars:
      SDK_PATH:
        sh: xcrun --sdk iphonesimulator --show-sdk-path
    cmds:
      - xcrun -sdk iphonesimulator clang -target arm64-apple-ios15.0-simulator -isysroot {{.SDK_PATH}} -framework Foundation -framework UIKit -framework WebKit -o {{.BIN_DIR}}/{{.APP_NAME}} build/ios/main.m
      - codesign --force --sign - "{{.BIN_DIR}}/{{.APP_NAME}}"

  package:
    summary: Packages a production build of the application into a '.app' bundle
    deps:
      - task: build
        vars:
          PRODUCTION: "true"
    cmds:
      - task: create:app:bundle

  create:app:bundle:
    summary: Creates an iOS '.app' bundle
    cmds:
      - rm -rf "{{.BIN_DIR}}/{{.APP_NAME}}.app"
      - mkdir -p "{{.BIN_DIR}}/{{.APP_NAME}}.app"
      - cp "{{.BIN_DIR}}/{{.APP_NAME}}" "{{.BIN_DIR}}/{{.APP_NAME}}.app/"
      - cp build/ios/Info.plist "{{.BIN_DIR}}/{{.APP_NAME}}.app/"
      - |
        # Compile asset catalog and embed icons in the app bundle
        APP_BUNDLE="{{.BIN_DIR}}/{{.APP_NAME}}.app"
        AC_IN="build/ios/xcode/main/Assets.xcassets"
        if [ -d "$AC_IN" ]; then
          TMP_AC=$(mktemp -d)
          xcrun actool \
            --compile "$TMP_AC" \
            --app-icon AppIcon \
            --platform iphonesimulator \
            --minimum-deployment-target 15.0 \
            --product-type com.apple.product-type.application \
            --target-device iphone \
            --target-device ipad \
            --output-partial-info-plist "$APP_BUNDLE/assetcatalog_generated_info.plist" \
            "$AC_IN"
          if [ -f "$TMP_AC/Assets.car" ]; then
            cp -f "$TMP_AC/Assets.car" "$APP_BUNDLE/Assets.car"
          fi
          rm -rf "$TMP_AC"
          if [ -f "$APP_BUNDLE/assetcatalog_generated_info.plist" ]; then
            /usr/libexec/PlistBuddy -c "Merge $APP_BUNDLE/assetcatalog_generated_info.plist" "$APP_BUNDLE/Info.plist" || true
          fi
        fi
      - codesign --force --sign - "{{.BIN_DIR}}/{{.APP_NAME}}.app"

  deploy-simulator:
    summary: Deploy to iOS Simulator
    deps: [package]
    cmds:
      - xcrun simctl terminate booted {{.BUNDLE_ID}} 2>/dev/null || true
      - xcrun simctl uninstall booted {{.BUNDLE_ID}} 2>/dev/null || true
      - xcrun simctl install booted "{{.BIN_DIR}}/{{.APP_NAME}}.app"
      - xcrun simctl launch booted {{.BUNDLE_ID}}

  compile:ios:
    summary: Compile the iOS executable from Go archive and main.m
    deps:
      - task: build
    vars:
      SDK_PATH:
        sh: xcrun --sdk iphonesimulator --show-sdk-path
    cmds:
      - |
        MAIN_M=build/ios/xcode/main/main.m
        if [ ! -f "$MAIN_M" ]; then
          MAIN_M=build/ios/main.m
        fi
        xcrun -sdk iphonesimulator clang \
          -target arm64-apple-ios15.0-simulator \
          -isysroot {{.SDK_PATH}} \
          -framework Foundation -framework UIKit -framework WebKit \
          -framework Security -framework CoreFoundation \
          -lresolv \
          -o "{{.BIN_DIR}}/{{.APP_NAME | lower}}" \
          "$MAIN_M" "{{.BIN_DIR}}/{{.APP_NAME}}.a"

  generate:ios:bindings:
    internal: true
    summary: Generates bindings for iOS with proper CGO flags
    sources:
      - "**/*.go"
      - go.mod
      - go.sum
    generates:
      - frontend/bindings/**/*
    vars:
      SDK_PATH:
        sh: xcrun --sdk iphonesimulator --show-sdk-path
    cmds:
      - wails3 generate bindings -f '{{.BUILD_FLAGS}}' -clean=false
    env:
      GOOS: ios
      CGO_ENABLED: 1
      GOARCH: '{{.ARCH | default "arm64"}}'
      CGO_CFLAGS: '-isysroot {{.SDK_PATH}} -target arm64-apple-ios15.0-simulator -mios-simulator-version-min=15.0'
      CGO_LDFLAGS: '-isysroot {{.SDK_PATH}} -target arm64-apple-ios15.0-simulator'

  ensure-simulator:
    internal: true
    summary: Ensure iOS Simulator is running and booted
    silent: true
    cmds:
      - |
        if ! xcrun simctl list devices booted | grep -q "Booted"; then
          echo "Starting iOS Simulator..."
          # Get first available iPhone device
          DEVICE_ID=$(xcrun simctl list devices available | grep "iPhone" | head -1 | grep -o "[A-F0-9-]\{36\}" || true)
          if [ -z "$DEVICE_ID" ]; then
            echo "No iPhone simulator found. Creating one..."
            RUNTIME=$(xcrun simctl list runtimes | grep iOS | tail -1 | awk '{print $NF}')
            DEVICE_ID=$(xcrun simctl create "iPhone 15 Pro" "iPhone 15 Pro" "$RUNTIME")
          fi
          # Boot the device
          echo "Booting device $DEVICE_ID..."
          xcrun simctl boot "$DEVICE_ID" 2>/dev/null || true
          # Open Simulator app
          open -a Simulator
          # Wait for boot (max 30 seconds)
          for i in {1..30}; do
            if xcrun simctl list devices booted | grep -q "Booted"; then
              echo "Simulator booted successfully"
              break
            fi
            sleep 1
          done
          # Final check
          if ! xcrun simctl list devices booted | grep -q "Booted"; then
            echo "Failed to boot simulator after 30 seconds"
            exit 1
          fi
        fi
    preconditions:
      - sh: command -v xcrun
        msg: "xcrun not found. Please run 'wails3 task ios:install:deps' to install iOS development dependencies"

  generate:ios:overlay:
    internal: true
    summary: Generate Go build overlay and iOS shim
    sources:
      - build/config.yml
    generates:
      - build/ios/xcode/overlay.json
      - build/ios/xcode/gen/main_ios.gen.go
    cmds:
      - wails3 ios overlay:gen -out build/ios/xcode/overlay.json -config build/config.yml

  generate:ios:xcode:
    internal: true
    summary: Generate iOS Xcode project structure and assets
    sources:
      - build/config.yml
      - build/appicon.png
    generates:
      - build/ios/xcode/main/main.m
      - build/ios/xcode/main/Assets.xcassets/**/*
      - build/ios/xcode/project.pbxproj
    cmds:
      - wails3 ios xcode:gen -outdir build/ios/xcode -config build/config.yml

  run:
    summary: Run the application in iOS Simulator
    deps:
      - task: ensure-simulator
      - task: compile:ios
    cmds:
      - rm -rf "{{.BIN_DIR}}/{{.APP_NAME}}.dev.app"
      - mkdir -p "{{.BIN_DIR}}/{{.APP_NAME}}.dev.app"
      - cp "{{.BIN_DIR}}/{{.APP_NAME | lower}}" "{{.BIN_DIR}}/{{.APP_NAME}}.dev.app/{{.APP_NAME | lower}}"
      - cp build/ios/Info.dev.plist "{{.BIN_DIR}}/{{.APP_NAME}}.dev.app/Info.plist"
      - |
        # Compile asset catalog and embed icons for dev bundle
        APP_BUNDLE="{{.BIN_DIR}}/{{.APP_NAME}}.dev.app"
        AC_IN="build/ios/xcode/main/Assets.xcassets"
        if [ -d "$AC_IN" ]; then
          TMP_AC=$(mktemp -d)
          xcrun actool \
            --compile "$TMP_AC" \
            --app-icon AppIcon \
            --platform iphonesimulator \
            --minimum-deployment-target 15.0 \
            --product-type com.apple.product-type.application \
            --target-device iphone \
            --target-device ipad \
            --output-partial-info-plist "$APP_BUNDLE/assetcatalog_generated_info.plist" \
            "$AC_IN"
          if [ -f "$TMP_AC/Assets.car" ]; then
            cp -f "$TMP_AC/Assets.car" "$APP_BUNDLE/Assets.car"
          fi
          rm -rf "$TMP_AC"
          if [ -f "$APP_BUNDLE/assetcatalog_generated_info.plist" ]; then
            /usr/libexec/PlistBuddy -c "Merge $APP_BUNDLE/assetcatalog_generated_info.plist" "$APP_BUNDLE/Info.plist" || true
          fi
        fi
      - codesign --force --sign - "{{.BIN_DIR}}/{{.APP_NAME}}.dev.app"
      - xcrun simctl terminate booted "{{.BUNDLE_ID}}.dev" 2>/dev/null || true
      - xcrun simctl uninstall booted "{{.BUNDLE_ID}}.dev" 2>/dev/null || true
      - xcrun simctl install booted "{{.BIN_DIR}}/{{.APP_NAME}}.dev.app"
      - xcrun simctl launch booted "{{.BUNDLE_ID}}.dev"

  xcode:
    summary: Open the generated Xcode project for this app
    cmds:
      - task: generate:ios:xcode
      - open build/ios/xcode/main.xcodeproj

  logs:
    summary: Stream iOS Simulator logs filtered to this app
    cmds:
      - |
        xcrun simctl spawn booted log stream \
          --level debug \
          --style compact \
          --predicate 'senderImagePath CONTAINS[c] "{{.APP_NAME | lower}}.app/" OR composedMessage CONTAINS[c] "{{.APP_NAME | lower}}" OR eventMessage CONTAINS[c] "{{.APP_NAME | lower}}" OR process == "{{.APP_NAME | lower}}" OR category CONTAINS[c] "{{.APP_NAME | lower}}"'

  logs:dev:
    summary: Stream logs for the dev bundle (used by 'task ios:run')
    cmds:
      - |
        xcrun simctl spawn booted log stream \
          --level debug \
          --style compact \
          --predicate 'senderImagePath CONTAINS[c] ".dev.app/" OR subsystem == "{{.BUNDLE_ID}}.dev" OR process == "{{.APP_NAME | lower}}"'

  logs:wide:
    summary: Wide log stream to help discover the exact process/bundle identifiers
    cmds:
      - |
        xcrun simctl spawn booted log stream \
          --level debug \
          --style compact \
          --predicate 'senderImagePath CONTAINS[c] ".app/"'
`, yamlString(meta.Windows.ProductIdentifier))
}

// renderIOSLaunchScreen 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderIOSLaunchScreen(meta metadata) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<document type="com.apple.InterfaceBuilder3.CocoaTouch.Storyboard.XIB" version="3.0" toolsVersion="21701" targetRuntime="iOS.CocoaTouch" propertyAccessControl="none" useAutolayout="YES" launchScreen="YES" useTraitCollections="YES" useSafeAreas="YES" colorMatched="YES" initialViewController="01J-lp-oVM">
    <device id="retina6_12" orientation="portrait" appearance="light"/>
    <dependencies>
        <deployment identifier="iOS"/>
        <plugIn identifier="com.apple.InterfaceBuilder.IBCocoaTouchPlugin" version="21678"/>
        <capability name="Safe area layout guides" minToolsVersion="9.0"/>
        <capability name="documents saved in the Xcode 8 format" minToolsVersion="8.0"/>
    </dependencies>
    <scenes>
        <!--View Controller-->
        <scene sceneID="EHf-IW-A2E">
            <objects>
                <viewController id="01J-lp-oVM" sceneMemberID="viewController">
                    <view key="view" contentMode="scaleToFill" id="Ze5-6b-2t3">
                        <rect key="frame" x="0.0" y="0.0" width="393" height="852"/>
                        <autoresizingMask key="autoresizingMask" widthSizable="YES" heightSizable="YES"/>
                        <subviews>
                            <label opaque="NO" clipsSubviews="YES" userInteractionEnabled="NO" contentMode="left" horizontalHuggingPriority="251" verticalHuggingPriority="251" text="%s" textAlignment="center" lineBreakMode="middleTruncation" baselineAdjustment="alignBaselines" minimumFontSize="18" translatesAutoresizingMaskIntoConstraints="NO" id="GJd-Yh-RWb">
                                <rect key="frame" x="0.0" y="397" width="393" height="43"/>
                                <fontDescription key="fontDescription" type="boldSystem" pointSize="36"/>
                                <nil key="textColor"/>
                                <nil key="highlightedColor"/>
                            </label>

                            <label opaque="NO" clipsSubviews="YES" userInteractionEnabled="NO" contentMode="left" horizontalHuggingPriority="251" verticalHuggingPriority="251" text="%s" textAlignment="center" lineBreakMode="tailTruncation" baselineAdjustment="alignBaselines" minimumFontSize="9" translatesAutoresizingMaskIntoConstraints="NO" id="MN2-I3-ftu">
                                <rect key="frame" x="0.0" y="448" width="393" height="21"/>
                                <fontDescription key="fontDescription" type="system" pointSize="17"/>
                                <nil key="textColor"/>
                                <nil key="highlightedColor"/>
                            </label>

                        </subviews>
                        <viewLayoutGuide key="safeArea" id="Bcu-3y-fUS"/>
                        <color key="backgroundColor" white="1" alpha="1" colorSpace="custom" customColorSpace="genericGamma22GrayColorSpace"/>
                        <constraints>
                            <constraint firstItem="Bcu-3y-fUS" firstAttribute="centerX" secondItem="GJd-Yh-RWb" secondAttribute="centerX" id="Q3B-4B-g5h"/>
                            <constraint firstItem="GJd-Yh-RWb" firstAttribute="centerY" secondItem="Ze5-6b-2t3" secondAttribute="bottom" multiplier="1/2" constant="-20" id="moa-c2-u7t"/>
                            <constraint firstItem="GJd-Yh-RWb" firstAttribute="leading" secondItem="Bcu-3y-fUS" secondAttribute="leading" symbolic="YES" id="x7j-FC-K8j"/>

                            <constraint firstItem="MN2-I3-ftu" firstAttribute="top" secondItem="GJd-Yh-RWb" secondAttribute="bottom" constant="8" symbolic="YES" id="cPy-rs-vsC"/>
                            <constraint firstItem="MN2-I3-ftu" firstAttribute="centerX" secondItem="Bcu-3y-fUS" secondAttribute="centerX" id="OQL-iM-xY6"/>
                            <constraint firstItem="MN2-I3-ftu" firstAttribute="leading" secondItem="Bcu-3y-fUS" secondAttribute="leading" symbolic="YES" id="Dti-5h-tvW"/>

                        </constraints>
                    </view>
                </viewController>
                <placeholder placeholderIdentifier="IBFirstResponder" id="iYj-Kq-Ea1" userLabel="First Responder" sceneMemberID="firstResponder"/>
            </objects>
            <point key="canvasLocation" x="53" y="375"/>
        </scene>
    </scenes>
</document>
`, xmlAttr(meta.AppName), xmlAttr(meta.Description))
}

// renderIOSProjectPBXProj 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderIOSProjectPBXProj(meta metadata) string {
	appName := pbxString(meta.AppName)
	companyName := pbxString(meta.CompanyName)
	productID := pbxString(meta.Windows.ProductIdentifier)
	return fmt.Sprintf(`// !$*UTF8*$!
{
	archiveVersion = 1;
	classes = {};
	objectVersion = 56;
	objects = {

/* Begin PBXBuildFile section */
		C0DEBEEF0000000000000001 /* main.m in Sources */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000002 /* main.m */; };
		C0DEBEEF00000000000000F1 /* UIKit.framework in Frameworks */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000101 /* UIKit.framework */; };
		C0DEBEEF00000000000000F2 /* Foundation.framework in Frameworks */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000102 /* Foundation.framework */; };
		C0DEBEEF00000000000000F3 /* WebKit.framework in Frameworks */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000103 /* WebKit.framework */; };
		C0DEBEEF00000000000000F4 /* Security.framework in Frameworks */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000104 /* Security.framework */; };
		C0DEBEEF00000000000000F5 /* CoreFoundation.framework in Frameworks */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000105 /* CoreFoundation.framework */; };
		C0DEBEEF00000000000000F6 /* libresolv.tbd in Frameworks */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000106 /* libresolv.tbd */; };
		C0DEBEEF00000000000000F7 /* %s.a in Frameworks */ = {isa = PBXBuildFile; fileRef = C0DEBEEF0000000000000107 /* %s.a */; };
/* End PBXBuildFile section */

/* Begin PBXFileReference section */
		C0DEBEEF0000000000000002 /* main.m */ = {isa = PBXFileReference; lastKnownFileType = sourcecode.c.objc; path = main.m; sourceTree = "<group>"; };
		C0DEBEEF0000000000000003 /* Info.plist */ = {isa = PBXFileReference; lastKnownFileType = text.plist.xml; path = Info.plist; sourceTree = "<group>"; };
		C0DEBEEF0000000000000004 /* %s.app */ = {isa = PBXFileReference; explicitFileType = wrapper.application; includeInIndex = 0; path = "%s.app"; sourceTree = BUILT_PRODUCTS_DIR; };
		C0DEBEEF0000000000000101 /* UIKit.framework */ = {isa = PBXFileReference; lastKnownFileType = wrapper.framework; name = UIKit.framework; path = System/Library/Frameworks/UIKit.framework; sourceTree = SDKROOT; };
		C0DEBEEF0000000000000102 /* Foundation.framework */ = {isa = PBXFileReference; lastKnownFileType = wrapper.framework; name = Foundation.framework; path = System/Library/Frameworks/Foundation.framework; sourceTree = SDKROOT; };
		C0DEBEEF0000000000000103 /* WebKit.framework */ = {isa = PBXFileReference; lastKnownFileType = wrapper.framework; name = WebKit.framework; path = System/Library/Frameworks/WebKit.framework; sourceTree = SDKROOT; };
		C0DEBEEF0000000000000104 /* Security.framework */ = {isa = PBXFileReference; lastKnownFileType = wrapper.framework; name = Security.framework; path = System/Library/Frameworks/Security.framework; sourceTree = SDKROOT; };
		C0DEBEEF0000000000000105 /* CoreFoundation.framework */ = {isa = PBXFileReference; lastKnownFileType = wrapper.framework; name = CoreFoundation.framework; path = System/Library/Frameworks/CoreFoundation.framework; sourceTree = SDKROOT; };
		C0DEBEEF0000000000000106 /* libresolv.tbd */ = {isa = PBXFileReference; lastKnownFileType = sourcecode.text-based-dylib-definition; name = libresolv.tbd; path = usr/lib/libresolv.tbd; sourceTree = SDKROOT; };
		C0DEBEEF0000000000000107 /* %s.a */ = {isa = PBXFileReference; lastKnownFileType = archive.ar; name = "%s.a"; path = ../../../bin/%s.a; sourceTree = SOURCE_ROOT; };
/* End PBXFileReference section */

/* Begin PBXGroup section */
		C0DEBEEF0000000000000010 = {
			isa = PBXGroup;
			children = (
				C0DEBEEF0000000000000020 /* Products */,
				C0DEBEEF0000000000000045 /* Frameworks */,
				C0DEBEEF0000000000000030 /* main */,
			);
			sourceTree = "<group>";
		};
		C0DEBEEF0000000000000020 /* Products */ = {
			isa = PBXGroup;
			children = (
				C0DEBEEF0000000000000004 /* %s.app */,
			);
			name = Products;
			sourceTree = "<group>";
		};
		C0DEBEEF0000000000000030 /* main */ = {
			isa = PBXGroup;
			children = (
				C0DEBEEF0000000000000002 /* main.m */,
				C0DEBEEF0000000000000003 /* Info.plist */,
			);
			path = main;
			sourceTree = SOURCE_ROOT;
		};
		C0DEBEEF0000000000000045 /* Frameworks */ = {
			isa = PBXGroup;
			children = (
				C0DEBEEF0000000000000101 /* UIKit.framework */,
				C0DEBEEF0000000000000102 /* Foundation.framework */,
				C0DEBEEF0000000000000103 /* WebKit.framework */,
				C0DEBEEF0000000000000104 /* Security.framework */,
				C0DEBEEF0000000000000105 /* CoreFoundation.framework */,
				C0DEBEEF0000000000000106 /* libresolv.tbd */,
				C0DEBEEF0000000000000107 /* %s.a */,
			);
			name = Frameworks;
			sourceTree = "<group>";
		};
/* End PBXGroup section */

/* Begin PBXNativeTarget section */
		C0DEBEEF0000000000000040 /* %s */ = {
			isa = PBXNativeTarget;
			buildConfigurationList = C0DEBEEF0000000000000070 /* Build configuration list for PBXNativeTarget "%s" */;
			buildPhases = (
				C0DEBEEF0000000000000055 /* Prebuild: Wails Go Archive */,
				C0DEBEEF0000000000000050 /* Sources */,
				C0DEBEEF0000000000000056 /* Frameworks */,
			);
			buildRules = (
			);
			dependencies = (
			);
			name = "%s";
			productName = "%s";
			productReference = C0DEBEEF0000000000000004 /* %s.app */;
			productType = "com.apple.product-type.application";
		};
/* End PBXNativeTarget section */

/* Begin PBXProject section */
		C0DEBEEF0000000000000060 /* Project object */ = {
			isa = PBXProject;
			attributes = {
				LastUpgradeCheck = 1500;
				ORGANIZATIONNAME = "%s";
				TargetAttributes = {
					C0DEBEEF0000000000000040 = {
						CreatedOnToolsVersion = 15.0;
					};
				};
			};
			buildConfigurationList = C0DEBEEF0000000000000080 /* Build configuration list for PBXProject "main" */;
			compatibilityVersion = "Xcode 15.0";
			developmentRegion = en;
			hasScannedForEncodings = 0;
			knownRegions = (
				en,
			);
			mainGroup = C0DEBEEF0000000000000010;
			productRefGroup = C0DEBEEF0000000000000020 /* Products */;
			projectDirPath = "";
			projectRoot = "";
			targets = (
				C0DEBEEF0000000000000040 /* %s */,
			);
		};
/* End PBXProject section */

/* Begin PBXFrameworksBuildPhase section */
		C0DEBEEF0000000000000056 /* Frameworks */ = {
			isa = PBXFrameworksBuildPhase;
			buildActionMask = 2147483647;
			files = (
				C0DEBEEF00000000000000F7 /* %s.a in Frameworks */,
				C0DEBEEF00000000000000F1 /* UIKit.framework in Frameworks */,
				C0DEBEEF00000000000000F2 /* Foundation.framework in Frameworks */,
				C0DEBEEF00000000000000F3 /* WebKit.framework in Frameworks */,
				C0DEBEEF00000000000000F4 /* Security.framework in Frameworks */,
				C0DEBEEF00000000000000F5 /* CoreFoundation.framework in Frameworks */,
				C0DEBEEF00000000000000F6 /* libresolv.tbd in Frameworks */,
			);
			runOnlyForDeploymentPostprocessing = 0;
		};
/* End PBXFrameworksBuildPhase section */

/* Begin PBXShellScriptBuildPhase section */
		C0DEBEEF0000000000000055 /* Prebuild: Wails Go Archive */ = {
			isa = PBXShellScriptBuildPhase;
			buildActionMask = 2147483647;
			files = (
			);
			inputFileListPaths = (
			);
			inputPaths = (
			);
			name = "Prebuild: Wails Go Archive";
			outputFileListPaths = (
			);
			outputPaths = (
			);
			runOnlyForDeploymentPostprocessing = 0;
			shellPath = /bin/sh;
			shellScript = "set -e\nAPP_ROOT=\"${PROJECT_DIR}/../../..\"\nSDK_PATH=$(xcrun --sdk iphonesimulator --show-sdk-path)\nexport GOOS=ios\nexport GOARCH=arm64\nexport CGO_ENABLED=1\nexport CGO_CFLAGS=\"-isysroot ${SDK_PATH} -target arm64-apple-ios15.0-simulator -mios-simulator-version-min=15.0\"\nexport CGO_LDFLAGS=\"-isysroot ${SDK_PATH} -target arm64-apple-ios15.0-simulator\"\ncd \"${APP_ROOT}\"\n# Ensure overlay exists\nif [ ! -f build/ios/xcode/overlay.json ]; then\n  wails3 ios overlay:gen -out build/ios/xcode/overlay.json -config build/config.yml || true\nfi\n# Build Go c-archive if missing or older than sources\nif [ ! -f bin/%s.a ]; then\n  echo \"Building Go c-archive...\"\n  go build -buildmode=c-archive -overlay build/ios/xcode/overlay.json -o bin/%s.a\nfi\n";
		};
/* End PBXShellScriptBuildPhase section */

/* Begin PBXSourcesBuildPhase section */
		C0DEBEEF0000000000000050 /* Sources */ = {
			isa = PBXSourcesBuildPhase;
			buildActionMask = 2147483647;
			files = (
				C0DEBEEF0000000000000001 /* main.m in Sources */,
			);
			runOnlyForDeploymentPostprocessing = 0;
		};
/* End PBXSourcesBuildPhase section */

/* Begin XCBuildConfiguration section */
		C0DEBEEF0000000000000090 /* Debug */ = {
			isa = XCBuildConfiguration;
			buildSettings = {
				INFOPLIST_FILE = main/Info.plist;
				IPHONEOS_DEPLOYMENT_TARGET = 15.0;
				PRODUCT_BUNDLE_IDENTIFIER = "%s";
				PRODUCT_NAME = "%s";
				CODE_SIGNING_ALLOWED = NO;
				SDKROOT = iphonesimulator;
			};
			name = Debug;
		};
		C0DEBEEF00000000000000A0 /* Release */ = {
			isa = XCBuildConfiguration;
			buildSettings = {
				INFOPLIST_FILE = main/Info.plist;
				IPHONEOS_DEPLOYMENT_TARGET = 15.0;
				PRODUCT_BUNDLE_IDENTIFIER = "%s";
				PRODUCT_NAME = "%s";
				CODE_SIGNING_ALLOWED = NO;
				SDKROOT = iphonesimulator;
			};
			name = Release;
		};
/* End XCBuildConfiguration section */

/* Begin XCConfigurationList section */
		C0DEBEEF0000000000000070 /* Build configuration list for PBXNativeTarget "%s" */ = {
			isa = XCConfigurationList;
			buildConfigurations = (
				C0DEBEEF0000000000000090 /* Debug */,
				C0DEBEEF00000000000000A0 /* Release */,
			);
			defaultConfigurationIsVisible = 0;
			defaultConfigurationName = Debug;
		};
		C0DEBEEF0000000000000080 /* Build configuration list for PBXProject "main" */ = {
			isa = XCConfigurationList;
			buildConfigurations = (
				C0DEBEEF0000000000000090 /* Debug */,
				C0DEBEEF00000000000000A0 /* Release */,
			);
			defaultConfigurationIsVisible = 0;
			defaultConfigurationName = Debug;
		};
/* End XCConfigurationList section */
	};
	rootObject = C0DEBEEF0000000000000060 /* Project object */;
}
`,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		appName,
		companyName,
		appName,
		appName,
		appName,
		appName,
		productID,
		appName,
		productID,
		appName,
		appName,
	)
}

// renderReleaseWorkflow 根据项目元数据渲染对应派生文件内容，返回完整文本供同步流程写入。
func renderReleaseWorkflow(meta metadata, wailsVersion string) string {
	return fmt.Sprintf(`name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  windows-amd64:
    name: Windows amd64
    runs-on: windows-latest

    env:
      GO_VERSION: "1.25.x"
      NODE_VERSION: "22"
      WAILS_VERSION: %s
      APP_NAME: %s
      ARCH: amd64
      GO_DESKTOP_LICENSE_MODE: required
      GO_DESKTOP_LICENSE_PUBLIC_KEY: ${{ vars.GO_DESKTOP_LICENSE_PUBLIC_KEY }}

    steps:
      - name: 检出代码
        uses: actions/checkout@v4

      - name: 解析版本
        id: version
        shell: pwsh
        run: |
          $tag = "${{ github.ref_name }}"
          if ($tag -notmatch '^v\d+(\.\d+){0,2}$') {
            throw "发布标签必须符合 vX、vX.Y 或 vX.Y.Z，当前是 $tag"
          }
          $parts = $tag.TrimStart('v').Split('.')
          while ($parts.Count -lt 3) {
            $parts += '0'
          }
          $version = $parts -join '.'
          "version=$version" >> $env:GITHUB_OUTPUT
          "APP_VERSION=$version" >> $env:GITHUB_ENV
          "APP_VERSION_MODE=github" >> $env:GITHUB_ENV

      - name: 校验授权配置
        shell: pwsh
        run: |
          if ([string]::IsNullOrWhiteSpace($env:GO_DESKTOP_LICENSE_PUBLIC_KEY)) {
            throw "GO_DESKTOP_LICENSE_PUBLIC_KEY 未配置，禁止发布授权版"
          }

      - name: 设置 Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: 设置 Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: frontend/package-lock.json

      - name: 安装 NSIS
        shell: pwsh
        run: |
          choco install nsis -y --no-progress
          $nsisPath = "${env:ProgramFiles(x86)}\NSIS"
          if (!(Test-Path (Join-Path $nsisPath "makensis.exe"))) {
            throw "找不到 makensis.exe：$nsisPath"
          }
          $nsisPath >> $env:GITHUB_PATH
          $env:PATH = "$nsisPath;$env:PATH"
          makensis /VERSION

      - name: 安装 Wails3
        shell: pwsh
        run: go install github.com/wailsapp/wails/v3/cmd/wails3@${{ env.WAILS_VERSION }}

      - name: 同步项目元数据
        shell: pwsh
        run: go run ./scripts/sync_project_metadata.go -sync

      - name: 安装前端依赖
        shell: pwsh
        run: npm ci
        working-directory: frontend

      - name: 生成 Wails 绑定
        shell: pwsh
        run: wails3 generate bindings -f '-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui -X main.appVersion=${{ steps.version.outputs.version }}"' -clean=false -ts

      - name: 前端类型检查
        shell: pwsh
        run: npx vue-tsc --noEmit
        working-directory: frontend

      - name: 构建前端
        shell: pwsh
        run: npm run build
        working-directory: frontend

      - name: 运行 Go 测试
        shell: pwsh
        run: |
          go test ./...
          Push-Location tests
          go test ./...
          Pop-Location

      - name: 打包 Windows 安装器
        shell: pwsh
        run: wails3 task package:github

      - name: 生成 SHA256
        id: assets
        shell: pwsh
        run: |
          $asset = "bin/${{ env.APP_NAME }}-v${{ steps.version.outputs.version }}-windows-${{ env.ARCH }}.exe"
          if (!(Test-Path $asset)) {
            throw "找不到安装器资产：$asset"
          }
          $sha = (Get-FileHash -Algorithm SHA256 -LiteralPath $asset).Hash.ToLowerInvariant()
          "$sha  $(Split-Path -Leaf $asset)" | Set-Content -NoNewline -Encoding ascii "$asset.sha256"
          "installer=$asset" >> $env:GITHUB_OUTPUT
          "checksum=$asset.sha256" >> $env:GITHUB_OUTPUT

      - name: 创建 GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: v${{ steps.version.outputs.version }}
          name: %s v${{ steps.version.outputs.version }}
          draft: false
          prerelease: false
          generate_release_notes: true
          files: |
            ${{ steps.assets.outputs.installer }}
            ${{ steps.assets.outputs.checksum }}
`,
		yamlString(wailsVersion),
		meta.AppName,
		meta.AppName,
	)
}

// mustWrite 执行必须成功的辅助步骤，失败时直接终止当前命令以避免继续写入错误结果。
func mustWrite(path string, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		exitf("创建目录失败 %s：%v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		exitf("写入文件失败 %s：%v", path, err)
	}
}

// goString 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func goString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		exitf("编码 Go 字符串失败：%v", err)
	}
	return string(encoded)
}

// nsisString 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func nsisString(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `$\"`) + `"`
}

// yamlString 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func yamlString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		exitf("编码 YAML 字符串失败：%v", err)
	}
	return string(encoded)
}

// shellString 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func shellString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		exitf("编码 Shell 字符串失败：%v", err)
	}
	return string(encoded)
}

// groovyString 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func groovyString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		exitf("编码 Gradle 字符串失败：%v", err)
	}
	return string(encoded)
}

// goSource 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func goSource(source string) string {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		exitf("格式化 Go 源码失败：%v", err)
	}
	return string(formatted)
}

// pbxString 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func pbxString(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`*/`, `* /`,
		`/*`, `/ *`,
	)
	return replacer.Replace(value)
}

// htmlText 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func htmlText(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&#34;",
		"'", "&#39;",
	)
	return replacer.Replace(value)
}

// xmlAttr 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func xmlAttr(value string) string {
	return htmlText(value)
}

// xmlText 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func xmlText(value string) string {
	return htmlText(value)
}

// fourPartVersion 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func fourPartVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0.0.0.0"
	}
	parts := strings.Split(value, ".")
	for len(parts) < 4 {
		parts = append(parts, "0")
	}
	if len(parts) > 4 {
		parts = parts[:4]
	}
	return strings.Join(parts, ".")
}

// windowsEnvInstallDir 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func windowsEnvInstallDir(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "$LOCALAPPDATA", "%LOCALAPPDATA%")
	value = strings.ReplaceAll(value, "$APPDATA", "%APPDATA%")
	return value
}

// linuxText 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func linuxText(value string) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ")
	return replacer.Replace(strings.TrimSpace(value))
}

// exitf 封装读取 project.metadata.json 并生成各平台派生配置、安装器配置、前端项目元数据和发布工作流 中的一段独立逻辑，调用方通过它复用同一业务规则。
func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
