// 文件职责：验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package build_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// projectMetadataFixture 定义 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 使用的数据实体，字段会直接参与校验、渲染、持久化或平台适配。
type projectMetadataFixture struct {
	AppName          string   `json:"appName"`        // AppName 保存 appName 对应的数据，供当前实体的调用方读取或持久化。
	DefaultVersion   string   `json:"defaultVersion"` // DefaultVersion 保存 defaultVersion 对应的数据，供当前实体的调用方读取或持久化。
	ModulePath       string   `json:"modulePath"`     // ModulePath 保存 modulePath 对应的数据，供当前实体的调用方读取或持久化。
	RepositoryURL    string   `json:"repositoryUrl"`  // RepositoryURL 保存 repositoryUrl 对应的数据，供当前实体的调用方读取或持久化。
	SettingsDefaults struct { // SettingsDefaults 保存 SettingsDefaults 对应的数据，供当前实体的调用方读取或持久化。
		UpdateCheckIntervalHours int  `json:"updateCheckIntervalHours"` // UpdateCheckIntervalHours 保存 updateCheckIntervalHours 对应的数据，供当前实体的调用方读取或持久化。
		MinimizeToTray           bool `json:"minimizeToTray"`           // MinimizeToTray 保存 minimizeToTray 对应的数据，供当前实体的调用方读取或持久化。
		LogRetentionDays         int  `json:"logRetentionDays"`         // LogRetentionDays 保存 logRetentionDays 对应的数据，供当前实体的调用方读取或持久化。
		AutoLaunch               bool `json:"autoLaunch"`               // AutoLaunch 保存 autoLaunch 对应的数据，供当前实体的调用方读取或持久化。
		CreateDesktopShortcut    bool `json:"createDesktopShortcut"`    // CreateDesktopShortcut 保存 createDesktopShortcut 对应的数据，供当前实体的调用方读取或持久化。
		LaunchHiddenToTray       bool `json:"launchHiddenToTray"`       // LaunchHiddenToTray 保存 launchHiddenToTray 对应的数据，供当前实体的调用方读取或持久化。
	} `json:"settingsDefaults"`
	Windows struct { // Windows 保存 Windows 对应的数据，供当前实体的调用方读取或持久化。
		SingleInstanceID  string `json:"singleInstanceId"`  // SingleInstanceID 保存 singleInstanceId 对应的数据，供当前实体的调用方读取或持久化。
		ProductIdentifier string `json:"productIdentifier"` // ProductIdentifier 保存 productIdentifier 对应的数据，供当前实体的调用方读取或持久化。
		WindowClass       string `json:"windowClass"`       // WindowClass 保存 windowClass 对应的数据，供当前实体的调用方读取或持久化。
	} `json:"windows"`
}

// TestScriptsDirectoryContainsNoGoTests 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestScriptsDirectoryContainsNoGoTests(t *testing.T) {
	var testFiles []string
	scriptsRoot := rootPath("scripts")
	if err := filepath.WalkDir(scriptsRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			testFiles = append(testFiles, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
	if len(testFiles) > 0 {
		t.Fatalf("scripts/ must not contain Go tests; move them to tests/: %v", testFiles)
	}
}

// TestSyncProjectMetadataPrintsCurrentDefaults 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestSyncProjectMetadataPrintsCurrentDefaults(t *testing.T) {
	for key, want := range map[string]string{
		"companyName":                    "chencn",
		"settings.updateInterval":        "3",
		"settings.minimizeToTray":        "true",
		"settings.autoLaunch":            "false",
		"settings.createDesktopShortcut": "true",
		"settings.launchHiddenToTray":    "false",
		"windows.windowClass":            "com.github.chencn.go-desktop-window",
	} {
		if got := runSyncProjectMetadata(t, "-print", key); got != want {
			t.Fatalf("metadata print %s = %q, want %q", key, got, want)
		}
	}
}

