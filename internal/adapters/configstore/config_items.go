// 文件职责：提供通用配置项元数据和值的 SQLite 操作。

package configstore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EnsureConfigItems 确保默认配置项存在。
// 已存在的配置项只刷新标题、描述、类型、默认值和排序，不覆盖用户当前 value。
func (s *Store) EnsureConfigItems(ctx context.Context, defaults []ConfigItem) error {
	if s == nil || s.db == nil {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range defaults {
		item = normaliseConfigItem(item)
		if item.Key == "" {
			continue
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO config_items (
				key, category, title, description, value_type,
				default_value, value, sort_order, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET
				category = excluded.category,
				title = excluded.title,
				description = excluded.description,
				value_type = excluded.value_type,
				default_value = excluded.default_value,
				sort_order = excluded.sort_order,
				updated_at = excluded.updated_at
		`, item.Key, item.Category, item.Title, item.Description, item.ValueType,
			item.DefaultValue, item.Value, item.SortOrder, item.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListConfigItems 返回全部配置项。
func (s *Store) ListConfigItems(ctx context.Context) ([]ConfigItem, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT key, category, title, description, value_type,
			default_value, value, sort_order, updated_at
		FROM config_items
		ORDER BY category ASC, sort_order ASC, key ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ConfigItem, 0)
	for rows.Next() {
		var item ConfigItem
		if err := rows.Scan(&item.Key, &item.Category, &item.Title, &item.Description, &item.ValueType,
			&item.DefaultValue, &item.Value, &item.SortOrder, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpsertConfigValue 更新单个配置项当前值。
func (s *Store) UpsertConfigValue(ctx context.Context, key string, value string) error {
	return s.UpsertConfigValues(ctx, map[string]string{key: value})
}

// UpsertConfigValues 批量更新配置项当前值。
func (s *Store) UpsertConfigValues(ctx context.Context, values map[string]string) error {
	if s == nil || s.db == nil {
		return nil
	}
	if len(values) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statement, err := tx.PrepareContext(ctx, `
		INSERT INTO config_items (
			key, category, title, description, value_type,
			default_value, value, sort_order, updated_at
		)
		VALUES (?, 'custom', ?, '', 'string', '', ?, 0, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return err
	}
	defer statement.Close()

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	now := time.Now().UTC().Format(time.RFC3339)
	for _, rawKey := range keys {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return errors.New("配置项 key 为空")
		}
		value := strings.TrimSpace(values[rawKey])
		if _, err := statement.ExecContext(ctx, key, key, value, now); err != nil {
			return fmt.Errorf("保存配置 %s 失败：%w", key, err)
		}
	}
	return tx.Commit()
}

func normaliseConfigItem(item ConfigItem) ConfigItem {
	item.Key = strings.TrimSpace(item.Key)
	item.Category = strings.TrimSpace(item.Category)
	item.Title = strings.TrimSpace(item.Title)
	item.Description = strings.TrimSpace(item.Description)
	item.ValueType = normaliseConfigValueType(item.ValueType)
	item.DefaultValue = strings.TrimSpace(item.DefaultValue)
	item.Value = strings.TrimSpace(item.Value)
	item.UpdatedAt = defaultTime(item.UpdatedAt)
	if item.Category == "" {
		item.Category = "app"
	}
	if item.Title == "" {
		item.Title = item.Key
	}
	if item.Value == "" {
		item.Value = item.DefaultValue
	}
	return item
}

func normaliseConfigValueType(valueType string) string {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "bool", "int", "string":
		return strings.ToLower(strings.TrimSpace(valueType))
	default:
		return "string"
	}
}
