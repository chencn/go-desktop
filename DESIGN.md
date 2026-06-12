# go-desktop 设计规范

本文档是 `go-desktop` 的 UI/UX、主题、组件和工程边界规范。`README.md` 只描述项目入口、目录、命令和维护流程；具体界面规则以本文档为准。

## 1. 当前结论

`go-desktop` 是一个基于 Wails3、Vue 3、TypeScript、Tailwind v4 和 shadcn-vue 源码组件的中文桌面工具项目。

界面目标：

- 桌面优先，信息密度高，布局克制。
- 保留中文产品体验，不做营销页式首屏。
- 主题、色彩、字号、密度和菜单风格可配置。
- 更新、日志、设置、关于各自职责清楚，不互相塞功能。

工程目标：

- shadcn-vue primitive first。
- 业务页面组合组件，不重造基础控件。
- 设置持久化走后端 SQLite KV。
- 日志写每日 JSONL 文件，SQLite 不保存日志。
- 更新状态只保存当前生命周期，不保存历史事件。

## 2. 技术栈和配置

核心技术：

| 层 | 技术 |
| --- | --- |
| 桌面运行时 | Wails3 |
| 后端 | Go |
| 前端 | Vue 3 + TypeScript + Vite |
| 样式 | Tailwind v4 + CSS variables |
| UI primitive | shadcn-vue + reka-ui |
| 图标 | `@lucide/vue` |
| 配置存储 | SQLite KV，表语义为 `config_items` |
| 日志 | 每日 JSONL 文件 + 内存 ring buffer |
| 更新 | GitHub Release 或本地静态 manifest |

shadcn-vue 配置以 [frontend/components.json](frontend/components.json) 为准：

```json
{
  "$schema": "https://shadcn-vue.com/schema.json",
  "style": "new-york",
  "typescript": true,
  "tailwind": {
    "config": "",
    "css": "src/styles.css",
    "baseColor": "neutral",
    "cssVariables": true
  },
  "iconLibrary": "lucide"
}
```

规则：

- `frontend/src/components/ui/` 只放 shadcn-vue CLI 生成的官方 primitive。
- `frontend/src/shared/ui/` 放项目兼容 wrapper、全局 `Ui*` 注册和少量跨页面 UI 适配。
- `frontend/src/features/**` 放业务页面、页面私有组件和页面私有 CSS。
- `frontend/src/lib/utils.ts` 是 `cn()` 唯一来源。
- `frontend/src/main.ts` 必须注册 `shared/ui/plugin.ts`。
- 本项目采用 shadcn-vue skill / CLI / docs 工作流：进入 UI 改造前先检查 `components.json`，再执行 `shadcn-vue info --json` 获取项目配置和已安装组件；新增或覆盖 primitive 前先用 `shadcn-vue docs <component>` 查询官方文档，再用 `shadcn-vue add <component>` 写入源码组件。
- 全局 Ui* 只在 `frontend/src/shared/ui/plugin.ts` 注册，业务页默认组合全局 `UiButton`、`UiCard`、`UiDialog`、`UiTooltip` 等 primitive，不在页面里重复造基础控件。

## 3. 信息架构

应用只有四个主页面，更新固定在右上角弹窗入口。

| 页面 | 职责 | 禁止 |
| --- | --- | --- |
| 概览 | 应用名称、版本、启动时间、日志健康摘要、主要入口 | 放更新主操作 |
| 日志 | 日志文件、筛选、统计、表格、分页、清理当前筛选范围 | 放设置项、更新策略 |
| 设置 | 可修改的业务设置和显示偏好 | 放只读信息、技术栈、路径、策略说明 |
| 关于 | 应用元数据、Release 来源、运行环境、本地路径、技术栈、显示偏好摘要 | 放可编辑控件 |
| 更新弹窗 | 检查、下载、校验、立即安装、下次启动安装、诊断 | 成为独立页面或导航项 |

导航规则：

