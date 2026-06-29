// 文件职责：验证前端模块边界、设计文档合同、shadcn/artistic 结构和更新/授权页面职责。

package frontend_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const frontendColorTokenFile = "frontend/src/colors.css"

var displayPreferencePaletteColors = []struct {
	token string
	value string
}{
	{token: "neutral", value: "oklch(0.45 0 0)"},
	{token: "stone", value: "oklch(0.444 0.011 73.639)"},
	{token: "zinc", value: "oklch(0.442 0.017 285.786)"},
	{token: "mauve", value: "oklch(0.5 0.09 315)"},
	{token: "olive", value: "oklch(0.48 0.08 120)"},
	{token: "mist", value: "oklch(0.55 0.08 235)"},
	{token: "taupe", value: "oklch(0.48 0.045 82)"},
	{token: "amber", value: "oklch(0.769 0.188 70.08)"},
	{token: "apple-blue", value: "#007aff"},
	{token: "blue", value: "#1677ff"},
	{token: "cyan", value: "oklch(0.609 0.126 221.723)"},
	{token: "emerald", value: "oklch(0.596 0.145 163.225)"},
	{token: "indigo", value: "oklch(0.511 0.262 276.966)"},
	{token: "orange", value: "oklch(0.646 0.222 41.116)"},
	{token: "pink", value: "oklch(0.656 0.241 354.308)"},
	{token: "rose", value: "oklch(0.586 0.253 17.585)"},
	{token: "sky", value: "oklch(0.588 0.158 241.966)"},
	{token: "teal", value: "oklch(0.6 0.118 184.704)"},
}

var rawColorLiteralPattern = regexp.MustCompile(`(?i)#[0-9a-f]{3,8}\b|oklch\([^;\n]*?\)|rgba?\([^;\n]*?\)|hsla?\([^;\n]*?\)`)
var rawNamedColorPattern = regexp.MustCompile(`(?i)(^|[^a-z0-9_-])(transparent|white|black)([^a-z0-9_-]|$)`)
var colorTokenDeclarationPattern = regexp.MustCompile(`(?m)^\s*(--color-[a-z0-9-]+):\s*([^;]+);`)
var colorTokenNameShapePattern = regexp.MustCompile(`^--color-(transparent|(display|black|white|value)-[a-z0-9-]+)$`)
var monochromeAlphaTokenNamePattern = regexp.MustCompile(`^--color-(black|white)-alpha-([0-9]{3})$`)
var rgbaColorValuePattern = regexp.MustCompile(`^rgba\(\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9.]+)\s*\)$`)

// TestAppRootStaysAsCompositionRoot 验证 App.vue 只装配全局状态、门禁和页面出口，不回流具体业务流程。
func TestAppRootStaysAsCompositionRoot(t *testing.T) {
	appRoot := readRootFile(t, "frontend", "src", "App.vue")

	for _, forbidden := range []string{
		"run" + "Release" + "Audit(",
		"downloadLatestUpdate(",
		"refreshLogs(",
		"persistSettings(",
	} {
		if strings.Contains(appRoot, forbidden) {
			t.Fatalf("frontend/src/App.vue should only compose pages and global providers; move %q into focused feature components", forbidden)
		}
	}

	if lines := strings.Count(appRoot, "\n") + 1; lines > 220 {
		t.Fatalf("frontend/src/App.vue is too large for a composition root: got %d lines, want <= 220", lines)
	}
}

// TestFrontendFeatureBoundariesExist 验证页面、store、shared/ui wrapper 和 shadcn primitive 的预期目录边界存在。
func TestFrontendFeatureBoundariesExist(t *testing.T) {
	for _, path := range []string{
		filepath.Join("frontend", "src", "App.vue"),
		filepath.Join("frontend", "src", "app", "display.ts"),
		filepath.Join("frontend", "src", "stores", "app.ts"),
		filepath.Join("frontend", "src", "features", "layout", "AppChrome.vue"),
		filepath.Join("frontend", "src", "features", "home", "HomePage.vue"),
		filepath.Join("frontend", "src", "features", "update", "UpdateStatusDialog.vue"),
		filepath.Join("frontend", "src", "features", "logs", "LogsPage.vue"),
		filepath.Join("frontend", "src", "features", "settings", "SettingsPage.vue"),
		filepath.Join("frontend", "src", "features", "about", "AboutPage.vue"),
		filepath.Join("frontend", "src", "components", "ui", "button", "Button.vue"),
		filepath.Join("frontend", "src", "components", "ui", "card", "Card.vue"),
		filepath.Join("frontend", "src", "components", "ui", "progress", "Progress.vue"),
		filepath.Join("frontend", "src", "components", "ui", "badge", "Badge.vue"),
		filepath.Join("frontend", "src", "components", "ui", "dialog", "Dialog.vue"),
		filepath.Join("frontend", "src", "components", "ui", "select", "Select.vue"),
		filepath.Join("frontend", "src", "components", "ui", "table", "Table.vue"),
		filepath.Join("frontend", "src", "components", "ui", "tooltip", "Tooltip.vue"),
		filepath.Join("frontend", "src", "shared", "ui", "Card.vue"),
		filepath.Join("frontend", "src", "shared", "ui", "CardTitle.vue"),
		filepath.Join("frontend", "src", "shared", "ui", "plugin.ts"),
	} {
		if _, err := os.Stat(rootPath(path)); err != nil {
			t.Fatalf("expected frontend boundary file %s to exist: %v", path, err)
		}
	}
}

// TestGoTestFilesStayInDedicatedTestsModule 验证 Go 测试不散落在生产模块。
func TestTestFilesStayInDedicatedTestsModule(t *testing.T) {
	var misplaced []string
	for _, root := range []string{
		filepath.Join("frontend", "src"),
		"scripts",
	} {
		err := filepath.WalkDir(rootPath(root), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			name := entry.Name()
			if strings.HasSuffix(name, "_test.go") ||
				strings.Contains(name, ".test.") ||
				strings.Contains(name, ".spec.") {
				rel, relErr := filepath.Rel(rootPath("."), path)
				if relErr != nil {
					rel = path
				}
				misplaced = append(misplaced, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s for misplaced tests: %v", root, err)
		}
	}
	if len(misplaced) > 0 {
		t.Fatalf("test files must live in the dedicated tests/ module, found outside tests/: %s", strings.Join(misplaced, ", "))
	}
}

// TestShadcnPrimitivesAreGloballyRegistered 验证全局 Ui* 注册集中在 shared/ui/plugin.ts，业务页不用重复 import primitive。
func TestShadcnPrimitivesAreGloballyRegistered(t *testing.T) {
	main := readRootFile(t, "frontend", "src", "main.ts")
	uiPlugin := readRootFile(t, "frontend", "src", "shared", "ui", "plugin.ts")

	for _, required := range []string{
		"uiPlugin",
		".use(uiPlugin)",
	} {
		if !strings.Contains(main, required) {
			t.Fatalf("frontend/src/main.ts should install the global UI plugin: missing %q", required)
		}
	}

	for _, required := range []string{
		"UiButton",
		"UiCard",
		"UiSelect",
		"UiSelectTrigger",
		"UiSelectContent",
		"UiSelectItem",
		"UiSelectValue",
		"UiProgress",
		"UiDialog",
		"UiTable",
		"UiTooltip",
		"app.component",
	} {
		if !strings.Contains(uiPlugin, required) {
			t.Fatalf("frontend/src/shared/ui/plugin.ts should globally register %q", required)
		}
	}
}

// TestDialogsDoNotCloseFromOutsideClick 验证项目级弹窗 wrapper 拦截外部点击关闭，避免业务弹窗被误关。
func TestDialogsDoNotCloseFromOutsideClick(t *testing.T) {
	dialog := readRootFile(t, "frontend", "src", "shared", "ui", "Dialog.vue")
	alertDialog := readRootFile(t, "frontend", "src", "shared", "ui", "AlertDialog.vue")
	alertDialogPrimitive := readRootFile(t, "frontend", "src", "components", "ui", "alert-dialog", "AlertDialogContent.vue")

	for _, content := range []struct {
		name string
		body string
	}{
		{name: "UiDialog", body: dialog},
		{name: "UiAlertDialog", body: alertDialog},
	} {
		if strings.Contains(content.body, `@pointer-down-outside="emit('close')"`) {
			t.Fatalf("%s must not close when users click outside the dialog", content.name)
		}
		if !strings.Contains(content.body, "@pointer-down-outside") || !strings.Contains(content.body, "event.preventDefault()") {
			t.Fatalf("%s should prevent outside pointer dismissal", content.name)
		}
	}
	if strings.Contains(alertDialogPrimitive, "@pointer-down-outside") {
		t.Fatalf("alert dialog primitive must stay CLI-owned; keep project outside-click behavior in shared/ui wrapper")
	}
}

// TestShadcnCompositionReplacesHandRolledControls 验证设置、日志、更新弹窗和顶栏继续组合 shadcn primitive，不回退成手写控件。
func TestShadcnCompositionReplacesHandRolledControls(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")

	for _, forbidden := range []string{
		"SegmentedControl",
		"segment-button",
		"swatch-button",
	} {
		if strings.Contains(settingsPage, forbidden) {
			t.Fatalf("settings display controls should use shadcn-style primitive composition, found %q", forbidden)
		}
	}

	for _, required := range []string{
		"UiSelect",
		"UiSelectTrigger",
		"UiSelectContent",
		"UiSelectItem",
		"UiSwitch",
		"color-dot-palette",
		"品牌辅助色 (Accent Color)",
		"is-managed-field",
		"is-managed-palette",
		"is-managed-selected",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should use shadcn-style primitive %q", required)
		}
	}
	settingsStyles := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.css")
	for _, required := range []string{
		`.aesthetic-field-col[aria-disabled="true"]`,
		".is-managed-palette",
		".color-dot-btn:disabled",
	} {
		if !strings.Contains(settingsStyles, required) {
			t.Fatalf("settings page should keep managed accent disabled styling in feature CSS for shadcn/artistic parity, missing %q", required)
		}
	}
	if strings.Contains(settingsPage, "UiNativeSelect") {
		t.Fatal("settings page should use shadcn Select for dropdowns instead of native select")
	}
	if strings.Contains(settingsPage, "setAccentColor") {
		t.Fatal("settings page must show artistic accent color as a managed disabled mirror, not edit accentColor directly")
	}
	for _, forbidden := range []string{
		"['default-translucent'",
		"['inverted-translucent'",
	} {
		if strings.Contains(settingsPage, forbidden) {
			t.Fatalf("settings page should only expose default/inverted menu options; persistence may still accept %q", forbidden)
		}
	}

	if strings.Contains(logsPage, `role="table"`) || strings.Contains(logsPage, "log-row") {
		t.Fatalf("logs page should use table primitives instead of a hand-rolled div table")
	}
	for _, required := range []string{"UiTable", "UiTableHeader", "UiTableBody", "UiTableRow", "UiTableHead", "UiTableCell"} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page should use table primitive %q", required)
		}
	}

	if strings.Contains(updateDialog, "dialog-layer") || strings.Contains(updateDialog, "update-dialog") {
		t.Fatalf("update dialog should compose the Dialog primitive instead of raw dialog wrappers")
	}
	if !strings.Contains(updateDialog, "UiDialog") {
		t.Fatalf("update dialog should use UiDialog")
	}
	if !strings.Contains(updateDialog, "scheduleDownloadedUpdateOnStartup") {
		t.Fatalf("update dialog should explicitly schedule next-start update instead of only closing the dialog")
	}

	if !strings.Contains(appChrome, "UiTooltip") {
		t.Fatalf("topbar icon buttons should use the tooltip primitive")
	}
}

// TestDesignDocumentsShadcnVueWorkflow 验证 DESIGN.md 记录 shadcn-vue 查询、添加和全局注册流程。
func TestDesignDocumentsShadcnVueWorkflow(t *testing.T) {
	design := readRootFile(t, "DESIGN.md")

	for _, required := range []string{
		"shadcn-vue skill",
		"components.json",
		"shadcn-vue info --json",
		"shadcn-vue docs",
		"shadcn-vue add",
		"全局 Ui*",
	} {
		if !strings.Contains(design, required) {
			t.Fatalf("DESIGN.md should document the shadcn-vue workflow and global UI policy, missing %q", required)
		}
	}
}

