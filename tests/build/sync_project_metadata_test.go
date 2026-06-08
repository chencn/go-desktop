// 文件职责：验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package build_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// projectMetadataFixture 定义 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 使用的数据实体，字段会直接参与校验、渲染、持久化或平台适配。
type projectMetadataFixture struct {
	AppName        string   `json:"appName"`        // AppName 保存 appName 对应的数据，供当前实体的调用方读取或持久化。
	DefaultVersion string   `json:"defaultVersion"` // DefaultVersion 保存 defaultVersion 对应的数据，供当前实体的调用方读取或持久化。
	ModulePath     string   `json:"modulePath"`     // ModulePath 保存 modulePath 对应的数据，供当前实体的调用方读取或持久化。
	RepositoryURL  string   `json:"repositoryUrl"`  // RepositoryURL 保存 repositoryUrl 对应的数据，供当前实体的调用方读取或持久化。
	Update         struct { // Update 保存 Update 对应的数据，供当前实体的调用方读取或持久化。
		DefaultSource     string `json:"defaultSource"`
		LocalBaseURL      string `json:"localBaseUrl"`
		LocalManifestPath string `json:"localManifestPath"`
	} `json:"update"`
	SettingsDefaults struct { // SettingsDefaults 保存 SettingsDefaults 对应的数据，供当前实体的调用方读取或持久化。
		GitHubProxyBase          string `json:"githubProxyBase"`          // GitHubProxyBase 保存 githubProxyBase 对应的数据，供当前实体的调用方读取或持久化。
		UpdateCheckIntervalHours int    `json:"updateCheckIntervalHours"` // UpdateCheckIntervalHours 保存 updateCheckIntervalHours 对应的数据，供当前实体的调用方读取或持久化。
		MinimizeToTray           bool   `json:"minimizeToTray"`           // MinimizeToTray 保存 minimizeToTray 对应的数据，供当前实体的调用方读取或持久化。
		LogRetentionDays         int    `json:"logRetentionDays"`         // LogRetentionDays 保存 logRetentionDays 对应的数据，供当前实体的调用方读取或持久化。
		AutoLaunch               bool   `json:"autoLaunch"`               // AutoLaunch 保存 autoLaunch 对应的数据，供当前实体的调用方读取或持久化。
		CreateDesktopShortcut    bool   `json:"createDesktopShortcut"`    // CreateDesktopShortcut 保存 createDesktopShortcut 对应的数据，供当前实体的调用方读取或持久化。
		LaunchHiddenToTray       bool   `json:"launchHiddenToTray"`       // LaunchHiddenToTray 保存 launchHiddenToTray 对应的数据，供当前实体的调用方读取或持久化。
	} `json:"settingsDefaults"`
	Windows struct { // Windows 保存 Windows 对应的数据，供当前实体的调用方读取或持久化。
		SingleInstanceID  string `json:"singleInstanceId"`  // SingleInstanceID 保存 singleInstanceId 对应的数据，供当前实体的调用方读取或持久化。
		ProductIdentifier string `json:"productIdentifier"` // ProductIdentifier 保存 productIdentifier 对应的数据，供当前实体的调用方读取或持久化。
		WindowClass       string `json:"windowClass"`       // WindowClass 保存 windowClass 对应的数据，供当前实体的调用方读取或持久化。
		UninstallKeyName  string `json:"uninstallKeyName"`  // UninstallKeyName 保存 uninstallKeyName 对应的数据，供当前实体的调用方读取或持久化。
	} `json:"windows"`
}

type localUpdateReleaseFixture struct {
	TagName    string                    `json:"tag_name"`
	Name       string                    `json:"name"`
	HTMLURL    string                    `json:"html_url"`
	Body       string                    `json:"body"`
	Draft      bool                      `json:"draft"`
	Prerelease bool                      `json:"prerelease"`
	Assets     []localUpdateAssetFixture `json:"assets"`
}

