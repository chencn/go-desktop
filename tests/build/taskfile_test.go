// 文件职责：验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package build_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommonTaskfileRunsNpmThroughWindowsEnvWrapper 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
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

// TestTaskfilesProvideWindowsDriveEnvFallbacks 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
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

// TestRootTestTaskRunsGoAndFrontendTests 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
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

// TestDevFrontendStartsWithoutReinstallingDependencies 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
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

// TestWailsDevStartsFrontendAfterBuild 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestWailsDevStartsFrontendAfterBuild(t *testing.T) {
	source := readRootFile(t, "build", "config.yml")
	buildIndex := strings.Index(source, "cmd: wails3 build DEV=true")
	frontendIndex := strings.Index(source, "cmd: wails3 task common:dev:frontend")
	runIndex := strings.Index(source, "cmd: wails3 task run")
	if buildIndex < 0 || frontendIndex < 0 || runIndex < 0 {
		t.Fatalf("build/config.yml 缺少 wails dev execute 链路：\n%s", source)
	}
	if !(buildIndex < frontendIndex && frontendIndex < runIndex) {
		t.Fatalf("wails dev execute 顺序必须是 build -> dev:frontend -> run，当前顺序错误")
	}
}

// TestEnvrunProvidesWindowsProcessEnvFallbacks 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
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

// TestTrayMenuOnlyShowsDisplayAndExit 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
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

// TestCloseToTrayDoesNotHijackMinimiseButton 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestCloseToTrayDoesNotHijackMinimiseButton(t *testing.T) {
	source := readRootFile(t, "main.go")
	if !strings.Contains(source, "WindowClosing") || !strings.Contains(source, "ShouldHideOnClose()") {
		t.Fatalf("main.go 应只在关闭窗口时判断是否隐藏到托盘")
	}
	if strings.Contains(source, "WindowMinimise") || strings.Contains(source, "窗口最小化到托盘") {
		t.Fatalf("点击最小化应保留在任务栏，不应注册最小化到托盘逻辑")
	}
}

// readRootFile 读取、解析或归一化 验证 taskfile_test.go 覆盖的生产行为、结构约束或构建脚本约束 需要的数据，并把结果返回给调用方。
func readRootFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
	if err != nil {
		t.Fatalf("读取仓库文件失败：%v", err)
	}
	return string(data)
}