// TestDesignDocumentsEngineeringHardRules 验证 DESIGN.md 和 .gitignore 继续声明测试、临时产物和界面工程硬约束。
func TestDesignDocumentsEngineeringHardRules(t *testing.T) {
	design := readRootFile(t, "DESIGN.md")
	gitignore := readRootFile(t, ".gitignore")

	for _, required := range []string{
		"工程硬约束",
		"测试只能放在独立 `tests/` 模块",
		"`scripts/` 只放可执行工具",
		"临时截图",
		"`.tmp/`",
		"PC 端 `1440×900` 和窄屏视口",
		"代码注释必须覆盖模块边界",
		"设置页只放能修改状态的控件",
		"只记录偏好但不改变实际界面/行为",
		"禁止远程字体加载",
	} {
		if !strings.Contains(design, required) {
			t.Fatalf("DESIGN.md should document engineering hard rule %q", required)
		}
	}

	for _, required := range []string{
		".tmp/*",
		"!.tmp/.gitkeep",
	} {
		if !strings.Contains(gitignore, required) {
			t.Fatalf(".gitignore should reserve .tmp for temporary artifacts: missing %q", required)
		}
	}
	if _, err := os.Stat(rootPath(filepath.Join(".tmp", ".gitkeep"))); err != nil {
		t.Fatalf("expected .tmp/.gitkeep to reserve temporary artifact directory: %v", err)
	}
}

// TestBootViewportBackplatePreventsOuterBlackFlash 验证首屏启动底板不依赖 Vue 挂载后的组件铺满视口。
func TestBootViewportBackplatePreventsOuterBlackFlash(t *testing.T) {
	indexHTML := strings.ReplaceAll(readRootFile(t, "frontend", "index.html"), "\r\n", "\n")
	globalStyles := strings.ReplaceAll(readRootFile(t, "frontend", "src", "styles.css"), "\r\n", "\n")
	design := readRootFile(t, "DESIGN.md")

	for _, required := range []string{
		"body::before",
		"position: fixed",
		"inset: 0",
		"z-index: -1",
		"background: var(--boot-background)",
		"#app,\n      .app-shell",
		"border: 0 !important",
		"box-shadow: none !important",
	} {
		if !strings.Contains(indexHTML, required) {
			t.Fatalf("frontend/index.html should keep boot viewport backplate rule, missing %q", required)
		}
	}
	if strings.Contains(indexHTML, "prefers-color-scheme") {
		t.Fatal("frontend/index.html should keep boot background under app display preferences instead of system color scheme")
	}

	for _, required := range []string{
		"html,\nbody,\n#app",
		"width: 100%",
		"min-width: 0",
		"background: var(--background)",
		"box-shadow: none",
		"html {\n  overflow: hidden;\n}",
	} {
		if !strings.Contains(globalStyles, required) {
			t.Fatalf("frontend/src/styles.css should keep runtime root viewport fallback, missing %q", required)
		}
	}

	for _, required := range []string{
		"首屏启动底板",
		"Vue 挂载空窗期",
		"body::before",
		"固定满屏底板",
		"不跟随 `prefers-color-scheme`",
	} {
		if !strings.Contains(design, required) {
			t.Fatalf("DESIGN.md should document boot viewport backplate rule, missing %q", required)
		}
	}
}

// TestDesignDocumentMatchesCurrentSettingsContract 验证 DESIGN.md 没有保留旧设置模型和过时测试命令。
func TestDesignDocumentMatchesCurrentSettingsContract(t *testing.T) {
	design := readRootFile(t, "DESIGN.md")

	for _, required := range []string{
		`export type BaseColor = "neutral" | "stone" | "zinc" | "mauve" | "olive" | "mist" | "taupe"`,
		"Theme / Accent / Chart Color",
		"Icon Library 暂不作为设置项",
		"固定使用 Lucide",
		"关闭到系统托盘",
		"`1 / 3 / 6 / 12 小时`",
		"go test ./...",
	} {
		if !strings.Contains(design, required) {
			t.Fatalf("DESIGN.md should describe current settings contract %q", required)
		}
	}

	for _, forbidden := range []string{
		`export type BaseColor = "slate"`,
		"五套官方 base color",
		"明确支持 shadcn-vue create 十项轴",
		"设置页分段控制",
		"设置页强调色色板",
		"按钮最小高度 `40px`",
		"go test ./frontend",
		"检查 icon library",
		"最小化到系统托盘",
	} {
		if strings.Contains(design, forbidden) {
			t.Fatalf("DESIGN.md should not keep stale or inconsistent contract %q", forbidden)
		}
	}
}

// TestUpdateIsGlobalIconNotStandalonePage 验证更新入口属于顶栏图标和弹窗，不重新变成导航页面。
func TestUpdateIsGlobalIconNotStandalonePage(t *testing.T) {
	appRoot := readRootFile(t, "frontend", "src", "App.vue")
	views := readRootFile(t, "frontend", "src", "shared", "views.ts")
	chrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")

	for _, forbidden := range []string{
		"UpdatePage",
		"key: 'update'",
		"activeView === 'update'",
		"更新管理",
	} {
		if strings.Contains(appRoot+views, forbidden) {
			t.Fatalf("update must be a top-right icon/dialog, not a standalone navigation page: found %q", forbidden)
		}
	}

	for _, required := range []string{
		"UpdateStatusDialog",
		"setThemeMode",
		"themeMode",
		"RefreshCw",
	} {
		if !strings.Contains(chrome, required) {
			t.Fatalf("frontend/src/features/layout/AppChrome.vue should own global theme and update entries, missing %q", required)
		}
	}
	if strings.Contains(chrome, "Bell") {
		t.Fatal("frontend/src/features/layout/AppChrome.vue should use a refresh/update icon for update status, not Bell")
	}
}

// TestUpdateHeaderIconReflectsLifecycleAndMotion 验证右上角更新图标按生命周期显示颜色和动效。
func TestUpdateHeaderIconReflectsLifecycleAndMotion(t *testing.T) {
	chrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")
	chromeStyles := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css")
	appStore := readRootFile(t, "frontend", "src", "stores", "app.ts")

	for _, required := range []string{
		`if (value === 'error') return 'is-danger'`,
		`if (['downloading', 'verifying', 'installing'].includes(value)) return 'is-busy'`,
		`if (['update_available', 'verified', 'pending_install'].includes(value)) return 'is-ready'`,
		`RefreshCw :class="cn(updateTone === 'is-danger' ? 'icon-tone-red' : updateTone === 'is-ready' ? 'icon-tone-green' : 'icon-tone-blue')"`,
	} {
		if !strings.Contains(chrome, required) {
			t.Fatalf("AppChrome.vue should keep update icon lifecycle mapping %q", required)
		}
	}

	for _, required := range []string{
		`@keyframes update-hover-spin`,
		`transform: rotate(360deg)`,
		`.update-icon-button.is-busy svg`,
		`animation: spin 1.2s linear infinite`,
		`animation-iteration-count: infinite !important`,
		`.update-icon-button:not(.is-busy):hover svg`,
		`animation: update-hover-spin 0.6s linear`,
		`animation-iteration-count: 1 !important`,
	} {
		if !strings.Contains(chromeStyles, required) {
			t.Fatalf("AppChrome.css should keep update icon motion rule %q", required)
		}
	}

	for _, required := range []string{
		`Events.On('update:status:changed'`,
		`updateStatusFromEventData(event.data)`,
		`isUpdateTerminalStatus(updateStatus?.status)`,
		`this.applyAction({ type: 'checkingSet', payload: false })`,
		`this.applyAction({ type: 'downloadingSet', payload: false })`,
	} {
		if !strings.Contains(appStore, required) {
			t.Fatalf("stores/app.ts should clear busy flags when update reaches terminal status: missing %q", required)
		}
	}
}

// TestFrontendInitialiseShowsMainWindowAfterLoading 验证前端初始化完成后才通知后端显示主窗口。
// 这条链路用于避免 Wails runtime ready 后主窗口先露出空白内容。
func TestFrontendInitialiseShowsMainWindowAfterLoading(t *testing.T) {
	appStore := readRootFile(t, "frontend", "src", "stores", "app.ts")
	wailsAPI := readRootFile(t, "frontend", "src", "api", "wails.ts")

	for _, required := range []string{
		"showMainWindow,",
	} {
		if !strings.Contains(appStore, required) {
			t.Fatalf("frontend/src/stores/app.ts 必须在初始化完成后调用 showMainWindow：缺少 %q", required)
		}
	}
	if calls := strings.Count(appStore, "await showMainWindow()"); calls < 2 {
		t.Fatalf("frontend/src/stores/app.ts 必须覆盖授权提前返回和正常初始化完成两条显示路径，当前 showMainWindow 调用次数=%d", calls)
	}
	for _, required := range []string{
		"export async function showMainWindow()",
		"binding('ShowMainWindow')",
	} {
		if !strings.Contains(wailsAPI, required) {
			t.Fatalf("frontend/src/api/wails.ts 必须封装 ShowMainWindow 绑定：缺少 %q", required)
		}
	}

	finallyIdx := strings.Index(appStore, "} finally {")
	if finallyIdx < 0 {
		t.Fatal("frontend/src/stores/app.ts 缺少初始化 finally 收尾逻辑")
	}
	finallyBlock := appStore[finallyIdx:]
	loadingSetIdx := strings.Index(finallyBlock, "type: 'loadingSet', payload: false")
	showMainWindowIdx := strings.Index(finallyBlock, "await showMainWindow()")
	if loadingSetIdx < 0 || showMainWindowIdx < 0 || showMainWindowIdx < loadingSetIdx {
		t.Fatal("showMainWindow 必须在 loadingSet:false 之后调用，确保启动数据加载完成后再显示窗口")
	}

	licenseCheckIdx := strings.Index(appStore, "licenseStatus.required && !licenseStatus.authorized")
	if licenseCheckIdx < 0 {
		t.Fatal("frontend/src/stores/app.ts 缺少授权状态提前返回逻辑")
	}
	licenseBlock := appStore[licenseCheckIdx:]
	if returnIdx := strings.Index(licenseBlock, "\n        return"); returnIdx > 0 {
		licenseBlock = licenseBlock[:returnIdx]
	}
	if !strings.Contains(licenseBlock, "await showMainWindow()") {
		t.Fatal("授权未通过的提前返回路径也必须调用 showMainWindow，否则授权页不可见")
	}
}

func TestUpdateDialogDoesNotInstallWhenOpened(t *testing.T) {
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")

	for _, required := range []string{
		"async function installNow()",
		"async function runPrimaryAction()",
		"await installNow()",
		`@click="runPrimaryAction"`,
	} {
		if !strings.Contains(updateDialog, required) {
			t.Fatalf("update dialog should keep explicit user-triggered install action %q", required)
		}
	}
	watchBlockStart := strings.Index(updateDialog, "watch(() => props.open")
	watchBlockEnd := strings.Index(updateDialog, "// isTransferState")
	if watchBlockStart < 0 || watchBlockEnd <= watchBlockStart {
		t.Fatalf("update dialog should keep an open watcher that only refreshes status")
	}
	if strings.Contains(updateDialog[watchBlockStart:watchBlockEnd], "installNow") {
		t.Fatalf("opening the update dialog must not start installation automatically")
	}
}

func TestUpdateDialogUsesUserFocusedStatusView(t *testing.T) {
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")
	updateStyles := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.css")

	for _, required := range []string{
		"user-version-line",
		"versionLine()",
		"更新包已准备好",
		"当前已是最新版本",
		"更新失败",
		"重新检查",
		`<p v-if="description">{{ description }}</p>`,
		"if (canInstall.value) return ''",
		"当前版本 ${currentVersion.value} · 最新版本 ${latestVersion.value}",
	} {
		if !strings.Contains(updateDialog, required) {
			t.Fatalf("update dialog should keep user-focused status view %q", required)
		}
	}

	for _, forbidden := range []string{
		`class="data-list"`,
		`safety-card neutral`,
		`<footer class="dialog-footer">`,
		"technical-details",
		"技术详情",
		"user-version-summary",
		"UiBadge",
		"trust-notice",
		"SHA256",
		"更新包已下载并通过校验，可以现在安装，也可以下次启动时再安装。",
	} {
		if strings.Contains(updateDialog, forbidden) {
			t.Fatalf("update dialog should not keep audit-style default content %q", forbidden)
		}
	}

	for _, required := range []string{
		".user-version-line",
	} {
		if !strings.Contains(updateStyles, required) {
			t.Fatalf("update dialog styles should support user-focused layout %q", required)
		}
	}
	for _, forbidden := range []string{
		".technical-details",
		".technical-row",
		".user-version-summary",
		".trust-notice",
	} {
		if strings.Contains(updateStyles, forbidden) {
			t.Fatalf("update dialog styles should not keep technical/card layout %q", forbidden)
		}
	}
}

