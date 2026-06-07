# DESIGN.md

## 0. 设计结论

当前项目是一个 **Vue 3 + TypeScript + Tailwind v4 + shadcn-vue 源码组件** 桌面应用项目。设计系统以 shadcn-vue 官方 CSS variables、源码组件、组件 composition 和本地主题 token 为准。

目标是一个现代桌面应用界面：克制、清楚、可配置主题和色彩，功能结构服务 `go-desktop` 原业务。

文档边界：

- `README.md` 是项目说明文档，说明项目定位、目录结构、数据约定、常用入口和维护入口。
- `DESIGN.md` 是产品、交互、视觉、组件和工程约束文档，作为 UI/UX 和工程边界的当前规范源。
- `frontend/src/components/ui/README.md` 只说明 shadcn-vue 生成目录边界，不承载项目整体说明。

官方依据：

- shadcn-vue Theming 文档：推荐 CSS variables；组件使用 `background`、`foreground`、`primary` 等语义 token；暗色模式通过覆盖同一组 token 实现。https://www.shadcn-vue.com/docs/theming
- shadcn-vue `components.json` 文档：Vue 版 schema 是 `https://shadcn-vue.com/schema.json`；`typescript: true`；Tailwind v4 的 `tailwind.config` 留空；推荐 `cssVariables: true`。create 页的运行时配置轴以当前 create 页面枚举为准。https://www.shadcn-vue.com/docs/components-json
- shadcn-vue Skills 文档：生成组件前应通过 CLI/docs/MCP 查询组件文档，结合 `components.json`、已安装组件、aliases 和项目上下文。https://www.shadcn-vue.com/docs/skills
- shadcn-vue `create` 页面：新项目配置不是只选基础色盘；当前创建器包含多条视觉轴。当前应用只把已经接入实际运行效果的轴暴露到设置页；字体族不作为设置项，Icon Library 未接入多图标包渲染前只作为 `components.json` 固定配置。https://www.shadcn-vue.com/create

## 1. 工程配置

### 1.1 shadcn 配置文件

