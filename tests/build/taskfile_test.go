// 文件职责：验证 Taskfile、envrun、发布工作流和桌面入口的构建/启动结构约束。

package build_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommonTaskfileRunsNpmThroughWindowsEnvWrapper 验证 npm 命令经过 envrun，避免 Windows/Codex 环境变量缺失。
func TestCommonTaskfileRunsNpmThroughWindowsEnvWrapper(t *testing.T) {
	source := readRootFile(t, "build", "Taskfile.yml")

	for _, want := range []string{
		"go run ../scripts/envrun npm version",
		"go run ../scripts/envrun npm install",
		"go run ../scripts/envrun npm run {{.BUILD_COMMAND}} -q",
		"go run ../scripts/envrun npm run dev -- --port {{.VITE_PORT}} --strictPort",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("build/Taskfile.yml should run npm through envrun, missing %q", want)
		}
	}
}

// TestTaskfilesProvideWindowsDriveEnvFallbacks 验证 Task 模板为 Windows 基础目录提供兜底值。
func TestTaskfilesProvideWindowsDriveEnvFallbacks(t *testing.T) {
	files := map[string]string{
		"Taskfile.yml":               readRootFile(t, "Taskfile.yml"),
		"build/Taskfile.yml":         readRootFile(t, "build", "Taskfile.yml"),
		"build/windows/Taskfile.yml": readRootFile(t, "build", "windows", "Taskfile.yml"),
	}

	for path, source := range files {
		for _, want := range []string{
			`SystemDrive: '{{if eq OS "windows"}}{{env "SystemDrive" | default "C:"}}{{end}}'`,
			`ProgramData: '{{if eq OS "windows"}}{{env "ProgramData" | default (printf "%s\\ProgramData" (env "SystemDrive" | default "C:"))}}{{end}}'`,
			`frontend/%SystemDrive%`,
		} {
			if !strings.Contains(source, want) {
				t.Fatalf("%s 缺少 Windows 运行时环境兜底或说明 %q", path, want)
			}
		}
	}
}

// TestRootTestTaskRunsGoAndFrontendTests 验证顶层 test 任务覆盖独立 Go tests 模块和前端 Vitest。
func TestRootTestTaskRunsGoAndFrontendTests(t *testing.T) {
	source := readRootFile(t, "Taskfile.yml")

	for _, want := range []string{
		"cd tests && go test ./...",
		"cd frontend && go run ../scripts/envrun npm test",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("Taskfile.yml test task should run Go and frontend tests, missing %q", want)
		}
	}
}

// TestRootDevCommandUsesEnvrun 验证本地开发启动也读取仓库 .env。
func TestRootDevCommandUsesEnvrun(t *testing.T) {
	source := readRootFile(t, "Taskfile.yml")

	for _, want := range []string{
		"go run ./scripts/envrun wails3 dev -config ./build/config.yml -port {{.VITE_PORT}}",
		"go run ./scripts/envrun wails3 task {{OS}}:build",
		"go run ./scripts/envrun wails3 task windows:package",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("Taskfile.yml build/dev/package 任务必须先经过 envrun 再进入 wails task，确保 .env 在 Task 模板展开前生效：缺少 %q", want)
		}
	}
}

// TestDevFrontendStartsWithoutReinstallingDependencies 验证 Wails dev 启动 Vite 时不重复安装依赖，避免拖慢启动窗口。
func TestDevFrontendStartsWithoutReinstallingDependencies(t *testing.T) {
	source := readRootFile(t, "build", "Taskfile.yml")
	start := strings.Index(source, "  dev:frontend:")
	if start < 0 {
		t.Fatalf("build/Taskfile.yml 缺少 common:dev:frontend 任务")
	}
	devFrontend := source[start:]
	if nextTask := strings.Index(devFrontend[1:], "\n  update:build-assets:"); nextTask >= 0 {
		devFrontend = devFrontend[:nextTask+1]
	}

	if strings.Contains(devFrontend, "install:frontend:deps") {
		t.Fatal("common:dev:frontend must not run npm install; wails dev builds first, then starts Vite before the app wait window expires")
	}
}

