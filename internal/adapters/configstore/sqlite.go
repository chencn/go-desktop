// ============================================================================
// 文件: internal/adapters/configstore/sqlite.go
// 描述: SQLite 数据库存储层
//
// 功能概述:
// - 管理 SQLite 数据库连接生命周期
// - 打开数据库时执行表结构迁移
// - 只提供 config_items 配置项持久化
// - 不保存日志、更新检查结果、更新事件或任何业务历史
// ============================================================================

package configstore

import (
	"context"       // 上下文包，用于数据库操作的超时控制
	"database/sql"  // 标准数据库接口
	"errors"        // 错误处理
	"os"            // 操作系统接口，用于目录创建
	"path/filepath" // 路径处理
	"strings"       // 字符串处理

	_ "modernc.org/sqlite" // SQLite 驱动
)

// Store 是 SQLite 数据库的封装
type Store struct {
	db *sql.DB // db 保存 db 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 数据库连接和迁移
// ============================================================================

// Open 打开或创建 SQLite 数据库
// 参数:
//   - path: 数据库文件路径
//
// 返回:
//   - *Store: 数据库存储实例
//   - error: 错误信息
func Open(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("SQLite 数据库路径为空")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

// Close 关闭数据库连接
// 返回:
//   - error: 错误信息
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
