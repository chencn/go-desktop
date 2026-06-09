// 文件职责：装配服务层持久化依赖并提供安全关闭入口。

package runtime

import (
	"fmt"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
)

// openStore 打开 SQLite 配置存储；路径为空或打开失败时保留 nil store，让读取走默认值、写入返回错误。
func (s *Runtime) openStore() {
	if s.databasePath == "" {
		return
	}
	store, err := configstore.Open(s.databasePath)
	if err != nil {
		s.store = nil
		s.RecordLogWithSeverity("storage", fmt.Sprintf("打开 SQLite 数据库失败：%s", err), "warning")
		return
	}
	s.store = store
}