// TestWailsDevStartsFrontendAfterBuild 验证 Wails dev 执行顺序是先构建、再启动前端、最后运行桌面壳。
func TestWailsDevStartsFrontendAfterBuild(t *testing.T) {
	source := readRootFile(t, "build", "config.yml")
	buildIndex := strings.Index(source, "cmd: go run ./scripts/envrun wails3 task windows:build DEV=true")
	frontendIndex := strings.Index(source, "cmd: wails3 task common:dev:frontend")
	runIndex := strings.Index(source, "cmd: wails3 task run")
	if buildIndex < 0 || frontendIndex < 0 || runIndex < 0 {
		t.Fatalf("build/config.yml 缺少 wails dev execute 链路：\n%s", source)
	}
	if !(buildIndex < frontendIndex && frontendIndex < runIndex) {
		t.Fatalf("wails dev execute 顺序必须是 build -> dev:frontend -> run，当前顺序错误")
	}
}

// TestEnvrunProvidesWindowsProcessEnvFallbacks 验证 envrun 补齐 Node/Go 在 Windows 下依赖的关键进程环境变量。
func TestEnvrunProvidesWindowsProcessEnvFallbacks(t *testing.T) {
	source := readRootFile(t, "scripts", "envrun", "main.go")

	for _, want := range []string{
		`exec.LookPath(command + ".cmd")`,
		`SystemRoot`,
		`SystemDrive`,
		`ProgramData`,
		`WINDIR`,
		`ComSpec`,
		`LOCALAPPDATA`,
		`APPDATA`,
		`GOCACHE`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("scripts/envrun/main.go should provide Windows env fallback %q", want)
		}
	}
}

// TestEnvrunLoadsDotEnvWhenPresent 验证 envrun 会读取仓库 .env，且进程环境变量优先于 .env。
func TestEnvrunLoadsDotEnvWhenPresent(t *testing.T) {
	source := readRootFile(t, "scripts", "envrun", "main.go")

	for _, want := range []string{
		"mergeDotEnv",
		"findDotEnv",
		".env",
		`strings.Cut(line, "=")`,
		"进程环境变量优先于 .env",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("scripts/envrun/main.go 必须读取可选 .env，并保持进程环境变量优先级：缺少 %q", want)
		}
	}
}

// TestWindowsBuildCommandsUseEnvrunForGoBuild 验证 Windows Go 构建命令会经过 envrun，从而读取本地 .env。
func TestWindowsBuildCommandsUseEnvrunForGoBuild(t *testing.T) {
	source := readRootFile(t, "build", "windows", "Taskfile.yml")

	for _, want := range []string{
		"go run ./scripts/envrun go build",
		"go run ../scripts/envrun go run ./windows/scripts/write_info_version.go",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("build/windows/Taskfile.yml 的 Windows 构建命令必须经过 envrun：缺少 %q", want)
		}
	}
}

// TestDotEnvExampleDocumentsLicenseVariables 验证仓库提供可提交的 .env 示例。
func TestDotEnvExampleDocumentsLicenseVariables(t *testing.T) {
	source := readRootFile(t, ".env.example")

	for _, want := range []string{
		"# 不配置 GO_DESKTOP_LICENSE_MODE 时授权关闭",
		"GO_DESKTOP_LICENSE_MODE=",
		"GO_DESKTOP_LICENSE_PUBLIC_KEY=",
		"GO_DESKTOP_LICENSE_MODE=required",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf(".env.example 必须说明授权环境变量和默认关闭规则：缺少 %q", want)
		}
	}
}

// TestWindowsBuildInjectsLicenseLdflags 验证授权构建变量会注入到 Windows 二进制。
func TestWindowsBuildInjectsLicenseLdflags(t *testing.T) {
	source := readRootFile(t, "build", "windows", "Taskfile.yml")

	for _, want := range []string{
		`-X main.licenseMode={{env "GO_DESKTOP_LICENSE_MODE"}}`,
		`-X main.licensePublicKey={{env "GO_DESKTOP_LICENSE_PUBLIC_KEY"}}`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("Windows 构建参数必须注入授权构建变量：缺少 %q", want)
		}
	}
}

