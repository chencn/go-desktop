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

var rawColorLiteralPattern = regexp.MustCompile(`(?i)#[0-9a-f]{3,8}\b|oklch\([^;\n]*?\)|rgba?\([^;\n]*?\)|hsla?\([^;\n]*?\)`)
var rawNamedColorPattern = regexp.MustCompile(`(?i)(^|[^a-z0-9_-])(transparent|white|black)([^a-z0-9_-]|$)`)
var colorTokenDeclarationPattern = regexp.MustCompile(`(?m)^\s*(--color-[a-z0-9-]+):\s*([^;]+);`)
var colorTokenNameShapePattern = regexp.MustCompile(`^--color-(transparent|(black|white|value)-[a-z0-9-]+)$`)
var monochromeAlphaTokenNamePattern = regexp.MustCompile(`^--color-(black|white)-alpha-([0-9]{3})$`)
var rgbaColorValuePattern = regexp.MustCompile(`^rgba\(\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9.]+)\s*\)$`)

// TestAppRootStaysAsCompositionRoot 验证 App.vue 只装配全局状态、门禁和页面出口，不回流具体业务流程。
func TestAppRootStaysAsCompositionRoot(t *testing.T) {
	appRoot := readRootFile(t, "frontend", "src", "App.vue")

	if lines := strings.Count(appRoot, "\n") + 1; lines > 220 {
		t.Fatalf("frontend/src/App.vue is too large for a composition root: got %d lines, want <= 220", lines)
	}
}

// TestFrontendFeatureBoundariesExist 验证页面、store 和 shared/ui wrapper 的预期目录边界存在。
func TestFrontendFeatureBoundariesExist(t *testing.T) {
	for _, path := range []string{
		filepath.Join("frontend", "src", "App.vue"),
		filepath.Join("frontend", "src", "app", "display.ts"),
		filepath.Join("frontend", "src", "app", "glass.ts"),
		filepath.Join("frontend", "src", "stores", "app.ts"),
		filepath.Join("frontend", "src", "styles", "liquid-glass.css"),
		filepath.Join("frontend", "src", "features", "shared", "GlassPanel.vue"),
		filepath.Join("frontend", "src", "features", "layout", "AppChrome.vue"),
		filepath.Join("frontend", "src", "features", "home", "HomePage.vue"),
		filepath.Join("frontend", "src", "features", "update", "UpdateStatusDialog.vue"),
		filepath.Join("frontend", "src", "features", "logs", "LogsPage.vue"),
		filepath.Join("frontend", "src", "features", "settings", "SettingsPage.vue"),
		filepath.Join("frontend", "src", "features", "about", "AboutPage.vue"),
		filepath.Join("frontend", "src", "features", "license", "LicenseCard.vue"),
	} {
		if _, err := os.Stat(rootPath(path)); err != nil {
			t.Fatalf("expected frontend boundary file %s to exist: %v", path, err)
		}
	}
	if _, err := os.Stat(rootPath(filepath.Join("frontend", "src", "shared", "ui", "plugin.ts"))); !os.IsNotExist(err) {
		t.Fatal("global Ui* registration should stay retired; import shared/ui components directly")
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

// TestGlassPanelIsDirectlyImported 验证全局 Ui* 注册已退役，玻璃卡片由业务页直接 import 项目组件。
func TestGlassPanelIsDirectlyImported(t *testing.T) {
	main := readRootFile(t, "frontend", "src", "main.ts")
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")

	if strings.Contains(main, "uiPlugin") {
		t.Fatal("frontend/src/main.ts should not install a global UI plugin anymore")
	}
	for _, required := range []string{
		"import GlassPanel from '@/features/shared/GlassPanel.vue'",
		"<GlassPanel",
	} {
		if !strings.Contains(homePage, required) {
			t.Fatalf("frontend/src/features/home/HomePage.vue should import the glass wrapper directly: missing %q", required)
		}
	}
}

// TestDialogsDoNotCloseFromOutsideClick 验证业务弹窗阻止外部点击关闭（Element Plus 配置），避免误关。
func TestDialogsDoNotCloseFromOutsideClick(t *testing.T) {
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")

	if !strings.Contains(updateDialog, `:close-on-click-modal="false"`) {
		t.Fatal("update dialog must prevent outside pointer dismissal via close-on-click-modal=false")
	}
}

// TestShadcnCompositionReplacesHandRolledControls 验证设置、日志、更新弹窗和顶栏继续组合 UI 组件库，不回退成手写控件。
func TestShadcnCompositionReplacesHandRolledControls(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")

	for _, required := range []string{
		"el-select",
		"el-option",
		"el-switch",
		"el-input",
		"el-radio-group",
		"el-radio-button",
		"AlertDialog",
		"apple-select",
		"apple-switch",
		"segmented-control",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should use Element Plus component %q", required)
		}
	}

	if strings.Contains(logsPage, `role="table"`) || strings.Contains(logsPage, "log-row") {
		t.Fatalf("logs page should use table primitives instead of a hand-rolled div table")
	}
	for _, required := range []string{"el-table", "el-table-column", `height="100%"`, "apple-table", "el-select", "el-option", "el-input", "AlertDialog", "segmented-control"} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page should use table/UI library primitive %q", required)
		}
	}

	if strings.Contains(updateDialog, `class="dialog-layer"`) || strings.Contains(updateDialog, `class="update-dialog"`) {
		t.Fatalf("update dialog should compose el-dialog instead of raw dialog wrappers")
	}
	if !strings.Contains(updateDialog, "el-dialog") {
		t.Fatalf("update dialog should use el-dialog")
	}
	if !strings.Contains(updateDialog, "scheduleDownloadedUpdateOnStartup") {
		t.Fatalf("update dialog should explicitly schedule next-start update instead of only closing the dialog")
	}

	// 外壳图标按钮用原生 title 提示：设计稿的顶栏按钮没有 EP tooltip 包装，悬浮文案由系统绘制。
	if !strings.Contains(appChrome, `:title="`) {
		t.Fatalf("topbar icon buttons should expose a title hint")
	}
}

