//go:build !windows

// ============================================================================
// 文件: internal/platform/shortcut/shortcut_other.go
// 描述: 非 Windows 桌面快捷方式占位实现
// ============================================================================

package shortcut

// EnsureShortcut 在非 Windows 平台返回不支持，调用方记录日志即可。
func EnsureShortcut(options ShortcutOptions) (string, error) {
	return "", ErrShortcutNotSupported
}

// ShortcutExists 在非 Windows 平台返回不支持，调用方记录日志即可。
func ShortcutExists(name string) (ShortcutStatus, error) {
	return ShortcutStatus{}, ErrShortcutNotSupported
}

// RemoveShortcut 在非 Windows 平台返回不支持，调用方记录日志即可。
func RemoveShortcut(name string) error {
	return ErrShortcutNotSupported
}
