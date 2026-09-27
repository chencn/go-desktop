# go-desktop 设计规范

本文档是 `go-desktop` 的 UI/UX、主题、组件和工程边界规范。`README.md` 只描述项目入口、目录、命令和维护流程；具体界面规则以本文档为准。

## 1. 当前结论

`go-desktop` 是一个基于 Wails3、Vue 3、TypeScript、Tailwind v4 和 Element Plus 组件库的中文桌面工具项目。

界面基线是 Apple iOS 26 / macOS 26 Liquid Glass HIG：单一一份设计令牌，桌面、平板、手机三端共用同一套外壳与原子，像素级对齐 `OpenDesign` 设计稿（mock 入口 `index.html`）。

界面目标：

- 桌面优先，信息密度高，布局克制；窄屏只换排布，不换观感令牌。
- 保留中文产品体验，不做营销页式首屏。
- 主题（亮暗模式）、全局控件尺寸和液态玻璃光学（极光舞台 / 折射风格 / 光强）可配置。
- 更新、日志、设置、关于、授权各自职责清楚，不互相塞功能。
- 所有间距/字号令牌以 `--sp-N` / `--fs-N` 命名，`N` 即默认档像素值，便于与设计稿逐条比对。

工程目标：

- Element Plus 组件优先，按需自动引入，观感由统一皮肤层改皮，不重造基础控件。
- 业务页面组合组件，不各自手写按钮、表格、弹窗、分段控件。
- 设置与显示偏好（含液态玻璃光学）统一持久化走后端 SQLite KV，前端不再保存第二份副本。
- 日志写每日 JSONL 文件，SQLite 不保存日志。
- 更新状态只保存当前生命周期，不保存历史事件。

## 2. 技术栈和配置

核心技术：

| 层 | 技术 |
| --- | --- |
| 桌面运行时 | Wails3 |
| 后端 | Go |
| 前端 | Vue 3 + TypeScript + Vite |
| 样式 | Tailwind v4 + CSS variables（Apple HIG 令牌层 + Element Plus 皮肤层 + 液态玻璃材质层） |
| UI 组件库 | Element Plus（按需注册） |
| 图标 | `@lucide/vue` |
| 配置存储 | SQLite KV，表语义为 `config_items` |
| 日志 | 每日 JSONL 文件 + 内存 ring buffer |
| 更新 | GitHub Release 或本地静态 manifest |

Element Plus 通过 `unplugin-vue-components` + `ElementPlusResolver` 按需注册：模板里的 `el-*` 组件自动带样式引入，命令式组件（`ElMessage`）的样式在 `frontend/src/main.ts` 手动引入。中文 locale 和全局尺寸由 `App.vue` 顶层的 `el-config-provider` 统一下发。

规则：

- `frontend/src/features/**` 放业务页面、页面私有组件和页面私有 CSS。
- `frontend/src/features/shared/` 放跨页复用的项目自有组件（`GlassPanel.vue`、`AlertDialog.vue`），由业务页直接 `import`，不做全局注册。
- `frontend/src/app/` 放应用级状态模块：`display.ts` 显示偏好、`glass.ts` 液态玻璃光学、`routes.ts` 视图映射、`state.ts` 启动门禁。
- `frontend/src/lib/utils.ts` 是 `cn()` 唯一来源。
- 全局 `Ui*` 注册已退役：`frontend/src/shared/ui/plugin.ts` 和 `frontend/src/components/ui/` 保持删除状态，基础控件一律直接使用 `el-*` 组件加皮肤类。
- Element Plus 观感改造只允许写在 `frontend/src/styles/element-plus.css` 的皮肤层，页面只挂 `.btn-apple` / `.apple-switch` / `.apple-select` / `.apple-field` / `.apple-table` / `.apple-pagination` / `.apple-dialog` / `.segmented-control` 等语义类，不在页面里重写 EP 内部结构。
- 本项目使用 element-plus-mcp 工作流：进入 UI 改造前先用 element-plus-mcp 查询当前项目实装版本的组件 API 和示例（`get_api` / `get_example`），再在页面里组合 `el-*` 组件。

