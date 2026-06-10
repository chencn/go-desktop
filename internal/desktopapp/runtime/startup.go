// 文件职责：按设置同步 Wails3 开机自启和当前用户桌面快捷方式。

package runtime

import (
	"errors"
	"fmt"

	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
	"github.com/chencn/go-desktop/internal/platform/shortcut"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// startupArg 标记本次启动来自开机自启入口。
const startupArg = "--startup"

// startupHiddenArg 是自启入口使用的最终隐藏参数。
const startupHiddenArg = "--startup-hidden"

// desktopShortcutArg 标记本次启动来自应用创建的桌面快捷图标。
const desktopShortcutArg = "--desktop-shortcut"

// StartupAutostartArguments 根据设置生成开机自启参数。
// 开机自启开启时始终写入 --startup 标记启动来源；隐藏到托盘开启时再写入 --startup-hidden。
func StartupAutostartArguments(settings Settings) []string {
	if !settings.AutoLaunch {
		return nil
	}
	args := []string{startupArg}
	if settings.LaunchHiddenToTray {
		args = append(args, startupHiddenArg)
	}
	return args
}

// ApplyStartupIntegrations 按当前设置执行启动期系统集成检查。
// 启动期只补齐缺失的自启项和桌面快捷方式；如果系统里已经存在则跳过，避免每次打开应用都覆盖注册表或 .lnk。
func (s *Runtime) ApplyStartupIntegrations() {
	defer s.RecoverPanic("启动期系统集成")
	if err := s.applyStartupIntegrations(false); err != nil {
		s.RecordLogWithSeverity("startup", fmt.Sprintf("启动期系统集成同步失败：%s", err), "warning")
	}
}

// applyStartupIntegrations 同步系统级启动集成。
// force=false 用于应用启动检查，已存在项直接跳过；force=true 用于用户保存设置，确保参数和快捷方式内容被修正。
func (s *Runtime) applyStartupIntegrations(force bool) error {
	s.lock.RLock()
	settings := s.settings
	wailsApp := s.wailsApp
	s.lock.RUnlock()
	if wailsApp == nil {
		return nil
	}

	return errors.Join(
		s.applyAutostartIntegration(wailsApp, settings, force),
		s.applyDesktopShortcutIntegration(settings, force),
	)
}

// applyChangedStartupIntegrations 只同步受设置保存影响的启动集成，避免无关保存触发桌面快捷方式副作用。
func (s *Runtime) applyChangedStartupIntegrations(previous Settings, next Settings) error {
	if s.options.StartupIntegrationApplier != nil {
		return s.options.StartupIntegrationApplier(previous, next)
	}
	s.lock.RLock()
	wailsApp := s.wailsApp
	s.lock.RUnlock()
	if wailsApp == nil {
		return nil
	}
	var errs []error
	if autostartSettingsChanged(previous, next) {
		errs = append(errs, s.applyAutostartIntegration(wailsApp, next, true))
	}
	if desktopShortcutSettingsChanged(previous, next) {
		errs = append(errs, s.applyDesktopShortcutIntegration(next, true))
	}
	return errors.Join(errs...)
}

// autostartSettingsChanged 判断保存设置时是否需要重写开机自启项。
// AutoLaunch 保持开启时也重写参数，保证旧自启项迁移到当前启动参数合同。
func autostartSettingsChanged(previous Settings, next Settings) bool {
	return previous.AutoLaunch != next.AutoLaunch || next.AutoLaunch
}

// desktopShortcutSettingsChanged 判断保存设置时是否需要创建或删除桌面快捷图标。
func desktopShortcutSettingsChanged(previous Settings, next Settings) bool {
	return previous.CreateDesktopShortcut != next.CreateDesktopShortcut
}

// applyAutostartIntegration 同步 Wails Autostart。
// force=false 只修补缺失项；force=true 来自保存设置，会重写自启参数以反映 LaunchHiddenToTray。
func (s *Runtime) applyAutostartIntegration(wailsApp *application.App, settings Settings, force bool) error {
	if settings.AutoLaunch {
		if !force {
			status, err := wailsApp.Autostart.Status()
			if err != nil {
				s.recordAutostartError("检查开机自启状态失败", err)
				return err
			}
			if status.Enabled {
				s.RecordLogWithSeverity("startup", "启动检查：开机自启已存在，未修改", "debug")
				return nil
			}
		}
		err := wailsApp.Autostart.EnableWithOptions(application.AutostartOptions{
			Identifier: metadata.WindowsSingleInstanceID,
			Arguments:  StartupAutostartArguments(settings),
		})
		if err != nil {
			s.recordAutostartError("启用开机自启失败", err)
			return err
		}
		if force {
			s.RecordLog("startup", "保存设置：开机自启设置已同步（当前为开启）")
		} else {
			s.RecordLog("startup", "启动检查：开机自启缺失，已补齐")
		}
		return nil
	}
	if !force {
		status, err := wailsApp.Autostart.Status()
		if err != nil {
			s.recordAutostartError("检查开机自启状态失败", err)
			return err
		}
		if !status.Enabled {
			s.RecordLogWithSeverity("startup", "启动检查：配置为不开机自启，系统自启项未启用", "debug")
			return nil
		}
	}
	if err := wailsApp.Autostart.Disable(); err != nil {
		s.recordAutostartError("关闭开机自启失败", err)
		return err
	}
	if force {
		s.RecordLog("startup", "保存设置：开机自启设置已同步（当前为关闭）")
	} else {
		s.RecordLog("startup", "启动检查：配置为不开机自启，已同步系统自启状态")
	}
	return nil
}

// applyDesktopShortcutIntegration 同步当前用户桌面快捷方式。
// 快捷方式会携带 --desktop-shortcut，便于启动日志区分入口来源。
func (s *Runtime) applyDesktopShortcutIntegration(settings Settings, force bool) error {
	if settings.CreateDesktopShortcut {
		if !force {
			status, err := shortcut.ShortcutExists(metadata.AppName)
			if err != nil {
				s.recordShortcutError("检查桌面快捷图标失败", err)
				return err
			}
			if status.Exists {
				s.RecordLogWithSeverity("shortcut", "启动检查：桌面快捷方式已存在，未修改", "debug")
				return nil
			}
		}
		s.recordCrashBreadcrumb("shortcut", "开始创建桌面快捷图标")
		path, err := shortcut.EnsureShortcut(shortcut.ShortcutOptions{
			Name:        metadata.AppName,
			Description: metadata.Description,
			Arguments:   []string{desktopShortcutArg},
		})
		if err != nil {
			s.recordShortcutError("创建桌面快捷图标失败", err)
			return err
		}
		s.recordCrashBreadcrumb("shortcut", "创建桌面快捷图标成功：%s", path)
		if force {
			s.RecordLog("shortcut", "保存设置：桌面快捷方式设置已同步（当前为创建）")
		} else {
			s.RecordLog("shortcut", "启动检查：桌面快捷方式缺失，已创建")
		}
		return nil
	}
	if !force {
		status, err := shortcut.ShortcutExists(metadata.AppName)
		if err != nil {
			s.recordShortcutError("检查桌面快捷图标失败", err)
			return err
		}
		if !status.Exists {
			s.RecordLogWithSeverity("shortcut", "启动检查：配置为不创建桌面快捷方式，快捷方式不存在", "debug")
			return nil
		}
	}
	s.recordCrashBreadcrumb("shortcut", "开始删除桌面快捷图标")
	if err := shortcut.RemoveShortcut(metadata.AppName); err != nil {
		s.recordShortcutError("删除桌面快捷图标失败", err)
		return err
	}
	s.recordCrashBreadcrumb("shortcut", "删除桌面快捷图标成功")
	if force {
		s.RecordLog("shortcut", "保存设置：桌面快捷方式设置已同步（当前为关闭）")
	} else {
		s.RecordLog("shortcut", "启动检查：配置为不创建桌面快捷方式，已同步桌面状态")
	}
	return nil
}

// recordAutostartError 记录自启平台错误；不支持自启的平台降级为 info。
func (s *Runtime) recordAutostartError(prefix string, err error) {
	severity := "warning"
	if errors.Is(err, application.ErrAutostartNotSupported) {
		severity = "info"
	}
	s.RecordLogWithSeverity("startup", fmt.Sprintf("%s：%s", prefix, err), severity)
}

// recordShortcutError 记录桌面快捷方式平台错误；不支持快捷方式的平台降级为 info。
func (s *Runtime) recordShortcutError(prefix string, err error) {
	severity := "warning"
	if errors.Is(err, shortcut.ErrShortcutNotSupported) {
		severity = "info"
	}
	s.RecordLogWithSeverity("shortcut", fmt.Sprintf("%s：%s", prefix, err), severity)
}
