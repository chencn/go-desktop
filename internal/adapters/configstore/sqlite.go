// Package configstore 提供 config_items 配置表的 SQLite 存储层。
// 它只承载通用配置 KV 的持久化和表结构迁移，不保存日志、更新历史或业务事件；
// 读取路径在 store 为 nil 时可降级（返回默认值），写入路径由上层决定是否报错。
package configstore

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // SQLite 驱动
)

// Store 是 config_items 表的 SQLite 封装；当前只承载配置项，不保存日志或业务历史。
type Store struct {
	db *sql.DB // db 是底层连接池，限制为单连接以匹配 SQLite 写入模型。
}

// ============================================================================
// 数据库连接和迁移
// ============================================================================

// Open 打开或创建 SQLite 数据库，并在返回前完成 config_items 迁移。
// path 为空会返回错误；目录不存在时会自动创建。
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

// Close 关闭数据库连接；nil Store 可安全调用。
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