type localUpdateAssetFixture struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
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
		"update.defaultSource":           "github",
		"update.localBaseUrl":            "http://www.xqchen.shop/exe/go-desktop",
		"update.localManifestPath":       "releases/latest.json",
		"windows.windowClass":            "com.github.chencn.go-desktop-window",
		"windows.uninstallKeyName":       "com.github.chencn.go-desktop",
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
			`go run ./scripts/resolve_app_version.go -mode '{{env "APP_VERSION_MODE" | default "local"}}' -config build/config.yml`,
			`package:github:`,
			`- task: windows:package`,
			`go build -o ".tmp/local-release-stage.exe" ./scripts/stage_local_update.go`,
			`.tmp/local-release-stage.exe -version "{{.APP_VERSION}}"`,
			`Remove-Item -LiteralPath '{{.BIN_DIR}}/{{.APP_NAME}}.exe','{{.BIN_DIR}}/{{.APP_NAME}}-v{{.APP_VERSION}}-windows-amd64.exe','{{.BIN_DIR}}/local-release-stage.exe','.tmp/local-release-stage.exe'`,
		},
		"frontend/src/shared/project.ts": {
			`"defaultSource": "github"`,
			`"localBaseUrl": "http://www.xqchen.shop/exe/go-desktop"`,
			`updateSource: projectMetadata.update.defaultSource`,
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
			`!define UNINST_KEY_NAME     "com.github.chencn.go-desktop"`,
			`!define APP_WINDOW_CLASS    "com.github.chencn.go-desktop-window"`,
			`!define APP_WINDOW_TITLE    "go-desktop"`,
		},
		".github/workflows/release.yml": {
			`- "v*"`,
			`"APP_VERSION_MODE=github" >> $env:GITHUB_ENV`,
			`wails3 generate bindings -f '-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui -X main.appVersion=${{ steps.version.outputs.version }}"' -clean=false -ts`,
			`run: wails3 task package:github`,
			`tag_name: v${{ steps.version.outputs.version }}`,
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
		{"DefaultUpdateSource", `"github"`},
		{"LocalUpdateBaseURL", `"http://www.xqchen.shop/exe/go-desktop"`},
		{"LocalUpdateManifestPath", `"releases/latest.json"`},
		{"DefaultGitHubProxyBase", `"https://gh-proxy.com"`},
		{"DefaultUpdateCheckIntervalHours", "3"},
		{"DefaultMinimizeToTray", "true"},
		{"DefaultAutoLaunch", "false"},
		{"DefaultCreateDesktopShortcut", "true"},
		{"DefaultLaunchHiddenToTray", "false"},
		{"WindowsSingleInstanceID", `"com.github.chencn.go-desktop"`},
		{"WindowsProductID", `"com.github.chencn.godesktop"`},
		{"WindowsWindowClass", `"com.github.chencn.go-desktop-window"`},
		{"WindowsUninstallKeyName", `"com.github.chencn.go-desktop"`},
	} {
		requireLineContaining(t, "internal/desktopapp/metadata/metadata.go", projectSource, fields...)
	}

	if meta.AppName != "go-desktop" || meta.DefaultVersion != "1.0.0" || meta.ModulePath != "github.com/chencn/go-desktop" {
		t.Fatalf("unexpected project metadata identity: %#v", meta)
	}
	if meta.Update.DefaultSource != "github" ||
		meta.Update.LocalBaseURL != "http://www.xqchen.shop/exe/go-desktop" ||
		meta.Update.LocalManifestPath != "releases/latest.json" {
		t.Fatalf("unexpected update metadata: %#v", meta.Update)
	}
	if meta.SettingsDefaults.GitHubProxyBase != "https://gh-proxy.com" ||
		meta.SettingsDefaults.UpdateCheckIntervalHours != 3 ||
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
		meta.Windows.UninstallKeyName != "com.github.chencn.go-desktop" ||
		meta.RepositoryURL != "https://github.com/chencn/go-desktop" {
		t.Fatalf("unexpected generated project integration metadata: %#v", meta)
	}
}