func TestUpdateDialogClosesAfterSchedulingNextStartup(t *testing.T) {
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")
	start := strings.Index(updateDialog, "async function scheduleOnStartup()")
	end := strings.Index(updateDialog, "// closeDialog")
	if start < 0 || end <= start {
		t.Fatalf("update dialog should keep scheduleOnStartup before closeDialog")
	}
	scheduleBlock := updateDialog[start:end]
	for _, required := range []string{
		"await appStore.scheduleDownloadedUpdateOnStartup()",
		"closeDialog()",
	} {
		if !strings.Contains(scheduleBlock, required) {
			t.Fatalf("scheduling next-start update should close the dialog after success: missing %q", required)
		}
	}
}

// TestFrontendHasLicenseGate 验证授权页是独立业务页面，App 根组件只负责门禁装配。
func TestFrontendHasLicenseGate(t *testing.T) {
	appRoot := readRootFile(t, "frontend", "src", "App.vue")
	appStore := readRootFile(t, "frontend", "src", "stores", "app.ts")
	wailsAPI := readRootFile(t, "frontend", "src", "api", "wails.ts")
	licensePage := readRootFile(t, "frontend", "src", "features", "license", "LicensePage.vue")
	licenseStyles := readRootFile(t, "frontend", "src", "features", "license", "LicensePage.css")

	for _, path := range []string{
		filepath.Join("frontend", "src", "features", "license", "LicensePage.vue"),
		filepath.Join("frontend", "src", "features", "license", "LicensePage.css"),
	} {
		if _, err := os.Stat(rootPath(path)); err != nil {
			t.Fatalf("expected license feature file %s to exist: %v", path, err)
		}
	}

	for _, required := range []string{
		"LicensePage",
		"licenseStatus?.required",
		"!appStore.licenseStatus?.authorized",
	} {
		if !strings.Contains(appRoot, required) {
			t.Fatalf("App.vue should gate the main UI behind license status: missing %q", required)
		}
	}

	for _, required := range []string{
		"getLicenseStatus",
		"activateLicense",
		"LicenseStatus",
		"defaultLicenseStatus",
	} {
		if !strings.Contains(wailsAPI, required) {
			t.Fatalf("frontend/src/api/wails.ts should expose license API helper %q", required)
		}
	}

	for _, required := range []string{
		"loadLicenseStatus",
		"activateLicenseKey",
		"failedLicenseStatus",
		"licenseStatus.required && !licenseStatus.authorized",
		"normaliseLicenseKey",
		"授权状态读取失败",
		"licenseStatusApplied",
		"licenseErrorSet",
	} {
		if !strings.Contains(appStore, required) {
			t.Fatalf("frontend/src/stores/app.ts should own license state flow %q", required)
		}
	}

	for _, required := range []string{
		"<textarea",
		`id="license-key"`,
		`v-model="licenseKey"`,
		`rows="5"`,
	} {
		if !strings.Contains(licensePage, required) {
			t.Fatalf("LicensePage.vue 授权码输入必须支持多行展示：缺少 %q", required)
		}
	}
	if strings.Contains(licensePage, `@keydown.enter.prevent="submitLicense"`) {
		t.Fatalf("LicensePage.vue 授权码多行输入不应拦截 Enter 直接提交")
	}
	for _, required := range []string{
		".license-textarea",
		"min-height",
		"resize: vertical",
		"overflow-wrap: anywhere",
	} {
		if !strings.Contains(licenseStyles, required) {
			t.Fatalf("LicensePage.css 授权码输入样式必须适配多行展示：缺少 %q", required)
		}
	}
}

// TestSettingsPageSeparatesDisplayPreferencesFromBackendSettings 验证显示偏好和后端设置在设置页保持独立保存链路。
func TestSettingsPageSeparatesDisplayPreferencesFromBackendSettings(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	displayState := readRootFile(t, "frontend", "src", "app", "display.ts")
	wailsAPI := readRootFile(t, "frontend", "src", "api", "wails.ts")
	appStore := readRootFile(t, "frontend", "src", "stores", "app.ts")

	for _, required := range []string{
		"显示偏好",
		"恢复当前方案默认预设",
		"setThemeMode",
		"setBaseColor",
		"setThemeColor",
		"setTextSize",
		"setRadius",
		"setDensity",
		"setCardBorder",
		"setIconTone",
		"updateCheckIntervalHours",
		"minimizeToTray",
		"alwaysOnTop",
		"logRetentionDays",
		"logLevel",
		"日志级别",
		"autoLaunch",
		"createDesktopShortcut",
		"launchHiddenToTray",
		`:disabled="!draft.autoLaunch"`,
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("frontend/src/features/settings/SettingsPage.vue should expose setting or display control %q", required)
		}
	}

	for _, required := range []string{
		"displayPreferenceDefaults",
		"displayScheme",
		"resetDisplayPreferences",
		"resetDisplayPreferencesForCurrentScheme",
		"hydrateDisplayPreferences",
		"exportDisplayPreferences",
		"profiles",
		"normaliseProfiles",
		"rememberCurrentProfile(displayScheme.value)",
		"dataset.displayScheme",
	} {
		if !strings.Contains(displayState, required) {
			t.Fatalf("frontend/src/app/display.ts should own display preference facade %q", required)
		}
	}

	for _, required := range []string{
		"显示方案",
		"asDisplayScheme",
		"artistic",
		"immediate: true",
		"settingsSaveDelayMs",
		"displaySaveTimer",
		"window.setTimeout",
		"window.clearTimeout",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should debounce backend persistence %q", required)
		}
	}

	for _, required := range []string{
		"GetDisplayPreferences",
		"SaveDisplayPreferences",
		"getDisplayPreferences",
		"saveDisplayPreferences",
		"displayPreferences",
		"persistDisplayPreferences",
	} {
		if !strings.Contains(wailsAPI+appStore, required) {
			t.Fatalf("frontend display preferences should persist through backend API/store %q", required)
		}
	}

	for _, forbidden := range []string{
		"localStorage",
		"getItem(",
		"setItem(",
	} {
		if strings.Contains(displayState, forbidden) {
			t.Fatalf("frontend/src/app/display.ts should not persist display preferences locally: found %q", forbidden)
		}
	}
}

// TestGeneratedBindingsExposeSettingsLogLevelAndDebugStats 保护 Wails 生成绑定和 Go 数据结构同步。
// 打包二进制使用 bindings 参与前端编译，旧绑定会让 logLevel/debug 字段在类型层丢失。
func TestGeneratedBindingsExposeSettingsLogLevelAndDebugStats(t *testing.T) {
	models := readRootFile(t, "frontend", "bindings", "github.com", "chencn", "go-desktop", "app", "models.ts")

	for _, required := range []string{
		`"displayScheme": string;`,
		`this["displayScheme"] = "";`,
		`"profiles": DisplayProfiles;`,
		`export class DisplayProfile`,
		`export class DisplayProfiles`,
		`"logLevel": string;`,
		`this["logLevel"] = "";`,
		`"alwaysOnTop": boolean;`,
		`this["alwaysOnTop"] = false;`,
		`"debug": number;`,
		`this["debug"] = 0;`,
	} {
		if !strings.Contains(models, required) {
			t.Fatalf("generated Wails models should expose log level/debug stats field %q", required)
		}
	}
}

func TestGeneratedAppBindingsDoNotExposeInternalPackages(t *testing.T) {
	bindingSources := strings.Join([]string{
		readRootFile(t, "frontend", "bindings", "github.com", "chencn", "go-desktop", "app", "api.ts"),
		readRootFile(t, "frontend", "bindings", "github.com", "chencn", "go-desktop", "app", "models.ts"),
	}, "\n")

	for _, forbidden := range []string{
		"../internal/desktopapp/runtime",
		"../internal/adapters/githubrelease",
		"../internal/githubrelease",
	} {
		if strings.Contains(bindingSources, forbidden) {
			t.Fatalf("generated app bindings must only expose app facade models, found internal package import %q", forbidden)
		}
	}
}

// TestSettingsPageOnlyContainsEditableSettings 验证设置页只放可编辑状态，不展示只读版本、路径或环境信息。
func TestSettingsPageOnlyContainsEditableSettings(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	settingsStyles := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.css")
	wailsAPI := readRootFile(t, "frontend", "src", "api", "wails.ts")
	projectMetadata := readRootFile(t, "frontend", "src", "shared", "project.ts")

	for _, forbidden := range []string{
		"Release 来源",
		"GitHub Owner",
		"GitHub Repo",
		"无网络策略",
		"SHA256 策略",
		"当前日志",
		"更新审计",
		"托盘菜单只保留",
		"最小化到系统托盘",
		"图标库",
		"当前运行时记录偏好",
	} {
		if strings.Contains(settingsPage, forbidden) {
			t.Fatalf("settings page should not show read-only/about information %q", forbidden)
		}
	}

	for _, required := range []string{
		"检查间隔",
		"关闭到系统托盘",
		"窗口置顶",
		"开机自启",
		"创建桌面快捷图标",
		"开机自启时隐藏到托盘",
		"保留周期",
		"const updateIntervalOptions = [1, 3, 6, 12]",
		"{{ hours }} 小时",
		"updateIntervalOptions",
		"normaliseUpdateCheckIntervalHours",
		"githubProxyBase",
		"GitHub 更新代理",
		`v-if="draft.updateSource === 'github'"`,
		`class="settings-row-item is-input-row"`,
		`persistSettingsPatch({ githubProxyBase: String($event) })`,
		`:checked="draft.alwaysOnTop"`,
		`persistSettingsPatch({ alwaysOnTop: $event })`,
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should keep editable setting %q", required)
		}
	}

	for _, source := range []struct {
		name    string
		content string
	}{
		{name: "frontend/src/api/wails.ts", content: wailsAPI},
		{name: "frontend/src/shared/project.ts", content: projectMetadata},
	} {
		for _, forbidden := range []string{"githubOwner", "githubRepo"} {
			if strings.Contains(source.content, forbidden) {
				t.Fatalf("%s should not expose repository metadata as Settings field %q", source.name, forbidden)
			}
		}
	}

	for _, required := range []string{
		"grid-template-columns: auto minmax(0, 1fr) auto",
		".settings-row-item.is-input-row",
		".settings-row-item.is-input-row .settings-control-input",
		"grid-column: 1 / -1",
		"justify-self: stretch",
	} {
		if !strings.Contains(settingsStyles, required) {
			t.Fatalf("settings page mobile rows should keep switches right-aligned and inputs full-width: missing %q", required)
		}
	}

	closeIndex := strings.Index(settingsPage, "<strong>关闭到系统托盘</strong>")
	alwaysOnTopIndex := strings.Index(settingsPage, "<strong>窗口置顶</strong>")
	autoLaunchIndex := strings.Index(settingsPage, "<strong>开机自启</strong>")
	intervalIndex := strings.Index(settingsPage, "<strong>自动更新检查间隔</strong>")
	retentionIndex := strings.Index(settingsPage, "<strong>日志保留周期</strong>")
	if closeIndex < 0 || alwaysOnTopIndex < 0 || autoLaunchIndex < 0 || intervalIndex < 0 || retentionIndex < 0 ||
		!(closeIndex < alwaysOnTopIndex && alwaysOnTopIndex < autoLaunchIndex && autoLaunchIndex < intervalIndex && intervalIndex < retentionIndex) {
		t.Fatalf("business settings should order window/startup behavior before time-based selects: close=%d top=%d auto=%d interval=%d retention=%d", closeIndex, alwaysOnTopIndex, autoLaunchIndex, intervalIndex, retentionIndex)
	}

	for _, forbidden := range []string{
		"保存设置",
		"save-bar",
	} {
		if strings.Contains(settingsPage, forbidden) {
			t.Fatalf("settings page should persist edits immediately and not expose %q", forbidden)
		}
	}
}