## 3. 信息架构

应用只有四个主页面，更新固定在右上角弹窗入口。

| 页面 | 职责 | 禁止 |
| --- | --- | --- |
| 概览 | 应用服务 / 数据库 / 网络 / WebView 运行状态卡、业务统计卡、样例图表 | 放更新主操作、快捷入口 |
| 日志 | 日志文件、筛选、统计、表格、分页、清理当前筛选范围 | 放设置项、更新策略 |
| 设置 | 可修改的业务设置和显示偏好 | 放只读信息、技术栈、路径、策略说明 |
| 关于 | 应用元数据、Release 来源、运行环境、本地路径、授权卡片（仅构建启用授权时显示） | 放可编辑控件 |
| 更新弹窗 | 检查、下载、校验、立即安装、下次启动安装、诊断 | 成为独立页面或导航项 |

导航规则：

- 侧栏导航数据只在 `frontend/src/shared/views.ts` 维护一份（`navigationGroups` / `navigation` / `pageTitle` / `pageSubtitle`），桌面侧栏、手机底部 TabBar 和顶栏标题都从那里取。
- 侧栏按 `核心功能 / 配置与系统` 分组，每项配 `.sq-icon-badge` 渐变徽标，只有图标 + 文案（不放日志数量角标）；`≥1024` 全宽、`768–1023` 收成 70px 图标导轨并可由顶栏汉堡键展开成 232px 浮层（`.tablet-open`）、`≤767` 变成抽屉（`.mobile-open`）。后两种浮层都要配遮罩，收起时遮罩与状态一起归零。
- 右上角是一条控制带：日夜切换 + 极光舞台 + 更新状态三枚 `.action-icon-btn`，分隔线，然后是苹果红绿灯，DOM 顺序与视觉顺序都是最小化→最大化→关闭（关闭在最右；为兼顾 Windows 桌面用户原生习惯，窗口控制按钮保持在右上角及此排列顺序，不改到左上角）。
- 更新状态按钮按状态显示 busy（图标自转）/ ready（状态点脉冲）/ danger（红色脉冲）视觉态。
- `≤767` 隐藏红绿灯与侧栏，改用底部 `mobile-tabbar`，侧栏由顶栏汉堡键经遮罩层唤出。
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
| `alwaysOnTop` | `window.always_on_top` | `false` | 窗口显示时置顶 |
| `logRetentionDays` | `log.retention_days` | `30` | `7 / 30 / 60 / 90 / 180 / 365 / 永不清理` |
| `logLevel` | `log.level` | `info` | `debug / info / warning / error` |
| `autoLaunch` | `startup.auto_launch` | `false` | 开机自启 |
| `createDesktopShortcut` | `startup.create_desktop_shortcut` | `true` | 创建桌面快捷图标 |
| `launchHiddenToTray` | `startup.launch_hidden_to_tray` | `false` | 开机自启时隐藏到托盘 |

业务设置页规则：

- 每一行只有一个可编辑控件；设计稿把所有行都当常驻设置，行与行之间不再互相隐藏或禁用（`launchHiddenToTray` 独立可编辑，只在开机自启时才生效）。
- `minimizeToTray` 只影响点击关闭按钮；点击最小化仍进入任务栏。
- `alwaysOnTop` 只影响窗口显示态；隐藏到托盘和自启隐藏策略保持独立。
- GitHub Release 的 owner/repo 来自项目元数据，不作为业务设置保存或修改。
- `githubProxyBase` 只影响 GitHub Release API、安装资产和 `.sha256` 下载；`local` 更新源不使用该代理。空串是合法取值（直连官方 API），后端按配置项是否存在回退默认值，因此清空代理的选择可以跨重启保留，不会被默认代理地址悄悄改回。
- 写配置失败必须返回错误，前端展示保存失败。
- 保存开机自启和桌面快捷方式时，同步 Windows 系统集成；系统集成失败要回滚内存和 SQLite 配置。