- 侧边栏和窄屏导航只包含 `概览 / 日志 / 设置 / 关于`。
- 右上角保留日夜切换按钮和更新状态按钮。
- 更新状态按钮按状态显示 busy / ready / danger 视觉态。
- 主窗口使用 Wails frameless 模式隐藏系统标题栏和系统最小化/最大化/关闭按钮，由 `AppChrome` 顶栏右上角固定渲染自定义窗口控制。
- 夜间切换和更新状态按钮固定在窗口控制按钮左侧，和最小化/最大化/关闭组成同一条右上控制带。
- 顶栏标题区域是窗口拖拽热区，所有按钮、导航和弹窗触发区域必须显式保持 no-drag。
- 自定义关闭按钮仍调用 Wails 窗口关闭链路，继续服从 `minimizeToTray` 的关闭到托盘规则；最小化按钮不进入托盘。
- Windows 主窗口使用 Wails frameless 原生装饰，`DisableFramelessWindowDecorations` 必须保持 `false`，由 Go 层 `CustomTheme.WindowTheme.BorderColour` 提供系统窗口外框色；前端 CSS 只负责 WebView 内部界面，不能承担操作系统外框。

## 4. 设置模型

设置分为两类：业务设置和显示偏好。两者都通过后端 API 保存到 SQLite 配置项，但语义边界不同。

### 4.1 业务设置

业务设置来自 `internal/desktopapp/settings`，当前字段：

| 字段 | Key | 默认值 | UI |
| --- | --- | --- | --- |
| `updateSource` | `update.source` | `github` | 更新源，下拉选择 `github / local` |
| `githubProxyBase` | `github.proxy_base` | `https://gh-proxy.com` | GitHub 更新代理，仅在更新源为 `github` 时展示 |
| `updateCheckIntervalHours` | `update.check_interval_hours` | `3` | 检查间隔，`1 / 3 / 6 / 12 小时` |
| `minimizeToTray` | `window.minimize_to_tray` | `true` | 关闭到系统托盘 |
| `logRetentionDays` | `log.retention_days` | `30` | `7 / 30 / 60 / 90 / 180 / 365 / 永不清理` |
| `logLevel` | `log.level` | `info` | `debug / info / warning / error` |
| `autoLaunch` | `startup.auto_launch` | `false` | 开机自启 |
| `createDesktopShortcut` | `startup.create_desktop_shortcut` | `true` | 创建桌面快捷图标 |
| `launchHiddenToTray` | `startup.launch_hidden_to_tray` | `false` | 开机自启时隐藏到托盘 |

业务设置页规则：

- 每一行只有一个可编辑控件。
- `launchHiddenToTray` 仅在 `autoLaunch` 开启时可编辑。
- `minimizeToTray` 只影响点击关闭按钮；点击最小化仍进入任务栏。
- GitHub Release 的 owner/repo 来自项目元数据，不作为业务设置保存或修改。
- `githubProxyBase` 只影响 GitHub Release API、安装资产和 `.sha256` 下载；`local` 更新源不使用该代理。
- 写配置失败必须返回错误，前端展示保存失败。
- 保存开机自启和桌面快捷方式时，同步 Windows 系统集成；系统集成失败要回滚内存和 SQLite 配置。

### 4.2 显示偏好

显示偏好来自 `frontend/src/app/display.ts` 和后端 `display.preferences.v2` JSON 配置项。它们控制 DOM token，不写入后端业务 `Settings` 结构。

| 轴 | 值 | DOM |
| --- | --- | --- |
| Display Scheme | `shadcn / artistic` | `data-display-scheme` |
| Mode | `light / dark` | `.dark`、`data-theme="day|night"` |
| UI Style | `reka / vega / nova / maia / lyra / mira / luma / sera` | `data-style` |
| Base Color | `neutral / stone / zinc / mauve / olive / mist / taupe` | `data-base-color` |
| Theme Color | 品牌色集合 | `data-theme-color` |
| Accent Color | 品牌色集合，设置页托管跟随 Theme Color | `data-accent-color` |
| Chart Color | 18 色集合 | `data-chart-color` |
| Icon Tone | `default / colorful` | `data-icon-tone` |
| Menu | 持久化支持 `default / inverted / default-translucent / inverted-translucent`；设置页只暴露 `default / inverted` | `data-menu` |
| Menu Accent | `subtle / bold`，设置页显示为侧边导航强调 | `data-menu-accent` |
| Radius | `default / none / small / medium / large` | `data-radius` + `--radius` |
| Density | `compact / comfortable` | `data-density` |
| Text Size | `small / normal / medium / large` | `data-text-size` |
| Card Border | `visible / soft / hidden` | `data-card-border` |