// TestSettingsPageUsesCurrentDisplayPreferenceControls 验证设置页只约束当前真实暴露的显示偏好控件。
func TestSettingsPageUsesCurrentDisplayPreferenceControls(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	displayState := readRootFile(t, "frontend", "src", "app", "display.ts")

	for _, required := range []string{
		"显示偏好",
		"显示方案",
		"主色彩模式",
		"中性灰阶色调",
		"品牌主题色",
		"品牌辅助色",
		"图表配色体系",
		"图标色彩风格",
		"圆角大小",
		"字体字号",
		"界面布局密度",
		"容器与卡片边框强度",
		"侧边导航风格",
		"侧边导航强调",
		"界面风格",
		"themeOptions",
		"displayColorOptions",
		"baseOptions",
		"brandColorOptions",
		"themeColorOptions",
		"iconToneOptions",
		"chartOptions",
		"radiusOptions",
		"textOptions",
		"densityOptions",
		"cardBorderOptions",
		"menuOptions",
		"menuAccentOptions",
		"styleOptions",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should expose current display preference control %q", required)
		}
	}

	displayOptionsStart := strings.Index(settingsPage, "const displayColorOptions")
	colorOptionsStart := strings.Index(settingsPage, "const colorOptions")
	if displayOptionsStart < 0 || colorOptionsStart < 0 || !(displayOptionsStart < colorOptionsStart) {
		t.Fatal("settings page should define displayColorOptions as the single source before derived color option arrays")
	}
	displayOptionsSource := settingsPage[displayOptionsStart:colorOptionsStart]
	for _, color := range displayPreferencePaletteColors {
		valueLiteral := `value: '` + color.token + `'`
		if count := strings.Count(displayOptionsSource, valueLiteral); count != 1 {
			t.Fatalf("displayColorOptions should define %q exactly once, got %d", color.token, count)
		}
		if count := strings.Count(settingsPage, valueLiteral); count != 1 {
			t.Fatalf("settings page should not repeat display color token %q outside displayColorOptions, got %d", color.token, count)
		}
	}
	for _, required := range []string{
		"const colorOptions: Array<[AccentColor, string]> = displayColorOptions.map(toColorOption)",
		"const baseOptions: Array<[BaseColor, string]> = displayColorOptions",
		".filter(isBaseColorOption)",
		"const brandColorOptions: Array<[ThemeColor, string]> = displayColorOptions.filter(isBrandColorOption).map(toColorOption)",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should derive display color options through shared variables: missing %q", required)
		}
	}

	for _, required := range []string{
		"hydrateDisplayPreferences",
		"exportDisplayPreferences",
		"setThemeColor",
		"setChartColor",
		"setIconTone",
		"setMenu",
		"setMenuAccent",
	} {
		if !strings.Contains(displayState, required) {
			t.Fatalf("display state should persist current display preference axis %q", required)
		}
	}

	sidebarStyleStart := strings.Index(settingsPage, "侧边导航风格 (Sidebar Style)")
	menuAccentStart := strings.Index(settingsPage, "侧边导航强调 (Menu Accent)")
	uiStyleStart := strings.Index(settingsPage, "界面风格 (UI Style)")
	if sidebarStyleStart < 0 || menuAccentStart < 0 || uiStyleStart < 0 || !(sidebarStyleStart < menuAccentStart && menuAccentStart < uiStyleStart) {
		t.Fatal("settings page should keep sidebar style, menu accent, then UI style")
	}
	sidebarStyleBlock := settingsPage[sidebarStyleStart:menuAccentStart]
	for _, required := range []string{
		`<div class="visual-segmented-control">`,
		`v-for="[value, label] in menuOptions"`,
		`:class="{ 'is-active': display.menu.value === value }"`,
		`@click="asMenu(value)"`,
	} {
		if !strings.Contains(sidebarStyleBlock, required) {
			t.Fatalf("sidebar style should use segmented buttons like density, missing %q", required)
		}
	}
	if strings.Contains(sidebarStyleBlock, "<UiSelect") {
		t.Fatal("sidebar style has only two visual modes and should not render as a dropdown")
	}
	menuAccentBlock := settingsPage[menuAccentStart:uiStyleStart]
	for _, required := range []string{
		`v-for="[value, label] in menuAccentOptions"`,
		`:class="{ 'is-active': display.menuAccent.value === value }"`,
		`@click="asMenuAccent(value)"`,
	} {
		if !strings.Contains(menuAccentBlock, required) {
			t.Fatalf("menu accent should use segmented buttons like density, missing %q", required)
		}
	}
	if strings.Contains(menuAccentBlock, "<UiSelect") {
		t.Fatal("menu accent has only two visual modes and should not render as a dropdown")
	}

	for _, forbidden := range []string{
		"go-desktop-icon-library",
		"setIconLibrary",
	} {
		if strings.Contains(displayState, forbidden) {
			t.Fatalf("display state should not keep inactive setting %q", forbidden)
		}
	}
}

// TestDisplayCssUsesCurrentColorAxes 验证 CSS dataset 和变量仍覆盖当前支持的颜色、菜单、密度和字号轴。
func TestDisplayCssUsesCurrentColorAxes(t *testing.T) {
	styles := strings.ReplaceAll(readRootFile(t, "frontend", "src", "styles.css"), "\r\n", "\n")

	for _, required := range []string{
		`:root[data-theme-color="neutral"] { --runtime-theme-color: var(--runtime-color-neutral); }`,
		`:root[data-accent-color="neutral"] { --runtime-accent-color: var(--runtime-color-neutral); }`,
		`:root[data-chart-color="neutral"] { --runtime-chart-color: var(--runtime-color-neutral); }`,
		`:root[data-theme-color="sky"] { --runtime-theme-color`,
		`:root[data-accent-color="sky"] { --runtime-accent-color`,
		`:root[data-chart-color="sky"] { --runtime-chart-color`,
		`:root[data-theme-color]`,
		`--primary: var(--runtime-theme-color);`,
		`:root[data-accent-color]`,
		`--accent: color-mix(in oklch, var(--runtime-accent-color)`,
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("styles.css should use current split color axis %q", required)
		}
	}

	for _, forbidden := range []string{
		`data-base-color="slate"`,
		`data-base-color="gray"`,
		`:root[data-base-color][data-accent-color=`,
		`:root[data-accent-color="blue"],`,
		":root[data-chart-color=\"blue\"] {\n  --chart-1:",
		":root[data-theme-color=\"blue\"] {\n  --primary:",
		`:root:not([data-theme-color="neutral"])[data-theme-color]`,
		`:root:not([data-accent-color="neutral"])[data-accent-color]`,
		`:root:not([data-chart-color="neutral"])[data-chart-color]`,
		"fonts.googleapis",
		"@import url(",
		"data-heading-font",
		"data-body-font",
	} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("styles.css should not keep stale color axis rule %q", forbidden)
		}
	}

	accentRuleStart := strings.Index(styles, `:root[data-accent-color]`)
	if accentRuleStart < 0 {
		t.Fatal("styles.css missing current accent color rule")
	}
	accentRuleEnd := strings.Index(styles[accentRuleStart:], "\n}")
	if accentRuleEnd < 0 {
		t.Fatal("styles.css accent color rule is malformed")
	}
	accentRule := styles[accentRuleStart : accentRuleStart+accentRuleEnd]
	if strings.Contains(accentRule, "--primary") || strings.Contains(accentRule, "--sidebar-primary") {
		t.Fatalf("accent color rule must not mutate primary tokens: %s", accentRule)
	}
}