### 4.2 显示偏好

显示偏好由五个轴组成，前端集中在 `frontend/src/app/display.ts`（光学三项委托给 `frontend/src/app/glass.ts`），后端落在 `display.preferences.v3` JSON 配置项。它们不写入后端业务 `Settings` 结构。

| 轴 | 值 | 生效方式 |
| --- | --- | --- |
| Mode | `light / dark` | `html.dark`，同时驱动 Element Plus 暗色变量和 Tailwind `dark:` 变体 |
| Size | `large / default / small` | `el-config-provider :size` 全局下发 |
| backdrop | `true / false`（默认 `false`） | `<html class="has-backdrop">` 点亮极光流光舞台并放开卡片折射 |
| lgStyle | `fresnel / frosted / sheen`（默认 `fresnel`） | `<html data-lg-style>` 切换折射风格 |
| lgIntensity | `30–100`（默认 `75`） | `<html style="--lg-intensity-factor">` 乘进 blur / alpha |

主色固定为 iOS System Blue（`#007aff`，暗色 `#0a84ff`，悬浮 `#1a86ff`，按压端 `#0062cc`），raw 值只在 `frontend/src/colors.css` 以 `--color-value-007aff` 写死一次；`styles.css` 先把它收进语义令牌 `--accent`，再用 `--el-color-primary: var(--accent)` 桥接给 Element Plus，light-3/5/7/8/9 与 dark-2 色阶用 `color-mix` 派生（亮色向白底相调、暗色向 `--surface` 相调），明暗两态自动跟随。**主色的所有派生（悬浮描边、激活底、聚焦光晕、按钮渐变中位、图表主蓝）一律写 `var(--accent)`，不得再直接引用 raw 蓝令牌**，否则换主色时派生会脱节。语义状态色是 iOS 系统色档且明暗同值（`--success #34c759` / `--warn #ff9500` / `--danger #ff3b30` / `--info #5ac8fa`——info 已从原来的纯灰 `#85858c` 换成系统青），只有 `--*-light` 淡底在暗色加档；暗色底是纯黑 `--bg #000`、`--surface #1a1c22`、`--surface-warm #24262e`。主题色方案（多色板切换）作为后续扩展预留。行内图标不再是偏好维度，一律走 `.sq-icon-badge` 语义渐变徽标。

五个轴是同一条链路，没有「前端专属」偏好：

```go
// internal/desktopapp/display/preferences.go
type PreferencesV3 struct {
    Version   int    `json:"version"`
    ThemeMode string `json:"themeMode"`
    Size      string `json:"size"`
    Backdrop  bool   `json:"backdrop"`
    LgStyle   string `json:"lgStyle"`
    LgIntensity int  `json:"lgIntensity"`
}
```

```ts
export type ThemeMode = "light" | "dark"
export type DisplaySize = "large" | "default" | "small"
export type DisplayPreferences = {
  backdrop: boolean
  lgIntensity: number
  lgStyle: GlassStyle
  size: DisplaySize
  themeMode: ThemeMode
}
```

后端 `NormalizeV3` 是权威校验：`lgStyle` 只接受 `fresnel / frosted / sheen`，`lgIntensity` 只接受 `30–100`（老数据缺字段读到的 `0` 同样越界），每个轴独立回自己的默认值，不会因光学项非法而牵连主题与尺寸；只有 `version != 3` 或 JSON 解析失败才整体回落 `DefaultV3()`。前端 `hydrateGlassPreferences` 在注入时逐项再校验一次，只为兼容旧 SQLite 数据缺字段。

显示偏好规则：