func TestResolveAppVersionNormalizesAndSelectsLargestCandidate(t *testing.T) {
	cmd := exec.Command("go", "run", "./scripts/resolve_app_version.go", "-mode", "local", "-config", "build/config.yml", "-version", "1.0", "-tag", "v1.2")
	cmd.Dir = rootPath()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve_app_version failed: %v\n%s", err, stderr.String())
	}
	if got := strings.TrimSpace(string(output)); got != "1.2.0" {
		t.Fatalf("expected normalized largest version 1.2.0, got %q", got)
	}
}

func TestResolveAppVersionGithubModeIgnoresBuildConfig(t *testing.T) {
	cmd := exec.Command("go", "run", "./scripts/resolve_app_version.go", "-mode", "github", "-config", filepath.Join(t.TempDir(), "missing.yml"), "-tag", "v0.9")
	cmd.Dir = rootPath()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve_app_version github mode failed: %v\n%s", err, stderr.String())
	}
	if got := strings.TrimSpace(string(output)); got != "0.9.0" {
		t.Fatalf("expected github tag version 0.9.0, got %q", got)
	}
}

func TestReleaseWorkflowGeneratesBindingsBeforeFrontendTypeCheck(t *testing.T) {
	source := readRootFile(t, ".github/workflows/release.yml")
	requireInOrder(t, ".github/workflows/release.yml", source,
		`"APP_VERSION_MODE=github" >> $env:GITHUB_ENV`,
		`- name: 生成 Wails 绑定`,
		`wails3 generate bindings -f '-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui -X main.appVersion=${{ steps.version.outputs.version }}"' -clean=false -ts`,
		`- name: 前端类型检查`,
	)
}

func TestReleaseWorkflowPublishesNsisPathBeforePackaging(t *testing.T) {
	source := readRootFile(t, ".github/workflows/release.yml")
	requireInOrder(t, ".github/workflows/release.yml", source,
		`- name: 安装 NSIS`,
		`choco install nsis -y --no-progress`,
		`$nsisPath = "${env:ProgramFiles(x86)}\NSIS"`,
		`$nsisPath >> $env:GITHUB_PATH`,
		`makensis /VERSION`,
		`- name: 打包 Windows 安装器`,
	)
}

func TestCleanupWorkflowKeepsRecentReleasesAndWorkflowLogs(t *testing.T) {
	source := readRootFile(t, ".github/workflows/cleanup.yml")
	for _, forbidden := range []string{
		"schedule:",
		"deleteWorkflowRun({",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(".github/workflows/cleanup.yml should not contain %q", forbidden)
		}
	}
	for _, required := range []string{
		"name: Cleanup GitHub History",
		"workflow_dispatch:",
		"description: \"保留最近多少个 Release 和 workflow run 日志\"",
		"default: \"5\"",
		"actions: write",
		"contents: write",
		"github.rest.repos.listReleases",
		"github.rest.repos.deleteRelease",
		"github.rest.actions.listWorkflowRunsForRepo",
		"github.rest.actions.deleteWorkflowRunLogs",
		"const staleReleases = releases.slice(keepCount)",
		"const staleRuns = runs.slice(keepCount)",
		"if (run.id === currentRunId)",
		"deleteReleaseTagsInput === true || deleteReleaseTagsInput === 'true'",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf(".github/workflows/cleanup.yml should contain %q", required)
		}
	}
}