// TestGeneratedProjectMetadataFilesUseCurrentDefaults 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestGeneratedProjectMetadataFilesUseCurrentDefaults(t *testing.T) {
	meta := readProjectMetadataFixture(t)

	for path, required := range map[string][]string{
		"Taskfile.yml": {
			`APP_NAME: "go-desktop"`,
			`APP_VERSION: '{{env "APP_VERSION" | default "0.0.1"}}'`,
		},
		"frontend/src/shared/project.ts": {
			`"singleInstanceId": "com.github.chencn.go-desktop"`,
			`"productIdentifier": "com.github.chencn.godesktop"`,
			`"updateCheckIntervalHours": 3`,
			`"autoLaunch": false`,
			`"createDesktopShortcut": true`,
			`"launchHiddenToTray": false`,
			`minimizeToTray: projectMetadata.settingsDefaults.minimizeToTray`,
			`autoLaunch: projectMetadata.settingsDefaults.autoLaunch`,
			`createDesktopShortcut: projectMetadata.settingsDefaults.createDesktopShortcut`,
			`launchHiddenToTray: projectMetadata.settingsDefaults.launchHiddenToTray`,
		},
		"build/windows/nsis/project_metadata.nsh": {
			`!define APP_WINDOW_CLASS    "com.github.chencn.go-desktop-window"`,
			`!define APP_WINDOW_TITLE    "go-desktop"`,
		},
		"build/android/app/build.gradle": {
			`applicationId "com.github.chencn.godesktop"`,
		},
		"build/ios/project.pbxproj": {
			`PRODUCT_BUNDLE_IDENTIFIER = "com.github.chencn.godesktop";`,
		},
		"build/windows/msix/app_manifest.xml": {
			`Name="com.github.chencn.godesktop"`,
			`<Application Id="com.github.chencn.godesktop"`,
		},
	} {
		source := readRootFile(t, path)
		for _, want := range required {
			if !strings.Contains(source, want) {
				t.Fatalf("%s should contain generated metadata %q", path, want)
			}
		}
	}

	projectSource := readRootFile(t, "internal/desktopapp/metadata/metadata.go")
	for _, fields := range [][]string{
		{"AppName", `"go-desktop"`},
		{"DefaultUpdateCheckIntervalHours", "3"},
		{"DefaultMinimizeToTray", "true"},
		{"DefaultAutoLaunch", "false"},
		{"DefaultCreateDesktopShortcut", "true"},
		{"DefaultLaunchHiddenToTray", "false"},
		{"WindowsSingleInstanceID", `"com.github.chencn.go-desktop"`},
		{"WindowsProductID", `"com.github.chencn.godesktop"`},
		{"WindowsWindowClass", `"com.github.chencn.go-desktop-window"`},
	} {
		requireLineContaining(t, "internal/desktopapp/metadata/metadata.go", projectSource, fields...)
	}

	if meta.AppName != "go-desktop" || meta.DefaultVersion != "0.0.1" || meta.ModulePath != "github.com/chencn/go-desktop" {
		t.Fatalf("unexpected project metadata identity: %#v", meta)
	}
	if meta.SettingsDefaults.UpdateCheckIntervalHours != 3 ||
		!meta.SettingsDefaults.MinimizeToTray ||
		meta.SettingsDefaults.LogRetentionDays != 30 ||
		meta.SettingsDefaults.AutoLaunch ||
		!meta.SettingsDefaults.CreateDesktopShortcut ||
		meta.SettingsDefaults.LaunchHiddenToTray {
		t.Fatalf("unexpected project settings defaults: %#v", meta.SettingsDefaults)
	}
	if meta.Windows.SingleInstanceID != "com.github.chencn.go-desktop" ||
		meta.Windows.ProductIdentifier != "com.github.chencn.godesktop" ||
		meta.Windows.WindowClass != "com.github.chencn.go-desktop-window" ||
		meta.RepositoryURL != "https://github.com/chencn/go-desktop" {
		t.Fatalf("unexpected generated project integration metadata: %#v", meta)
	}
}

// runSyncProjectMetadata 封装 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 中的一段独立逻辑，调用方通过它复用同一业务规则。
func runSyncProjectMetadata(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "./scripts/sync_project_metadata.go"}, args...)...)
	cmd.Dir = rootPath()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("sync_project_metadata failed: %v\n%s", err, stderr.String())
	}
	return strings.TrimSpace(string(output))
}

// readProjectMetadataFixture 读取、解析或归一化 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 需要的数据，并把结果返回给调用方。
func readProjectMetadataFixture(t *testing.T) projectMetadataFixture {
	t.Helper()
	var meta projectMetadataFixture
	data, err := os.ReadFile(rootPath("project.metadata.json"))
	if err != nil {
		t.Fatalf("read project.metadata.json: %v", err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parse project.metadata.json: %v", err)
	}
	return meta
}

// requireLineContaining 封装 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 中的一段独立逻辑，调用方通过它复用同一业务规则。
func requireLineContaining(t *testing.T, path string, source string, fields ...string) {
	t.Helper()
	for _, line := range strings.Split(source, "\n") {
		matches := true
		for _, field := range fields {
			if !strings.Contains(line, field) {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}
	t.Fatalf("%s should contain one line with fields %v", path, fields)
}

// rootPath 封装 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 中的一段独立逻辑，调用方通过它复用同一业务规则。
func rootPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}
