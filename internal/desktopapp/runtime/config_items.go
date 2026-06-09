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
	"context"
	"fmt"
	"time"

	"github.com/chencn/go-desktop/internal/adapters/configstore"
	"github.com/chencn/go-desktop/internal/desktopapp/display"
	appsettings "github.com/chencn/go-desktop/internal/desktopapp/settings"
)

// configSaveTimeout 是无 deadline 调用写入 SQLite 配置项时的兜底超时。
const configSaveTimeout = 5 * time.Second

// configItemMap 是按配置 key 建索引后的 SQLite 配置项集合。
type configItemMap map[string]configstore.ConfigItem

// allConfigDefinitions 合并所有配置定义，供 Runtime 启动时一次性确保默认值。
func allConfigDefinitions() []configstore.ConfigItem {
	definitions := appsettings.Definitions()
	definitions = append(definitions, display.Definitions()...)
	definitions = append(definitions, licenseDefinitions()...)
	return definitions
}

// licenseDefinitions 声明授权状态配置项；这里仅提供元数据和值槽位，授权校验逻辑在 license 模块。
func licenseDefinitions() []configstore.ConfigItem {
	return []configstore.ConfigItem{
		{Key: licenseKeyConfig, Category: "license", Title: "授权码", Description: "当前设备保存的授权码", ValueType: "string", SortOrder: 900},
		{Key: licenseDeviceCodeConfig, Category: "license", Title: "授权设备码", Description: "最近一次授权成功时使用的设备码", ValueType: "string", SortOrder: 901},
		{Key: licenseValidatedAtConfig, Category: "license", Title: "授权校验时间", Description: "最近一次授权码校验通过的 UTC 时间", ValueType: "string", SortOrder: 902},
		{Key: licenseLastErrorConfig, Category: "license", Title: "授权错误", Description: "最近一次授权校验失败原因", ValueType: "string", SortOrder: 903},
	}
}

// ensureConfigDefaults 把缺失配置项写入 SQLite，并刷新已有配置项的展示元数据。
// 已存在 value 不会被默认值覆盖；成功后 Runtime 生命周期内只执行一次。
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

// configItemsByKey 确保默认项后读取全部配置项并按 key 建索引；存储未打开时返回空集合。
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
// 存储不可用属于写入失败，需要向 API 调用方返回错误，不能静默降级。
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