- 显示偏好持久化使用 `display.preferences.v3` JSON；旧版本 key（v1/v2）不再读取，解析失败或版本不匹配回退默认值。
- 显示偏好不使用拆分 `display.*` KV。
- 光学三项不允许再写浏览器本地存储：`glass.ts` 只保留状态与 DOM 同步，导出 `exportGlassPreferences()` 交给 `display.ts` 合成快照，后端是唯一的持久化出口。
- 切换后立即更新 DOM / provider，再异步保存到 SQLite 配置项。
- 保存失败时显示错误，并把已乐观切换的控件回滚到原值（顶栏极光开关、主题切换同此规则）。
- 切换不允许触发页面 reload。
- 主色控制 `el-*` 组件主色、选中态、焦点环和关键进度。
- 行内图标语义色固定，以 `.sq-icon-badge`（侧栏、分组标题、设置行、服务卡）和 `.stat-icon-wrap`（统计卡）两类渐变徽标提供，色板 token 为 `--tile-*` 11 颗：**两端一律取 Apple 官方系统色的两个外观档**（亮端 = Dark 档、暗端 = Light 档，不允许出现官方表之外的 hex）（blue `#0a84ff→#007aff`、indigo `#5e5ce6→#5856d6`、purple `#bf5af2→#af52de`、pink `#ff375f→#ff2d55`、red `#ff453a→#ff3b30`、orange `#ff9f0a→#ff9500`、yellow `#ffd60a→#ffcc00`、green `#30d158→#34c759`、teal `#40c8e0→#30b0c7`、cyan `#64d2ff→#32ade6`、gray `#98989f→#8e8e93`），145deg 只做立体差，禁止再把暗端压到 `#1x/#2x` 深色（那是整颗图标发闷发灰的根因），明暗同一色板。材质外壳是 `--tile-edge` / `--tile-shadow` / `--tile-svg-drop` / `--tile-filter` 四条；卡片与图标的高程描边用 `--elev-rim`（上缘镜面 + 0.5px 菲涅尔切面）。白色符号在黄/青/绿/橙这类亮档上的固有对比只有 1.5–2.4（色彩本身决定，加深底色会牺牲"阳光"），所以 `--tile-svg-drop` 用两层投影分离：贴边硬投影定形 + 零偏移描边把符号从底色里抠出来。维护在 `frontend/src/styles.css` 与 `frontend/src/styles/layout.css`。
- 材质取向是通透与高光，不是哑光：设计稿在 `.action-icon-btn` / `.modal-close-btn` 上明写的 `backdrop-filter` 采样必须保留，底色按稿子的通透档（日 `.65`、夜 `.06`），浮起靠上缘镜面而不是实色填充；暗色徽标不再叠 `saturate(.82) brightness(.92)` 的去饱和滤镜（`--tile-filter: none`），高程由 `--elev-rim` 与更暗的边缘承担。任何合规/性能整改都不许以牺牲这层光泽为代价；若两者冲突，摆证据让用户拍板。
- 弹窗自成材质档 `--dialog-fill` / `--dialog-blur`（日 `.86` 白 + `blur(30px) saturate(210%) brightness(1.05)`，夜 `rgba(26,28,34,.88)`），遮罩相应减淡到 `--overlay-fill` 日 `.28` / 夜 `.55`；页眉页脚保持 `--surface-warm` 近白实底，不得改成透明（透明会透出暗遮罩而发灰）。依据是 Apple 对玻璃的两条硬性要求：更厚的材质保证细部文字的对比，材质之上用 vibrant 中性色。
- 玻璃面板内部不许再铺彩色底：`.update-info-banner` 原按稿子用 `--accent-light`（8% Action Blue），实测在近白玻璃上形成"白底压淡蓝"的脏色块，已改走中性 `--banner-fill` / `--banner-edge`（日 `black/.05` + `.06` 切面，夜 `white/.08` + `.12`）。彩色只保留在图标与小徽标（`.log-level-badge`、`.always-on-top-badge`、主按钮）这一层级，出自 HIG 原话 "Be judicious with your use of color in controls and navigation so they stay legible and allow your content to infuse them and shine through."
- 五个偏好轴切换都必须立即改变实际渲染效果，不允许出现只记录不改观感的假设置。
- 禁止远程字体加载，字体族固定系统字体。

### 4.3 首屏启动底板