品牌色集合：

`neutral`、`amber`、`apple-blue`、`blue`、`cyan`、`emerald`、`indigo`、`orange`、`pink`、`rose`、`sky`、`teal`。

18 色 token 兼容集合：

`neutral`、`stone`、`zinc`、`mauve`、`olive`、`mist`、`taupe`、`amber`、`apple-blue`、`blue`、`cyan`、`emerald`、`indigo`、`orange`、`pink`、`rose`、`sky`、`teal`。

两套显示方案默认 profile：

| 字段 | shadcn 默认 | artistic 默认 |
| --- | --- | --- |
| 界面风格 (uiStyle) | 极简 Vega (`vega`) | 极简 Vega (`vega`) |
| 基础色调 (baseColor) | 灰阶 (`neutral`) | 灰阶 (`neutral`) |
| 品牌主题色 (themeColor) | 灰阶 (`neutral`) | Apple 蓝 (`apple-blue`) |
| 品牌辅助色 (accentColor) | 灰阶 (`neutral`) | Apple 蓝 (`apple-blue`) |
| 图表颜色 (chartColor) | 灰阶 (`neutral`) | Apple 蓝 (`apple-blue`) |
| 图标色调 (iconTone) | 默认颜色 (`default`) | 彩色图标 (`colorful`) |
| 侧边导航风格 (menu) | 默认 (`default`) | 默认 (`default`) |
| 侧边导航强调 (menuAccent) | 轻强调 (`subtle`) | 强强调 (`bold`) |
| 圆角大小 (radius) | 中 (`medium`) | 中 (`medium`) |
| 界面密度 (density) | 舒展 (`comfortable`) | 舒展 (`comfortable`) |
| 字号 (textSize) | 正常 (`normal`) | 正常 (`normal`) |
| 卡片边框 (cardBorder) | 清晰 (`visible`) | 清晰 (`visible`) |

当前前端类型契约：

```ts
export type BaseColor = "neutral" | "stone" | "zinc" | "mauve" | "olive" | "mist" | "taupe"
```

Theme / Accent / Chart Color 是持久化模型中的三条显示轴：Theme 控制主强调，Accent 控制辅助强调，Chart Color 只控制统计图表 token。设置页品牌主题色只展示品牌色集合；中性灰阶色调里除 `neutral` 外，`stone`、`zinc`、`mauve`、`olive`、`mist`、`taupe` 只属于 Base Color，不出现在品牌主题色里。设置页品牌辅助色显示为 disabled 托管项，跟随品牌主题色，视觉使用同色系浅一号，不提供独立选择入口。Icon Library 暂不作为设置项，未接入多图标包渲染前固定使用 Lucide。

显示偏好规则：

- 显示偏好持久化使用 `display.preferences.v2` JSON。
- `displayScheme=shadcn` 使用 `profiles.shadcn`，设置页当前不提供品牌辅助色独立选择入口。
- `displayScheme=artistic` 使用 `profiles.artistic` 并叠加温暖落日毛玻璃主题覆盖；覆盖范围包括常用控件和应用骨架，不只是颜色。
- Artistic 下使用同一套显示偏好 token；主题层通过 `frontend/src/styles/artistic-scheme/**` 解释这些 token，不在业务页面里重造控件。
- 切换显示方案只切换当前方案，不把一个方案的 profile 写进另一个方案；切回原方案必须恢复该方案之前的偏好。
- 显示偏好不使用拆分 `display.*` KV。
- 切换后立即更新 DOM，再异步保存到 SQLite 配置项。
- 保存失败时显示错误，但不阻塞即时预览。
- 切换不允许触发页面 reload。
- `themeColor` 控制主按钮、选中态、焦点环和关键进度。
- `accentColor` 控制次级强调和 hover / focus 辅助强调；设置页由 `themeColor` 托管，当前只显示 disabled 的品牌辅助色浅色盘。
- `chartColor` 只服务图表和统计色，不偷用 Theme 或 Accent。
- Icon Library 暂不作为设置项，未接入多图标包渲染前固定使用 Lucide。
- 禁止远程字体加载，字体族固定系统字体。
- Artistic 主题的外观范围包括按钮、输入、shadcn Select、原生 select 兜底、开关、卡片、表格、弹窗、Badge、菜单、顶栏、focus、hover、active 和暗色状态。
- Artistic 主题默认主色和图表色为 Apple 蓝，基础色调为 Neutral，并叠加毛玻璃面板；具体变量以 `frontend/src/styles/artistic-scheme/common.css` 为准。

