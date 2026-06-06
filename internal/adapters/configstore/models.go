// 文件职责：定义 SQLite 存储层对外使用的数据模型。

package configstore

// ConfigItem 是通用配置项结构。
// value/default_value 都以字符串保存，业务层负责按 value_type 做类型转换和枚举校验。
type ConfigItem struct {
	Key          string // 稳定配置键，例如 startup.auto_launch
	Category     string // 配置分组，例如 startup、display、update
	Title        string // 前端展示标题
	Description  string // 前端展示描述
	ValueType    string // 值类型：bool、int、string
	DefaultValue string // 字符串形式默认值
	Value        string // 字符串形式当前值
	SortOrder    int    // 同一分组内排序
	UpdatedAt    string // 更新时间（RFC3339 UTC）
}