刷新或冷启动时，Vue 挂载会接管并短暂替换 `#app` 内的静态 loading；在路由门禁、显示偏好、页面 chunk 和 `AppChrome` 完成渲染前，会出现极短的 Vue 挂载空窗期。这个阶段的黑边不是卡片、表格、侧栏或内部组件边框，而是 WebView 最外层露出了默认背景。

规则：

- `frontend/index.html` 必须内联 `body::before`，用 `position: fixed`、`inset: 0` 和 `background: var(--boot-background)` 作为独立于 Vue 生命周期的固定满屏底板。
- `html`、`body`、`#app` 必须在内联启动 CSS 中提前声明 `width: 100%`、`min-width: 0`、`background: var(--boot-background)`，并清掉边框、轮廓和阴影；启动期占位用 `.boot-loading` / `.boot-spinner`，配色与字体栈沿用 Apple HIG 底板，不引入额外组件样式。
- `frontend/src/styles.css` 必须在运行期继续给 `html`、`body`、`#app` 声明根级背景、尺寸和外层兜底，不能只依赖页面组件铺满视口。
- 启动底色不跟随 `prefers-color-scheme`，因为项目显示偏好由持久化配置接管；在配置加载前先使用日间默认底板，避免系统暗色导致刷新瞬间黑底。

## 5. 主题和 CSS 归属规则

`frontend/src/styles.css` 只放 Tailwind import、主题 token、全局 reset、focus 和根级媒体变量，只允许承载：

- Tailwind v4 import 和 `@custom-variant dark`。
- `@theme inline` token 映射（Tailwind 只读别名）。
- Apple HIG 语义令牌：三层表面阶梯、玻璃材质、极光舞台、按钮填充、前景/边框/主色/语义色/图表色、圆角、深度、动效、间距 `--sp-N`、字号 `--fs-N`、外壳几何。
- 旧语义别名到 HIG 令牌的桥接（`--background: var(--bg)` 等），只为兼容 Tailwind 工具类。
- `--el-*` Element Plus 变量映射与 `color-mix` 色阶派生（含 `html.dark` 暗色重算）。
- `:root`、`html.dark`、显示尺寸 `html[data-display-size]`。
- 全局 reset、focus、滚动条、reduced motion。

`frontend/src/styles/layout.css` 只允许承载：

- 窗口骨架：`.app-window`、`.app-topbar`、`.app-body`、`.app-sidebar`、`.app-main-viewport`、`.view-content`。
- 顶栏原子：`.topbar-left`、`.topbar-right`、`.topbar-actions`、`.action-icon-btn`、`.topbar-divider`、`.traffic-lights-right`、`.traffic-btn`。
- 移动端外壳：`.mobile-tabbar`、`.tabbar-item`、`.mobile-sidebar-backdrop`。
- 跨页面布局 primitive，例如 `.split-header`、`.section-title-row`、`.topbar-title-line`、`.detail-row`、`.view-content--fill`。
- 跨页面图标与徽标原子，例如 `.sq-icon-badge` 用到的 `--tile-*` 渐变色板、`.badge-apple`、`.ver-badge`、`.nav-item-badge`（`.nav-item-badge` 只属于日志页筛选按钮的命中数角标，侧栏导航不放日志数量）。
- 全部三端断点。

`.lg` / `.lg-surface` / `.lg-light` / `.lg-sheen` / `.lg-content` 三层光学基元不在本仓维护：它们逐字节拷贝自上游 `D:\liquid-glass\src\glass.css`，落在 `frontend/src/styles/liquid-glass.upstream.css`，`main.ts` 里先于派生层引入。上游升级时整文件重新拷贝，禁止手抄进派生文件、禁止改它的色值与降级规则。`frontend/src/styles/liquid-glass.css` 只写派生层：极光流光舞台 `.stage-backdrop`、业务卡片的材质档位与折射风格档；派生层不得给 `.lg-surface`/`.lg-light`/`.lg-sheen` 写 `display`/`visibility`（那会盖掉上游的材质降级），也不得给宿主 `.lg` 加 `opacity`/`filter`/`backdrop-filter`（上游硬性要求宿主保持无滤镜，采样只在 `.lg-surface` 层做）。

