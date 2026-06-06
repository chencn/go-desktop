# go-desktop

`go-desktop` 是一个基于 Wails3、Vue 3、Tailwind v4 和 shadcn-vue 的中文桌面应用模板/工具项目。后端负责桌面生命周期、配置持久化、文件日志、更新检查和安装器启动；前端负责桌面 UI、设置、日志、更新弹窗和关于页。

这份 README 只说明项目是什么、目录怎么读、常用命令怎么跑。视觉、交互、组件边界和工程硬约束看 `DESIGN.md`。

## 项目结构

- `main.go`：Wails3 应用入口，组装 runtime、窗口、托盘和启动期流程。
- `internal/desktopapp/runtime/`：桌面应用运行时和 Wails API 实现。
- `internal/adapters/`：配置存储、文件日志、GitHub Release 等外部适配层。
- `internal/platform/`：路径、安装器、快捷方式、进程检测等平台能力。
- `frontend/`：Vue 3 + TypeScript + Vite 前端。
- `frontend/src/components/ui/`：shadcn-vue CLI 生成组件目录，只能通过 CLI 覆盖。
- `frontend/src/shared/ui/`：项目 UI 兼容层和全局 `Ui*` 注册。
- `frontend/src/features/`：业务页面和页面私有组件。
- `tests/`：独立 Go 测试模块，生产目录旁边不放 Go `_test.go`。
- `build/`：Wails 平台构建配置、图标、NSIS/MSIX 等安装器资源。
- `scripts/`：项目工具脚本，例如元数据同步和 Windows npm 环境包装。
- `project.metadata.json`：产品元数据源。
- `.tmp/`：临时截图、临时日志和一次性输出目录。

## 数据约定

运行数据默认放在可执行文件所在目录的 `data/` 下，开发兜底为当前工作目录的 `data/`。

- `data/go-desktop.db`：SQLite，只保存 `config_items` 配置项。
- `data/logs/`：每日文件日志。
- `data/updates/`：更新缓存目录，`pending.json` 保存“下次启动安装”的待安装状态。

SQLite 不保存日志、更新历史或更新事件。读配置失败时后端降级为默认值；写配置失败必须返回错误，前端展示保存失败。

## 常用入口

- `wails3 dev -config ./build/config.yml -port 9245`：完整开发模式。
- `wails3 task common:dev:frontend`：只启动前端 Vite dev server。
- `wails3 task common:build:frontend`：只构建前端。
- `wails3 task common:generate:bindings`：生成 Wails TypeScript 绑定。
- `wails3 task windows:build`：Windows 构建。
- `wails3 task windows:package`：Windows 打包，默认走 NSIS。
- `cd tests && go test ./...`：独立 Go 测试模块入口。
- `cd frontend && npm run build`：前端构建入口；日常优先走 Taskfile 包装命令。

## 更新链路

右上角更新入口触发检查和弹窗流程，不提供独立更新页面。

- `CheckUpdate()`：检查 GitHub Release；发现可更新且有 SHA256 时可触发后台下载。
- `DownloadUpdate()`：下载并校验安装包，校验通过后返回 `verified`，等待用户选择安装时机。
- `InstallDownloadedUpdate()`：启动静默安装器，成功启动后退出当前应用。
- `ScheduleDownloadedUpdateOnStartup()`：把已校验安装包标记为下次启动安装，并写入 `data/updates/pending.json`。
- `InstallPendingUpdateOnStartup()`：启动期读取 `pending.json`，成功或失败后清理待安装状态。

安装包路径必须在 `data/updates/` 更新缓存目录内，避免外部路径被启动期安装流程消费。

## 元数据同步

`project.metadata.json` 是产品元数据源，包含应用名、仓库、版本兜底、Windows 标识、安装路径和默认设置。

`scripts/sync_project_metadata.go` 不是打包入口。它只在运行下面命令时同步生成内容：

```powershell
go run ./scripts/sync_project_metadata.go -sync
```

同步范围包括根 `Taskfile.yml`、前端 HTML、Go/TypeScript 元数据、安装器配置、平台配置和 release workflow。根 `Taskfile.yml` 文件头标了“生成”，所以改根 `Taskfile.yml` 的固定模板时，要同步改 `scripts/sync_project_metadata.go`。`build/Taskfile.yml` 是通用构建任务文件，不由这个同步脚本生成。

## 前端组件边界

项目使用 shadcn-vue 源码组件，但 `frontend/src/components/ui/` 不是业务组件目录。

- shadcn-vue 官方 primitive：放在 `frontend/src/components/ui/`，通过 `shadcn-vue add ... --overwrite` 更新。
- 项目兼容 wrapper 和全局 `Ui*` 注册：放在 `frontend/src/shared/ui/`。
- 页面私有控件：放在对应 `frontend/src/features/**`。
- 具体视觉、交互和组件规则：看 `DESIGN.md`。

## Windows 环境兜底

部分 Windows 自动化或精简 shell 会缺少 `SystemDrive`、`ProgramData`、`SystemRoot` 等变量。Wails、Node、WebView 或 Windows 运行时如果拿到未展开的 `%SystemDrive%\ProgramData`，可能会在当前工作目录下创建 `frontend/%SystemDrive%/ProgramData/...`。

仓库在这些位置做兜底：

- `scripts/envrun/main.go`：包装 npm 子进程，补齐 Windows 基础环境变量。
- `Taskfile.yml`：根 `dev` 和 `test` 任务的环境兜底。
- `build/Taskfile.yml`：前端安装、构建、绑定生成和 dev server 的环境兜底。
- `build/windows/Taskfile.yml`：Windows 构建和 syso 生成的环境兜底。

`frontend/%SystemDrive%/` 已在 `.gitignore` 中忽略；它是异常环境下的缓存副产物，不是源码。确认没有前端 dev server、Wails 进程或构建命令正在运行后，可以删除该目录清理现场，但真正防止复现的是上面的环境兜底。