// TestGitHubReleaseSetsLicenseEnvironment 验证官方 GitHub Release 打包会启用授权模式、注入公钥并拒绝空公钥发布。
func TestGitHubReleaseSetsLicenseEnvironment(t *testing.T) {
	source := readRootFile(t, ".github", "workflows", "release.yml")

	for _, want := range []string{
		"GO_DESKTOP_LICENSE_MODE: required",
		"GO_DESKTOP_LICENSE_PUBLIC_KEY: ${{ vars.GO_DESKTOP_LICENSE_PUBLIC_KEY }}",
		"- name: 校验授权配置",
		"[string]::IsNullOrWhiteSpace($env:GO_DESKTOP_LICENSE_PUBLIC_KEY)",
		"GO_DESKTOP_LICENSE_PUBLIC_KEY 未配置，禁止发布授权版",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("release workflow 必须设置官方授权发行版环境变量：缺少 %q", want)
		}
	}
}

// TestTrayMenuOnlyShowsDisplayAndExit 验证托盘菜单只保留窗口显示和退出，不承载更新等业务入口。
func TestTrayMenuOnlyShowsDisplayAndExit(t *testing.T) {
	source := readRootFile(t, "main.go")
	start := strings.Index(source, "trayMenu := wailsApp.NewMenu()")
	end := strings.Index(source, "systemTray.SetMenu(trayMenu)")
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("main.go 缺少可检查的托盘菜单结构")
	}
	trayMenu := source[start:end]

	for _, want := range []string{
		`trayMenu.Add("显示")`,
		`trayMenu.Add("退出")`,
	} {
		if !strings.Contains(trayMenu, want) {
			t.Fatalf("托盘菜单必须只保留显示和退出，缺少 %q", want)
		}
	}
	for _, forbidden := range []string{
		"隐藏",
		"检查更新",
		"CheckUpdate",
	} {
		if strings.Contains(trayMenu, forbidden) {
			t.Fatalf("托盘菜单只允许显示和退出，不应包含 %q", forbidden)
		}
	}
}

// TestCloseToTrayDoesNotHijackMinimiseButton 验证关闭到托盘只挂在关闭事件，不劫持最小化按钮。
func TestCloseToTrayDoesNotHijackMinimiseButton(t *testing.T) {
	source := readRootFile(t, "main.go")
	if !strings.Contains(source, "WindowClosing") || !strings.Contains(source, "ShouldHideOnClose()") {
		t.Fatalf("main.go 应只在关闭窗口时判断是否隐藏到托盘")
	}
	if strings.Contains(source, "WindowMinimise") || strings.Contains(source, "窗口最小化到托盘") {
		t.Fatalf("点击最小化应保留在任务栏，不应注册最小化到托盘逻辑")
	}
}

// TestSplashWindowShowsAfterNavigationCompleted 验证原生 splash 先隐藏创建，避免 WebView2 导航完成前露出白底窗口。
func TestSplashWindowShowsAfterNavigationCompleted(t *testing.T) {
	source := readRootFile(t, "main.go")
	start := strings.Index(source, `Name:             "splash"`)
	end := strings.Index(source, `crashReporter.Phase("创建主窗口")`)
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("main.go 缺少可检查的 splash 创建结构")
	}
	splashBlock := source[start:end]
	for _, want := range []string{
		"Hidden:           true",
		"splashWindow.OnWindowEvent(events.Windows.WebViewNavigationCompleted",
		"splashWindow.Show()",
	} {
		if !strings.Contains(splashBlock, want) {
			t.Fatalf("splash 必须先隐藏创建并在 WebView 导航完成后显示，缺少 %q", want)
		}
	}
}

func readRootFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
	if err != nil {
		t.Fatalf("读取仓库文件失败：%v", err)
	}
	return string(data)
}
