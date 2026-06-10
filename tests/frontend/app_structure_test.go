// 文件职责：验证前端模块边界、设计文档合同、shadcn/artistic 结构和更新/授权页面职责。

package frontend_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		"UiNativeSelect",
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

	for _, forbidden := range []string{
		"Release 来源",
		"GitHub Owner",
		"GitHub Repo",
		"GitHub API 代理",
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
		"开机自启",
		"创建桌面快捷图标",
		"开机自启时隐藏到托盘",
		"保留周期",
		"const updateIntervalOptions = [1, 3, 6, 12]",
		"{{ hours }} 小时",
		"updateIntervalOptions",
		"normaliseUpdateCheckIntervalHours",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should keep editable setting %q", required)
		}
	}

	closeIndex := strings.Index(settingsPage, "<strong>关闭到系统托盘</strong>")
	autoLaunchIndex := strings.Index(settingsPage, "<strong>开机自启</strong>")
	intervalIndex := strings.Index(settingsPage, "<strong>自动更新检查间隔</strong>")
	retentionIndex := strings.Index(settingsPage, "<strong>每日日志保留周期</strong>")
	if closeIndex < 0 || autoLaunchIndex < 0 || intervalIndex < 0 || retentionIndex < 0 || !(closeIndex < autoLaunchIndex && autoLaunchIndex < intervalIndex && intervalIndex < retentionIndex) {
		t.Fatalf("business settings should order window/startup behavior before time-based selects: close=%d auto=%d interval=%d retention=%d", closeIndex, autoLaunchIndex, intervalIndex, retentionIndex)
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
		"组件高矮风格",
		"themeOptions",
		"baseOptions",
		"themeColorOptions",
		"iconToneOptions",
		"chartOptions",
		"radiusOptions",
		"textOptions",
		"densityOptions",
		"cardBorderOptions",
		"menuOptions",
		"styleOptions",
	} {
		if !strings.Contains(settingsPage, required) {
			t.Fatalf("settings page should expose current display preference control %q", required)
		}
	}

	for _, required := range []string{
		"hydrateDisplayPreferences",
		"exportDisplayPreferences",
		"setThemeColor",
		"setChartColor",
		"setIconTone",
		"setMenu",
	} {
		if !strings.Contains(displayState, required) {
			t.Fatalf("display state should persist current display preference axis %q", required)
		}
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
		`:root[data-theme-color="yellow"] { --runtime-theme-color`,
		`:root[data-accent-color="yellow"] { --runtime-accent-color`,
		`:root[data-chart-color="yellow"] { --runtime-chart-color`,
		`:root:not([data-theme-color="neutral"])[data-theme-color]`,
		`--primary: var(--runtime-theme-color);`,
		`:root:not([data-accent-color="neutral"])[data-accent-color]`,
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
		"fonts.googleapis",
		"@import url(",
		"data-heading-font",
		"data-body-font",
	} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("styles.css should not keep stale color axis rule %q", forbidden)
		}
	}

	accentRuleStart := strings.Index(styles, `:root:not([data-accent-color="neutral"])[data-accent-color]`)
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