// TestWideWindowPagesShareViewportGutters 验证设置页与关于页通栏铺在 .app-main-viewport 的栏距内。
// 设计稿把两页量宽限制在 880px 并居中，那属于 1400px 假窗口；真实窗口最大化后会把整页缩成居中小票，
// 与首页/日志页的边距明显不一致，因此这里明确禁止再回到居中限宽。
func TestWideWindowPagesShareViewportGutters(t *testing.T) {
	for _, page := range []struct {
		name  string
		shell string
		style string
	}{
		{name: "settings", shell: ".settings-stack", style: "frontend/src/features/settings/SettingsPage.css"},
		{name: "about", shell: ".about-container", style: "frontend/src/features/about/AboutPage.css"},
	} {
		css := readRootFile(t, filepath.FromSlash(page.style))
		start := strings.Index(css, page.shell+" {")
		if start < 0 {
			t.Fatalf("%s should define the page shell %q", page.name, page.shell)
		}
		end := strings.Index(css[start:], "\n}")
		if end < 0 {
			t.Fatalf("%s page shell rule is malformed", page.name)
		}
		block := css[start : start+end]
		// 居中回退是真实回归点；max-width 本身不是错，通栏页也可以有可读性上限，因此不做字面量禁令。
		if strings.Contains(block, "margin: 0 auto") {
			t.Fatalf("%s page shell should fill the viewport gutters instead of centering itself", page.name)
		}
		if !strings.Contains(block, "min-width: 0") {
			t.Fatalf("%s page shell should keep min-width: 0 so long content can shrink inside the viewport", page.name)
		}
	}
}

// TestShellMeasuresAndDialogCenteringSkin 锁定三处真实窗口适配：弹窗垂直居中、下拉触发框向左扩张、
// 侧栏宽度随视口夹在上下限之间。设计稿的固定值只属于它的 1400px 假窗口，直接照抄会在最大化窗口走样。
func TestShellMeasuresAndDialogCenteringSkin(t *testing.T) {
	styles := readRootFile(t, "frontend", "src", "styles.css")
	elementSkin := readRootFile(t, "frontend", "src", "styles", "element-plus.css")
	settingsStyles := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.css")

	for _, required := range []string{
		// EP 的 .el-overlay-dialog 只是定高滚动容器，必须补 flex 才会真正垂直居中。
		`.el-overlay-dialog:has(.apple-dialog) {`,
		`align-items: center;`,
		`justify-content: center;`,
		// 设计稿 .dialog-header 是两端对齐，缺 flex 会让页头关闭钮贴着标题漂移。
		`.el-dialog.apple-dialog .el-dialog__header {`,
		// 设计稿 .dialog-footer 是右对齐 + 10px 间距；EP 默认只给 text-align，两枚按钮会挤在一起。
		`.el-dialog.apple-dialog .el-dialog__footer {`,
		`justify-content: flex-end;`,
		`gap: var(--sp-10);`,
		`.el-dialog.apple-dialog .el-dialog__footer .el-button + .el-button {`,
		`margin-left: 0;`,
		// 下拉文案量宽交给页面，触发框才能跟着文字向左扩张而不是整体平移。
		`max-width: var(--apple-select-label-width, var(--sp-220));`,
	} {
		if !strings.Contains(elementSkin, required) {
			t.Fatalf("element-plus.css should keep the real-window dialog/select skin rule %q", required)
		}
	}

	for _, required := range []string{
		"--sidebar-width: clamp(",
		"calc(232px * var(--ui-scale) + (100vw - 1360px) * 0.12)",
		"calc(196px * var(--ui-scale))",
		"calc(300px * var(--ui-scale))",
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("styles.css should keep the fluid sidebar width measure %q", required)
		}
	}

	if !strings.Contains(settingsStyles, "--apple-select-label-width: 100%") {
		t.Fatal("settings rows should let the select trigger grow leftward with its selected label")
	}
}

// TestSegmentedControlSkinNeutralizesElementPlusDefaults 验证分段器皮肤压住 Element Plus 的自带观感：
// EP 的 .el-radio-button__inner 自带 1px outline 与白底，选中态又用 is-active 写成主色实心胶囊，
// 直接照抄设计稿的 .segment-btn 规则会被组件默认样式盖住，因此配色必须走 EP 自己的变量。
func TestSegmentedControlSkinNeutralizesElementPlusDefaults(t *testing.T) {
	elementSkin := readRootFile(t, "frontend", "src", "styles", "element-plus.css")

	for _, required := range []string{
		`.segmented-control .el-radio-button .el-radio-button__inner {`,
		`outline: none;`,
		`.segmented-control .el-radio-button {`,
		`--el-radio-button-checked-bg-color: var(--bg);`,
		`--el-radio-button-checked-text-color: var(--fg);`,
		`--el-radio-button-checked-border-color: var(--color-transparent);`,
		`.el-radio-group.segmented-control .el-radio-button.is-active .el-radio-button__inner {`,
		`box-shadow: var(--segment-knob-shadow) !important;`,
		// 日志级别页签只让文字按级别变色，数量胶囊恒定主色底（设计稿 .log-tab-btn.active strong）。
		`.segmented-control.is-tone-tabs .el-radio-button {`,
		`--el-radio-button-checked-text-color: var(--danger);`,
		`.el-radio-group.segmented-control.is-tone-tabs .el-radio-button.is-active .el-radio-button__inner strong {`,
		`background: var(--accent);`,
	} {
		if !strings.Contains(elementSkin, required) {
			t.Fatalf("element-plus.css segmented skin should keep the design knob instead of EP defaults: missing %q", required)
		}
	}

	// 全项目只有 §3 一套分段器材质：§4 的级别页签只准加数量胶囊与级别文字色，
	// 不许另起轨道底色/采样/滑块投影/悬浮底色，否则同一个项目里两处分段器会长得不一样。
	toneStart := strings.Index(elementSkin, "4. 日志级别页签")
	toneEnd := strings.Index(elementSkin, "5. iOS 玻璃下拉选择器")
	if toneStart < 0 || toneEnd < 0 || toneEnd < toneStart {
		t.Fatal("element-plus.css should keep the log level tab section delimited between §4 and §5")
	}
	toneSection := elementSkin[toneStart:toneEnd]
	for _, forbidden := range []string{
		"background: var(--surface)",
		"backdrop-filter: none",
		"box-shadow: none",
		"box-shadow: 0 var(--sp-1)",
	} {
		if strings.Contains(toneSection, forbidden) {
			t.Fatalf("log level tabs must inherit the §3 segmented track material, found %q", forbidden)
		}
	}

	// EP 的选中态挂在 is-active 上，皮肤不能再依赖 :checked 兄弟选择器，否则优先级不够被蓝底盖回。
	if strings.Contains(elementSkin, "__original-radio:checked + .el-radio-button__inner") {
		t.Fatal("element-plus.css should drive the checked segment through .is-active, not the hidden input sibling")
	}
}

