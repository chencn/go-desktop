//go:build windows

// ============================================================================
// 文件: internal/platform/shortcut/shortcut_windows.go
// 描述: Windows 桌面快捷方式实现
//
// 功能概述:
// - 使用当前用户 Desktop 已知目录定位快捷方式路径
// - 使用 Windows COM WScript.Shell.CreateShortcut 创建 .lnk
// - 只删除本应用同名快捷方式，避免扫描或清理用户其他文件
// ============================================================================

package shortcut

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

// comAlreadyInitializedHResult 表示当前线程已经初始化过 COM；这是成功状态，也需要配对 CoUninitialize。
const comAlreadyInitializedHResult uintptr = 0x00000001

// EnsureShortcut 在当前用户桌面创建或刷新应用快捷方式。
// 返回创建的 .lnk 绝对路径；调用方负责记录日志。
func EnsureShortcut(options ShortcutOptions) (string, error) {
	name := shortcutName(options.Name)
	path, err := shortcutPath(name)
	if err != nil {
		return "", err
	}
	executable, err := resolvedExecutable()
	if err != nil {
		return "", err
	}
	if err := writeShortcut(path, executable, options.Description, options.Arguments); err != nil {
		return "", err
	}
	return path, nil
}

// ShortcutExists 查询当前用户桌面上本应用快捷方式是否已经存在。
// 该函数只检查本应用管理的同名 .lnk，不读取或修改快捷方式内容。
func ShortcutExists(name string) (ShortcutStatus, error) {
	path, err := shortcutPath(shortcutName(name))
	if err != nil {
		return ShortcutStatus{}, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ShortcutStatus{Path: path, Exists: false}, nil
		}
		return ShortcutStatus{Path: path, Exists: false}, fmt.Errorf("stat desktop shortcut: %w", err)
	}
	return ShortcutStatus{Path: path, Exists: true}, nil
}

// RemoveShortcut 删除当前用户桌面上的本应用快捷方式。
func RemoveShortcut(name string) error {
	path, err := shortcutPath(shortcutName(name))
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove desktop shortcut: %w", err)
	}
	return nil
}

// shortcutName 规范化快捷方式文件名；空名称回退到 go-desktop，并强制补 .lnk。
func shortcutName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "go-desktop"
	}
	name = strings.TrimSuffix(name, ".lnk")
	return name + ".lnk"
}

// shortcutPath 返回当前用户 Desktop 已知目录下的快捷方式路径。
func shortcutPath(name string) (string, error) {
	desktopDir, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", fmt.Errorf("resolve desktop directory: %w", err)
	}
	return filepath.Join(desktopDir, name), nil
}

// resolvedExecutable 返回当前进程可执行文件路径，能解析符号链接时优先使用真实路径。
func resolvedExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil && strings.TrimSpace(resolved) != "" {
		return resolved, nil
	}
	return executable, nil
}

// writeShortcut 在独立协程里写 .lnk，内部会锁定 OS 线程以满足 COM STA 要求。
func writeShortcut(path string, executable string, description string, arguments []string) error {
	result := make(chan error, 1)
	go func() {
		result <- writeShortcutOnLockedThread(path, executable, description, arguments)
	}()
	return <-result
}

// writeShortcutOnLockedThread 在固定 OS 线程里执行完整 COM 生命周期，避免 Go 协程迁移导致 Wails 回调进程崩溃。
func writeShortcutOnLockedThread(path string, executable string, description string, arguments []string) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("write desktop shortcut COM: %v", recovered)
		}
	}()

	if err := initializeShortcutCOM(); err != nil {
		return err
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return fmt.Errorf("create WScript.Shell: %w", err)
	}
	defer unknown.Release()

	dispatch, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query WScript.Shell dispatch: %w", err)
	}
	defer dispatch.Release()

	shortcutVariant, err := oleutil.CallMethod(dispatch, "CreateShortcut", path)
	if err != nil {
		return fmt.Errorf("create shortcut object: %w", err)
	}
	defer shortcutVariant.Clear()

	shortcut := shortcutVariant.ToIDispatch()
	if shortcut == nil {
		return fmt.Errorf("create shortcut object: dispatch is nil")
	}

	if err := putShortcutProperty(shortcut, "TargetPath", executable); err != nil {
		return err
	}
	if err := putShortcutProperty(shortcut, "WorkingDirectory", filepath.Dir(executable)); err != nil {
		return err
	}
	if err := putShortcutProperty(shortcut, "IconLocation", executable+",0"); err != nil {
		return err
	}
	if argumentLine := shortcutArguments(arguments); argumentLine != "" {
		if err := putShortcutProperty(shortcut, "Arguments", argumentLine); err != nil {
			return err
		}
	}
	if strings.TrimSpace(description) != "" {
		if err := putShortcutProperty(shortcut, "Description", description); err != nil {
			return err
		}
	}
	if err := callShortcutMethod(shortcut, "Save"); err != nil {
		return err
	}
	return nil
}

// putShortcutProperty 设置 WScript shortcut 属性，并把 COM 错误附带属性名返回。
func putShortcutProperty(shortcut *ole.IDispatch, name string, value any) error {
	result, err := oleutil.PutProperty(shortcut, name, value)
	if err := clearShortcutResult(result, err); err != nil {
		return fmt.Errorf("set shortcut %s: %w", name, err)
	}
	return nil
}

// callShortcutMethod 调用 WScript shortcut 方法，并清理 COM 返回值。
func callShortcutMethod(shortcut *ole.IDispatch, name string, params ...any) error {
	result, err := oleutil.CallMethod(shortcut, name, params...)
	if err := clearShortcutResult(result, err); err != nil {
		return fmt.Errorf("call shortcut %s: %w", name, err)
	}
	return nil
}

// clearShortcutResult 优先返回业务调用错误；只有业务调用成功时才暴露 VARIANT 清理错误。
func clearShortcutResult(result *ole.VARIANT, err error) error {
	if result == nil {
		return err
	}
	if clearErr := result.Clear(); err == nil && clearErr != nil {
		return clearErr
	}
	return err
}

// shortcutArguments 跳过空参数并使用 Windows 命令行转义规则拼成 Arguments 字段。
func shortcutArguments(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		parts = append(parts, syscall.EscapeArg(arg))
	}
	return strings.Join(parts, " ")
}

// initializeShortcutCOM 用 STA 模型初始化当前线程；S_FALSE 代表已初始化成功，仍按成功处理。
func initializeShortcutCOM() error {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		var oleErr *ole.OleError
		if errors.As(err, &oleErr) && oleErr.Code() == comAlreadyInitializedHResult {
			return nil
		}
		return fmt.Errorf("initialize COM: %w", err)
	}
	return nil
}
