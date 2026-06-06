// 文件职责：装配服务层持久化依赖并提供安全关闭入口。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package runtime

import (
	"fmt"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
)

// openStore 读取、解析或归一化 装配服务层持久化依赖并提供安全关闭入口 需要的数据，并把结果返回给调用方。
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
