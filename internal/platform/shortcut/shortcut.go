// Package shortcut 为运行时提供桌面快捷方式的平台抽象。
// 当前只有 Windows 实现，其他平台统一返回 ErrShortcutNotSupported。

package shortcut

import "errors"

// ErrShortcutNotSupported 表示当前平台暂不支持运行时管理桌面快捷方式。
var ErrShortcutNotSupported = errors.New("desktop shortcut is not supported on this platform")

// ShortcutOptions 定义桌面快捷方式创建参数。
// Name 不含扩展名；Arguments 会按平台规则转义后写入快捷方式。
type ShortcutOptions struct {
	Name        string   // 快捷方式名称，不含扩展名
	Description string   // 快捷方式描述
	Arguments   []string // 启动参数
}

// ShortcutStatus 描述当前用户桌面上目标快捷方式的存在状态。
// Path 总是尽量返回本应用管理的 .lnk 绝对路径，Exists 表示该文件当前是否存在。
type ShortcutStatus struct {
	Path   string // 快捷方式绝对路径
	Exists bool   // 是否已经存在
}