## 5. 主题和 CSS 归属规则

`frontend/src/styles.css` 只放 Tailwind import、主题 token、全局 reset、focus 和根级媒体变量，只允许承载：

- Tailwind v4 import。
- `@theme inline` token 映射。
- shadcn-vue CSS variables。
- `:root`、`.dark`、显示偏好 token。
- 全局 reset、focus、reduced motion。

`frontend/src/styles/layout.css` 只允许承载：

- 跨页面布局 primitive，例如 `.page-stack`、`.content-grid`、`.split-header`。
- 跨页面图标语义工具，例如 `.nav-icon`、`.data-icon`、`.icon-tone-*`。

页面私有 CSS 必须放在 `frontend/src/features/**` 相邻文件中，例如：

- `frontend/src/features/home/HomePage.css`
- `frontend/src/features/logs/LogsPage.css`
- `frontend/src/features/settings/SettingsPage.css`
- `frontend/src/features/update/UpdateStatusDialog.css`

主题/组件覆盖必须放在 `frontend/src/styles/artistic-scheme/**`，例如：

- `[data-slot]` primitive 覆盖放到对应 `components/*.css`。
- shadcn Select 的 `[data-slot="select-*"]` 外观、选中项和原生 select 兜底放到 `components/select.css`；禁止在主题外重写下拉视觉。
- `.settings-*`、`.app-sidebar`、`.topbar` 等非 shadcn primitive 的 artistic 项目级覆盖放到 `common.css`；页面布局仍保留在 `features/settings/SettingsPage.css`。

禁止项：

- 页面前缀样式进入 `styles.css`，例如 `.settings-*`、`.log-*`、`.about-*`。
- 组件选择器进入 `styles.css`，例如页面私有下拉选择器、`[data-slot]`。
- 在业务页面手写一套普通按钮、表格、弹窗、分段控件。
- 卡片嵌套卡片。
- 远程字体 `@import url(...)`。
- 布局依赖负 letter spacing，`letter-spacing` 保持 `0`。

## 6. 组件规则

优先级：

1. 已有 `frontend/src/components/ui/*` primitive。
2. shadcn-vue 官方组件，通过 CLI 添加或覆盖。
3. `frontend/src/shared/ui/*` 组合官方 primitive 做项目 wrapper。
4. 页面私有特殊组件放 `frontend/src/features/**`。

当前主要组件：

| 组件 | 位置 | 用途 |
| --- | --- | --- |
| `UiButton` | `shared/ui` 全局注册 | 主操作、次操作、图标按钮 |
| `UiCard` | `shared/ui` / `components/ui/card` | 页面内容分组 |
| `UiBadge` | `components/ui/badge` | 状态、版本、计数 |
| `UiSwitch` | `components/ui/switch` | 二元设置 |
| `UiSelect` / `UiSelectTrigger` / `UiSelectContent` / `UiSelectItem` | `components/ui/select` | 设置页和常规下拉 |
| `UiNativeSelect` | `shared/ui/NativeSelect.vue` | 原生 select 兼容兜底 |
| 设置页色盘按钮 | `features/settings/SettingsPage.vue` | 平铺 swatch 选择主题色和图表色 |
| `UiDialog` | `components/ui/dialog` | 更新状态弹窗 |
| `UiAlertDialog` | `components/ui/alert-dialog` | 清空日志、危险确认 |
| `UiTooltip` | `components/ui/tooltip` | 顶栏图标按钮解释 |
| `UiTable` | `components/ui/table` | 日志表 |
| `UiProgress` | `components/ui/progress` | 下载和校验进度 |

