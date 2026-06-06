// 文件职责：维护 SQLite 表结构迁移。

package configstore

import "context"

// migrate 执行数据库表结构迁移。
func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS config_items (
			key TEXT PRIMARY KEY,
			category TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			value_type TEXT NOT NULL,
			default_value TEXT NOT NULL,
			value TEXT NOT NULL,
			sort_order INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_config_items_category_sort ON config_items(category, sort_order, key)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