// TestFrontendColorsUseSingleTokenFile 验证项目自有前端颜色只能在全局 token 文件写死。
func TestFrontendColorsUseSingleTokenFile(t *testing.T) {
	mainTS := readRootFile(t, "frontend", "src", "main.ts")
	styles := readRootFile(t, "frontend", "src", "styles.css")
	colorTokens := strings.ReplaceAll(readRootFile(t, filepath.FromSlash(frontendColorTokenFile)), "\r\n", "\n")

	if !strings.Contains(mainTS, `import './colors.css'`) {
		t.Fatal("frontend/src/main.ts should import the global color token file before project styles")
	}
	if strings.Index(mainTS, `import './colors.css'`) > strings.Index(mainTS, `import './styles.css'`) {
		t.Fatal("global color token file must load before frontend/src/styles.css")
	}

	if !strings.Contains(styles, "var(--color-display-apple-blue)") {
		t.Fatal("styles.css should consume display palette colors through the global color token file")
	}

	for _, required := range []string{
		"Naming rules:",
		"--color-display-<token>",
		"--color-transparent",
		"--color-(white|black)-solid",
		"--color-(white|black)-alpha-080",
		"alpha 后缀固定三位",
		"--color-value-<raw-value-slug>",
		"同一个 raw 色值只能定义一次",
		"frontend/src/components/ui/** 是 shadcn primitive 目录，结构测试会跳过。",
	} {
		if !strings.Contains(colorTokens, required) {
			t.Fatalf("%s should document color token naming rule %q", frontendColorTokenFile, required)
		}
	}
	if _, err := os.Stat(rootPath(filepath.Join("frontend", "src", "styles", "tokens", "colors.css"))); err == nil {
		t.Fatal("global color token file should stay beside frontend/src/styles.css, not under frontend/src/styles/tokens")
	} else if !os.IsNotExist(err) {
		t.Fatalf("check old color token path: %v", err)
	}

	valueOwners := map[string]string{}
	for _, match := range colorTokenDeclarationPattern.FindAllStringSubmatch(colorTokens, -1) {
		name := match[1]
		value := strings.TrimSpace(match[2])
		assertColorTokenNameMatchesRules(t, name, value)
		canonicalValue := canonicalColorValue(value)
		if previous, exists := valueOwners[canonicalValue]; exists {
			t.Fatalf("raw color value %q is defined by both %s and %s", value, previous, name)
		}
		valueOwners[canonicalValue] = name
	}
	if len(valueOwners) == 0 {
		t.Fatal("global color token file should define color variables")
	}

	var violations []string
	err := filepath.WalkDir(rootPath(filepath.Join("frontend", "src")), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			rel := filepath.ToSlash(mustRelRoot(t, path))
			if rel == "frontend/src/components/ui" {
				return filepath.SkipDir
			}
			return nil
		}
		if !hasAnySuffix(entry.Name(), ".css", ".vue", ".ts") {
			return nil
		}
		rel := filepath.ToSlash(mustRelRoot(t, path))
		if rel == frontendColorTokenFile {
			return nil
		}
		source := strings.ReplaceAll(readRootFile(t, filepath.FromSlash(rel)), "\r\n", "\n")
		if strings.Contains(source, "--color-var(") || strings.Contains(source, "var(--color-var") {
			violations = append(violations, rel+": invalid nested color var")
		}
		for _, match := range rawColorLiteralPattern.FindAllString(source, -1) {
			violations = append(violations, rel+": "+match)
		}
		for _, match := range rawNamedColorPattern.FindAllStringSubmatch(source, -1) {
			violations = append(violations, rel+": "+strings.TrimSpace(match[2]))
		}
		for _, line := range strings.Split(source, "\n") {
			if strings.Contains(line, "color-mix(") && (strings.Contains(line, " white") || strings.Contains(line, " black")) {
				violations = append(violations, rel+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan frontend colors: %v", err)
	}
	if len(violations) > 0 {
		limit := len(violations)
		if limit > 20 {
			limit = 20
		}
		t.Fatalf("frontend raw colors must live in %s; first violations:\n%s", frontendColorTokenFile, strings.Join(violations[:limit], "\n"))
	}
}

// TestDisplayPreferencePaletteUsesSingleRuntimeColorSource 锁定品牌主题色和中性灰阶色调共 18 色只在 runtime 变量处写死。
func TestDisplayPreferencePaletteUsesSingleRuntimeColorSource(t *testing.T) {
	styles := strings.ReplaceAll(readRootFile(t, "frontend", "src", "styles.css"), "\r\n", "\n")
	colorTokens := strings.ReplaceAll(readRootFile(t, filepath.FromSlash(frontendColorTokenFile)), "\r\n", "\n")
	settingsStyles := strings.ReplaceAll(readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.css"), "\r\n", "\n")

	if got, want := len(displayPreferencePaletteColors), 18; got != want {
		t.Fatalf("display preference palette should explicitly cover %d colors, got %d", want, got)
	}

	seen := map[string]bool{}
	for _, color := range displayPreferencePaletteColors {
		if seen[color.token] {
			t.Fatalf("display preference palette token %q is duplicated in the test contract", color.token)
		}
		seen[color.token] = true

		runtimeVar := "--runtime-color-" + color.token
		colorTokenVar := "--color-display-" + color.token
		runtimeDeclaration := runtimeVar + ": var(" + colorTokenVar + ");"
		colorTokenDeclaration := colorTokenVar + ": " + color.value + ";"
		if count := strings.Count(colorTokens, colorTokenDeclaration); count != 1 {
			t.Fatalf("display color %q should be hard-coded exactly once in %s, got %d", color.token, frontendColorTokenFile, count)
		}
		if count := strings.Count(styles, runtimeDeclaration); count != 1 {
			t.Fatalf("runtime color %q should reference %s exactly once in styles.css, got %d", color.token, colorTokenVar, count)
		}

		if strings.Contains(settingsStyles, color.value) {
			t.Fatalf("SettingsPage.css must not repeat hard-coded display palette value %q; use var(%s)", color.value, runtimeVar)
		}

		swatchRule := `.color-palette-circle[data-accent="` + color.token + `"] { background: var(` + runtimeVar + `); }`
		if !strings.Contains(settingsStyles, swatchRule) {
			t.Fatalf("SettingsPage.css should render palette swatch %q through %s", color.token, runtimeVar)
		}

		for _, axis := range []string{"theme", "accent", "chart"} {
			axisRule := `:root[data-` + axis + `-color="` + color.token + `"] { --runtime-` + axis + `-color: var(` + runtimeVar + `);`
			if !strings.Contains(styles, axisRule) {
				t.Fatalf("styles.css should map %s color %q through %s", axis, color.token, runtimeVar)
			}
		}
	}
}

// TestColorfulIconToneStaysSemanticAndSkipsActiveNavigation 验证彩色图标只用于语义点缀，不覆盖当前激活导航状态。
func TestColorfulIconToneStaysSemanticAndSkipsActiveNavigation(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")
	homeStyles := readRootFile(t, "frontend", "src", "features", "home", "HomePage.css")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")
	appChromeStyles := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css")
	artisticCommonStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "common.css")

	for _, required := range []string{
		`data-icon icon-tone-orange`,
		`data-icon icon-tone-cyan`,
		`data-icon icon-tone-green`,
		`data-icon icon-tone-purple`,
		`data-icon icon-tone-blue`,
		`data-icon icon-tone-amber`,
		`data-icon icon-tone-red`,
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should give display preference icons semantic tone %q", required)
		}
	}
	for _, required := range []string{
		"data-icon icon-tone-indigo",
		"data-icon icon-tone-green",
		"data-icon icon-tone-orange",
		"software-status-icon ${item.tone}",
		"status-inline is-${item.status}",
	} {
		if !strings.Contains(homePage, required) {
			t.Fatalf("home page data icons should use semantic colorful tone %q", required)
		}
	}
	for _, required := range []string{
		"props.activeView !== item.key && item.tone",
		`.sidebar-item.is-active .nav-icon`,
		`.compact-nav-item.is-active svg`,
	} {
		if !strings.Contains(appChrome+appChromeStyles+layoutStyles, required) {
			t.Fatalf("active navigation icons should not keep colorful tone: missing %q", required)
		}
	}
	for _, required := range []string{
		`:root[data-icon-tone="colorful"] :where(.nav-icon, .data-icon, .software-status-icon).icon-tone-orange`,
		`:root[data-icon-tone="colorful"] svg.icon-tone-orange`,
		`:root[data-display-scheme="artistic"][data-icon-tone="colorful"] :where(.nav-icon, .data-icon, .software-status-icon).icon-tone-orange`,
	} {
		if !strings.Contains(layoutStyles+artisticCommonStyles, required) {
			t.Fatalf("semantic icon tone CSS must be gated by the icon color style setting, missing %q", required)
		}
	}
	for _, forbidden := range []string{
		`.software-status-icon.icon-tone-indigo`,
		`.software-status-icon.icon-tone-green`,
		`.software-status-icon.icon-tone-orange`,
		`.software-status-icon.icon-tone-purple`,
	} {
		if strings.Contains(homeStyles, forbidden) {
			t.Fatalf("home page icon tone styles must respect data-icon-tone instead of always coloring icons, found %q", forbidden)
		}
	}
	var ungatedRules []string
	err := filepath.WalkDir(rootPath(filepath.Join("frontend", "src")), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".css" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel := mustRelRoot(t, path)
		for index, line := range strings.Split(string(content), "\n") {
			if strings.Contains(line, ".icon-tone-") && !strings.Contains(line, `data-icon-tone="colorful"`) {
				ungatedRules = append(ungatedRules, fmt.Sprintf("%s:%d %s", filepath.ToSlash(rel), index+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan frontend icon tone CSS: %v", err)
	}
	if len(ungatedRules) > 0 {
		t.Fatalf("icon tone CSS must be gated by data-icon-tone=\"colorful\"; first violations:\n%s", strings.Join(ungatedRules[:min(len(ungatedRules), 8)], "\n"))
	}
}

// TestHomePageFocusesOnRuntimeStatusAndBusinessStats 验证首页职责只承载软件运行状态和业务统计，不回退成快捷入口页。
func TestHomePageFocusesOnRuntimeStatusAndBusinessStats(t *testing.T) {
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")
	homeStyles := readRootFile(t, "frontend", "src", "features", "home", "HomePage.css")
	appRoot := readRootFile(t, "frontend", "src", "App.vue")
	views := readRootFile(t, "frontend", "src", "shared", "views.ts")

	for _, required := range []string{
		"software-status-grid",
		"software-status-card",
		"WebView",
		"应用服务",
		"SQLite 数据库",
		"网络",
		"business-stats-grid",
		"demoStats",
		"demo-bar-chart",
		".demo-bar-fill",
		"distribution-list",
		"正常",
		"异常",
		"检测中",
		"软件运行状态、业务统计和样例图表",
		".status-inline.is-ok",
		".status-inline.is-warning",
		".status-inline.is-error",
	} {
		if !strings.Contains(homePage+views+homeStyles, required) {
			t.Fatalf("home page should focus on runtime status and business stats: missing %q", required)
		}
	}

	for _, forbidden := range []string{
		"activeViewProps",
		"workflowCards",
		"workflow-grid",
		"排查运行问题",
		"调整应用行为",
		"查看运行信息",
		"关键入口",
		"网络通道",
		"Release 通道",
		"latestUpdateCheck",
		"updateStatus",
		"checking",
		"GetUpdateStatus",
		"linear-gradient(180deg, var(--chart-2), var(--chart-1))",
	} {
		if strings.Contains(homePage+appRoot+views+homeStyles, forbidden) {
			t.Fatalf("home page should not fall back to quick-entry workflow content: found %q", forbidden)
		}
	}

	demoBarFillStart := strings.Index(homeStyles, ".demo-bar-fill")
	if demoBarFillStart < 0 {
		t.Fatal("home page trend chart should style .demo-bar-fill")
	}
	demoBarFillEnd := strings.Index(homeStyles[demoBarFillStart:], "\n}")
	if demoBarFillEnd < 0 {
		t.Fatal("home page trend chart .demo-bar-fill rule is malformed")
	}
	demoBarFillRule := homeStyles[demoBarFillStart : demoBarFillStart+demoBarFillEnd]
	if !strings.Contains(demoBarFillRule, "background:") {
		t.Fatalf("home page trend chart should keep a visible fill style, got: %s", demoBarFillRule)
	}
}

// TestAboutPageOwnsRuntimeReleaseAndTechInformation 验证关于页只承载只读运行、Release、本地数据和技术栈信息。
func TestAboutPageOwnsRuntimeReleaseAndTechInformation(t *testing.T) {
	aboutPage := readRootFile(t, "frontend", "src", "features", "about", "AboutPage.vue")

	for _, required := range []string{
		"about-overview",
		"about-section-grid",
		"应用信息",
		"运行状态",
		"运行时长",
		"Release 来源",
		"本地数据",
		"技术栈",
		"桌面运行层",
		"前端界面",
		"数据与发布",
		"GitHub Owner",
		"GitHub Repo",
		"API 代理",
	} {
		if !strings.Contains(aboutPage, required) {
			t.Fatalf("about page should own runtime/release/tech information %q", required)
		}
	}

	for _, forbidden := range []string{
		"界面系统",
		"组件风格",
		"显示偏好",
		"Vega",
	} {
		if strings.Contains(aboutPage, forbidden) {
			t.Fatalf("about page should not duplicate editable display-system settings: found %q", forbidden)
		}
	}
}

// TestLogsPageUsesThemeAlignedPageLayout 验证日志页保留专注模式，同时服从主题 token 和 shadcn Table 结构。
func TestLogsPageUsesThemeAlignedPageLayout(t *testing.T) {
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	logStyles := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.css")
	appChromeStyles := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css")
	alertDialogPrimitive := readRootFile(t, "frontend", "src", "components", "ui", "alert-dialog", "AlertDialogContent.vue")

	for _, required := range []string{
		"<Teleport to=\"body\"",
		":disabled=\"!fullscreen\"",
		`cn('page-stack log-page'`,
		`<header class="split-header log-page-header">`,
		`<div class="log-page-main">`,
		"filtersOpen = ref(false)",
		"fullscreen = ref(false)",
		"aria-expanded",
		"aria-pressed",
		"Maximize2",
		"专注模式",
		"退出专注",
		"classList.toggle('is-log-fullscreen', enabled)",
		"classList.remove('is-log-fullscreen')",
		"log-filter-panel",
		"log-message-cell",
		"log-col-message",
		"UiAlertDialog",
	} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page should keep themed fullscreen structure %q", required)
		}
	}

	for _, required := range []string{
		`SlidersHorizontal class="icon-tone-indigo"`,
		`TimerReset class="icon-tone-indigo"`,
		`RefreshCw class="icon-tone-green"`,
		`Maximize2 class="icon-tone-gray"`,
		`FileText class="icon-tone-gray"`,
		`Search class="icon-tone-gray"`,
	} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page tool icons should keep semantic icon tone %q", required)
		}
	}

	for _, required := range []string{
		".log-page",
		".log-page-main",
		".log-filter-panel",
		".log-stats-strip",
		".log-stream-panel",
		`.log-table[data-slot="table"]`,
		`.log-stream-panel [data-slot="table-container"]`,
		`.log-table [data-slot="table-head"]`,
		`.log-table [data-slot="table-cell"]`,
		".log-message-cell",
		".log-col-message",
		":global(:root.is-log-fullscreen .app-shell)",
		":global(:root.is-log-fullscreen) .log-fullscreen",
		".page-stack.log-fullscreen",
		".log-fullscreen .log-page-header",
		".log-fullscreen .log-page-main",
		".log-fullscreen.has-open-filters .log-page-main",
		`.log-fullscreen .log-stream-panel [data-slot="table-container"]`,
		`.content-scroll.is-logs-view {
    align-content: start;
    overflow: auto;
  }`,
		`height: min(44vh, 360px)`,
		"z-index: 40",
		"height: 100dvh",
		"background: var(--background)",
	} {
		if !strings.Contains(logStyles+appChromeStyles, required) {
			t.Fatalf("logs page themed layout should define %q", required)
		}
	}
	for _, required := range []string{
		`data-slot="alert-dialog-content"`,
		`z-50`,
	} {
		if !strings.Contains(alertDialogPrimitive, required) {
			t.Fatalf("log fullscreen z-index should stay below alert dialog overlay/content, missing alert dialog primitive marker %q", required)
		}
	}
	if strings.Contains(logStyles, "z-index: 2147483647") {
		t.Fatal("log fullscreen must not use max z-index; alert dialogs still need to appear above focused logs")
	}
	mobileLogsStart := strings.Index(logStyles, "@media (max-width: 980px)")
	if mobileLogsStart < 0 {
		t.Fatal("logs page should keep a mobile-only layout override")
	}
	mobileLogStyles := logStyles[mobileLogsStart:]
	for _, required := range []string{
		`display: flex;`,
		`flex-direction: column;`,
		`align-content: start;`,
		`align-self: start;`,
		`height: min(44vh, 360px);`,
		`overflow-x: auto !important;`,
		`min-width: 560px;`,
	} {
		if !strings.Contains(mobileLogStyles, required) {
			t.Fatalf("logs page mobile layout should define %q", required)
		}
	}

	for _, forbidden := range []string{
		"log-workbench",
		"log-workbench-grid",
		"log-workbench-main",
		"log-side-panel",
		"log-stat-list",
	} {
		if strings.Contains(logsPage, forbidden) || strings.Contains(logStyles, forbidden) {
			t.Fatalf("logs page should not carry stale workbench/sidebar styling %q", forbidden)
		}
	}

	for _, forbidden := range []string{
		".ui-table-wrap",
		".ui-table-head",
		".ui-table-cell",
		".ui-table-row",
	} {
		if strings.Contains(logStyles, forbidden) {
			t.Fatalf("logs page should target shadcn table data-slot instead of stale class %q", forbidden)
		}
	}
}

// TestLogsPageKeepsCloudInspiredPaginationAndTableDetails 验证日志页吸收 cloud-checkin 的分页信息密度，同时保留运行时日志表格模型。
func TestLogsPageKeepsCloudInspiredPaginationAndTableDetails(t *testing.T) {
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	logStyles := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.css")

	for _, required := range []string{
		"calculateLogPageSize",
		"ResizeObserver",
		"logTableRef",
		"logPaginationRef",
		"totalPages",
		"displayedLogPage",
		"displayedPageSize",
		"watch(logPageSize",
		"logLayoutReady ? `共 ${appStore.logTotal} 条，每页 ${displayedPageSize} 条，当前第 ${displayedLogPage} / ${totalPages} 页` : ''",
		`ref="logTableRef"`,
		`ref="logPaginationRef"`,
		`class="log-footer log-pagination-card"`,
		`<UiTableRow v-if="displayedLogs.length === 0" class="log-empty-row">`,
		`<UiTableCell colspan="4" class="log-empty-cell">{{ logLayoutReady ? '暂无匹配日志' : '' }}</UiTableCell>`,
		`class="log-level-badge"`,
		"logLevelClass",
	} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page should keep cloud-inspired pagination/table detail %q", required)
		}
	}

	for _, required := range []string{
		".log-table-shell",
		".log-pagination-card",
		".log-pagination-summary",
		".log-empty-cell",
		".log-level-badge",
		".log-level-badge.is-error",
		".log-level-badge.is-warning",
		".log-level-badge.is-info",
		".log-level-badge.is-debug",
	} {
		if !strings.Contains(logStyles, required) {
			t.Fatalf("logs page styles should keep cloud-inspired pagination/table detail %q", required)
		}
	}

	for _, forbidden := range []string{
		"const logPageSize = 50",
		`<div v-if="appStore.logs.length === 0" class="empty-state">`,
	} {
		if strings.Contains(logsPage, forbidden) {
			t.Fatalf("logs page should not keep stale pagination/table detail %q", forbidden)
		}
	}
}

