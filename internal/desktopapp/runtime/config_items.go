// ============================================================================
// 文件: config_items.go
// 描述: 应用配置 KV 定义和读写辅助
//
// 功能概述:
// - 统一定义业务设置、启动设置、显示偏好的 SQLite KV 元数据
// - 在 Runtime 启动时写入缺失默认值
// - 提供 typed facade 到字符串 KV 的转换辅助
// ============================================================================

package runtime

import (
	"context" // 上下文包，用于 SQLite 操作
	"fmt"     // 格式化错误信息
	"time"    // 配置保存超时控制

	"github.com/chencn/go-desktop/internal/adapters/configstore" // SQLite 配置项结构
	"github.com/chencn/go-desktop/internal/desktopapp/display"
	appsettings "github.com/chencn/go-desktop/internal/desktopapp/settings"
)

const configSaveTimeout = 5 * time.Second

// configItemMap 是按配置 key 建索引后的 SQLite 配置项集合。
type configItemMap map[string]configstore.ConfigItem

// allConfigDefinitions 合并所有配置定义，供 Runtime 启动时一次性确保默认值。
func allConfigDefinitions() []configstore.ConfigItem {
	definitions := appsettings.Definitions()
	definitions = append(definitions, display.Definitions()...)
	return definitions
}

// ensureConfigDefaults 把缺失配置项写入 SQLite，并刷新已有配置项的展示元数据。
func (s *Runtime) ensureConfigDefaults(ctx context.Context) error {
	s.lock.RLock()
	store := s.store
	ensured := s.configDefaultsEnsured
	s.lock.RUnlock()
	if store == nil || ensured {
		return nil
	}
	items := allConfigDefinitions()
	if err := store.EnsureConfigItems(ctx, items); err != nil {
		return err
	}
	s.lock.Lock()
	if s.store == store {
		s.configDefaultsEnsured = true
	}
	s.lock.Unlock()
	return nil
}

// configItemsByKey 读取全部配置项并按 key 建索引。
func (s *Runtime) configItemsByKey(ctx context.Context) (configItemMap, error) {
	if err := s.ensureConfigDefaults(ctx); err != nil {
		return nil, err
	}
	s.lock.RLock()
	store := s.store
	s.lock.RUnlock()
	if store == nil {
		return configItemMap{}, nil
	}
	items, err := store.ListConfigItems(ctx)
	if err != nil {
		return nil, err
	}
	byKey := make(configItemMap, len(items))
	for _, item := range items {
		byKey[item.Key] = item
	}
	return byKey, nil
}

// saveConfigValues 批量保存配置值；调用方负责先做 typed 标准化。
func (s *Runtime) saveConfigValues(ctx context.Context, values map[string]string) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, configSaveTimeout)
		defer cancel()
	}
	if err := s.ensureConfigDefaults(ctx); err != nil {
		return err
	}
	s.lock.RLock()
	store := s.store
	s.lock.RUnlock()
	if store == nil {
		return fmt.Errorf("配置存储不可用")
	}
	s.RecordLogWithSeverity("settings", fmt.Sprintf("保存配置项：准备写入 %d 项", len(values)), "debug")
	if err := store.UpsertConfigValues(ctx, values); err != nil {
		return fmt.Errorf("保存配置失败：%w", err)
	}
	return nil
}
