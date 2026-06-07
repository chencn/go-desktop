# go-desktop

`go-desktop` 是一个基于 Wails3、Go、Vue 3、TypeScript、Tailwind v4 和 shadcn-vue 的中文桌面工具项目。

后端负责桌面生命周期、配置持久化、文件日志、更新检查、安装器启动和 Windows 系统集成；前端负责概览、日志、设置、关于、更新弹窗和显示偏好。

视觉、交互、组件边界和工程硬约束见 [DESIGN.md](DESIGN.md)。

## 项目结构

| 路径 | 说明 |
| --- | --- |
| `main.go` | Wails3 应用入口，组装 runtime、窗口、托盘和启动期流程 |
| `app/` | 暴露给 Wails 的应用服务入口 |
| `internal/desktopapp/runtime/` | 运行时状态、Wails API、日志、设置、更新、窗口控制 |
| `internal/desktopapp/settings/` | 业务设置定义、默认值、归一化和 SQLite KV 映射 |
| `internal/desktopapp/display/` | 显示偏好定义、默认值和 SQLite KV 映射 |
| `internal/adapters/` | SQLite 配置存储、文件日志、GitHub Release 检查等适配层 |
| `internal/platform/` | 路径、安装器、快捷方式、进程检测等平台能力 |
| `frontend/` | Vue 3 + TypeScript + Vite 前端 |
| `frontend/src/components/ui/` | shadcn-vue CLI 生成组件目录 |
| `frontend/src/shared/ui/` | 项目 UI wrapper、兼容层和全局 `Ui*` 注册 |
| `frontend/src/features/` | 业务页面、页面私有组件和页面 CSS |
| `tests/` | 独立 Go 测试模块和前端测试入口 |
| `build/` | Wails 平台配置、图标、NSIS/MSIX 等安装器资源 |
| `scripts/` | 元数据同步、版本解析、本地更新 staging、Windows npm 环境包装 |
| `project.metadata.json` | 产品元数据源 |
| `.tmp/` | 临时截图、临时日志和一次性输出 |

## 数据约定

运行数据默认放在可执行文件所在目录的 `data/` 下，开发环境兜底为当前工作目录的 `data/`。

| 路径 | 用途 |
| --- | --- |
| `data/go-desktop.db` | SQLite KV 配置库，只保存 `config_items` |
| `data/logs/` | 每日 JSONL 文件日志 |
| `data/updates/` | 更新安装包缓存、verified / pending 状态 |
| `data/updates/pending.json` | “下次启动安装”的待安装状态 |

SQLite 不保存日志、更新历史或更新事件。读配置失败时后端降级为默认值；写配置失败必须返回错误，前端展示保存失败。

## 常用命令

```powershell
wails3 dev -config ./build/config.yml -port 9245
```

完整开发模式。

```powershell
wails3 task common:dev:frontend
```

只启动前端 Vite dev server。

```powershell
wails3 task common:build:frontend
```

只构建前端。

```powershell
wails3 task common:generate:bindings
```

生成 Wails TypeScript 绑定。

```powershell
wails3 task windows:build
wails3 task windows:package
```

Windows 构建和打包，打包默认走 NSIS。

```powershell
cd D:\app\go\go-desktop\tests
go test ./...
```

独立 Go 测试模块入口。

```powershell
cd D:\app\go\go-desktop\frontend
npm run build
```

前端构建入口；日常优先使用 Wails Taskfile 包装命令。

## 前端入口

主页面：

- `概览`：应用运行状态、版本、启动时间、日志健康摘要。
- `日志`：日志文件、筛选、统计、表格、分页和清理。
- `设置`：业务设置和显示偏好。
- `关于`：应用元数据、Release 来源、运行环境、本地路径和技术栈。

更新不是独立页面。右上角更新按钮打开 `UpdateStatusDialog`，负责检查、下载、校验、立即安装和下次启动安装。

显示偏好由 `frontend/src/app/display.ts` 管理，包含：

- 亮暗模式。
- shadcn-vue create 相关轴：Style、Base Color、Theme Color、Chart Color、Radius、Menu、Menu Accent。
- 项目附加轴：Accent Color、Icon Tone、Text Size、Density、Card Border。

显示偏好切换会立即写 DOM token，并异步保存到后端 SQLite KV。

## 后端设置

业务设置定义在 `internal/desktopapp/settings/settings.go`。