// TestLogsPageKeepsFileSelectorInsideFilterPanel 验证日志文件/日期选择收进折叠筛选，避免顶部常驻摘要挤占日志流。
func TestLogsPageKeepsFileSelectorInsideFilterPanel(t *testing.T) {
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	logStyles := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.css")

	selectorIndex := strings.Index(logsPage, "日期/日志文件")
	filterIndex := strings.Index(logsPage, "log-filter-panel")
	if selectorIndex < 0 || filterIndex < 0 {
		t.Fatal("logs page should keep the date/file selector inside the filter panel")
	}
	if selectorIndex < filterIndex {
		t.Fatal("log file/date selector must live inside the collapsed filter panel")
	}

	for _, required := range []string{
		".log-toolbar",
		".log-file-field",
		"grid-template-columns: minmax(220px, 1.35fr) minmax(140px, 0.85fr) minmax(120px, 0.75fr) minmax(240px, 1.4fr)",
		"@container (max-width: 820px)",
		"grid-template-columns: minmax(0, 1fr) minmax(0, 1fr)",
		"@container (max-width: 560px)",
	} {
		if !strings.Contains(logStyles, required) {
			t.Fatalf("logs page layout should define %q", required)
		}
	}

	for _, removed := range []string{
		"log-file-selector",
		"log-source-banner",
		"log-source-summary",
		"log-file-path",
	} {
		if strings.Contains(logsPage, removed) || strings.Contains(logStyles, removed) {
			t.Fatalf("logs page should not keep the removed persistent log source summary %q", removed)
		}
	}
}

// TestMenuAccentCssOnlyUsesSupportedValues 验证菜单强调样式只依赖当前支持的 CSS dataset 值。
func TestMenuAccentCssOnlyUsesSupportedValues(t *testing.T) {
	displayState := readRootFile(t, "frontend", "src", "app", "display.ts")
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	appChromeStyles := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css")
	artisticSchemeStyles := readArtisticSchemeStyles(t)

	for _, required := range []string{
		"export type MenuAccent = 'subtle' | 'bold'",
		`:root[data-menu-accent="bold"]`,
		`:root[data-display-scheme="artistic"]`,
	} {
		if !strings.Contains(displayState+settingsPage+appChromeStyles+artisticSchemeStyles, required) {
			t.Fatalf("menu accent should expose and style the supported values only: missing %q", required)
		}
	}

	for _, forbidden := range []string{
		`data-menu-accent="solid"`,
		`data-menu-accent="outline"`,
		"['solid',",
		"['outline',",
	} {
		if strings.Contains(displayState+settingsPage+appChromeStyles+artisticSchemeStyles, forbidden) {
			t.Fatalf("menu accent should not keep unsupported legacy value %q", forbidden)
		}
	}
}

// TestSidebarInvertedStyleFlipsBackgroundAndForeground 锁定侧边导航反色必须同时反背景和文字。
func TestSidebarInvertedStyleFlipsBackgroundAndForeground(t *testing.T) {
	appChromeStyles := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css")
	artisticSidebarStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "sidebar.css")

	for _, required := range []string{
		`:root[data-menu="inverted"] .app-sidebar`,
		`--sidebar: var(--foreground);`,
		`--sidebar-foreground: var(--background);`,
		`background-color: var(--sidebar);`,
		`color: var(--sidebar-foreground);`,
	} {
		if !strings.Contains(appChromeStyles, required) {
			t.Fatalf("base sidebar inverted style should flip background and foreground together: missing %q", required)
		}
	}

	for _, required := range []string{
		`:root[data-display-scheme="artistic"][data-menu="inverted"] .app-sidebar`,
		`--sidebar: var(--color-value-rgba-20-16-25-0p95) !important;`,
		`--sidebar-foreground: var(--color-white-alpha-800) !important;`,
		`background: var(--sidebar) !important;`,
		`background-color: var(--sidebar) !important;`,
		`color: var(--color-white-alpha-800) !important;`,
	} {
		if !strings.Contains(artisticSidebarStyles, required) {
			t.Fatalf("artistic sidebar inverted style should not only flip text: missing %q", required)
		}
	}
}

// TestTopbarUsesSharedNavigationAndResponsiveUtilityRow 验证顶栏复用 shared views 导航数据，并在窄屏保留工具区布局。
func TestTopbarUsesSharedNavigationAndResponsiveUtilityRow(t *testing.T) {
	appRoot := readRootFile(t, "frontend", "src", "App.vue")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")
	routes := readRootFile(t, "frontend", "src", "app", "routes.ts")
	views := readRootFile(t, "frontend", "src", "shared", "views.ts")
	appChromeStyles := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css")

	for _, required := range []string{
		"viewComponents",
		"activeViewComponent",
	} {
		if !strings.Contains(appRoot+routes, required) {
			t.Fatalf("App routing should be delegated through app/routes.ts: missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"import HomePage",
		"import LogsPage",
		"import SettingsPage",
		"import AboutPage",
		"v-else-if=\"activeView === 'logs'\"",
	} {
		if strings.Contains(appRoot, forbidden) {
			t.Fatalf("App.vue should not own page import/switch boilerplate: found %q", forbidden)
		}
	}
	for _, required := range []string{
		"pageSubtitle",
		"isWindowControlsDisabled",
		":disabled=\"isWindowControlsDisabled\"",
		"if (isWindowControlsDisabled.value) return",
		"topbar-utility",
		"topbar-actions",
		"window-controls",
		"compact-nav",
		"grid-template-columns: minmax(0, 1fr) auto",
		"grid-template-rows: auto auto",
		"padding-right: var(--window-controls-width)",
		"top: 36px",
		"right: 0",
		"width: calc(var(--window-controls-width) / 3 * 2)",
		".topbar-actions > button",
		"width: calc(var(--window-controls-width) / 3)",
		"grid-column: 1 / -1",
		"grid-row: 1",
		"grid-row: 2",
		"justify-self: stretch",
	} {
		if !strings.Contains(appChrome+views+appChromeStyles, required) {
			t.Fatalf("responsive topbar should share navigation metadata and utility row %q", required)
		}
	}
	if strings.Contains(appChromeStyles, ".window-controls {\n    position: static") {
		t.Fatal("mobile topbar must keep native window controls at the original absolute top-right position")
	}
}

// TestCssOwnershipKeepsBusinessStylesOutOfGlobalTheme 验证公共 CSS 只承载主题 token 和少量跨页面 primitive，页面/组件样式必须归属到对应文件。
func TestCssOwnershipKeepsBusinessStylesOutOfGlobalTheme(t *testing.T) {
	mainTS := readRootFile(t, "frontend", "src", "main.ts")
	styles := readRootFile(t, "frontend", "src", "styles.css")
	artisticSchemeEntry := readRootFile(t, "frontend", "src", "styles", "artistic-scheme.css")
	artisticSchemeStyles := readArtisticSchemeStyles(t)
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")
	aboutPage := readRootFile(t, "frontend", "src", "features", "about", "AboutPage.vue")
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")
	primitiveCard := readRootFile(t, "frontend", "src", "components", "ui", "card", "Card.vue")
	primitiveCardTitle := readRootFile(t, "frontend", "src", "components", "ui", "card", "CardTitle.vue")
	sharedCard := readRootFile(t, "frontend", "src", "shared", "ui", "Card.vue")
	sharedCardTitle := readRootFile(t, "frontend", "src", "shared", "ui", "CardTitle.vue")
	uiPlugin := readRootFile(t, "frontend", "src", "shared", "ui", "plugin.ts")
	featureStyles := strings.Join([]string{
		readRootFile(t, "frontend", "src", "features", "about", "AboutPage.css"),
		readRootFile(t, "frontend", "src", "features", "home", "HomePage.css"),
		readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css"),
		readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.css"),
		readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.css"),
		readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.css"),
	}, "\n")
	design := readRootFile(t, "DESIGN.md")

	for _, required := range []string{
		"@theme inline",
		"--color-sidebar",
		"--runtime-color-sky",
		":root[data-card-border=\"visible\"]",
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("styles.css should keep theme/reset contract %q", required)
		}
	}

	for _, forbidden := range []string{
		".app-shell",
		".about-",
		".software-",
		".business-",
		".settings-",
		".preference-",
		".log-page",
		".dialog-header",
		".status-pill",
		".custom-select-",
		".ui-dialog-content",
		".ui-switch",
		`data-display-scheme="artistic"`,
		`[data-slot="button"]`,
	} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("styles.css should not own page/component selector %q", forbidden)
		}
	}

	for _, forbidden := range []string{
		".ui-",
		".ui-dialog-layer",
		".ui-dialog-content",
		`data-display-scheme="artistic"`,
		"--antd-",
	} {
		if strings.Contains(featureStyles, forbidden) {
			t.Fatalf("feature CSS should not reach into component implementation selector %q", forbidden)
		}
	}

	for _, forbidden := range []string{
		"--card-border",
		"--font-heading",
		"<style scoped>",
	} {
		if strings.Contains(primitiveCard+primitiveCardTitle, forbidden) {
			t.Fatalf("components/ui/card primitives should not carry project style patch %q", forbidden)
		}
	}

	for _, required := range []string{
		".page-stack",
		".content-grid",
		".split-header",
		".section-title-row",
		".data-icon",
		".icon-tone-indigo",
	} {
		if !strings.Contains(layoutStyles, required) {
			t.Fatalf("layout.css should keep shared layout/icon primitive %q", required)
		}
	}

	for _, required := range []string{
		`import './styles/layout.css'`,
		`import './styles/artistic-scheme.css'`,
		`<style scoped src="./HomePage.css">`,
		`<style scoped src="./AboutPage.css">`,
		`<style scoped src="./SettingsPage.css">`,
		`<style scoped src="./LogsPage.css">`,
		`<style scoped src="./UpdateStatusDialog.css">`,
		"settings-control-switch",
		"settings-control-select",
		"CardCompat",
		"CardTitleCompat",
		"--card-border",
		"--font-heading",
		"CSS 归属规则",
		"`frontend/src/styles.css` 只放 Tailwind",
	} {
		if !strings.Contains(mainTS+homePage+aboutPage+settingsPage+logsPage+updateDialog+sharedCard+sharedCardTitle+uiPlugin+design, required) {
			t.Fatalf("frontend CSS ownership should be documented and wired: missing %q", required)
		}
	}

	if strings.Index(mainTS, `import './styles/layout.css'`) > strings.Index(mainTS, `import './styles/artistic-scheme.css'`) {
		t.Fatal("frontend/src/main.ts should import artistic-scheme.css after base and layout styles")
	}

	for _, required := range []string{
		`Artistic 方案入口`,
		`@import "./artistic-scheme/common.css"`,
		`@import "./artistic-scheme/components/button.css"`,
		`@import "./artistic-scheme/components/input.css"`,
		`@import "./artistic-scheme/components/select.css"`,
		`@import "./artistic-scheme/components/switch.css"`,
		`@import "./artistic-scheme/components/card.css"`,
		`@import "./artistic-scheme/components/table.css"`,
		`@import "./artistic-scheme/components/dialog.css"`,
		`@import "./artistic-scheme/components/alert-dialog.css"`,
		`@import "./artistic-scheme/components/badge.css"`,
		`@import "./artistic-scheme/components/progress.css"`,
		`@import "./artistic-scheme/components/tooltip.css"`,
	} {
		if !strings.Contains(artisticSchemeEntry, required) {
			t.Fatalf("artistic-scheme.css should be an import-only theme entry: missing %q", required)
		}
	}

	for _, forbidden := range []string{
		`@import "./artistic-scheme/components/settings.css"`,
		`:root[data-display-scheme="artistic"]`,
		`[data-slot="`,
		`.app-sidebar`,
		`.preference-row`,
	} {
		if strings.Contains(artisticSchemeEntry, forbidden) {
			t.Fatalf("artistic-scheme.css should not own concrete theme rules after folder split: found %q", forbidden)
		}
	}

	for _, required := range []string{
		`Artistic 方案 common`,
		`:root[data-display-scheme="artistic"]`,
		`--artistic-primary: var(--runtime-theme-color, var(--runtime-color-apple-blue))`,
		`--artistic-success: var(--color-value-10b981)`,
		`--artistic-warning: var(--color-value-f59e0b)`,
		`--artistic-error: var(--color-value-ef4444)`,
		`--artistic-shadow`,
		`[data-slot="button"]`,
		`[data-slot="badge"]`,
		`[data-slot="card"]`,
		`[data-slot="input"]`,
		`[data-slot="switch"]`,
		`[data-slot="table-container"]`,
		`[data-slot="dialog-content"]`,
		`[data-slot="alert-dialog-content"]`,
		`[data-slot="progress"]`,
		`[data-slot="tooltip-content"]`,
		`.app-sidebar`,
		`.sidebar-item`,
		`.compact-nav-item`,
		`.topbar`,
		`.settings-row-item`,
		`.aesthetic-field-col`,
	} {
		if !strings.Contains(artisticSchemeStyles, required) {
			t.Fatalf("frontend/src/styles/artistic-scheme should centralise artistic theme rule %q", required)
		}
	}
}

