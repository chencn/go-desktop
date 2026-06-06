// 文件职责：验证 SQLite 只承担配置项持久化，不保存日志、更新检查或更新事件业务数据。
// 说明：本文件位于独立 tests 模块，避免生产目录散落 Go 测试。

package configstore_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
)

// TestStoreCreatesOnlyConfigItemsTable 验证 SQLite schema 只创建 config_items 和对应索引。
func TestStoreCreatesOnlyConfigItemsTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	store, err := configstore.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}
	if len(tables) != 1 || tables[0] != "config_items" {
		t.Fatalf("SQLite should only create config_items, got %#v", tables)
	}
}

// TestStoreEnsuresConfigItemsWithoutOverwritingUserValues 验证默认配置刷新不会覆盖用户值。
func TestStoreEnsuresConfigItemsWithoutOverwritingUserValues(t *testing.T) {
	store, err := configstore.Open(filepath.Join(t.TempDir(), "go-desktop.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	defaults := []configstore.ConfigItem{
		{
			Key:          "startup.auto_launch",
			Category:     "startup",
			Title:        "开机自启",
			Description:  "登录 Windows 后自动启动应用。",
			ValueType:    "bool",
			DefaultValue: "false",
			Value:        "false",
			SortOrder:    10,
		},
		{
			Key:          "display.theme_mode",
			Category:     "display",
			Title:        "主题模式",
			Description:  "控制亮色或暗色主题。",
			ValueType:    "string",
			DefaultValue: "light",
			Value:        "light",
			SortOrder:    110,
		},
	}
	if err := store.EnsureConfigItems(context.Background(), defaults); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	if err := store.UpsertConfigValue(context.Background(), "startup.auto_launch", "true"); err != nil {
		t.Fatalf("update config value: %v", err)
	}
	defaults[0].Title = "开机自动启动"
	defaults[0].DefaultValue = "false"
	defaults[0].Value = "false"
	if err := store.EnsureConfigItems(context.Background(), defaults); err != nil {
		t.Fatalf("refresh defaults: %v", err)
	}

	items, err := store.ListConfigItems(context.Background())
	if err != nil {
		t.Fatalf("list config items: %v", err)
	}
	byKey := map[string]configstore.ConfigItem{}
	for _, item := range items {
		byKey[item.Key] = item
	}
	if byKey["startup.auto_launch"].Value != "true" {
		t.Fatalf("expected user value to survive default refresh, got %#v", byKey["startup.auto_launch"])
	}
	if byKey["startup.auto_launch"].Title != "开机自动启动" {
		t.Fatalf("expected metadata refresh, got %#v", byKey["startup.auto_launch"])
	}
	if byKey["display.theme_mode"].Value != "light" || byKey["display.theme_mode"].DefaultValue != "light" {
		t.Fatalf("expected display preference default to persist, got %#v", byKey["display.theme_mode"])
	}
}

// TestStoreUpsertsConfigValuesInBatch 验证配置保存可以在一个批次里写入多个 key。
func TestStoreUpsertsConfigValuesInBatch(t *testing.T) {
	store, err := configstore.Open(filepath.Join(t.TempDir(), "go-desktop.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.UpsertConfigValues(context.Background(), map[string]string{
		"display.theme_mode":  "dark",
		"startup.auto_launch": "true",
	}); err != nil {
		t.Fatalf("upsert config values: %v", err)
	}

	items, err := store.ListConfigItems(context.Background())
	if err != nil {
		t.Fatalf("list config items: %v", err)
	}
	byKey := map[string]configstore.ConfigItem{}
	for _, item := range items {
		byKey[item.Key] = item
	}
	if byKey["display.theme_mode"].Value != "dark" || byKey["startup.auto_launch"].Value != "true" {
		t.Fatalf("expected batched config values to persist, got %#v", byKey)
	}
}