func TestStageLocalUpdateWritesGithubCompatibleStaticFiles(t *testing.T) {
	tempDir := t.TempDir()
	payload := []byte("fake windows installer")
	installerPath := filepath.Join(tempDir, "source.exe")
	if err := os.WriteFile(installerPath, payload, 0o644); err != nil {
		t.Fatalf("write installer fixture: %v", err)
	}
	metadataPath := filepath.Join(tempDir, "project.metadata.json")
	metadataJSON := `{
  "appName": "go-desktop",
  "update": {
    "localBaseUrl": "http://www.xqchen.shop/exe/go-desktop",
    "localManifestPath": "releases/latest.json"
  }
}`
	if err := os.WriteFile(metadataPath, []byte(metadataJSON), 0o644); err != nil {
		t.Fatalf("write metadata fixture: %v", err)
	}

	outDir := filepath.Join(tempDir, "go-desktop")
	helperPath := filepath.Join(tempDir, "local-release-stage.exe")
	buildCmd := exec.Command("go", "build", "-o", helperPath, "./scripts/stage_local_update.go")
	buildCmd.Dir = rootPath()
	var buildStderr bytes.Buffer
	buildCmd.Stderr = &buildStderr
	if output, err := buildCmd.Output(); err != nil {
		t.Fatalf("build stage_local_update failed: %v\nstdout: %s\nstderr: %s", err, output, buildStderr.String())
	}

	cmd := exec.Command(helperPath,
		"-metadata", metadataPath,
		"-version", "1",
		"-installer", installerPath,
		"-out", outDir,
		"-arch", "amd64",
	)
	cmd.Dir = rootPath()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if output, err := cmd.Output(); err != nil {
		t.Fatalf("stage_local_update failed: %v\nstdout: %s\nstderr: %s", err, output, stderr.String())
	}

	versionTag := "v1.0.0"
	assetName := "go-desktop-v1.0.0-windows-amd64.exe"
	shaName := assetName + ".sha256"
	releaseURL := "http://www.xqchen.shop/exe/go-desktop/releases/download/v1.0.0/"
	downloadDir := filepath.Join(outDir, "releases", "download", versionTag)

	requireFileContent(t, filepath.Join(downloadDir, assetName), string(payload))
	sum := sha256.Sum256(payload)
	expectedSha := hex.EncodeToString(sum[:])
	requireFileContent(t, filepath.Join(downloadDir, shaName), expectedSha+"  "+assetName)

	data, err := os.ReadFile(filepath.Join(outDir, "releases", "latest.json"))
	if err != nil {
		t.Fatalf("read generated latest.json: %v", err)
	}
	var releases []localUpdateReleaseFixture
	if err := json.Unmarshal(data, &releases); err != nil {
		t.Fatalf("latest.json must be GitHub List releases compatible array: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("latest.json should contain one release, got %d", len(releases))
	}
	release := releases[0]
	if release.TagName != versionTag ||
		release.Name != "go-desktop "+versionTag ||
		release.HTMLURL != releaseURL ||
		release.Draft ||
		release.Prerelease ||
		len(release.Assets) != 2 {
		t.Fatalf("unexpected generated local release: %#v", release)
	}
	if release.Assets[0].Name != assetName ||
		release.Assets[0].Size != int64(len(payload)) ||
		release.Assets[0].Digest != "sha256:"+expectedSha ||
		release.Assets[0].BrowserDownloadURL != releaseURL+assetName {
		t.Fatalf("unexpected installer asset: %#v", release.Assets[0])
	}
	if release.Assets[1].Name != shaName ||
		release.Assets[1].Size != int64(len(expectedSha)+2+len(assetName)) ||
		release.Assets[1].BrowserDownloadURL != releaseURL+shaName {
		t.Fatalf("unexpected sha256 asset: %#v", release.Assets[1])
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

func requireFileContent(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}
	if got := string(data); got != want {
		t.Fatalf("%s content = %q, want %q", path, got, want)
	}
}

func requireInOrder(t *testing.T, path string, source string, values ...string) {
	t.Helper()
	offset := 0
	for _, value := range values {
		index := strings.Index(source[offset:], value)
		if index < 0 {
			t.Fatalf("%s should contain %q after offset %d", path, value, offset)
		}
		offset += index + len(value)
	}
}

// rootPath 封装 验证 sync_project_metadata_test.go 覆盖的生产行为、结构约束或构建脚本约束 中的一段独立逻辑，调用方通过它复用同一业务规则。
func rootPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}