// TestColorfulIconToneStaysSemanticAndSkipsActiveNavigation 验证彩色图标只用于语义点缀，不覆盖当前激活导航状态。
func TestColorfulIconToneStaysSemanticAndSkipsActiveNavigation(t *testing.T) {
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	homePage := readRootFile(t, "frontend", "src", "features", "home", "HomePage.vue")
	appChrome := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.vue")
	layoutStyles := readRootFile(t, "frontend", "src", "styles", "layout.css")
	appChromeStyles := readRootFile(t, "frontend", "src", "features", "layout", "AppChrome.css")

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

	for _, required := range []string{
		"<Teleport to=\"body\"",
		":disabled=\"!fullscreen\"",
		`cn('page-stack log-page'`,
		"<UiCard>",
		`UiCardContent class="log-page-main"`,
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
		`.log-fullscreen > [data-slot="card"]`,
		".log-fullscreen .log-page-main",
		".log-fullscreen.has-open-filters .log-page-main",
		`.log-fullscreen .log-stream-panel [data-slot="table-container"]`,
		"z-index: 2147483647",
		"height: 100dvh",
		"background: var(--background)",
	} {
		if !strings.Contains(logStyles+appChromeStyles, required) {
			t.Fatalf("logs page themed layout should define %q", required)
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
		"topbar-utility",
		"topbar-actions",
		"compact-nav",
		"grid-template-columns: minmax(0, 1fr) auto",
		"display: contents",
		"grid-column: 2",
		"grid-row: 1",
		"grid-column: 1 / -1",
		"grid-row: 2",
		"justify-self: stretch",
	} {
		if !strings.Contains(appChrome+views+appChromeStyles, required) {
			t.Fatalf("responsive topbar should share navigation metadata and utility row %q", required)
		}
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
		"--runtime-color-yellow",
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
		`--artistic-primary: #f97316`,
		`--artistic-success: #10b981`,
		`--artistic-warning: #f59e0b`,
		`--artistic-error: #ef4444`,
		`--artistic-shadow`,
		`[data-slot="button"]`,
		`[data-slot="badge"]`,
		`[data-slot="card"]`,
		`[data-slot="input"]`,
		`.ui-native-select`,
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
		`.ui-native-select`,
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
		`border-radius: 999px`,
		`:root[data-display-scheme="artistic"] [data-slot="switch-thumb"]`,
		`inset-inline-start:`,
		`translate: none !important`,
		`width:`,
		`height:`,
		`background: #ffffff !important`,
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
		`linear-gradient`,
	} {
		if !strings.Contains(buttonStyles, required) {
			t.Fatalf("button.css should use artistic control detail %q", required)
		}
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
	} {
		if !strings.Contains(dialogStyles+alertDialogStyles, required) {
			t.Fatalf("dialog styles should keep artistic modal detail %q", required)
		}
	}

	for _, required := range []string{
		`border-radius: 999px`,
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
func TestArtisticSchemeStylesNativeSelectOnly(t *testing.T) {
	nativeSelect := readRootFile(t, "frontend", "src", "shared", "ui", "NativeSelect.vue")
	settingsPage := readRootFile(t, "frontend", "src", "features", "settings", "SettingsPage.vue")
	selectStyles := readRootFile(t, "frontend", "src", "styles", "artistic-scheme", "components", "select.css")

	for _, required := range []string{
		`<select`,
		`ui-native-select`,
		`v-bind="delegatedAttrs"`,
		`@change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"`,
	} {
		if !strings.Contains(nativeSelect, required) {
			t.Fatalf("NativeSelect should stay as the original native select wrapper: missing %q", required)
		}
	}

	for _, forbidden := range []string{
		`MutationObserver`,
		`document.addEventListener`,
		`role="listbox"`,
		`@pointerdown.stop`,
		`ChevronDown`,
	} {
		if strings.Contains(nativeSelect, forbidden) {
			t.Fatalf("NativeSelect must not implement a custom dropdown in Vue; CSS-only override required, found %q", forbidden)
		}
	}

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
		`:root[data-display-scheme="artistic"] .ui-native-select`,
		`:root[data-display-scheme="artistic"] [data-slot="select-trigger"]`,
		`:root[data-display-scheme="artistic"] [data-slot="select-content"]`,
		`:root[data-display-scheme="artistic"] [data-slot="select-item"]`,
		`:root[data-display-scheme="artistic"] [data-slot="select-item"][data-state="checked"]`,
		`:root[data-display-scheme="artistic"] .ui-native-select:hover`,
		`:root[data-display-scheme="artistic"] [data-slot="select-trigger"]:hover`,
		`:root[data-display-scheme="artistic"] .ui-native-select:focus-visible`,
		`:root[data-display-scheme="artistic"] [data-slot="select-trigger"]:focus-visible`,
		`:root[data-display-scheme="artistic"] .ui-native-select:disabled`,
		`appearance: none !important`,
		`background-color: var(--card)`,
		`box-shadow: var(--artistic-shadow-lg)`,
		`background: color-mix(in srgb, var(--runtime-accent-color`,
		`color: var(--runtime-accent-color`,
		`font-weight: 500`,
		`padding: 0 32px 0 12px !important`,
		`background-image: url("data:image/svg+xml`,
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
		`.ui-native-select option`,
		`option:checked`,
		`.ui-native-select option {
  background-color: var(--runtime-accent-color`,
	} {
		if strings.Contains(selectStyles, forbidden) {
			t.Fatalf("components/select.css must not hide native Select or depend on Vue-rendered shell: found %q", forbidden)
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
	return string(data)
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
