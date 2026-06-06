# go-desktop 前端

这里是 Wails3 桌面应用的 Vue 3 + TypeScript + Vite 前端。前端通过 `frontend/bindings/` 和 `frontend/src/api/wails.ts` 调用 Go 后端服务。

## 推荐入口

日常不要把裸 `npm` 当成主入口。优先走 Wails/Taskfile：

- `wails3 dev -config ./build/config.yml -port 9245`：完整开发模式。
- `wails3 task common:dev:frontend`：只启动 Vite dev server。
- `wails3 task common:build:frontend`：只构建前端。
- `wails3 task common:generate:bindings`：生成 Wails TypeScript 绑定。

这些任务会在 Windows 下补齐必要环境变量，再调用 `scripts/envrun/main.go` 包装 npm 子进程。

## `%SystemDrive%` 目录是什么

如果当前 Windows shell 缺少 `SystemDrive` 或 `ProgramData`，某些 Windows 运行时会把 `%SystemDrive%\ProgramData\...` 当成未展开的相对路径。因为 Vite/npm 任务的工作目录是 `frontend/`，缓存就会落成：

```text
frontend/%SystemDrive%/ProgramData/Microsoft/Windows/Caches/...
```

这个目录不是源码，也不是 Wails 绑定。它是异常环境变量导致的 Windows 缓存副产物，已被 `.gitignore` 忽略。

## 兜底在哪里

- `build/Taskfile.yml`：前端依赖安装、构建、绑定生成和 dev server 的任务级环境。
- `Taskfile.yml`：根 `dev` 任务的环境；这个文件由 `scripts/sync_project_metadata.go` 生成。
- `build/windows/Taskfile.yml`：Windows 构建和 syso 生成任务。
- `scripts/envrun/main.go`：npm 子进程环境包装器。

如果以后改根 `Taskfile.yml` 的生成内容，同步改 `scripts/sync_project_metadata.go`，否则运行 `go run ./scripts/sync_project_metadata.go -sync` 后会覆盖根 Taskfile。

## 清理规则

确认没有前端 dev server、Wails 进程或构建命令正在运行后，可以删除 `frontend/%SystemDrive%/` 清理异常缓存。但这只是清理现场；真正防止复现的是上面的 `SystemDrive` 和 `ProgramData` 环境兜底。
