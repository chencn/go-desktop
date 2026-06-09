package runtime

import (
	"fmt"
	"runtime/debug"
)

// recoverError 把 Wails API 方法内的 panic 转成 error，避免进入 Wails 默认 fatal 退出流程。
func (api *API) recoverError(operation string, errp *error) {
	recovered := recover()
	if recovered == nil {
		return
	}
	err := recoveredError(operation, recovered)
	if api != nil && api.runtime != nil {
		api.runtime.recordRecoveredPanic(operation, recovered)
	}
	if errp != nil {
		*errp = err
	}
}

// RecoverPanic 用于 Wails 回调、托盘菜单和启动期任务，把 panic 记入日志并阻断该回调继续向外抛。
func (s *Runtime) RecoverPanic(operation string) {
	if recovered := recover(); recovered != nil {
		s.recordRecoveredPanic(operation, recovered)
	}
}

// recordRecoveredPanic 尽力写入 panic 日志；日志链路本身异常时会被吞掉，避免二次 panic。
func (s *Runtime) recordRecoveredPanic(operation string, recovered any) {
	if s == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	s.RecordLogWithSeverity("panic", fmt.Sprintf("%s panic：%v\n%s", operation, recovered, string(debug.Stack())), "error")
}

// recoveredError 把 panic 值转换为带操作名的 error，供 Wails API 返回给前端。
func recoveredError(operation string, recovered any) error {
	if err, ok := recovered.(error); ok {
		return fmt.Errorf("%s发生异常：%w", operation, err)
	}
	return fmt.Errorf("%s发生异常：%v", operation, recovered)
}