当前字段：

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `updateSource` | `github` | 更新源，支持 `github` 和 `local` |
| `githubOwner` | 元数据 owner | GitHub Release owner |
| `githubRepo` | 元数据 repo | GitHub Release repo |
| `githubProxyBase` | 空 | GitHub API 代理地址 |
| `updateCheckIntervalHours` | `3` | 自动检查间隔，允许 `1 / 3 / 6 / 12` |
| `minimizeToTray` | `true` | 点击关闭按钮时隐藏到托盘 |
| `logRetentionDays` | `30` | 每日文件日志保留周期，`-1` 表示永不清理 |
| `logLevel` | `info` | 最小日志级别 |
| `autoLaunch` | `false` | 开机自启 |
| `createDesktopShortcut` | `true` | 创建桌面快捷图标 |
| `launchHiddenToTray` | `false` | 开机自启时隐藏到托盘 |

保存设置时会先归一化，再写 SQLite KV；开机自启和桌面快捷方式还会同步 Windows 系统集成。系统集成失败时后端会回滚内存设置和 SQLite 配置。

## 更新链路

更新源：

- `github`：读取 GitHub Release。
- `local`：读取 `project.metadata.json` 中 `update.localBaseUrl + update.localManifestPath` 拼出的静态 manifest。

主要 API：

| API | 说明 |
| --- | --- |
| `CheckUpdate()` | 检查更新；发现可更新且有 SHA256 时可自动下载并校验 |
| `DownloadUpdate()` | 下载并校验最近一次检查结果 |
| `GetUpdateStatus()` | 返回当前更新生命周期状态 |
| `InstallDownloadedUpdate()` | 重新校验本地安装包后启动静默安装器 |
| `ScheduleDownloadedUpdateOnStartup()` | 写入 `data/updates/pending.json`，下次启动安装 |
| `InstallPendingUpdateOnStartup()` | 启动期读取 pending 状态并安装 |

安全规则：

- 缺 SHA256 时禁止下载和安装。
- 安装前必须重新计算 SHA256。
- pending 安装包路径必须在 `data/updates/` 内。
- 安装成功启动、安装失败、校验失败或 pending 读取失败后必须清理待安装状态。
- 更新状态只表示当前生命周期，不保存历史。

## 元数据同步

`project.metadata.json` 是产品元数据源，包含应用名、仓库、默认版本、Windows 标识、安装路径、更新源和默认设置。

同步命令：

```powershell
go run ./scripts/sync_project_metadata.go -sync
```

同步范围包括：

- 根 `Taskfile.yml`
- 前端 HTML / TypeScript 元数据
- Go 元数据
- 安装器配置
- 平台配置
- release workflow

根 `Taskfile.yml` 文件头标了生成来源。修改根 `Taskfile.yml` 的固定模板时，要同步修改 `scripts/sync_project_metadata.go`。`build/Taskfile.yml` 是通用构建任务文件，不由该同步脚本生成。

## 组件边界

项目使用 shadcn-vue 源码组件，但目录边界必须固定：

- `frontend/src/components/ui/`：shadcn-vue CLI 生成或覆盖的官方 primitive。
- `frontend/src/shared/ui/`：项目 wrapper、旧 API 兼容层、`Ui*` 全局注册。
- `frontend/src/features/**`：业务页面、页面私有组件、页面私有样式。

规则：

- 业务页面默认使用全局 `Ui*` 组件。
- 普通按钮、弹窗、表格、tooltip、progress 优先使用 shadcn primitive。
- `components/ui` 内禁止放业务组件、业务 class、全局注册和项目 wrapper。
- 颜色选择使用 `features/settings/SettingsColorSelect.vue`。
- 简单下拉使用 `shared/ui/NativeSelect.vue`。

## Windows 环境兜底

部分 Windows 自动化或精简 shell 会缺少 `SystemDrive`、`ProgramData`、`SystemRoot` 等变量。Wails、Node、WebView 或 Windows 运行时如果拿到未展开的 `%SystemDrive%\ProgramData`，可能在当前工作目录下创建 `frontend/%SystemDrive%/ProgramData/...`。

仓库在这些位置做兜底：

- `scripts/envrun/main.go`
- 根 `Taskfile.yml`
- `build/Taskfile.yml`
- `build/windows/Taskfile.yml`

`frontend/%SystemDrive%/` 已在 `.gitignore` 中忽略；它是异常环境下的缓存副产物，不是源码。

## 维护规则

- 先读代码，先找根因。
- 精确修改，避免无关重写。
- 测试只放 `tests/` 独立模块。
- 临时截图、调试日志和一次性输出只写 `.tmp/`。
- UI 调试优先 Browser / Chrome 插件。
- 未经明确确认，不做删除、覆盖、回滚、清理类危险操作。