// TestArtisticSchemeComponentCssImportsStayScoped 验证 artistic 主题入口为每个 shadcn primitive 预留覆盖文件，暂未定制的组件允许空文件占位。
func TestArtisticSchemeComponentCssCoversCurrentPrimitives(t *testing.T) {
	entry := readRootFile(t, "frontend", "src", "styles", "artistic-scheme.css")
	uiRoot := rootPath(filepath.Join("frontend", "src", "components", "ui"))
	componentCssRoot := rootPath(filepath.Join("frontend", "src", "styles", "artistic-scheme", "components"))

	entries, err := os.ReadDir(uiRoot)
	if err != nil {
		t.Fatalf("read shadcn primitive directory: %v", err)
	}

	for _, entryDir := range entries {
		if !entryDir.IsDir() {
			continue
		}
		name := entryDir.Name()
		cssPath := filepath.Join(componentCssRoot, name+".css")
		if _, err := os.Stat(cssPath); err != nil {
			t.Fatalf("artistic scheme should provide component override placeholder for frontend/src/components/ui/%s at %s: %v", name, cssPath, err)
		}
		importPath := `@import "./artistic-scheme/components/` + name + `.css"`
		if !strings.Contains(entry, importPath) {
			t.Fatalf("artistic-scheme.css should import component override for %s: missing %q", name, importPath)
		}
	}
}

// TestArtisticSchemeKeepsSwitchAsTrack 验证 artistic 主题不能把 Switch 当作整行表单容器拉伸。
func TestArtisticSchemeKeepsSwitchAsTrack(t *testing.T) {
	switchStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "switch.css")
	commonStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "common.css")

	for _, required := range []string{
		`:root[data-display-scheme="artistic"] [data-slot="switch"]`,
		`min-width:`,
		`height:`,
		`border-radius: var(--radius-full)`,
		`:root[data-display-scheme="artistic"] [data-slot="switch-thumb"]`,
		`inset-inline-start:`,
		`translate: none !important`,
		`width:`,
		`height:`,
		`background: var(--color-white-solid) !important`,
		`:root[data-display-scheme="artistic"] [data-slot="switch"][data-state="checked"] [data-slot="switch-thumb"]`,
	} {
		if !strings.Contains(switchStyles, required) {
			t.Fatalf("components/switch.css should keep switch controls as artistic switch tracks: missing %q", required)
		}
	}

	for _, required := range []string{
		`:root[data-display-scheme="artistic"] .settings-control-switch`,
		`justify-self: end`,
	} {
		if !strings.Contains(commonStyles, required) {
			t.Fatalf("common.css should keep settings switch controls as artistic switch tracks: missing %q", required)
		}
	}

	for _, forbidden := range []string{
		`.settings-control-switch {
    grid-column: 1 / -1;
    width: 100%;`,
	} {
		if strings.Contains(commonStyles, forbidden) {
			t.Fatalf("artistic switch must not be stretched to full row width: found %q", forbidden)
		}
	}
}

// TestArtisticSchemeComponentDetailsMatchThemeTokens 验证 artistic 主题覆盖的不只是颜色，还包括边框、间距、行高、阴影和状态细节。
func TestArtisticSchemeComponentDetailsMatchThemeTokens(t *testing.T) {
	commonStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "common.css")
	artisticSchemeStyles := readArtisticSchemeStyles(t)
	buttonStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "button.css")
	cardStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "card.css")
	dialogStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "dialog.css")
	alertDialogStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "alert-dialog.css")
	badgeStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "badge.css")
	tableStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "table.css")
	tooltipStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "tooltip.css")

	for _, required := range []string{
		`--artistic-primary`,
		`--artistic-shadow`,
		`--artistic-glass-border`,
		`background-image`,
	} {
		if !strings.Contains(commonStyles, required) {
			t.Fatalf("common.css should keep artistic detail token %q", required)
		}
	}
	if !strings.Contains(artisticSchemeStyles, `backdrop-filter`) {
		t.Fatal("artistic scheme component styles should keep glass detail backdrop-filter")
	}

	for _, required := range []string{
		`var(--control-height`,
		`background: var(--artistic-primary)`,
	} {
		if !strings.Contains(buttonStyles, required) {
			t.Fatalf("button.css should use artistic control detail %q", required)
		}
	}
	// 品牌填充不允许再掺固定色渐变；主题色必须原样生效。
	if strings.Contains(buttonStyles, "linear-gradient") {
		t.Fatal("button.css should fill primary buttons with the solid theme color instead of a gradient")
	}

	for _, required := range []string{
		`var(--artistic-glass-border)`,
		`backdrop-filter`,
		`var(--artistic-shadow)`,
	} {
		if !strings.Contains(cardStyles, required) {
			t.Fatalf("card.css should keep artistic card detail %q", required)
		}
	}

	for _, required := range []string{
		`var(--artistic-glass-border)`,
		`var(--artistic-shadow-lg)`,
		`justify-content: flex-end`,
		`:root[data-display-scheme="artistic"] [data-slot="dialog-content"]:not(.ui-dialog-content-top-right)`,
		`translate: -50% -50%`,
	} {
		if !strings.Contains(dialogStyles+alertDialogStyles, required) {
			t.Fatalf("dialog styles should keep artistic modal detail %q", required)
		}
	}

	for _, required := range []string{
		`border-radius: var(--radius-full)`,
		`color-mix`,
	} {
		if !strings.Contains(badgeStyles, required) {
			t.Fatalf("badge.css should keep artistic badge detail %q", required)
		}
	}

	for _, required := range []string{
		`overflow: auto`,
		`text-align: start`,
		`overflow-wrap: break-word`,
	} {
		if !strings.Contains(tableStyles, required) {
			t.Fatalf("table.css should keep table detail %q", required)
		}
	}

	for _, required := range []string{
		`background: var(--popover)`,
		`box-shadow: var(--artistic-shadow-lg)`,
		`word-wrap: break-word`,
	} {
		if !strings.Contains(tooltipStyles, required) {
			t.Fatalf("tooltip.css should keep artistic tooltip detail %q", required)
		}
	}

}

// TestArtisticSchemeStylesShadcnSelect 验证设置页下拉使用 shadcn Select，选中项颜色由主题 CSS 覆盖。
func TestArtisticSchemeStylesShadcnSelect(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	selectStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "select.css")

	if strings.Contains(settingsPage, "UiNativeSelect") {
		t.Fatal("settings page dropdowns should use shadcn Select, not UiNativeSelect")
	}
	for _, required := range []string{
		`<UiSelect`,
		`<UiSelectTrigger class="settings-control-select"`,
		`<UiSelectContent>`,
		`<UiSelectItem`,
		`<UiSelectValue`,
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should compose shadcn Select primitive: missing %q", required)
		}
	}

	for _, required := range []string{
		`:root[data-display-scheme="artistic"] [data-slot="select-trigger"]`,
		`:root[data-display-scheme="artistic"] [data-slot="select-content"]`,
		`:root[data-display-scheme="artistic"] [data-slot="select-item"]`,
		`:root[data-display-scheme="artistic"] [data-slot="select-item"][data-state="checked"]`,
		`:root[data-display-scheme="artistic"] [data-slot="select-trigger"]:hover`,
		`:root[data-display-scheme="artistic"] [data-slot="select-trigger"]:focus-visible`,
		`background-color: var(--card)`,
		`box-shadow: var(--artistic-shadow-lg)`,
		`background: color-mix(in srgb, var(--runtime-accent-color`,
		`color: var(--runtime-accent-color`,
		`font-weight: var(--fw-medium)`,
		`:root[data-display-scheme="artistic"] .preference-color-menu`,
		`:root[data-display-scheme="artistic"] .preference-color-option`,
		`:root[data-display-scheme="artistic"] .preference-color-option.is-selected`,
	} {
		if !strings.Contains(selectStyles, required) {
			t.Fatalf("components/select.css should style shadcn Select and color dropdown with artistic CSS: missing %q", required)
		}
	}

	for _, forbidden := range []string{
		`opacity: 0;`,
		`z-index: -1`,
		`option:checked`,
	} {
		if strings.Contains(selectStyles, forbidden) {
			t.Fatalf("components/select.css must not hide Select or depend on Vue-rendered shell: found %q", forbidden)
		}
	}

	for _, forbidden := range []string{
		`.custom-select-`,
	} {
		if strings.Contains(selectStyles, forbidden) {
			t.Fatalf("components/select.css should not keep dead CustomSelect contract %q", forbidden)
		}
	}
}

// TestDesignDocumentsCommentAndLoggingRequirements 验证 DESIGN.md 继续覆盖注释、日志和临时调试产物规则。
func TestDesignDocumentsCommentAndLoggingRequirements(t *testing.T) {
	design := readRootFile(t, "DESIGN.md")

	for _, required := range []string{
		"代码注释必须覆盖",
		"变量、结构体字段、测试用例",
		"日志必须覆盖运行时、窗口、设置、更新、存储、单实例和进程级错误",
		"`log`、`slog`、`stdout`、`stderr`",
		"日志界面默认折叠筛选",
		"日志表格优先保证内容列可读",
		"临时截图、浏览器截图、调试日志、一次性输出必须写入 `.tmp/`",
	} {
		if !strings.Contains(design, required) {
			t.Fatalf("DESIGN.md should document comment/logging hard rule %q", required)
		}
	}
}

// TestFrontendPackageUsesVueStack 验证前端依赖和 test script 保持 Vue/Pinia/Vitest 栈，不回退到 React。
func TestFrontendPackageUsesVueStack(t *testing.T) {
	packageJSON := readRootFile(t, "frontend", "package.json")

	for _, required := range []string{
		"\"vue\"",
		"\"pinia\"",
		"\"@lucide/vue\"",
		"\"@vitejs/plugin-vue\"",
		"\"vue-tsc\"",
		"\"test\": \"node ./node_modules/vitest/vitest.mjs run tests/frontend --root ..\"",
	} {
		if !strings.Contains(packageJSON, required) {
			t.Fatalf("frontend/package.json should include Vue stack dependency %s", required)
		}
	}

	for _, forbidden := range []string{
		"\"react\"",
		"\"react-dom\"",
		"\"lucide-react\"",
		"\"@vitejs/plugin-react\"",
	} {
		if strings.Contains(packageJSON, forbidden) {
			t.Fatalf("frontend/package.json should not keep React dependency %s after Vue migration", forbidden)
		}
	}
}