`frontend/src/styles/element-plus.css` 承载 Element Plus 皮肤层：把 `el-*` 组件捏成设计稿观感的语义类（`.btn-apple`、`.apple-switch`、`.apple-select`、`.apple-field`、`.apple-table`、`.apple-pagination`、`.apple-dialog`、`.apple-alert`、`.apple-progress`、`.segmented-control`）。它必须在 `main.ts` 里排在最后引入，以便覆盖按需引入的 EP 组件样式。

页面私有 CSS 必须放在 `frontend/src/features/**` 相邻文件中，例如：

- `frontend/src/features/home/HomePage.css`
- `frontend/src/features/logs/LogsPage.css`
- `frontend/src/features/settings/SettingsPage.css`
- `frontend/src/features/update/UpdateStatusDialog.css`

页面私有 CSS 只写本页结构、宽度与栅格，控件观感一律挂皮肤类；`raw` 色值只能出现在 `frontend/src/colors.css`，其它文件通过 `var(--color-*)` 或语义令牌引用。唯一的例外是 `*.upstream.css`（逐字节拷贝的上游文件）：它的色值归上游，`.tmp/check-css.mjs` 与 `TestFrontendColorsUseSingleTokenFile` 对它显式豁免，不允许为了过审计去改写它。

禁止项：

- 页面前缀样式进入 `styles.css`，例如 `.settings-*`、`.log-*`、`.about-*`。
- 组件选择器进入 `styles.css`，例如页面私有下拉选择器。
- 在业务页面手写一套普通按钮、表格、弹窗、分段控件。
- 在页面 CSS 重写 `.el-*` 内部结构，需要改皮时改 `styles/element-plus.css`。
- 卡片嵌套卡片。
- 远程字体 `@import url(...)`。
- 布局依赖负 letter spacing；表头字距只允许 `.apple-table` 皮肤层的 `0.04em`。

## 6. 组件规则

优先级：

1. Element Plus `el-*` 组件（按需自动引入）+ `styles/element-plus.css` 皮肤类，设计稿的每一个控件观感都在皮肤层落地。
2. `frontend/src/features/shared/*` 项目自有组件（`GlassPanel.vue` 玻璃卡片容器、`AlertDialog.vue` 二次确认弹窗），由页面直接 import。
3. 页面私有特殊组件放 `frontend/src/features/**`。

当前主要组件：

| 组件 | 位置 | 用途 |
| --- | --- | --- |
| `el-button` + `.btn-apple` | Element Plus + 皮肤层 | 主/次/危险按钮，弹窗等宽按钮用 `.is-alert-action` |
| `el-switch` + `.apple-switch` | Element Plus + 皮肤层 | 44×26 iOS 阻尼开关 |
| `el-radio-group` + `.segmented-control` | Element Plus + 皮肤层 | iOS 分段器（主题、尺寸、玻璃风格）与日志级别页签 `.is-tone-tabs`；两处分段器共用 §3 同一套轨道与滑块材质，`.is-tone-tabs` 只加数量胶囊与级别文字色 |
| `el-select` + `.apple-select` | Element Plus + 皮肤层 | iOS 玻璃下拉，浮层用 `.ios-select-popover` |
| `el-input` / `el-textarea` + `.apple-field` | Element Plus + 皮肤层 | 文本输入、带图标搜索、授权码多行等宽输入 |
| `el-table` + `.apple-table` | Element Plus + 皮肤层 | 日志表（吸顶表头、行悬停主色底） |
| `el-pagination` + `.apple-pagination` | Element Plus + 皮肤层 | 28×28 方键分页 |
| `el-progress` + `.apple-progress` | Element Plus + 皮肤层 | 8px 药丸下载进度 |
| `el-dialog` + `.apple-dialog` | Element Plus + 皮肤层 | 更新状态弹窗与二次确认弹窗 |
| `el-alert` + `.apple-alert` | Element Plus + 皮肤层 | 错误横幅 |
| `ElMessage` / `.apple-toast` | Element Plus 命令式 | 设置与观感变更的即时反馈 |
| `GlassPanel` | `features/shared/GlassPanel.vue` | 首页服务卡 / 指标卡的液态玻璃容器 |
| `AlertDialog` | `features/shared/AlertDialog.vue` | 清空日志等危险确认（替代 `ElMessageBox`） |