当前项目已提供 `frontend/components.json`。shadcn CLI / skill 读取项目上下文时必须以该文件为准：

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
  "iconLibrary": "lucide",
  "aliases": {
    "components": "@/components",
    "ui": "@/components/ui",
    "lib": "@/lib",
    "utils": "@/lib/utils",
    "composables": "@/composables"
  }
}
```

配套 TypeScript 路径：

```json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    },
    "moduleResolution": "Bundler"
  }
}
```

shadcn-vue skill 使用规则：

- 进入 UI 组件改造前先检查 `frontend/components.json`，确认 `style`、`baseColor`、`cssVariables`、aliases 和 iconLibrary。
- 优先执行 `shadcn-vue info --json` 获取项目配置和已安装组件；再用 `shadcn-vue docs <component>` 或 `shadcn-vue search <keyword>` 拉取官方组件文档；最后用 `shadcn-vue add <component>` 添加源码组件。
- `@shadcn` 是新版 CLI 内置 registry，不允许在 `components.json` 里覆盖。
- `frontend/src/components/ui/*` 只能通过 `shadcn-vue add ... --overwrite` 或等价 CLI 流程更新；CLI 失败时先修 npm cache / Windows env，不把业务补丁手写进该目录。
- 本项目的 shadcn-vue skill 目标不是生成一次性页面代码，而是让组件查询、组件 composition、源码 primitive、主题 token 和业务页组合保持一致。

全局注册规则：

- `frontend/src/shared/ui/plugin.ts` 负责注册全局 `Ui*` primitive 和项目兼容 wrapper，例如 `UiButton`、`UiCard`、`UiDialog`、`UiNativeSelect`、`UiProgress`、`UiTable`、`UiTooltip`。
- 全局 Ui* 是本项目业务页使用 shadcn primitive 或兼容层的默认方式。
- `frontend/src/main.ts` 必须 `.use(uiPlugin)`，业务页默认直接使用全局 `Ui*`，避免每页重复 import 同一批基础 primitive。
- 图标、业务组件、页面私有小组件仍局部 import；复杂业务模块不放进全局注册。
- 全局名统一带 `Ui` 前缀，避免和原生标签、未来官方组件名或页面局部组件冲突。

规则：

- `frontend/src/components/ui/*` 是 shadcn-vue CLI 专属源码目录，只放 CLI 生成或覆盖的官方 primitive；禁止放业务组件、业务 class、全局注册和项目兼容 wrapper。
- `frontend/src/components/ui/README.md` 必须保留该边界说明。
- `frontend/src/shared/ui/*` 是项目 UI 兼容层，放 `Ui*` 全局注册、旧 API 适配、NativeSelect 这类项目 wrapper。
- `frontend/src/features/*` 是业务页面层，可组合 shadcn primitives；页面私有控件例如 `SettingsColorSelect.vue` 留在对应 feature 目录。
- `frontend/tsconfig.json` 排除 `src/components/ui/**/*.vue`，避免框架生成 Vue 文件被当作业务源码全量检查；前端构建仍负责验证实际打包链路。
- `lib/utils.ts` 是 `cn()` 唯一来源，使用 `clsx + tailwind-merge`。
- 新增 shadcn-vue 组件前先查官方 docs，例如 Button、Field、Sidebar、Dialog、Popover、Table、Alert Dialog、Toggle Group。
- 本项目明确采用 shadcn-vue skill/CLI/docs 工作流。生成或修改组件前必须先确认 `components.json`、项目信息、已安装组件、token、aliases 和官方 docs；没有可用 skill 时才按本文档手动模拟。

### 1.2 shadcn primitive-first 规则

所有新增或重构的交互控件先按 shadcn 官方组件模型处理，禁止在业务页面里优先手写控件样式。

优先级：

1. 已存在 `frontend/src/components/ui/*` Vue primitive：直接复用。
2. shadcn-vue 官方已有组件：用 CLI / docs / skill 拉取或覆盖到 `components/ui`。
3. 多个 primitive 可组合解决：在 `shared/ui` 兼容层或业务层组合，不在 CSS 里另造一套控件体系。
4. 官方没有、业务确实特殊：页面私有控件放 `features/**`，跨页面兼容 wrapper 放 `shared/ui`，不写进 `components/ui`。

禁止项：

- `features/*` 里直接写裸 `<button class="...">` 实现普通命令按钮、图标按钮、分段按钮、色板按钮。
- 用 `window.confirm`、手写 fixed overlay、手写 popover、手写 dropdown、手写 tablist 替代官方 Dialog / AlertDialog / Popover / DropdownMenu / Tabs。
- 用原生 `<select>` 承担主要产品选择器；当前只允许通过 `shared/ui/NativeSelect.vue` 的 `UiNativeSelect` 使用，复杂选择器后续补官方 Select 结构。
- 为某一页单独发明 `.segment-button`、`.swatch-button` 或独立日志表格这类可复用控件样式而不沉淀 primitive。

允许项：

- `features/*` 可以写页面布局 class，例如 grid、page-stack、workbench、响应式容器。
- 业务语义小组件可以留在页面文件内，例如只负责拼装数据的 `Metric`、`DataRow`。
- `UiTable class="log-table"` 这类页面级 modifier 只能用于日志页列宽、换行和全屏工作区布局；表格结构、行、表头和单元格必须继续来自 Table primitive。
- 侧边栏和窄屏导航允许使用原生 `<button>`，但只能承担导航语义，必须有稳定 active 态、图标含义和无障碍标题；普通命令仍使用 `UiButton`。
- `components/ui/*` 内部可以使用原生 HTML 元素，因为它们是 shadcn-vue CLI primitive 的实现层；项目代码不手写改该目录。

### 1.3 CSS 归属规则

公共 CSS 只承担主题、token、reset 和极少跨页面 primitive，禁止把页面和组件私有样式继续堆进全局入口。

- `frontend/src/styles.css` 只放 Tailwind import、`@theme inline`、`:root` 主题变量、显示偏好 token、全局 reset、focus/reduced-motion 和根级媒体变量。
- `frontend/src/styles/layout.css` 只放跨页面布局 primitive 和语义图标工具，例如 `.page-stack`、`.content-grid`、`.split-header`、`.section-title-row`、`.nav-icon`、`.data-icon`、`.icon-tone-*`。
- 页面私有样式必须和页面放在一起：`features/**/<Page>.css` 或页面 SFC `<style scoped>`，并由页面通过 `<style scoped src="./<Page>.css">` 引入。
- 页面业务前缀禁止进入 `styles.css`：`.about-*`、`.software-*`、`.business-*`、`.settings-*`、`.preference-*`、`.log-*`、`.dialog-*`、`.status-pill`。
- 组件私有样式必须写回组件所属文件：CLI primitive 的官方样式只允许随 `components/ui/*` 生成物进入；项目级组件样式放 `shared/ui/*` wrapper 或业务组件相邻 CSS，禁止手写业务补丁进 `components/ui/*`。
- 新增样式先判断 owner：主题 token 进 `styles.css`，跨页面布局 primitive 进 `styles/layout.css`，页面布局进 `features/**`，组件结构和状态进组件文件。

当前替换优先级：

| 当前问题 | 目标 shadcn primitive |
| --- | --- |
| 侧边栏 / 顶栏导航裸 `<button>` | `Button`，后续可评估 `Sidebar` |
| 设置页非颜色选项 | `NativeSelect` |
| 设置页颜色选项 | `SettingsColorSelect` 页面组件 |
| 原生 `select` | 官方 `Select` 或 `Native Select` |
| 清空日志 `window.confirm` | `AlertDialog` |
| 更新状态弹出层 | `Dialog` |
| 日志列表 | `Table` |
| 图标按钮缺说明 | `Tooltip` |

### 1.4 工程硬约束

这些规则是仓库级硬约束，不是建议：

- 测试只能放在独立 `tests/` 模块；禁止把 `_test.go` 写进 `app/`、`internal/`、`scripts/` 或生产目录旁边。测试需要验证工具脚本时，放到 `tests/` 下通过外部行为、生成结果或独立 helper 验证。
- `scripts/` 只放可执行工具和工具依赖代码，不放测试用例、截图、临时日志或一次性调试脚本。
- 临时截图、浏览器截图、调试日志、一次性输出必须写入 `.tmp/`；根目录、`scripts/`、`frontend/src/` 和生产目录禁止堆临时文件。
- `.tmp/` 默认被 git ignore，只允许保留 `.tmp/.gitkeep`；需要提交的测试 fixture 必须放到 `tests/fixtures` 或明确的测试资源目录。
- 前端视觉验证必须同时覆盖 PC 端 `1440×900` 和窄屏视口；截图统一写入 `.tmp/`，禁止只用当前窄窗口截图代表桌面端。
- 代码注释必须覆盖模块边界、导出 API、结构体字段、页面状态变量、测试用例意图、失败原因、复杂流程和工程约束；禁止无注释的大段复杂逻辑，也禁止“给变量赋值”这类噪音注释。
- 变量、结构体字段、测试用例只要承载业务语义或约束，就必须用注释写清楚为什么存在、影响范围和默认值；临时循环变量这类自明细节不写废话注释。
- 日志必须覆盖运行时、窗口、设置、更新、存储、单实例和进程级错误；`log`、`slog`、`stdout`、`stderr` 都必须接入统一日志框架并写每日 JSONL 文件，内存 ring buffer 只服务当前前端视图；SQLite 只保存 `config_items` 配置项，禁止保存日志、更新历史或更新事件。
- 待安装更新状态不写 SQLite；选择“下次启动再更新”时，只允许持久化到 `data/updates/pending.json`，安装成功、安装启动失败、校验失败或读取失败后必须清理。
- UI/运行时调试默认使用 Browser/Chrome 插件；禁止引入 Playwright 作为仓库依赖或脚本。
- 删除、覆盖、回滚、移动会导致原路径消失的文件前，必须说明影响范围并等用户明确回复“同意”。
- 设置页只放能修改状态的控件；只读信息、策略说明、运行状态、计数、路径、Release 来源、技术栈都放关于页或诊断弹窗。
- 设置页禁止放“只记录偏好但不改变实际界面/行为”的假设置；例如图标库未接入多图标包渲染前，不得出现在设置页。
- 用户可见文案只描述当前控件会改什么；实现约束例如“托盘菜单只包含显示/退出”写进设计和测试，不写到设置页文案。
- 禁止远程字体加载；设置页不提供字体族选择，应用统一使用系统默认字体。

## 2. 主题模型

主题由多条轴组成，全部前端本地状态即可，不写入后端 `Settings`。高频模式切换放在右上角，完整调参放在设置页的“显示偏好”分区：

| Axis | Values | DOM | Storage |
| --- | --- | --- | --- |
| Mode | `light` / `dark` | `html.classList.toggle("dark")` | `display.theme_mode` |
| Style | `reka` / `vega` / `nova` / `maia` / `lyra` / `mira` / `luma` / `sera` | `html[data-style]` | `display.ui_style` |
| Base Color | `neutral` / `stone` / `zinc` / `mauve` / `olive` / `mist` / `taupe` | `html[data-base-color]` | `display.base_color` |
| Theme | create 颜色集合 24 项 | `html[data-theme-color]` | `display.theme_color` |
| Chart Color | create 颜色集合 24 项 | `html[data-chart-color]` | `display.chart_color` |
| Icon Tone | `default` / `colorful` | `html[data-icon-tone]` | `display.icon_tone` |
| Radius | `default` / `none` / `small` / `medium` / `large` | `html[data-radius]` and `--radius` | `display.radius` |
| Menu | `default` / `inverted` / `default-translucent` / `inverted-translucent` | `html[data-menu]` | `display.menu` |
| Menu Accent | `subtle` / `bold` | `html[data-menu-accent]` | `display.menu_accent` |
| Accent | create 颜色集合 24 项 | `html[data-accent-color]` | `display.accent_color` |
| Card Border | `visible` / `soft` / `hidden` | `html[data-card-border]` | `display.card_border` |
| Density | `compact` / `comfortable` | `html[data-density]` | `display.density` |
| Text Size | `small` / `normal` / `medium` / `large` | `html[data-text-size]` | `display.text_size` |

兼容规则：

- 同时写入 `data-theme="day|night"` 供本地 CSS 选择器使用；主题判断以 shadcn 官方 `.dark` 为准。
- `light/dark` 只控制亮暗，不控制 Theme 品牌色。
- `baseColor` 控制完整中性色底盘：`background`、`foreground`、`card`、`popover`、`secondary`、`muted`、`accent`、`border`、`input`、`sidebar` 都要响应，亮色和暗色都必须覆盖。
- `themeColor` 覆盖高强调区域：主按钮、选中导航、右上角更新 icon、focus ring、progress、switch、active toggle、link、sidebar-primary。日志级别和危险态继续使用语义 token，不被 Theme 覆盖。
- `chartColor` 独立覆盖 `--chart-*`，不能偷用 Theme 或 Accent。
- `iconTone` 默认 `default`，图标使用当前主题语义色；切到 `colorful` 时才允许按图标含义使用彩色语义色。
- `menu` 和 `menuAccent` 必须改变侧栏菜单外观和 active / hover 强度，不能只是更新持久化值。
- `radius` 只通过 `--radius` 派生，不允许卡片、按钮、输入框各自硬编码一套圆角。
- `cardBorder` 要覆盖所有 Card primitive、列表边框、日志表、设置偏好列表、弹窗和 popover。输入框、危险区域和 focus ring 保留必要边界。

## 3. 官方 Token 基线

### 3.1 Tailwind v4 映射

`frontend/src/styles.css` 必须保留 `@theme inline`，并映射完整 shadcn token：

```css
@import "tailwindcss";
@custom-variant dark (&:is(.dark *));

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover);
  --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive);
  --color-destructive-foreground: var(--destructive-foreground);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-ring: var(--ring);
  --color-chart-1: var(--chart-1);
  --color-chart-2: var(--chart-2);
  --color-chart-3: var(--chart-3);
  --color-chart-4: var(--chart-4);
  --color-chart-5: var(--chart-5);
  --color-sidebar: var(--sidebar);
  --color-sidebar-foreground: var(--sidebar-foreground);
  --color-sidebar-primary: var(--sidebar-primary);
  --color-sidebar-primary-foreground: var(--sidebar-primary-foreground);
  --color-sidebar-accent: var(--sidebar-accent);
  --color-sidebar-accent-foreground: var(--sidebar-accent-foreground);
  --color-sidebar-border: var(--sidebar-border);
  --color-sidebar-ring: var(--sidebar-ring);
  --radius-sm: calc(var(--radius) - 4px);
  --radius-md: calc(var(--radius) - 2px);
  --radius-lg: var(--radius);
  --radius-xl: calc(var(--radius) + 4px);
}
```

禁止在首屏 CSS 中引入远程字体 `@import url(...)`；内网或代理不可用时它会阻塞首屏。应用统一使用系统默认字体。

### 3.2 Neutral Light

```css
:root {
  --radius: 0.625rem;
  --background: oklch(1 0 0);
  --foreground: oklch(0.145 0 0);
  --card: oklch(1 0 0);
  --card-foreground: oklch(0.145 0 0);
  --popover: oklch(1 0 0);
  --popover-foreground: oklch(0.145 0 0);
  --primary: oklch(0.205 0 0);
  --primary-foreground: oklch(0.985 0 0);
  --secondary: oklch(0.97 0 0);
  --secondary-foreground: oklch(0.205 0 0);
  --muted: oklch(0.97 0 0);
  --muted-foreground: oklch(0.556 0 0);
  --accent: oklch(0.97 0 0);
  --accent-foreground: oklch(0.205 0 0);
  --destructive: oklch(0.577 0.245 27.325);
  --destructive-foreground: oklch(0.985 0 0);
  --border: oklch(0.922 0 0);
  --input: oklch(0.922 0 0);
  --ring: oklch(0.708 0 0);
  --chart-1: oklch(0.646 0.222 41.116);
  --chart-2: oklch(0.6 0.118 184.704);
  --chart-3: oklch(0.398 0.07 227.392);
  --chart-4: oklch(0.828 0.189 84.429);
  --chart-5: oklch(0.769 0.188 70.08);
  --sidebar: oklch(0.985 0 0);
  --sidebar-foreground: oklch(0.145 0 0);
  --sidebar-primary: oklch(0.205 0 0);
  --sidebar-primary-foreground: oklch(0.985 0 0);
  --sidebar-accent: oklch(0.97 0 0);
  --sidebar-accent-foreground: oklch(0.205 0 0);
  --sidebar-border: oklch(0.922 0 0);
  --sidebar-ring: oklch(0.708 0 0);
}
```

### 3.3 Neutral Dark

```css
.dark {
  --background: oklch(0.145 0 0);
  --foreground: oklch(0.985 0 0);
  --card: oklch(0.205 0 0);
  --card-foreground: oklch(0.985 0 0);
  --popover: oklch(0.205 0 0);
  --popover-foreground: oklch(0.985 0 0);
  --primary: oklch(0.922 0 0);
  --primary-foreground: oklch(0.205 0 0);
  --secondary: oklch(0.269 0 0);
  --secondary-foreground: oklch(0.985 0 0);
  --muted: oklch(0.269 0 0);
  --muted-foreground: oklch(0.708 0 0);
  --accent: oklch(0.269 0 0);
  --accent-foreground: oklch(0.985 0 0);
  --destructive: oklch(0.704 0.191 22.216);
  --destructive-foreground: oklch(0.985 0 0);
  --border: oklch(1 0 0 / 10%);
  --input: oklch(1 0 0 / 15%);
  --ring: oklch(0.556 0 0);
  --chart-1: oklch(0.488 0.243 264.376);
  --chart-2: oklch(0.696 0.17 162.48);
  --chart-3: oklch(0.769 0.188 70.08);
  --chart-4: oklch(0.627 0.265 303.9);
  --chart-5: oklch(0.645 0.246 16.439);
  --sidebar: oklch(0.205 0 0);
  --sidebar-foreground: oklch(0.985 0 0);
  --sidebar-primary: oklch(0.488 0.243 264.376);
  --sidebar-primary-foreground: oklch(0.985 0 0);
  --sidebar-accent: oklch(0.269 0 0);
  --sidebar-accent-foreground: oklch(0.985 0 0);
  --sidebar-border: oklch(1 0 0 / 10%);
  --sidebar-ring: oklch(0.556 0 0);
}
```

## 4. 可切换色彩

### 4.1 Base Color

运行时允许切换本项目实际接入的 base color 名称：

```ts
export type BaseColor = "neutral" | "stone" | "zinc" | "mauve" | "olive" | "mist" | "taupe"
```

实现方式：

- 每个 base color 都提供 light/dark token set。
- 切换时只改 `html[data-base-color]`。
- 组件只能使用 `bg-background text-foreground border-border` 这类语义 class。
- 不允许组件写 `bg-neutral-*`、`bg-zinc-*` 这种硬编码 base color。

首版必须落地 `neutral`、`stone`、`zinc`、`mauve`、`olive`、`mist`、`taupe` 七套 base color，并覆盖 light / dark 的完整中性色 token。

### 4.2 Theme / Accent / Chart Color

运行时颜色分成三条轴，禁止互相偷用：

- `themeColor` 覆盖 `--primary`、`--ring`、`--sidebar-primary`、progress、switch 和高强调选中态。
- `accentColor` 覆盖 `--accent`、`--accent-foreground`、`--sidebar-accent`、hover / focus 辅助强调，不覆盖 `--primary`。
- `chartColor` 独立覆盖 `--chart-*`，只给统计和可视化使用。
- 三条颜色轴都使用同一套 create 颜色集合 24 项：`neutral`、`stone`、`zinc`、`mauve`、`olive`、`mist`、`taupe`、`amber`、`blue`、`cyan`、`emerald`、`fuchsia`、`green`、`indigo`、`lime`、`orange`、`pink`、`purple`、`red`、`rose`、`sky`、`teal`、`violet`、`yellow`。

规则：

- `neutral` 使用默认中性语义 token，不额外制造彩色倾向。
- Theme 用于主按钮、选中态、焦点环、当前导航和关键进度。
- Accent 用于次级强调，不得把主按钮和当前导航改成另一套色。
- Chart Color 不能偷用 Theme 或 Accent。
- 日志级别、错误、警告、成功继续用 `destructive` 或自定义语义 token，不能被 accent 覆盖。

## 5. 设置页显示偏好 UI

shadcn-vue create 轴里已经接入运行时效果的项、亮暗模式、强调色、字号、密度和卡片边框属于全局显示偏好。它们放在设置页顶部的“显示偏好”分区，DOM token 立即响应，持久化进入后端 SQLite KV，不进入后端业务 `Settings` 结构。

位置：

- 桌面宽度：`SettingsPage.vue` 顶部第一张宽卡片。
- Style、Base Color、Theme、Chart Color、Radius、Menu、Menu Accent 在同一分区完整出现。
- 亮暗模式、强调色、图标颜色、字号、密度、卡片边框是本项目附加显示偏好，也放在同一分区。
- Icon Library 当前仍使用 Lucide 实际渲染，没有接入 Tabler/HugeIcons/Phosphor/Remix Icon 包，因此不是设置页控件。
- 更新检查频率、窗口行为、日志保留是后端业务设置，和显示偏好分开。
- `Release 来源` 是只读说明，不放设置页，归到关于页。
- `AppChrome.vue` 右上角必须保留日间/夜间切换 icon 和更新状态 icon；完整调参放设置页。

组件：

- 非颜色选项统一使用 `UiNativeSelect`，避免横铺选项造成设置页不可读。
- 颜色选项统一使用 `SettingsColorSelect` 页面组件；闭合态和展开列表每个选项都必须显示 swatch。
- Style：选项 `Reka / Vega / Nova / Maia / Lyra / Mira / Luma / Sera`。
- Base Color：选项 `Neutral / Stone / Zinc / Mauve / Olive / Mist / Taupe`。
- Theme：选项使用 create 颜色集合 24 项。
- Chart Color：选项使用 create 颜色集合 24 项。
- Icon Tone：选项 `默认颜色 / 彩色图标`，默认 `默认颜色`；彩色图标必须按图标含义使用稳定语义色。
- 彩色图标只用于信息类别、状态和入口，不覆盖激活导航、主按钮、危险按钮、关闭按钮等需要继承控件状态色的图标。
- 语义色规则：更新/刷新/主要流程用 indigo 或 green，日志/存储用 orange，视觉主题用 purple，窗口/布局/中性结构用 gray，危险/错误只在非危险按钮背景上用 red。
- Radius：选项 `默认 / 无 / 小 / 中 / 大`。
- Menu：选项 `Default / Inverted / Default Translucent / Inverted Translucent`。
- Menu Accent：选项 `Subtle / Bold`。
- Mode：选项 `亮色 / 暗色`。
- Accent：选项使用 create 颜色集合 24 项。
- Text Size：选项 `小 / 正常 / 中 / 大`。
- Density：选项 `紧凑 / 舒展`。
- Card Border：选项 `清晰 / 柔和 / 隐藏`。

交互规则：

- 点击立即写 DOM token，并异步排队保存到 SQLite KV。
- 后端保存失败时显示错误，但不阻塞当前页面的即时预览。
- 切换不能触发页面重新加载。
- 切换后当前页面不应产生布局跳动；字号变化允许行高变化，但不允许横向溢出。
- “隐藏边框”不能让危险区域、输入框、日志表失去可读边界；这些控件仍保留必要 focus / input border。

## 6. 系统字体和字号

```css
:root {
  --font-sans: system-ui, sans-serif;
  --font-heading: inherit;
  --font-mono: ui-monospace, monospace;
  --fs-title: 24px;
  --fs-section: 18px;
  --fs-body: 14px;
  --fs-caption: 12px;
  --fs-mono: 12px;
}

:root[data-text-size="small"] {
  --fs-title: 22px;
  --fs-section: 17px;
  --fs-body: 13px;
  --fs-caption: 11px;
  --fs-mono: 11px;
}

:root[data-text-size="medium"] {
  --fs-title: 25px;
  --fs-section: 19px;
  --fs-body: 15px;
  --fs-caption: 13px;
  --fs-mono: 13px;
}

:root[data-text-size="large"] {
  --fs-title: 28px;
  --fs-section: 21px;
  --fs-body: 16px;
  --fs-caption: 14px;
  --fs-mono: 14px;
}
```

规则：

- 禁止远程字体加载，禁止把外部字体服务作为首屏依赖。
- 设置页不提供字体族选择；字体族统一使用系统默认字体。
- `letter-spacing: 0`。
- 日志用 `--font-mono` 和 `--fs-mono`。
- 大字号下路径、日志、版本号必须 `min-width: 0` 和 `overflow-wrap: anywhere`。

## 7. 组件规范

必须优先使用或补齐这些 shadcn primitive。业务页面不得绕过 primitive 自造同类控件：

| Component | 文件 | 用途 |
| --- | --- | --- |
| `Button` | `components/ui/button/*.vue` | 主操作、次操作、危险操作 |
| `Card` | `components/ui/card/*.vue` | 页面内容分组 |
| `Badge` | `components/ui/badge/*.vue` | 状态、版本、计数 |
| `Input` | `components/ui/input` | 表单输入 |
| `Switch` | `components/ui/switch/*.vue` | 二元设置 |
| `Progress` | `components/ui/progress/*.vue` | 下载和校验进度 |
| `NativeSelect` | `shared/ui/NativeSelect.vue` | 检查间隔、保留周期、日志筛选 |
| `SettingsColorSelect` | `features/settings/SettingsColorSelect.vue` | 设置页带 swatch 的颜色选择 |
| `AlertDialog` | `components/ui/alert-dialog` | 清空日志、危险操作确认 |
| `Dialog` | `components/ui/dialog` | 更新状态弹窗、通用模态承载 |
| `Field` / `Label` | `shared/ui/Field.vue` / `shared/ui/Label.vue` | 表单字段结构 |
| `ToggleGroup` | `components/ui/toggle-group` | 少量互斥分段控制 |
| `Tooltip` | `components/ui/tooltip` | 顶栏图标按钮解释 |
| `Table` | `components/ui/table` | 日志流、表格型数据 |

后续按需补齐官方组件：

| Component | 用途 |
| --- | --- |
| `Select` | 搜索、分组或复杂选择器 |
| `Separator` | 页面局部分隔 |
| `DropdownMenu` 或 `Popover` | 轻量菜单或非模态信息层 |
| `Tabs` | 同页多分区切换 |

组件规则：

- shadcn primitive 必须只吃 props 和 className，不 import `app/store`。
- `shared/ui` 兼容层只能适配项目 API 和官方组件差异，不承载业务流程。
- 业务页面不得直接写一堆裸 `<button className="...">`，能用 `Button` 就用 `Button`。
- 表单字段优先使用 `UiField` / `UiLabel` 结构；复杂表单再引入官方 Form / FieldGroup。
- 选择器当前使用 `UiNativeSelect`；需要搜索、分组或复杂弹层时补官方 `Select` 结构。
- 设置页颜色选择必须使用 `SettingsColorSelect`；其它页面需要同类控件时先评估是否抽到 `shared/ui`，不得写进 `components/ui`。
- 分段控制统一使用 `ToggleGroup`，业务页不再定义 `SegmentedControl`。
- 弹出层统一通过 `Dialog` / `AlertDialog` composition，禁止手写 fixed overlay 和 `window.confirm`。
- 表格型数据统一使用 `Table` primitive，业务页不再用 `role="table"` 的 div 网格。
- 图标使用 `@lucide/vue`，大小默认 `16-20px`。
- 按钮默认高度 `36px`，紧凑按钮 `32px`，图标按钮宽高一致。
- 卡片 radius 使用 `rounded-lg` 或 token，不做随机大圆角。
- 阴影保持 `shadow-sm`；深色模式主要靠 token 和 border，不靠黑色重阴影。

## 8. 页面功能结构

### 概览

只做：

- 应用名称、版本、启动时间、日志健康摘要。
- 进入日志、设置、关于的任务入口。
- 可以显示更新状态摘要，但不能放检查/下载/安装主操作。
- 不放检查更新按钮。

### 更新弹窗

更新不再是独立页面，不出现在侧边栏或窄屏导航。唯一入口是右上角 update icon，点击打开 Popover/Dialog：

- 检查 Release。
- 判断当前版本 / 最新版本。
- SHA256 是否存在。
- 后台静默下载和校验进度。
- 下载校验完成后提供“马上更新”和“下次启动再更新”。
- Release 诊断、缓存路径、最近一次检查结果可以折叠在弹窗内；不展示更新历史或事件列表。

按钮规则：

- 空闲或失败：主按钮 `检查更新`。
- 可更新且未下载：后端可后台下载；UI 展示进度和安全状态。
- 已校验：主按钮 `马上更新`，次按钮 `下次启动再更新`。
- 缺 SHA256 时禁止下载和安装，只展示原因。

后端规则：

- `CheckUpdate()` 检查到可更新且有 SHA256 时可以后台下载，但不能自动启动安装器。
- `DownloadUpdate()` 只下载并校验，校验通过返回 `verified`，不直接退出应用。
- `ScheduleDownloadedUpdateOnStartup()` 把已校验安装包标记为下次启动安装，并把路径、版本和 SHA256 写入 `data/updates/pending.json`。
- `InstallPendingUpdateOnStartup()` 启动期读取 `pending.json`，安装成功、安装启动失败、校验失败或 pending 读取失败后必须清理该文件。
- `InstallDownloadedUpdate()` 只在用户点击“马上更新”或启动期 pending 策略触发时调用。
- pending 安装包路径必须在更新缓存目录 `data/updates/` 内，禁止启动期消费缓存目录外的安装包路径。

### 日志

只做：

- 来源、级别、关键词筛选。
- 统计摘要。
- 日志流。
- 分页。
- 当前筛选范围清理。

布局规则：

- 日志页是工作台，默认让日志流成为首屏主体，不做左右割裂的大空布局。
- 日志界面默认折叠筛选；筛选按钮显示当前筛选数量，展开后出现来源、级别、关键词、重置和清空当前视图。
- 自动刷新、手动刷新和专注模式属于日志页顶层工具；专注模式必须把剩余空间优先给日志表格，不再让筛选和统计挤掉日志流。
- 中间是统计条和日志表格；日志表格优先保证内容列可读，时间、来源、级别固定宽度，内容列占满剩余宽度并允许换行。
- 底部是分页和匹配数量。
- 清空日志必须使用 `AlertDialog`，禁止 `window.confirm`。

### 设置

只放可修改项，包含两个分区。

显示偏好分区：

- 主题模式。
- base color。
- theme color。
- accent color。
- chart color。
- 图标颜色。
- 菜单。
- 菜单强调。
- 字号。
- 圆角。
- 密度。
- 卡片边框强度。
- 这些偏好通过后端保存到 SQLite KV，不写浏览器本地存储。

后端业务设置分区：

- `updateCheckIntervalHours`
- `minimizeToTray`
- `logRetentionDays`
- `autoLaunch`
- `createDesktopShortcut`
- `launchHiddenToTray`

布局规则：

- 后端业务设置压成一个紧凑卡片，顺序固定为窗口行为、启动设置、检查间隔、日志保留；检查间隔和日志保留两个时间类设置必须相邻。
- 每一行必须有且只有一个可编辑控件。
- 不在业务设置卡片里塞策略说明、统计计数或只读状态。
- `minimizeToTray` 字段的用户可见文案必须是“关闭到系统托盘”；它只影响点击关闭按钮时隐藏窗口，点击最小化仍应进入任务栏。
- `autoLaunch` 字段的用户可见文案必须是“开机自启”，默认关闭。
- `createDesktopShortcut` 字段的用户可见文案必须是“创建桌面快捷图标”，默认开启。
- `launchHiddenToTray` 字段的用户可见文案必须是“开机自启时隐藏到托盘”；控件只在开机自启开启时生效。
- `updateCheckIntervalHours` 选项固定为 `1 / 3 / 6 / 12 小时`，默认值为 `3 小时`。

禁止放：

- 更新检查主操作。
- 日志清理主操作。
- 关于页技术栈信息。
- 只读信息卡片。只读的 Release 来源、版本、路径、运行环境都属于关于或更新弹窗诊断。
- 只有描述、不能修改的“伪设置”。
- “托盘菜单只包含显示/退出”这类实现约束。

### 关于

只做：

- 应用元数据。
- Release 来源只读摘要。
- 运行环境。
- 设置文件、SQLite、缓存路径、待安装更新文件。
- 技术栈。
- 当前 Style、Base Color、Theme、Chart Color、图标实现、图标颜色、Radius、Menu、Menu Accent 的只读摘要。

## 9. 布局和响应式

桌面优先，不做手机精修。

| Width | 规则 |
| --- | --- |
| `>= 1600px` | sidebar `280px`，页面最大 `1560px`，双列可保留 |
| `1280-1599px` | sidebar `256px`，内容区不居中过窄 |
| `1100-1279px` | sidebar `220px`，复杂页面单列 |
| `980-1099px` | sidebar 可保留或收窄；内容必须单列 |
| `< 980px` | 隐藏 sidebar，顶部 compact nav，只保证可用 |

规则：

- `.app-shell` 固定两列：`sidebar + minmax(0, 1fr)`。
- `.content-scroll` 是唯一页面滚动容器。
- `.page-stack` 允许最大宽度，但不能造成两侧大空地；1280 宽下内容应占右侧大部分。
- `Card` 不嵌套 `Card`。
- 表格、日志、路径可内部横滚，但不能撑破 app shell。
- 所有 grid 子项必须 `min-width: 0`。

## 10. 验收清单

设计验收：

- `DESIGN.md` 以 shadcn 官方 CSS variables 和本项目 token 为当前方案。
- 明确写入 shadcn 官方 CSS variables、`.dark`、`@theme inline`、`components.json`。
- 明确支持已接入运行时效果的 shadcn-vue create 轴：Style、Base Color、Theme、Chart Color、Radius、Menu、Menu Accent。
- 明确写清 Icon Library 暂不作为设置项；未接入多图标包渲染前固定使用 Lucide。
- 明确支持本项目附加显示偏好：light/dark mode、accent color、icon tone、text size、density、card border。
- 显示偏好放在设置页，但不写后端 Settings。
- 左下角 sidebar footer 不显示技术栈；技术栈放关于页。
- 所有页面职责清楚，功能不乱塞。
- 工程硬约束写入本文档：测试只进 `tests/`、临时文件只进 `.tmp/`、禁止远程字体加载、设置页只放可改控件。

代码验收：

- `frontend/components.json` 存在，并能被 shadcn skill/CLI 读取。
- `styles.css` 暴露完整 shadcn token，包括 chart 和 sidebar。
- `styles.css` 不包含页面/组件私有选择器；页面 CSS 归属 `features/**`，项目级组件 CSS 归属 `shared/ui/*` 或业务组件相邻 CSS。
- `styles/layout.css` 只包含跨页面布局 primitive 和语义图标工具，不承载业务页面样式。
- `shared/ui/Card.vue` 和 `shared/ui/CardTitle.vue` 承接 `card_border` 与标题字体 token；`components/ui/card/*` 保持 shadcn-vue CLI primitive，不手写项目 style。
- `useDisplayPreferences()` 管理 mode/style/base/theme/chart/icon-tone/radius/menu/menu-accent/accent/text-size/density/card-border。
- `SettingsPage.vue` 同时展示显示偏好和后端 `Settings`，但两者状态边界清楚。
- `SettingsPage.vue` 的颜色选择使用 `SettingsColorSelect`，业务页不手写散落的 swatch dropdown。
- 设置页后端业务设置只包含 `updateCheckIntervalHours`、`minimizeToTray`、`logRetentionDays`、`logLevel`、`autoLaunch`、`createDesktopShortcut`、`launchHiddenToTray` 七个可改项。
- 设置页业务设置顺序固定为“关闭到系统托盘 / 开机自启 / 开机自启时隐藏到托盘 / 创建桌面快捷图标 / 检查间隔 / 保留周期 / 日志级别”。
- `AboutPage.vue` 展示 Vue 3、Tailwind、shadcn-vue、shadcn-vue skill 和只读运行信息。
- `components/ui` 是 shadcn-vue CLI 专属目录，至少包含 Button、Card、Badge、Input、Switch、Progress、AlertDialog、Dialog、Select、Toggle、ToggleGroup、Tooltip、Table 等官方 primitive。
- `components/ui/README.md` 明确禁止业务组件、业务 class、全局注册和兼容 wrapper 进入该目录。
- `shared/ui/plugin.ts` 全局注册 `Ui*` primitive / wrapper，业务页默认不重复 import 基础 shadcn 组件。
- `.tmp/.gitkeep` 保留临时目录，截图和临时日志不散落到工程根目录。
- `README.md` 是项目说明文档，必须覆盖目录结构、数据约定、常用入口、更新链路、元数据同步和前端组件边界。
- 侧边栏和窄屏导航没有独立更新页面。
- 右上角保留日间/夜间切换 icon 和更新 icon。
- 彩色图标模式下，激活的侧边栏和窄屏导航图标继承选中态颜色，不使用语义彩色。
- 托盘菜单只包含 `显示`、`退出`。

浏览器验收：

- 检查宽度：`1600 / 1280 / 1100 / 980`。
- 检查模式：light / dark。
- 检查 base color：neutral / stone / zinc / mauve / olive / mist / taupe。
- 检查 theme：create 颜色集合 24 项。
- 检查 chart color：create 颜色集合 24 项。
- 检查远程字体：首屏 CSS 不包含 `fonts.googleapis` 或 `@import url(...)`。
- 检查 icon tone：默认颜色 / 彩色图标。
- 检查 accent：create 颜色集合 24 项。
- 检查 radius：default / none / small / medium / large。
- 检查 menu/menu accent：侧栏菜单外观和选中强度响应。
- 检查字号：small / normal / medium / large。
- 检查 density：compact / comfortable。
- 五个页面都不能横向撑破 app shell。

验证命令：

```powershell
cd D:\app\go\go-desktop\tests
go test ./...
```

前端验证优先使用 TypeScript/Vite API 或可用的 npm 脚本：

```powershell
cd D:\app\go\go-desktop\frontend
npm run build
```