// TestDesignDocumentsElementPlusWorkflow 验证 DESIGN.md 记录 Element Plus 查询、按需注册和全局注册流程。
func TestDesignDocumentsShadcnVueWorkflow(t *testing.T) {
	design := readRootFile(t, "DESIGN.md")

	for _, required := range []string{
		"element-plus-mcp",
		"unplugin-vue-components",
		"ElementPlusResolver",
		"el-config-provider",
		"全局 `Ui*`",
	} {
		if !strings.Contains(design, required) {
			t.Fatalf("DESIGN.md should document the Element Plus workflow and global UI policy, missing %q", required)
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
		"html,\n      body,\n      #app {",
		"border: 0 !important",
		"box-shadow: none !important",
		".boot-spinner",
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
		"background: var(--bg)",
		"box-shadow: none",
		"  overflow: hidden;\n  /* 桌面应用观感",
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

// TestDesignDocumentMatchesCurrentSettingsContract 验证 DESIGN.md 描述当前设置契约。
func TestDesignDocumentMatchesCurrentSettingsContract(t *testing.T) {
	design := readRootFile(t, "DESIGN.md")

	for _, required := range []string{
		`export type DisplaySize = "large" | "default" | "small"`,
		"显示偏好持久化使用 `display.preferences.v3` JSON",
		"固定使用 Lucide",
		"关闭到系统托盘",
		"`1 / 3 / 6 / 12 小时`",
		"go test ./...",
	} {
		if !strings.Contains(design, required) {
			t.Fatalf("DESIGN.md should describe current settings contract %q", required)
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
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")
	appStore := readRootFile(t, "frontend", "src", "stores", "app.ts")

	for _, required := range []string{
		`if (value === 'error') return 'is-danger'`,
		`if (['downloading', 'verifying', 'installing'].includes(value)) return 'is-busy'`,
		`if (['update_available', 'verified', 'pending_install'].includes(value)) return 'is-ready'`,
		`cn('action-icon-btn', updateTone)`,
		"<RefreshCw :size=\"16\" />",
	} {
		if !strings.Contains(chrome, required) {
			t.Fatalf("AppChrome.vue should keep update icon lifecycle mapping %q", required)
		}
	}

	// 生命周期动效归全局原子层：忙碌时图标自转，就绪/失败时状态点脉冲。
	for _, required := range []string{
		`.action-icon-btn.is-busy svg`,
		`animation: spin-update 900ms linear infinite`,
		`@keyframes spin-update {`,
		`transform: rotate(360deg)`,
		`.action-icon-btn.is-ready::after,`,
		`animation: pulse-update 2s infinite ease-in-out`,
	} {
		if !strings.Contains(layoutStyles, required) {
			t.Fatalf("layout.css should keep update icon motion rule %q", required)
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
		"invoke('ShowMainWindow')",
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
		"update-versions",
		"update-info-banner",
		`<strong class="ver-badge">v{{ currentVersion }}</strong>`,
		`<strong class="ver-badge ok">v{{ latestVersion }}</strong>`,
		"当前已是最新",
		"更新失败",
		"重新检查",
		`<p v-if="description" class="update-message">{{ description }}</p>`,
		"if (canInstall.value) return '安装包已下载并通过 SHA-256 校验，可以立即安装。'",
		// 版本横幅只在服务端版本严格更高时才画目标版本，版本相同不能摆出「可升级」的样子。
		"const hasLatest = computed(() => isVersionAhead(latestVersion.value, currentVersion.value))",
		"function isVersionAhead(candidate: string, baseline: string)",
		// 已知存在更高版本时，文案和主按钮都必须指向「立即更新」，不能继续显示技术性的检查消息或「检查更新」。
		"if (status.value === 'update_available') return `发现新版本 v${latestVersion.value}，点击「立即更新」开始下载并校验。`",
		"if (status.value === 'update_available') return '立即更新'",
	} {
		if !strings.Contains(updateDialog, required) {
			t.Fatalf("update dialog should keep user-focused status view %q", required)
		}
	}

	// 更新状态弹窗只讲当前状态与进度，Release 变更日志不再展示。
	for _, forbidden := range []string{
		"releaseNoteItems",
		"变更日志",
		`class="update-notes"`,
	} {
		if strings.Contains(updateDialog, forbidden) {
			t.Fatalf("update dialog should not render the release changelog: found %q", forbidden)
		}
	}

	for _, required := range []string{
		".update-versions",
		".update-info-banner",
	} {
		if !strings.Contains(updateStyles, required) {
			t.Fatalf("update dialog styles should support user-focused layout %q", required)
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
	licenseCard := readRootFile(t, "frontend", "src", "features", "license", "LicenseCard.vue")
	licenseStyles := readRootFile(t, "frontend", "src", "features", "license", "LicensePage.css")
	elementSkin := readRootFile(t, "frontend", "src", "styles", "element-plus.css")

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
		"import LicenseCard from './LicenseCard.vue'",
		"<LicenseCard />",
	} {
		if !strings.Contains(licensePage, required) {
			t.Fatalf("LicensePage.vue 只负责闸门装配，授权卡必须由 LicenseCard 承载：缺少 %q", required)
		}
	}

	for _, required := range []string{
		`type="textarea"`,
		`id="license-key"`,
		`v-model="licenseKey"`,
		`:rows="4"`,
		`class="apple-field is-mono"`,
	} {
		if !strings.Contains(licenseCard, required) {
			t.Fatalf("LicenseCard.vue 授权码输入必须支持多行展示：缺少 %q", required)
		}
	}
	if strings.Contains(licenseCard, `@keydown.enter.prevent="submitLicense"`) {
		t.Fatalf("LicenseCard.vue 授权码多行输入不应拦截 Enter 直接提交")
	}
	for _, required := range []string{
		".license-shell",
		"min-height: 100%",
	} {
		if !strings.Contains(licenseStyles, required) {
			t.Fatalf("LicensePage.css 授权闸门必须铺满视口：缺少 %q", required)
		}
	}
	// 授权码输入的多行观感归 Element Plus 皮肤层：等宽字体与竖向缩放只在 .el-* 内部声明。
	for _, required := range []string{
		".el-textarea.apple-field.is-mono .el-textarea__inner",
		"font-family: var(--font-mono)",
		"resize: vertical",
	} {
		if !strings.Contains(elementSkin, required) {
			t.Fatalf("element-plus.css 授权码输入样式必须适配多行展示：缺少 %q", required)
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
		"恢复默认外观",
		"setThemeMode",
		"setSize",
		"updateCheckIntervalHours",
		"minimizeToTray",
		"alwaysOnTop",
		"logRetentionDays",
		"logLevel",
		"日志级别",
		"autoLaunch",
		"createDesktopShortcut",
		"launchHiddenToTray",
		// 设计稿把每一行都当成常驻设置：开关只受设置加载状态门控，不再互相隐藏或禁用。
		`:disabled="!settingsReady"`,
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("frontend/src/features/settings/SettingsPage.vue should expose setting or display control %q", required)
		}
	}

	for _, required := range []string{
		"displayPreferenceDefaults",
		"resetDisplayPreferences",
		"hydrateDisplayPreferences",
		"exportDisplayPreferences",
		"classList.toggle('dark'",
	} {
		if !strings.Contains(displayState, required) {
			t.Fatalf("frontend/src/app/display.ts should own display preference facade %q", required)
		}
	}

	for _, required := range []string{
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
		"localStorage.",
		"sessionStorage.",
		".setItem(",
	} {
		if strings.Contains(tsCodeOnly(displayState), forbidden) {
			t.Fatalf("frontend/src/app/display.ts should persist display preferences through the backend only: found %q", forbidden)
		}
	}
}

// TestGeneratedBindingsExposeSettingsLogLevelAndDebugStats 保护 Wails 生成绑定和 Go 数据结构同步。
// 打包二进制使用 bindings 参与前端编译，旧绑定会让 logLevel/debug 字段在类型层丢失。
func TestGeneratedBindingsExposeSettingsLogLevelAndDebugStats(t *testing.T) {
	models := readRootFile(t, "frontend", "bindings", "github.com", "chencn", "go-desktop", "app", "models.ts")

	for _, required := range []string{
		`"themeMode": string;`,
		`this["themeMode"] = "";`,
		`"size": string;`,
		`this["size"] = "";`,
		`"logLevel": string;`,
		`this["logLevel"] = "";`,
		`"alwaysOnTop": boolean;`,
		`this["alwaysOnTop"] = false;`,
		// 液态玻璃三项也进了绑定：打包二进制用 bindings 编译，缺字段会让偏好在前端类型层丢失。
		`"backdrop": boolean;`,
		`this["backdrop"] = false;`,
		`"lgStyle": string;`,
		`this["lgStyle"] = "";`,
		`"lgIntensity": number;`,
		`this["lgIntensity"] = 0;`,
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

// TestSettingsPageOnlyContainsEditableSettings 验证设置页保留可编辑项的装配契约。
func TestSettingsPageOnlyContainsEditableSettings(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	settingsStyles := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.css")
	wailsAPI := readRootFile(t, "frontend", "src", "api", "wails.ts")
	projectMetadata := readRootFile(t, "frontend", "src", "shared", "project.ts")

	for _, required := range []string{
		"检查间隔",
		"关闭到系统托盘",
		"窗口置顶",
		"开机自启",
		"创建桌面快捷图标",
		"开机自启时隐藏到托盘",
		"保留周期",
		"const updateIntervalOptions = [1, 3, 6, 12]",
		"`${hours} 小时`",
		"[-1, '永不清理']",
		"updateIntervalOptions",
		"normaliseUpdateCheckIntervalHours",
		"githubProxyBase",
		"GitHub 更新代理",
		`:model-value="draft.githubProxyBase"`,
		`class="settings-row-item"`,
		`persistSettingsPatch({ githubProxyBase: String($event) })`,
		`:model-value="draft.alwaysOnTop"`,
		`persistSettingsPatch({ alwaysOnTop: Boolean($event) })`,
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
		"@media (max-width: 767px)",
		"flex-direction: column",
		".settings-row-item:has(.apple-switch)",
		"--apple-select-width: 100%",
		"justify-content: space-between",
	} {
		if !strings.Contains(settingsStyles, required) {
			t.Fatalf("settings page mobile rows should stack compound rows while switch rows stay one line: missing %q", required)
		}
	}

	// 设计稿逐行给定顺序：业务行为在前、时间与日志策略居中、外观偏好独立成组，行序即信息架构。
	mockRowOrder := []string{
		"关闭到系统托盘",
		"窗口置顶",
		"开机自启",
		"自启时隐藏到系统托盘",
		"创建桌面快捷图标",
		"系统更新源",
		"GitHub 更新代理加速",
		"自动更新检查间隔",
		"日志保留周期",
		"控制台日志级别",
		"主题模式",
		"控件尺寸",
		"极光折射流光背景",
		"液态玻璃折射风格",
		"液态折射光强",
	}
	previousIndex := -1
	for _, label := range mockRowOrder {
		index := strings.Index(settingsPage, `<span class="settings-row-label">`+label+`</span>`)
		if index < 0 {
			t.Fatalf("settings page should keep the design row %q", label)
		}
		if index < previousIndex {
			t.Fatalf("settings row %q should follow the design order (previous=%d, current=%d)", label, previousIndex, index)
		}
		previousIndex = index
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
		"主题模式",
		"控件尺寸",
		"themeOptions",
		"sizeOptions",
		"asThemeMode",
		"asSize",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should expose current display preference control %q", required)
		}
	}

	for _, required := range []string{
		"hydrateDisplayPreferences",
		"exportDisplayPreferences",
		"setThemeMode",
		"setSize",
		"resetDisplayPreferences",
	} {
		if !strings.Contains(displayState, required) {
			t.Fatalf("display state should persist current display preference axis %q", required)
		}
	}
}

// TestGlassPreferencesPersistThroughBackend 锁定液态玻璃三项偏好的持久化链路：
// backdrop / lgStyle / lgIntensity 与主题、尺寸一样走后端 display.preferences.v3，前端不再另设 localStorage 副本。
func TestGlassPreferencesPersistThroughBackend(t *testing.T) {
	domain := readRootFile(t, "internal", "desktopapp", "display", "preferences.go")
	runtimeAPI := readRootFile(t, "internal", "desktopapp", "runtime", "display_preferences.go")
	serviceAPI := readRootFile(t, "app", "service.go")
	displayState := readRootFile(t, "frontend", "src", "app", "display.ts")
	glassState := readRootFile(t, "frontend", "src", "app", "glass.ts")
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")

	for _, required := range []string{
		"Backdrop bool `json:\"backdrop\"`",
		"LgStyle string `json:\"lgStyle\"`",
		"LgIntensity int `json:\"lgIntensity\"`",
		"DefaultGlassStyle = \"fresnel\"",
		"GlassIntensityMin = 30",
		"GlassIntensityMax = 100",
		"allowedGlassStyles = stringSet(\"fresnel\", \"frosted\", \"sheen\")",
	} {
		if !strings.Contains(domain, required) {
			t.Fatalf("display/preferences.go should model the glass axes %q", required)
		}
	}

	for _, source := range []struct{ name, content string }{
		{name: "runtime/display_preferences.go", content: runtimeAPI},
		{name: "app/service.go", content: serviceAPI},
	} {
		for _, required := range []string{"Backdrop:", "LgStyle:", "LgIntensity:"} {
			if !strings.Contains(source.content, required) {
				t.Fatalf("%s should carry the glass axes through the typed facade: missing %q", source.name, required)
			}
		}
	}

	for _, required := range []string{
		"hydrateGlassPreferences(preferences)",
		"...exportGlassPreferences()",
		"resetGlassPreferences()",
		"lgStyle: GlassStyle",
		"lgIntensity: number",
	} {
		if !strings.Contains(displayState, required) {
			t.Fatalf("app/display.ts should compose the glass axes into the persisted snapshot: missing %q", required)
		}
	}

	// 光学偏好只允许一条持久化链路；再留一份本地存储副本会让前后端口径分叉，切档后重启就“回滚”。
	// 断言收窄到真实存储 API 调用，注释里提到这些技术名词不算违规。
	for _, forbidden := range []string{"localStorage.", "sessionStorage.", "indexedDB.", ".setItem(", ".getItem("} {
		if strings.Contains(tsCodeOnly(glassState), forbidden) {
			t.Fatalf("app/glass.ts must persist nothing on its own (the backend owns the glass axes): found %q", forbidden)
		}
	}

	for _, required := range []string{
		"asGlassBackdrop(Boolean($event))",
		"glass.setStyle(value as GlassStyle)",
		"glass.setIntensity((event.target as HTMLInputElement).valueAsNumber)",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should drive the glass rows through the shared persistence path: missing %q", required)
		}
	}
	if count := strings.Count(settingsPage, "persistDisplayPreferences()"); count < 5 {
		t.Fatalf("theme, size and the three glass axes should all persist the same snapshot, got %d calls", count)
	}
	if !strings.Contains(appChrome, "glass.setBackdrop(next)") || !strings.Contains(appChrome, "await appStore.persistDisplayPreferences()") {
		t.Fatal("the topbar aurora toggle must persist the glass preference like the theme toggle does")
	}
}

// TestDisplayCssUsesCurrentColorAxes 验证 styles.css 把 HIG 语义令牌桥接到 Element Plus 变量，
// 主色固定 iOS System Blue 且色阶全部由 color-mix 从 --accent 派生，明暗两态都跟随 html.dark。
func TestDisplayCssUsesCurrentColorAxes(t *testing.T) {
	styles := strings.ReplaceAll(readRootFile(t, "frontend", "src", "styles.css"), "\r\n", "\n")

	for _, required := range []string{
		`--accent: var(--color-value-007aff);`,
		`--el-color-primary: var(--accent);`,
		`--el-color-primary-light-3: color-mix(in srgb, var(--el-color-primary) 70%, var(--color-white-solid));`,
		`--el-color-primary-light-9: color-mix(in srgb, var(--el-color-primary) 10%, var(--color-white-solid));`,
		`--el-color-primary-dark-2: color-mix(in srgb, var(--el-color-primary) 80%, var(--color-black-solid));`,
		`--el-bg-color: var(--bg);`,
		`--el-bg-color-page: var(--surface);`,
		`--el-text-color-primary: var(--fg);`,
		`--el-border-color-lighter: var(--border-soft);`,
		`--el-color-danger: var(--danger);`,
		`--el-component-size: var(--control-height);`,
		`html.dark {`,
		`--accent: var(--color-value-0a84ff);`,
		// 夜间色阶向深空底色相调，否则 EP 默认往白里混会得出近白悬浮底。
		`--el-color-primary-light-3: color-mix(in srgb, var(--el-color-primary) 70%, var(--surface));`,
		`--el-color-primary-light-9: color-mix(in srgb, var(--el-color-primary) 10%, var(--surface));`,
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("styles.css should bridge HIG tokens to Element Plus variables %q", required)
		}
	}

	// 语义令牌只允许从 colors.css 的 raw 变量派生，反向桥接会让色值口径分叉；远程字体由 DESIGN.md 明令禁止。
	for _, forbidden := range []string{
		`--background: var(--el-bg-color-page);`,
		`--primary: var(--el-color-primary);`,
		"fonts.googleapis",
		"@import url(",
	} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("styles.css should not bridge tokens backwards or load remote fonts: found %q", forbidden)
		}
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

	if !strings.Contains(styles, "var(--color-value-007aff)") {
		t.Fatal("styles.css should consume raw palette colors through the global color token file")
	}

	for _, required := range []string{
		"Naming rules:",
		"--color-transparent: 只用于 transparent。",
		"--color-transparent",
		"--color-(white|black)-solid",
		"--color-(white|black)-alpha-080",
		"alpha 后缀固定三位",
		"--color-value-<raw-value-slug>",
		"同一个 raw 色值只能定义一次",
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
			return nil
		}
		if !hasAnySuffix(entry.Name(), ".css", ".vue", ".ts") {
			return nil
		}
		rel := filepath.ToSlash(mustRelRoot(t, path))
		if rel == frontendColorTokenFile {
			return nil
		}
		// *.upstream.css 是逐字节拷贝的上游材质文件（见 frontend/src/styles/liquid-glass.upstream.css），
		// 色值由上游定义、升级时整文件重拷，不允许为了过本审计去改写上游文件，因此这里显式豁免。
		if strings.HasSuffix(rel, ".upstream.css") {
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

// TestDisplayPreferencePaletteUsesSingleRuntimeColorSource 锁定主色只在 colors.css 写死一次、styles.css 只通过变量引用。
func TestDisplayPreferencePaletteUsesSingleRuntimeColorSource(t *testing.T) {
	styles := strings.ReplaceAll(readRootFile(t, "frontend", "src", "styles.css"), "\r\n", "\n")
	colorTokens := strings.ReplaceAll(readRootFile(t, filepath.FromSlash(frontendColorTokenFile)), "\r\n", "\n")
	settingsStyles := strings.ReplaceAll(readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.css"), "\r\n", "\n")

	colorTokenVar := "--color-value-007aff"
	colorTokenDeclaration := colorTokenVar + ": #007aff;"
	if count := strings.Count(colorTokens, colorTokenDeclaration); count != 1 {
		t.Fatalf("iOS system blue should be hard-coded exactly once in %s, got %d", frontendColorTokenFile, count)
	}
	// 主色链只允许一次写死：colors.css 出 raw 值 -> --accent 语义令牌 -> Element Plus 桥接。
	accentRule := "--accent: var(" + colorTokenVar + ");"
	if count := strings.Count(styles, accentRule); count != 1 {
		t.Fatalf("styles.css should define the semantic accent through %s exactly once, got %d", colorTokenVar, count)
	}
	primaryRule := "--el-color-primary: var(--accent);"
	if count := strings.Count(styles, primaryRule); count != 1 {
		t.Fatalf("styles.css should bridge Element Plus primary through the accent token exactly once, got %d", count)
	}
	if strings.Contains(settingsStyles, "#007aff") {
		t.Fatalf("SettingsPage.css must not repeat hard-coded iOS system blue %q; use var(%s)", "#007aff", colorTokenVar)
	}
}

// TestColorfulIconToneStaysSemanticAndSkipsActiveNavigation 验证行内图标统一走 .sq-icon-badge 渐变徽标，
// 色相只由语义 tone 类名决定，且不再是可配置的显示偏好维度。
func TestColorfulIconToneStaysSemanticAndSkipsActiveNavigation(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")
	homeStyles := readRootFile(t, "frontend", "src", "features", "home", "HomePage.css")

	for _, required := range []string{
		`sq-icon-badge size-md indigo`,
		`sq-icon-badge size-md blue`,
		`sq-icon-badge size-md teal`,
		`sq-icon-badge size-md purple`,
		`sq-icon-badge size-md cyan`,
		`sq-icon-badge size-md orange`,
		`sq-icon-badge size-md green`,
		`sq-icon-badge size-md gray`,
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should give each row a semantic badge tone %q", required)
		}
	}
	for _, required := range []string{
		"sq-icon-badge size-md ${item.tone}",
		"stat-icon-wrap ${stat.tone}",
		"badge-apple ${softwareSummary.tone}",
		"tone: 'indigo'",
		"tone: 'green'",
		"tone: 'orange'",
		"tone: 'purple'",
	} {
		if !strings.Contains(homePage, required) {
			t.Fatalf("home page icons should resolve tone through the badge class %q", required)
		}
	}
	// 侧栏导航徽标的 tone 来自 shared views 元数据，激活态只改底和文字色，不换图标配色。
	for _, required := range []string{
		"cn('sq-icon-badge', 'size-md', item.tone)",
		`cn('nav-item', props.activeView === item.key && 'active')`,
		`.nav-item.active {`,
	} {
		if !strings.Contains(appChrome+layoutStyles, required) {
			t.Fatalf("navigation badge should keep tone-driven gradient and a text-only active state: missing %q", required)
		}
	}
	// 设计稿的侧栏条目只有图标 + 文字：日志数量属于筛选结果，只能出现在日志页的筛选按钮上。
	if strings.Contains(appChrome, "nav-item-badge") {
		t.Fatal("AppChrome.vue sidebar navigation must not carry a log count badge")
	}
	if !strings.Contains(logsPage, `class="nav-item-badge"`) {
		t.Fatal("LogsPage.vue should keep the active filter count badge on the filter button")
	}
	// 渐变色板集中定义在令牌层，页面只写 tone 类名。
	for _, required := range []string{
		`.sq-icon-badge.indigo {`,
		`.sq-icon-badge.green {`,
		`background: var(--tile-indigo);`,
		`background: var(--tile-green);`,
		`box-shadow: var(--tile-shadow);`,
	} {
		if !strings.Contains(layoutStyles, required) {
			t.Fatalf("layout.css should keep the centralized gradient badge palette %q", required)
		}
	}
	for _, required := range []string{
		`.stat-icon-wrap.indigo {`,
		`.stat-icon-wrap.orange {`,
		`box-shadow: var(--tile-shadow-stat);`,
	} {
		if !strings.Contains(homeStyles, required) {
			t.Fatalf("home stat cards should reuse the gradient tile tokens %q", required)
		}
	}
}

// TestHomePageFocusesOnRuntimeStatusAndBusinessStats 验证首页职责只承载软件运行状态和业务统计，不回退成快捷入口页。
func TestHomePageFocusesOnRuntimeStatusAndBusinessStats(t *testing.T) {
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")
	homeStyles := readRootFile(t, "frontend", "src", "features", "home", "HomePage.css")
	appRoot := readRootFile(t, "frontend", "src", "App.vue")
	views := readRootFile(t, "frontend", "src", "shared", "views.ts")

	for _, required := range []string{
		"services-grid",
		"GlassPanel",
		"stat-card",
		"WebView",
		"应用服务",
		"SQLite 数据库",
		"网络",
		"stats-grid",
		"dashboard-grid",
		"demoStats",
		"demo-bar-chart",
		".demo-bar-fill",
		"distribution-list",
		".dist-track",
		".dist-bar",
		"正常",
		"异常",
		"检测中",
		"软件运行状态、业务统计和样例图表",
		"badge-apple ${softwareSummary.tone}",
	} {
		if !strings.Contains(homePage+views+homeStyles, required) {
			t.Fatalf("home page should focus on runtime status and business stats: missing %q", required)
		}
	}

	for _, forbidden := range []string{
		"workflowCards",
		"workflow-grid",
		"GetUpdateStatus",
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
		"about-container",
		"about-hero-card",
		"meta-grid-2",
		"details-card",
		"detail-row",
		"运行时长",
		"Release 来源",
		"本地数据与路径",
		"运行时与平台",
		"Release 与分发",
		"Go 核心",
		"Wails 框架",
		"公开仓库",
		"API 代理",
	} {
		if !strings.Contains(aboutPage, required) {
			t.Fatalf("about page should own runtime/release/tech information %q", required)
		}
	}
}

// TestLogsPageUsesThemeAlignedPageLayout 验证日志页保留专注模式，同时服从全局主题 token 与 el-table 结构。
func TestLogsPageUsesThemeAlignedPageLayout(t *testing.T) {
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	logStyles := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.css")
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")

	for _, required := range []string{
		`class="log-page"`,
		`<section class="log-command-card" aria-label="日志筛选工具条">`,
		`class="log-command-toolbar"`,
		`class="log-command-tabs segmented-control is-tone-tabs"`,
		`el-radio-button`,
		`class="log-command-search"`,
		`applySeverityFilter(String($event))`,
		"filtersOpen = ref(false)",
		"fullscreen = ref(false)",
		"aria-expanded",
		"aria-pressed",
		"Maximize2",
		"专注模式",
		"退出专注",
		"classList.toggle('is-log-focus', enabled)",
		"classList.remove('is-log-focus')",
		"log-filter-panel",
		"log-collapsed-toolbar",
		"log-mobile-list",
		"log-mobile-card",
		"log-message-cell",
		"log-stream-panel",
		"AlertDialog",
	} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page should keep themed fullscreen structure %q", required)
		}
	}

	for _, required := range []string{
		`RefreshCw class="log-tool-icon is-success"`,
		`TimerReset class="log-tool-icon is-indigo"`,
		`SlidersHorizontal class="log-tool-icon is-indigo"`,
		`Maximize2 class="log-tool-icon"`,
		`Search class="log-search-icon"`,
	} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page tool icons should keep semantic icon tone %q", required)
		}
	}

	for _, required := range []string{
		".log-page",
		".log-command-card",
		".log-command-toolbar",
		".log-command-search",
		".log-filter-panel",
		".log-collapsed-toolbar",
		".log-stream-panel",
		".log-mobile-list",
		".log-mobile-card",
		".log-mobile-card__meta",
		".log-table-shell",
		".log-pagination-card",
	} {
		if !strings.Contains(logStyles, required) {
			t.Fatalf("logs page themed layout should define %q", required)
		}
	}
	// 专注模式由 LogsPage 在 <html> 上切 is-log-focus，外壳在 layout.css 里收起导航列与顶栏。
	// 顶栏只收成一条透明拖拽带、三颗窗口控制钮仍留在 DOM 里等悬停浮出：
	// 无边框窗口的拖拽区与最小化/最大化/关闭全挂在顶栏上，整条 display:none 会把窗口控制一起砍掉。
	for _, required := range []string{
		"html.is-log-focus .app-sidebar",
		"html.is-log-focus .mobile-tabbar",
		"html.is-log-focus .app-topbar",
		"html.is-log-focus .topbar-left",
		"html.is-log-focus .traffic-lights-right",
		"html.is-log-focus .app-topbar:hover .traffic-lights-right",
	} {
		if !strings.Contains(layoutStyles, required) {
			t.Fatalf("layout.css should collapse shell chrome but keep window controls in log focus mode: missing %q", required)
		}
	}
	if !strings.Contains(layoutStyles, "display: none") {
		t.Fatal("layout.css should collapse the shell navigation with display:none in log focus mode")
	}
	if strings.Contains(logStyles, "z-index: 2147483647") {
		t.Fatal("log fullscreen must not use max z-index; AlertDialog overlays still need to appear above focused logs")
	}
	mobileStart := strings.Index(logStyles, "@media (max-width: 767px)")
	if mobileStart < 0 {
		t.Fatal("logs page should keep the shared phone breakpoint override")
	}
	mobileStyles := logStyles[mobileStart:]
	for _, required := range []string{
		`.log-command-toolbar {
    flex-direction: column;`,
		`.log-command-toolbar > .log-command-tabs {
    width: 100%;`,
		`--apple-select-width: 100%;`,
		`.log-table-shell {
    display: none;
  }`,
		`.log-mobile-list {
    display: flex;`,
	} {
		if !strings.Contains(mobileStyles, required) {
			t.Fatalf("logs page mobile layout should define %q", required)
		}
	}
}

// TestLogsPageKeepsDensePaginationAndTableDetails 验证日志页保留分页信息密度与运行时日志表格模型，
// 表格观感统一由 element-plus.css 的 .apple-table 皮肤提供。
func TestLogsPageKeepsDensePaginationAndTableDetails(t *testing.T) {
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	logStyles := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.css")
	elementPlusStyles := readRootFile(t, "frontend", "src", "styles", "element-plus.css")

	for _, required := range []string{
		"calculateLogPageSize",
		"ResizeObserver",
		"logTableRef",
		"logListRef",
		"logPaginationRef",
		"totalPages",
		"displayedLogPage",
		"displayedPageSize",
		"watch(logPageSize",
		// 设计稿的分页摘要要显式交代后端最低门禁，否则筛空时无法区分「没有日志」和「被门禁挡住」。
		"`每页 ${displayedPageSize.value} 条，当前第 ${displayedLogPage.value} / ${totalPages.value} 页 (门禁过滤后共 ${appStore.logTotal} 条记录，系统最低门禁: ≥ ${gate})`",
		"`暂无匹配日志 (门禁级别: ≥ ${gate})`",
		`layout="prev, pager, next"`,
		`ref="logTableRef"`,
		`ref="logListRef"`,
		`ref="logPaginationRef"`,
		`class="log-pagination-card"`,
		`class="log-mobile-card"`,
		`class="log-mobile-card__message"`,
		`class="log-time-cell"`,
		`class="log-scope-tag"`,
		`class="log-message-cell"`,
		"if (!isDesktop) return 12",
		`:empty-text="logLayoutReady ? emptyLogsText : ''"`,
		"`log-level-badge ${logLevelClass(row.severity)}`",
		"logLevelClass",
	} {
		if !strings.Contains(logsPage, required) {
			t.Fatalf("logs page should keep dense pagination/table detail %q", required)
		}
	}

	for _, required := range []string{
		".log-table-shell",
		".log-pagination-card",
		".log-mobile-list",
		".log-mobile-card",
		".log-mobile-card__message",
		".log-time-cell",
		".log-scope-tag",
		".log-level-badge",
		".log-level-badge.is-error",
		".log-level-badge.is-warning",
		".log-level-badge.is-info",
		".log-level-badge.is-debug",
	} {
		if !strings.Contains(logStyles, required) {
			t.Fatalf("logs page styles should keep dense pagination/table detail %q", required)
		}
	}
	for _, required := range []string{
		`.el-table.apple-table th.el-table__cell`,
		`.el-table.apple-table td.el-table__cell`,
		`--el-table-border-color: var(--border-soft);`,
		`--el-table-header-bg-color: var(--table-head-fill);`,
		`--el-table-row-hover-bg-color: var(--table-hover);`,
		// 设计稿 .apple-table 有偶数行斑马底，EP 靠 stripe 开关 + fill-color-lighter 才有同等观感
		`--el-fill-color-lighter: var(--table-zebra);`,
		`text-transform: uppercase;`,
		`letter-spacing: 0.05em;`,
	} {
		if !strings.Contains(elementPlusStyles, required) {
			t.Fatalf("logs page table should mirror the shared apple-table skin %q", required)
		}
	}
	if !strings.Contains(logsPage, "stripe") {
		t.Fatalf("logs page table should enable stripe so the design's even-row zebra background shows up")
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
		".log-filter-panel",
		".log-collapsed-toolbar",
		".filter-item-label",
		".log-command-toolbar",
		".log-command-search",
		"min-width: calc(180px * var(--ui-scale))",
		"flex-wrap: wrap",
		"@media (max-width: 767px)",
	} {
		if !strings.Contains(logStyles, required) {
			t.Fatalf("logs page layout should define %q", required)
		}
	}
}

// TestTopbarUsesSharedNavigationAndResponsiveUtilityRow 验证顶栏复用 shared views 元数据，桌面工具区与手机 TabBar 各自成行。
func TestTopbarUsesSharedNavigationAndResponsiveUtilityRow(t *testing.T) {
	appRoot := readRootFile(t, "frontend", "src", "App.vue")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")
	routes := readRootFile(t, "frontend", "src", "app", "routes.ts")
	views := readRootFile(t, "frontend", "src", "shared", "views.ts")
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")

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
		"navigationGroups",
		"isWindowControlsDisabled",
		`:disabled="isWindowControlsDisabled"`,
		"topbar-left",
		"topbar-right",
		"topbar-actions",
		"topbar-title-group",
		"traffic-lights-right",
		"mobile-tabbar",
		"mobile-sidebar-backdrop",
		".app-topbar {",
		".topbar-left {",
		".topbar-right {",
		".traffic-lights-right {",
		".mobile-tabbar {",
		"--topbar-height",
		"--tabbar-height",
	} {
		if !strings.Contains(appChrome+views+layoutStyles, required) {
			t.Fatalf("responsive topbar should share navigation metadata and utility row %q", required)
		}
	}
	// 手机端外壳：侧栏变抽屉、出现底部 TabBar，断点由 layout.css 单独持有。
	for _, required := range []string{
		"@media (max-width: 767px)",
		"transform: translateX(-100%)",
		".app-sidebar.mobile-open {",
		"width: var(--sidebar-drawer-width)",
		`.mobile-tabbar {
    display: block;`,
	} {
		if !strings.Contains(layoutStyles[strings.Index(layoutStyles, "@media (max-width: 767px)"):], required) {
			t.Fatalf("layout.css phone breakpoint should switch the shell to drawer + tabbar: missing %q", required)
		}
	}
}

// TestCssOwnershipKeepsBusinessStylesOutOfGlobalTheme 验证公共 CSS 只承载主题 token 和少量跨页面 primitive，页面/组件样式必须归属到对应文件。
func TestCssOwnershipKeepsBusinessStylesOutOfGlobalTheme(t *testing.T) {
	mainTS := readRootFile(t, "frontend", "src", "main.ts")
	styles := readRootFile(t, "frontend", "src", "styles.css")
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")
	aboutPage := readRootFile(t, "frontend", "src", "features", "about", "AboutPage.vue")
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	logsPage := readRootFile(t, "frontend", "src", "features", "logs", "LogsPage.vue")
	updateDialog := readRootFile(t, "frontend", "src", "features", "update", "UpdateStatusDialog.vue")
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
		"--el-color-primary",
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

	for _, required := range []string{
		".app-window",
		".app-topbar",
		".app-sidebar",
		".mobile-tabbar",
		".sq-icon-badge",
		".badge-apple",
		".split-header",
		".section-title-row",
		".traffic-btn",
		".detail-row",
		".ver-badge",
	} {
		if !strings.Contains(layoutStyles, required) {
			t.Fatalf("layout.css should keep shared layout/icon primitive %q", required)
		}
	}

	// 断点集中在 layout.css，页面 CSS 只允许出现本页在手机端的微调。
	for _, required := range []string{
		"@media (max-width: 1023px)",
		"@media (max-width: 767px)",
	} {
		if !strings.Contains(layoutStyles, required) {
			t.Fatalf("layout.css should own the tablet/phone breakpoints %q", required)
		}
	}

	for _, required := range []string{
		`import './styles/layout.css'`,
		`import './styles/liquid-glass.css'`,
		`import './styles/element-plus.css'`,
		`<style scoped src="./HomePage.css">`,
		`<style scoped src="./AboutPage.css">`,
		`<style scoped src="./SettingsPage.css">`,
		`<style scoped src="./LogsPage.css">`,
		`<style scoped src="./UpdateStatusDialog.css">`,
		"class=\"apple-switch\"",
		"class=\"apple-select\"",
		"import GlassPanel from '@/features/shared/GlassPanel.vue'",
		"--card-border",
		"CSS 归属规则",
		"`frontend/src/styles.css` 只放 Tailwind",
	} {
		if !strings.Contains(mainTS+styles+homePage+aboutPage+settingsPage+logsPage+updateDialog+design, required) {
			t.Fatalf("frontend CSS ownership should be documented and wired: missing %q", required)
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
	} {
		if strings.Contains(appChrome, forbidden) {
			t.Fatalf("frontend/src/features/layout/AppChrome.vue should keep repository information out of the global header: found %q", forbidden)
		}
	}
}

// TestHomePageDoesNotExposeUpdateWorkflowActions 验证首页不承担更新检查、安装或下次启动更新操作。
func TestHomePageDoesNotExposeUpdateWorkflowActions(t *testing.T) {
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")

	for _, forbidden := range []string{
		"检查更新",
		"马上更新",
		"下次启动",
	} {
		if strings.Contains(homePage, forbidden) {
			t.Fatalf("frontend/src/features/home/HomePage.vue should keep update workflow actions in the global update dialog: found %q", forbidden)
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

// tsCodeOnly 抹掉 TS 源码里的注释内容（保留行数），用于「不得调用某 API」这类禁令断言。
// 注释描述的是实现意图，不是行为：把 "localStorage" 写在注释里不会让偏好多出第二份存储。
func tsCodeOnly(source string) string {
	block := regexp.MustCompile(`(?s)/\*.*?\*/`)
	line := regexp.MustCompile(`//[^\n]*`)
	kept := block.ReplaceAllStringFunc(source, func(match string) string {
		return strings.Repeat("\n", strings.Count(match, "\n"))
	})
	return line.ReplaceAllString(kept, "")
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
		t.Fatalf("color token %q should use a documented --color-black/white/value name", name)
	}

	switch {
	case name == "--color-transparent":
		if strings.ToLower(value) != "transparent" {
			t.Fatalf("%s should define transparent, got %q", name, value)
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

func rootPath(path string) string {
	return filepath.Join("..", "..", path)
}