组件规则：

- `el-*` 组件与 `features/shared` 组件不 import store，状态由页面下发。
- 危险确认使用 `AlertDialog` 组件，禁止 `window.confirm` 与命令式 `ElMessageBox`。
- 表格型数据一律 `el-table`，日志页同样走 `.apple-table`；行高测量依赖 `el-table` 的 DOM，不再手写 `<table>`。
- 顶栏图标按钮用原生 `title` 提示，不包 `el-tooltip`（设计稿的顶栏按钮没有 tooltip 包装层）。
- 图标默认 `13-18px`，固定使用 Lucide。
- 控件高度由令牌给出：默认 `--control-height`（32px）、加高 `--control-height-lg`（36px）、手机触控 `--control-height-touch`（38px），图标按钮宽高一致。
- Element Plus 自带的组件级 CSS 变量（例如分段器的 `--el-radio-button-checked-*`）声明在组件自身元素上，皮肤要改这些配色时必须落在同一个元素，靠上层容器覆写会被组件自身声明盖掉。
- 皮肤层禁止使用会改变盒子尺寸的 `:active` 位移/缩放于任何承载浮层定位的触发框（下拉框已因此出现浮层错位），按钮等非浮层锚点不受此限制。

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
- 更新弹窗只讲当前版本、最新版本、状态说明和下载进度；不渲染 Release 变更日志列表（`releaseNotes` 不参与弹窗展示）。

## 8. 日志模型

日志来源：

- 文件日志：`data/logs/*.log`，内容为每日 JSONL。
- 内存 ring buffer：只服务当前前端视图和即时反馈。
- SQLite：只保存配置项，不保存日志、日志历史、更新历史。
- `crash.log` 是 Runtime 创建前的早期崩溃兜底文件；启动时先导入上次异常退出尾部，再裁剪到最近尾部，避免无限增长。

日志页规则：

- 支持来源、级别、关键词、日志文件筛选；筛选面板默认折叠，命中条件数显示在筛选按钮的 `.nav-item-badge` 角标上。
- 级别页签是 `.segmented-control.is-tone-tabs` 分段器，页签右侧挂该级别的数量胶囊；激活页签只按级别换文字色（debug 灰 / warning 橙 / error 红），滑块和胶囊保持主色底。**全项目只有 §3 一套分段器材质**：`.is-tone-tabs` 不得覆写轨道底色、采样、内距、圆角、描边、滑块投影或悬浮反馈，只允许加数量胶囊与级别文字色这两类语义差异（手机端为容纳五个页签额外允许横向滚动布局）。
- 分页摘要必须交代后端最低门禁：`每页 N 条，当前第 P / T 页 (门禁过滤后共 X 条记录，系统最低门禁: ≥ LEVEL)`，空态同理写成 `暂无匹配日志 (门禁级别: ≥ LEVEL)`，让用户能区分「没有日志」和「被门禁挡住」。
- 表格用 `el-table` + `.apple-table`：表头吸顶、11px 大写、`letter-spacing: 0.04em`、行悬停主色底；级别列用 `.log-level-badge`、模块列用 `.log-scope-tag`，时间列 `.log-time-cell` 等宽。
- 手机端表格换成 `.log-mobile-card` 卡片列表，`.log-table-shell` 隐藏。
- 空态放在表格 body 内，不在表格外额外拆一块空状态。
- 路径、日志内容、版本号必须 `min-width: 0` 和 `overflow-wrap: anywhere`。
- 清空日志必须使用 `AlertDialog` 组件二次确认。
- 自动刷新、手动刷新、筛选和专注模式属于日志页工具条；专注模式在 `<html>` 上切 `is-log-focus`，由 `styles/layout.css` 收起侧栏、底部 TabBar 与顶栏内容，组件卸载时必须移除该类。顶栏不能整条 `display: none`：无边框窗口的拖拽区（`--wails-draggable: drag`）和三颗窗口控制钮都挂在它上面，所以专注态把 `.app-topbar` 收成 14px 透明拖拽带（去底、去采样、去描边去阴影），标题与右侧动作区隐藏，红绿灯绝对定位在右上角、`opacity: 0` + `pointer-events: none`，指针进入顶栏才浮出。

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