// TestHeaderDoesNotExposeRepositoryOrGlobalUpdateAction 验证顶栏不展示仓库信息，也不塞入额外更新操作。
func TestHeaderDoesNotExposeRepositoryOrGlobalUpdateAction(t *testing.T) {
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")

	for _, forbidden := range []string{
		"state.appInfo?.repository",
		"projectMetadata.repositoryUrl",
		"go-desktop-text-size",
		"React</Badge>",
		"shadcn</Badge>",
	} {
		if strings.Contains(appChrome, forbidden) {
			t.Fatalf("frontend/src/features/layout/AppChrome.vue should keep repository and update actions out of the global header: found %q", forbidden)
		}
	}
}

// TestHomePageDoesNotExposeUpdateWorkflowActions 验证首页不承担更新检查、安装或下次启动更新操作。
func TestHomePageDoesNotExposeUpdateWorkflowActions(t *testing.T) {
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")

	for _, forbidden := range []string{
		"run" + "Release" + "Audit(",
		"检查更新",
		"马上更新",
		"下次启动",
	} {
		if strings.Contains(homePage, forbidden) {
			t.Fatalf("frontend/src/features/home/HomePage.vue should keep update workflow actions in the global update dialog: found %q", forbidden)
		}
	}
}

// TestFrontendDoesNotTrackUpdateHistoryOrEvents 验证前端只展示当前更新检查结果，不保留历史、事件或审计状态。
func TestFrontendDoesNotTrackUpdateHistoryOrEvents(t *testing.T) {
	sources := strings.Join([]string{
		readRootFile(t, "frontend", "src", "api", "wails.ts"),
		readRootFile(t, "frontend", "src", "app", "state.ts"),
		readRootFile(t, "frontend", "src", "stores", "app.ts"),
		readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue"),
	}, "\n")

	for _, forbidden := range []string{
		"Release" + "Audit",
		"Audit" + "Result",
		"List" + "Release" + "Audits",
		"list" + "Release" + "Audits",
		"run" + "Release" + "Audit",
		"Update" + "Event",
		"update" + "Events",
		"List" + "Update" + "Events",
		"list" + "Update" + "Events",
	} {
		if strings.Contains(sources, forbidden) {
			t.Fatalf("frontend update flow must not expose history/event/audit concepts: found %q", forbidden)
		}
	}
}

func readRootFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(rootPath(filepath.Join(parts...)))
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Join(parts...), err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func mustRelRoot(t *testing.T, path string) string {
	t.Helper()
	rel, err := filepath.Rel(rootPath("."), path)
	if err != nil {
		t.Fatalf("resolve relative path for %s: %v", path, err)
	}
	return rel
}

func hasAnySuffix(value string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

func assertColorTokenNameMatchesRules(t *testing.T, name string, value string) {
	t.Helper()
	if !colorTokenNameShapePattern.MatchString(name) {
		t.Fatalf("color token %q should use a documented --color-display/black/white/value name", name)
	}

	switch {
	case name == "--color-transparent":
		if strings.ToLower(value) != "transparent" {
			t.Fatalf("%s should define transparent, got %q", name, value)
		}
	case strings.HasPrefix(name, "--color-display-"):
		token := strings.TrimPrefix(name, "--color-display-")
		expectedValue, ok := displayPaletteValueForToken(token)
		if !ok {
			t.Fatalf("display color token %q is not part of the locked 18-color display palette", name)
		}
		if value != expectedValue {
			t.Fatalf("display color token %q should define %q, got %q", name, expectedValue, value)
		}
	case name == "--color-black-solid":
		if strings.ToLower(value) != "#000000" {
			t.Fatalf("%s should define #000000, got %q", name, value)
		}
	case name == "--color-white-solid":
		if strings.ToLower(value) != "#ffffff" {
			t.Fatalf("%s should define #ffffff, got %q", name, value)
		}
	case strings.HasPrefix(name, "--color-black-alpha-") || strings.HasPrefix(name, "--color-white-alpha-"):
		assertMonochromeAlphaTokenNameMatchesValue(t, name, value)
	case strings.HasPrefix(name, "--color-value-"):
		slug, ok := colorValueSlug(value)
		if !ok {
			t.Fatalf("value-derived color token %q has unsupported raw value %q", name, value)
		}
		if expected := "--color-value-" + slug; name != expected {
			t.Fatalf("value-derived color token %q should be named %q for raw value %q", name, expected, value)
		}
	default:
		t.Fatalf("color token %q does not match any documented naming rule", name)
	}
}

func displayPaletteValueForToken(token string) (string, bool) {
	for _, color := range displayPreferencePaletteColors {
		if color.token == token {
			return color.value, true
		}
	}
	return "", false
}

func assertMonochromeAlphaTokenNameMatchesValue(t *testing.T, name string, value string) {
	t.Helper()
	nameMatch := monochromeAlphaTokenNamePattern.FindStringSubmatch(name)
	if nameMatch == nil {
		t.Fatalf("monochrome alpha token %q should use --color-(white|black)-alpha-000 naming", name)
	}
	valueMatch := rgbaColorValuePattern.FindStringSubmatch(strings.ToLower(value))
	if valueMatch == nil {
		t.Fatalf("monochrome alpha token %q should define rgba(...), got %q", name, value)
	}

	expectedChannel := "0"
	if nameMatch[1] == "white" {
		expectedChannel = "255"
	}
	for _, channel := range valueMatch[1:4] {
		if channel != expectedChannel {
			t.Fatalf("%s should use rgba(%s, %s, %s, alpha), got %q", name, expectedChannel, expectedChannel, expectedChannel, value)
		}
	}

	alpha, err := strconv.ParseFloat(valueMatch[4], 64)
	if err != nil {
		t.Fatalf("parse alpha for %s: %v", name, err)
	}
	expectedSuffix := fmt.Sprintf("%03d", int(alpha*1000+0.5))
	if nameMatch[2] != expectedSuffix {
		t.Fatalf("%s alpha suffix should be %s for %q", name, expectedSuffix, value)
	}
}

func colorValueSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(value, "#"):
		return strings.TrimPrefix(value, "#"), true
	case strings.HasPrefix(value, "oklch(") && strings.HasSuffix(value, ")"):
		return "oklch-" + functionalColorSlug(strings.TrimSuffix(strings.TrimPrefix(value, "oklch("), ")")), true
	case strings.HasPrefix(value, "rgb(") && strings.HasSuffix(value, ")"):
		inner := strings.TrimSuffix(strings.TrimPrefix(value, "rgb("), ")")
		colorPart, alphaPart, hasAlpha := strings.Cut(inner, "/")
		components := slugFields(colorPart)
		if len(components) == 0 {
			return "", false
		}
		slug := "rgb-" + strings.Join(components, "-")
		if hasAlpha {
			slug += "-alpha-" + slugColorComponent(alphaPart)
		}
		return slug, true
	case strings.HasPrefix(value, "rgba(") && strings.HasSuffix(value, ")"):
		inner := strings.TrimSuffix(strings.TrimPrefix(value, "rgba("), ")")
		parts := strings.Split(inner, ",")
		if len(parts) != 4 {
			return "", false
		}
		for index, part := range parts {
			parts[index] = slugColorComponent(part)
		}
		return "rgba-" + strings.Join(parts, "-"), true
	default:
		return "", false
	}
}

func functionalColorSlug(value string) string {
	colorPart, alphaPart, hasAlpha := strings.Cut(value, "/")
	slug := strings.Join(slugFields(colorPart), "-")
	if hasAlpha {
		slug += "-alpha-" + slugColorComponent(alphaPart)
	}
	return slug
}

func slugFields(value string) []string {
	fields := strings.Fields(value)
	for index, field := range fields {
		fields[index] = slugColorComponent(field)
	}
	return fields
}

func slugColorComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, ".", "p")
	value = strings.ReplaceAll(value, "%", "pct")
	return value
}

func canonicalColorValue(value string) string {
	value = strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
	if canonical, ok := canonicalHexColorValue(value); ok {
		return canonical
	}
	if canonical, ok := canonicalRGBColorValue(value); ok {
		return canonical
	}
	if canonical, ok := canonicalWhiteOklchValue(value); ok {
		return canonical
	}
	return value
}

func canonicalHexColorValue(value string) (string, bool) {
	if !strings.HasPrefix(value, "#") {
		return "", false
	}
	hex := strings.TrimPrefix(value, "#")
	if len(hex) == 3 {
		hex = strings.Repeat(hex[0:1], 2) + strings.Repeat(hex[1:2], 2) + strings.Repeat(hex[2:3], 2)
	}
	if len(hex) != 6 {
		return "", false
	}
	red, errRed := strconv.ParseInt(hex[0:2], 16, 64)
	green, errGreen := strconv.ParseInt(hex[2:4], 16, 64)
	blue, errBlue := strconv.ParseInt(hex[4:6], 16, 64)
	if errRed != nil || errGreen != nil || errBlue != nil {
		return "", false
	}
	return fmt.Sprintf("rgba(%d,%d,%d,1)", red, green, blue), true
}

func canonicalRGBColorValue(value string) (string, bool) {
	if strings.HasPrefix(value, "rgb(") && strings.HasSuffix(value, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(value, "rgb("), ")")
		colorPart, alphaPart, hasAlpha := strings.Cut(inner, "/")
		channels := strings.Fields(strings.TrimSpace(colorPart))
		if len(channels) != 3 {
			return "", false
		}
		alpha := "1"
		if hasAlpha {
			alpha = normaliseAlphaValue(alphaPart)
		}
		return canonicalRGBAChannels(channels[0], channels[1], channels[2], alpha), true
	}
	if strings.HasPrefix(value, "rgba(") && strings.HasSuffix(value, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(value, "rgba("), ")")
		parts := strings.Split(inner, ",")
		if len(parts) != 4 {
			return "", false
		}
		return canonicalRGBAChannels(parts[0], parts[1], parts[2], normaliseAlphaValue(parts[3])), true
	}
	return "", false
}

func canonicalRGBAChannels(red string, green string, blue string, alpha string) string {
	return fmt.Sprintf(
		"rgba(%s,%s,%s,%s)",
		strings.TrimSpace(red),
		strings.TrimSpace(green),
		strings.TrimSpace(blue),
		strings.TrimSpace(alpha),
	)
}

func canonicalWhiteOklchValue(value string) (string, bool) {
	if !strings.HasPrefix(value, "oklch(") || !strings.HasSuffix(value, ")") {
		return "", false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(value, "oklch("), ")")
	colorPart, alphaPart, hasAlpha := strings.Cut(inner, "/")
	if strings.TrimSpace(colorPart) != "1 0 0" {
		return "", false
	}
	alpha := "1"
	if hasAlpha {
		alpha = normaliseAlphaValue(alphaPart)
	}
	return "rgba(255,255,255," + alpha + ")", true
}

func normaliseAlphaValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "%") {
		percent, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
		if err != nil {
			return value
		}
		return strconv.FormatFloat(percent/100, 'f', -1, 64)
	}
	alpha, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	return strconv.FormatFloat(alpha, 'f', -1, 64)
}

func readArtisticComponentStylesByFile(t *testing.T) map[string]string {
	t.Helper()
	componentDir := rootPath(filepath.Join("frontend", "src", "styles", "artistic-scheme", "components"))
	entries, err := os.ReadDir(componentDir)
	if err != nil {
		t.Fatalf("read artistic component CSS directory: %v", err)
	}

	files := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("frontend", "src", "styles", "artistic-scheme", "components", entry.Name()))
		files[rel] = readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", entry.Name())
	}
	return files
}

// readArtisticSchemeStyles 汇总 artistic 方案目录下的 common 和组件覆盖 CSS，供结构测试检查集中归属。
func readArtisticSchemeStyles(t *testing.T) string {
	t.Helper()
	var files []string
	files = append(files, readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "common.css"))

	componentDir := rootPath(filepath.Join("frontend", "src", "styles", "artistic-scheme", "components"))
	entries, err := os.ReadDir(componentDir)
	if err != nil {
		t.Fatalf("read artistic component CSS directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}
		files = append(files, readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", entry.Name()))
	}
	return strings.Join(files, "\n")
}

func rootPath(path string) string {
	return filepath.Join("..", "..", path)
}