组件规则：

- shadcn primitive 不 import store。
- `shared/ui` wrapper 只做项目 API 适配，不承载业务流程。
- 页面业务组件可以局部 import 图标和私有小组件。
- 普通命令按钮优先 `UiButton`。
- 危险确认使用 `AlertDialog`，禁止 `window.confirm`。
- 表格型数据使用 `Table` primitive。
- 简单选择器优先使用 shadcn Select；只有必须保留浏览器原生行为时才使用 `UiNativeSelect` 兜底。
- 图标默认 `16-20px`。
- 按钮默认高度 `36px`，紧凑按钮 `32px`，图标按钮宽高一致。
- `displayScheme=artistic` 时，常用控件按 artistic 主题覆盖：按钮、输入、shadcn Select、原生 select 兜底、开关、卡片、表格、弹窗、Badge、菜单、顶栏和暗色状态都在 `frontend/src/styles/artistic-scheme/**` 内维护。

## 7. 更新链路

更新只维护当前状态，不保存历史。

后端 API：

| API | 行为 |
| --- | --- |
| `CheckUpdate()` | 检查 GitHub Release 或 local manifest；发现新版本且有 SHA256 时可后台下载 |
| `DownloadUpdate()` | 下载并校验最近一次检查结果 |
| `GetUpdateStatus()` | 返回当前生命周期状态；必要时读取已校验安装包状态 |
| `InstallDownloadedUpdate()` | 校验本地文件后启动静默安装器，成功启动后退出应用 |
| `ScheduleDownloadedUpdateOnStartup()` | 写入 `data/updates/pending.json`，下次启动安装 |
| `InstallPendingUpdateOnStartup()` | 启动期消费 pending 状态，成功或失败后清理 |

状态：

`idle`、`update_available`、`downloading`、`verifying`、`verified`、`pending_install`、`installing`、`install_started`、`no_update`、`skipped`、`error`。

规则：

- 缺 SHA256 时禁止下载和安装。
- 已校验安装包状态可以持久化，但只表示当前可安装包，不是历史记录。
- pending 更新只写 `data/updates/pending.json`，不写 SQLite。
- 启动期只允许消费 `data/updates/` 缓存目录内的安装包。
- 启动期清理 `data/updates/` 中版本小于等于当前应用版本的 `pending.json` / `verified.json` 和版本目录。
- 安装前必须重新计算 SHA256。
- SHA256 不匹配时删除本地安装包并清理 pending / verified 状态。
- 更新入口不出现在侧边栏和窄屏导航。

## 8. 日志模型

日志来源：

- 文件日志：`data/logs/*.log`，内容为每日 JSONL。
- 内存 ring buffer：只服务当前前端视图和即时反馈。
- SQLite：只保存配置项，不保存日志、日志历史、更新历史。
- `crash.log` 是 Runtime 创建前的早期崩溃兜底文件；启动时先导入上次异常退出尾部，再裁剪到最近尾部，避免无限增长。

日志页规则：

- 支持来源、级别、关键词、日志文件筛选。
- 支持统计摘要、动态 pageSize 分页和当前筛选范围清理；分页条显示总数、每页条数和当前页/总页。
- 表格时间、来源、级别列固定宽度，内容列占满剩余宽度并允许换行；级别列使用 `UiBadge` 表达状态。
- 空态放在表格 body 内，不在表格外额外拆一块空状态。
- 路径、日志内容、版本号必须 `min-width: 0` 和 `overflow-wrap: anywhere`。
- 清空日志必须使用 `AlertDialog`。
- 自动刷新、手动刷新和专注模式属于日志页顶层工具。

## 9. 数据和路径

运行数据默认放在可执行文件所在目录的 `data/` 下，开发兜底为当前工作目录的 `data/`。