外壳几何来自设计稿，但按真实窗口（可最大化、可缩放）而非假窗口定尺：

| 区域 | 规则 |
| --- | --- |
| 顶栏 | 高 `--topbar-height`（52px），液态玻璃材质，红绿灯在右端，顺序为最小化→最大化→关闭 |
| 侧栏 | `--sidebar-width: clamp(196px, 232px + (100vw - 1360px) * 0.12, 300px)`，1360px 视口正好等于设计稿 232px，更宽才向右扩张，窄屏不低于 196px；折叠态 `--sidebar-collapsed-width`（68px） |
| 内容视口 | `.app-main-viewport` 唯一滚动容器，栏距 `--page-gutter-y/x`（24px / 28px） |
| 平板 `<= 1023px` | 侧栏收成 70px 图标栏（44×44 方形透镜胶囊），隐藏文字与页脚文案，栅格降为两列；汉堡键把侧栏展开成 232px 浮层 `.tablet-open` 并压暗遮罩 |
| 手机 `<= 767px` | 侧栏变抽屉（`translateX(-100%)` 收起，稿子基底写 260px，但 `.mobile-open` 的特异度更高，展开态实测 232px）+ 底部 TabBar，视口栏距改 `16px 14px calc(tabbar+20px)` |

规则：

- `.app-window` 固定两列：sidebar + content；`.app-main-viewport` 是唯一页面滚动容器。
- 所有 grid / flex 子项必须 `min-width: 0`。
- 四个页面共用同一套视口栏距：概览、日志、设置、关于全部通栏铺开。设计稿把设置与关于限宽 880px 并居中，那属于 1400px 假窗口，真实最大化窗口下会缩成居中小票，因此这两页不再 `max-width` / `margin: 0 auto`。
- 桌面弹窗由 `.el-overlay-dialog:has(.apple-dialog)` 用 flex 真正垂直+水平居中，窄屏（`≤767px`）自动收口为 iOS 26 标准的贴底 Bottom Sheet（带顶部手势抓手条与 `env(safe-area-inset-bottom)` 安全区垫高）；更新弹窗页头的关闭钮是设计稿的 `.modal-close-btn`（28px 中性玻璃圆盘 + 白色 X，外扩补足 44px 触控区），不是红绿灯第三颗红珠，靠 `.el-dialog__header` 的 `space-between` 钉在右端，不允许用绝对定位的默认 `headerbtn`（会随标题长度漂移）。
- 日志表、路径和长文本可以内部横滚或换行，但不能撑破 `.app-window`。
- 五个主要交互区域不能出现横向溢出：概览、日志、设置、关于、更新弹窗。
- 视觉验证必须覆盖 `1440×900`、最大化宽屏、平板 `768-1023px`、手机 `<= 767px`，且亮/暗与清爽/极光四种组合。

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

- Element Plus 通过 `unplugin-vue-components` 按需注册，`frontend/src/components/ui` 保持移除状态。
- 全局 `Ui*` 注册保持退役：`shared/ui/plugin.ts` 不存在，项目自有组件由页面直接 import。
- 控件观感只写在 `styles/element-plus.css`，页面 CSS 不出现 `.el-*` 内部选择器。
- 设置页业务设置和显示偏好边界清楚，后端只存业务设置与 `display.preferences.v3`。
- 更新入口只在右上角图标和弹窗。
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
