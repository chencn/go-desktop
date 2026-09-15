// 文件职责：Runtime 状态容器定义。集中持有设置、日志、更新、授权、窗口和 Wails API 状态。
// 装配与生命周期见 service.go，DTO 定义见 types.go。

package runtime

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
	"github.com/chencn/go-desktop/internal/adapters/filelog"
	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	"github.com/chencn/go-desktop/internal/desktopapp/display"
	updater "github.com/chencn/go-desktop/internal/desktopapp/update"
	processutil "github.com/chencn/go-desktop/internal/platform/process"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Runtime 是桌面应用的进程级状态容器。
// lock 保护可变状态；耗时更新操作额外由 updateOperationLock 串行化。
type Runtime struct {
	// options 初始化配置（只读，初始化后不变）
	options ServiceOptions

	// releaseChecker 是默认 GitHub 源检查器；非默认设置会由 updateChecker 临时构造检查器。
	releaseChecker *githubrelease.Checker

	// startedAt 应用启动时间（UTC）
	// 用于计算运行时长和日志时间戳
	startedAt time.Time

	// databasePath 数据库文件路径
	databasePath string

	// logger 是应用统一结构化日志入口。
	logger *slog.Logger

	// shuttingDown 表示运行时正在释放资源，禁止重新打开日志文件。
	shuttingDown bool

	// logLevel 允许设置变更时动态调整 slog 过滤级别。
	logLevel *slog.LevelVar

	// logWriter 是每日文件 writer，只负责按日期切文件。
	logWriter *filelog.DailyWriter

	// logDirPath 文件日志目录。
	logDirPath string

	// logFilePattern 描述每日文件命名规则，便于环境信息展示和排查。
	logFilePattern string

	// logCleanupStop 停止启动期日志保留清理任务。
	logCleanupStop context.CancelFunc

	// crashReporter 最早期崩溃日志器
	// error/panic 和 Wails 生命周期异常会同步写入，避免 Runtime 日志不可用时丢线索
	crashReporter *CrashReporter

	// processCapture 当前 stdout/stderr 捕获器
	// 用于把非 log/slog 的进程标准流输出也接入日志页
	processCapture *processutil.StreamCapture

	// processLogRestore 记录安装进程日志捕获前的全局 log/slog 状态
	// Shutdown 会用它恢复标准库日志和结构化日志，避免留下跨运行时副作用
	processLogRestore *processLogRestore

	// cachePath 是更新包和更新状态文件的根目录。
	cachePath string

	// store SQLite 数据库实例
	// 只用于持久化配置项
	store *configstore.Store

	// configDefaultsEnsured 标记当前 store 是否已经写入过配置默认项元数据
	configDefaultsEnsured bool // configDefaultsEnsured 避免每次设置保存都重复刷新全部默认配置项。

	// updateManager 处理安装包下载、校验和安装器启动。
	updateManager *updater.Manager

	// updateOperationLock 串行化下载、安排和安装，避免同一 .download 临时文件和 pending/verified 状态被并发改写。
	updateOperationLock sync.Mutex

	// licenseMode 授权模式；只有 required 会触发授权。
	licenseMode string

	// licensePublicKey 授权公钥，来自构建期注入。
	licensePublicKey string

	// licenseDeviceCode 可选设备码覆盖，主要用于测试。
	licenseDeviceCode string

	// licenseDeviceCodeSource 可选设备码生成函数，主要用于测试。
	licenseDeviceCodeSource func() string

	// lock 保护 Runtime 可变状态；不要在持锁期间执行网络、文件下载或 Wails 退出等耗时副作用。
	lock sync.RWMutex

	// wailsApp Wails 应用实例
	// 用于访问窗口、托盘等 GUI 功能
	wailsApp *application.App

	// mainWindow 主窗口实例
	// 用于窗口操作（显示、隐藏、事件发送等）
	mainWindow *application.WebviewWindow

	// splashWindow 启动加载窗口实例
	// 前端 initialise 完成后首次调用 ShowMainWindow 时自动关闭
	splashWindow *application.WebviewWindow

	// settings 当前应用设置
	// 包含 GitHub 配置、更新间隔、日志保留等
	settings Settings

	// updateSchedulerStop 停止后台更新检查任务。
	updateSchedulerStop context.CancelFunc

	// displayPreferences 当前显示偏好
	// 由 SQLite KV 配置项加载，前端只通过 typed facade 读取和保存
	displayPreferencesV2 display.PreferencesV2 // displayPreferencesV2 保存完整显示偏好 JSON profile。
	displayPreferences   DisplayPreferences    // displayPreferences 保存当前方案生效偏好，供前端读取。

	// logs 内存中的日志条目
	// 用于当前前端视图和文件不可读时兜底
	logs []LogEntry

	// logViewClearedAt 记录当前前端视图的清空时间，不删除文件日志。
	logViewClearedAt map[string]time.Time

	// latestUpdateCheck 是当前进程最近一次更新检查结果，只供 DownloadUpdate 消费，不从磁盘恢复。
	latestUpdateCheck githubrelease.CheckResult

	// hasUpdateCheck 标记 latestUpdateCheck 是否可用；重启后不从数据库恢复。
	hasUpdateCheck bool

	// updateState 当前内存更新状态；静止态可由 verified.json 恢复。
	updateState UpdateStatus

	// forceQuit 强制退出标志
	// 为 true 时，关闭窗口直接退出而不是隐藏到托盘
	forceQuit bool

	// secondStart 多实例启动记录
	// 当用户第二次启动应用时记录参数
	secondStart []SecondInstanceRecord
}