| 路径 | 用途 |
| --- | --- |
| `data/go-desktop.db` | SQLite KV 配置 |
| `data/logs/` | 每日 JSONL 日志和早期 `crash.log` |
| `data/updates/` | 更新安装包、verified / pending 状态 |
| `data/updates/pending.json` | 下次启动安装状态 |

读配置失败：

- 后端降级为默认值。
- 日志记录 warning。

写配置失败：

- 必须向前端返回错误。
- 前端显示保存失败。

## 10. 布局和响应式

桌面优先，窄屏只保证可用。

| 宽度 | 规则 |
| --- | --- |
| `>= 1600px` | sidebar 约 `280px`，页面最大宽度可扩到 `1560px` |
| `1280-1599px` | sidebar 约 `256px`，内容区不能居中过窄 |
| `1100-1279px` | sidebar 可收窄，复杂页面单列 |
| `980-1099px` | 内容必须单列 |
| `< 980px` | 隐藏 sidebar，显示顶部 compact nav |

规则：

- `.app-shell` 固定两列：sidebar + content。
- `.content-scroll` 是唯一页面滚动容器。
- 所有 grid 子项必须 `min-width: 0`。
- 日志表、路径和长文本可以内部横滚或换行，但不能撑破 app shell。
- 五个主要交互区域不能出现横向溢出：概览、日志、设置、关于、更新弹窗。

## 11. 工程硬约束

- 先读代码，先找根因，禁止未核对就下结论。
- 精确修改，避免无关重写。
- 测试只能放在独立 `tests/` 模块；生产目录旁边不放 Go `_test.go`。
- `scripts/` 只放可执行工具和工具依赖代码，不放测试用例、截图、临时日志或一次性调试脚本。
- 临时截图、浏览器截图、调试日志、一次性输出必须写入 `.tmp/`。
- PC 端 `1440×900` 和窄屏视口都要覆盖前端视觉验证。
- 代码注释必须覆盖模块边界、导出 API、结构体字段、页面状态变量、测试用例意图、失败原因、复杂流程和工程约束。
- 变量、结构体字段、测试用例只要承载业务语义或约束，就必须说明存在原因、影响范围和默认值。
- 日志必须覆盖运行时、窗口、设置、更新、存储、单实例和进程级错误；`log`、`slog`、`stdout`、`stderr` 都必须接入统一日志框架并写每日 JSONL 文件。
- 日志界面默认折叠筛选，日志表格优先保证内容列可读。
- 设置页只放能修改状态的控件；只读信息、路径、Release 来源和技术栈放关于页或诊断弹窗。
- 禁止只记录偏好但不改变实际界面/行为的假设置。
- 禁止远程字体加载。
- UI 调试优先 Browser / Chrome 插件，禁止把 Playwright 引入仓库依赖或脚本。
- 未经确认不执行删除、覆盖、回滚、清理类危险操作。
- 未经用户要求不自动运行测试。
- `project.metadata.json` 是产品元数据源；同步生成内容由 `scripts/sync_project_metadata.go -sync` 负责。
- 根 `Taskfile.yml` 的生成模板变更必须同步维护 `scripts/sync_project_metadata.go`。

## 12. 验收清单

文档验收：

- README 覆盖项目定位、目录结构、数据约定、命令、更新链路、元数据同步和组件边界。
- DESIGN 覆盖页面职责、设置模型、显示偏好、组件规则、日志、更新和工程硬约束。
- DESIGN 不把未落地的功能写成已实现能力。

代码验收：

- `frontend/components.json` 存在并可被 shadcn-vue CLI 读取。
- `components/ui` 保持 CLI primitive 专属。
- `shared/ui/plugin.ts` 注册全局 `Ui*`。
- 设置页业务设置和显示偏好边界清楚。
- 更新入口只在右上角弹窗。
- SQLite 只保存配置项。
- 日志文件写入 `data/logs/`。
- pending 更新写入 `data/updates/pending.json`。

可选验证命令：

```powershell
cd D:\app\go\go-desktop\tests
go test ./...
```

```powershell
cd D:\app\go\go-desktop\frontend
npm run build
```
