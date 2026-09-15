// 文件职责：应用信息与运行环境 DTO 的读取实现，供 app facade 和前端诊断页使用。

package runtime

import (
	goruntime "runtime"
	"time"
)

// GetAppInfo API 方法，返回应用元数据。
// 前端调用: wails.App.GetAppInfo()
func (api *API) GetAppInfo() (info AppInfo, err error) {
	defer api.recoverError("读取应用信息", &err)
	api.runtime.RecordLogWithSeverity("api-trace", "GetAppInfo：后端收到请求", "debug")
	info = api.runtime.GetAppInfo()
	api.runtime.RecordLogWithSeverity("api-trace", "GetAppInfo：后端返回成功", "debug")
	return info, nil
}

// GetAppInfo 返回应用元数据（内部实现）。
func (s *Runtime) GetAppInfo() AppInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return AppInfo{
		Name:        s.options.AppName,
		Version:     s.options.Version,
		Description: s.options.Description,
		Repository:  s.options.Repository,
		StartedAt:   s.startedAt.Format(time.RFC3339),
	}
}

// GetEnvironmentInfo API 方法，返回进程运行环境快照。
// 前端调用: wails.App.GetEnvironmentInfo()
func (api *API) GetEnvironmentInfo() (info EnvironmentInfo, err error) {
	defer api.recoverError("读取运行环境", &err)
	api.runtime.RecordLogWithSeverity("api-trace", "GetEnvironmentInfo：后端收到请求", "debug")
	info = api.runtime.GetEnvironmentInfo()
	api.runtime.RecordLogWithSeverity("api-trace", "GetEnvironmentInfo：后端返回成功", "debug")
	return info, nil
}

// GetEnvironmentInfo 返回进程运行环境快照（内部实现）。
func (s *Runtime) GetEnvironmentInfo() EnvironmentInfo {
	s.lock.RLock()
	storeReady := s.store != nil && s.configDefaultsEnsured
	info := EnvironmentInfo{
		OS:              goruntime.GOOS,
		Arch:            goruntime.GOARCH,
		GoVersion:       goruntime.Version(),
		WailsVersion:    moduleVersion("github.com/wailsapp/wails/v3"),
		DatabasePath:    s.databasePath,
		DatabaseReady:   storeReady,
		DatabaseStatus:  databaseStatus(storeReady, s.databasePath),
		DatabaseMessage: databaseStatusMessage(storeReady, s.databasePath),
		CachePath:       s.cachePath,
	}
	s.lock.RUnlock()
	info.LogFilePath = s.currentLogFilePath()
	return info
}

// databaseStatus 把配置库就绪状态映射为紧凑 UI 状态码。
func databaseStatus(ready bool, path string) string {
	if path == "" {
		return "disabled"
	}
	if ready {
		return "ok"
	}
	return "error"
}

// databaseStatusMessage 返回 databaseStatus 的本地化说明。
func databaseStatusMessage(ready bool, path string) string {
	if path == "" {
		return "未配置 SQLite 数据库路径。"
	}
	if ready {
		return "SQLite 配置库已打开，默认配置已初始化。"
	}
	return "SQLite 配置库未打开或默认配置未完成。"
}
