// 文件职责：提供更新检查、下载、校验、安排安装和启动安装能力。
// 说明：更新链路只维护当前进程状态，不保存历史或事件数据。

package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	"github.com/chencn/go-desktop/internal/common/semver"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
	updater "github.com/chencn/go-desktop/internal/desktopapp/update"
)

// CheckUpdate API 方法，检查 GitHub Release 更新。
func (api *API) CheckUpdate() (result githubrelease.CheckResult, err error) {
	defer api.recoverError("检查更新", &err)
	if err := api.requireAuthorized(); err != nil {
		return githubrelease.CheckResult{}, err
	}
	return api.runtime.CheckUpdate(), nil
}

// CheckUpdate 检查更新并记录当前进程最近一次检查结果。
func (s *Runtime) CheckUpdate() githubrelease.CheckResult {
	settings := s.SettingsSnapshot()
	checker := s.updateChecker(settings)

	result := checker.Check(context.Background())
	s.RecordUpdateCheckResult(result)
	s.RecordLog("update", result.Message)
	if result.Status == githubrelease.StatusUpdateAvailable && strings.TrimSpace(result.Sha256) != "" {
		s.RecordLog("update", "发现可更新版本，开始自动下载并校验")
		s.downloadUpdateForCheck(result, true)
	}
	return result
}

func releaseAssetNames(version string) []string {
	return []string{
		metadata.WindowsInstallerAssetName(version),
		metadata.WindowsInstallerAssetNameWithoutV(version),
		metadata.WindowsSetupAssetName(version),
		metadata.WindowsSetupAssetNameWithoutV(version),
	}
}

func (s *Runtime) updateChecker(settings Settings) *githubrelease.Checker {
	if settings.UpdateSource == "local" {
		return githubrelease.NewChecker(githubrelease.Config{
			ManifestURL:    localUpdateManifestURL(s.options.LocalUpdateBaseURL, s.options.LocalUpdateManifestPath),
			Source:         "local",
			CurrentVersion: s.options.Version,
			UserAgent:      metadata.UserAgent,
			APIVersion:     metadata.GitHubAPIVersion,
			AssetNames:     releaseAssetNames,
		})
	}
	checker := s.releaseChecker
	if checker != nil && settings.GitHubOwner == metadata.GitHubOwner && settings.GitHubRepo == metadata.GitHubRepo && settings.GitHubProxyBase == "" {
		return checker
	}
	return githubrelease.NewChecker(githubrelease.Config{
		Owner:          settings.GitHubOwner,
		Repo:           settings.GitHubRepo,
		ProxyBase:      settings.GitHubProxyBase,
		Source:         "github",
		CurrentVersion: s.options.Version,
		UserAgent:      metadata.UserAgent,
		APIVersion:     metadata.GitHubAPIVersion,
		AssetNames:     releaseAssetNames,
	})
}

func localUpdateManifestURL(baseURL, manifestPath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	manifestPath = strings.TrimLeft(strings.TrimSpace(manifestPath), "/")
	if baseURL == "" {
		return manifestPath
	}
	if manifestPath == "" {
		return baseURL
	}
	return baseURL + "/" + manifestPath
}

// RecordUpdateCheckResult 保存当前进程最近一次更新检查结果。
func (s *Runtime) RecordUpdateCheckResult(result githubrelease.CheckResult) {
	s.lock.Lock()
	s.latestUpdateCheck = result
	s.hasUpdateCheck = true
	s.lock.Unlock()
	s.setUpdateStatus(statusFromCheckResult(result))
}

// GetUpdateStatus API 方法，返回当前更新状态。
func (api *API) GetUpdateStatus() (status UpdateStatus, err error) {
	defer api.recoverError("读取更新状态", &err)
	if err := api.requireAuthorized(); err != nil {
		return UpdateStatus{}, err
	}
	api.runtime.RecordLogWithSeverity("api-trace", "GetUpdateStatus：后端收到请求", "debug")
	status = api.runtime.GetUpdateStatus()
	api.runtime.RecordLogWithSeverity("api-trace", fmt.Sprintf("GetUpdateStatus：后端返回成功 status=%q", status.Status), "debug")
	return status, nil
}

// GetUpdateStatus 返回当前更新状态。
func (s *Runtime) GetUpdateStatus() UpdateStatus {
	s.lock.RLock()
	state := s.updateState
	s.lock.RUnlock()
	if shouldReturnMemoryUpdateStatus(state) {
		return state
	}
	verified, found, err := s.loadVerifiedUpdate()
	if err != nil {
		if found {
			s.clearVerifiedUpdate()
		}
		s.RecordLogWithSeverity("update", fmt.Sprintf("读取已校验更新状态失败：%s", err), "warning")
		return state
	}
	if !found {
		return state
	}
	settings := s.SettingsSnapshot()
	if verified.Source != "" && settings.UpdateSource != "" && verified.Source != settings.UpdateSource {
		return state
	}
	if semver.Compare(verified.Version, s.options.Version) <= 0 {
		s.clearVerifiedUpdate()
		return state
	}
	s.setUpdateStatus(verified)
	return verified
}

// DownloadUpdate API 方法，下载并校验最近一次检查发现的更新包。
func (api *API) DownloadUpdate() (status UpdateStatus, err error) {
	defer api.recoverError("下载更新", &err)
	if err := api.requireAuthorized(); err != nil {
		return UpdateStatus{}, err
	}
	return api.runtime.DownloadUpdate(), nil
}

// DownloadUpdate 下载并校验更新包。
func (s *Runtime) DownloadUpdate() UpdateStatus {
	check, ok := s.latestCheckResult()
	return s.downloadUpdateForCheck(check, ok)
}

func (s *Runtime) downloadUpdateForCheck(check githubrelease.CheckResult, ok bool) UpdateStatus {
	s.updateOperationLock.Lock()
	defer s.updateOperationLock.Unlock()

	if !ok {
		return s.failUpdate("missing_update_check", "请先检查更新。")
	}
	if check.Status == githubrelease.StatusIgnored {
		status := statusFromCheckResult(check)
		s.setUpdateStatus(status)
		return status
	}
	if check.Status != githubrelease.StatusUpdateAvailable {
		status := statusFromCheckResult(check)
		s.setUpdateStatus(status)
		return status
	}
	if strings.TrimSpace(check.Sha256) == "" {
		return s.failUpdateFromStatus(statusFromCheckResult(check), "sha256_missing", "发现新版本，但缺少 SHA256 校验信息，已禁止下载和安装。")
	}
	if strings.TrimSpace(check.AssetDownloadURL) == "" || strings.TrimSpace(check.AssetName) == "" {
		return s.failUpdateFromStatus(statusFromCheckResult(check), "asset_missing", "发现新版本，但安装资产信息不完整。")
	}

	s.setUpdateStatus(UpdateStatus{
		Status:    "downloading",
		Message:   "正在下载安装包...",
		Version:   check.LatestVersion,
		AssetName: check.AssetName,
		Sha256:    check.Sha256,
		Source:    check.Source,
		UpdatedAt: nowRFC3339(),
	})
	s.RecordLog("update", fmt.Sprintf("开始下载更新：%s", check.AssetName))

	result, err := s.updateManager.DownloadAndVerify(context.Background(), updater.ReleaseAsset{
		LatestVersion:    check.LatestVersion,
		TagName:          check.TagName,
		AssetName:        check.AssetName,
		AssetSizeBytes:   check.AssetSizeBytes,
		AssetDownloadURL: check.AssetDownloadURL,
		Sha256:           check.Sha256,
	}, func(progress updater.Progress) {
		status := "downloading"
		message := "正在下载安装包..."
		if progress.Stage == "verifying" {
			status = "verifying"
			message = "正在校验 SHA256..."
		}
		s.setUpdateStatus(UpdateStatus{
			Status:          status,
			Message:         message,
			Version:         check.LatestVersion,
			AssetName:       check.AssetName,
			DownloadedBytes: progress.DownloadedBytes,
			TotalBytes:      progress.TotalBytes,
			ProgressPercent: progressPercent(progress.DownloadedBytes, progress.TotalBytes),
			Sha256:          check.Sha256,
			Source:          check.Source,
			UpdatedAt:       nowRFC3339(),
		})
	})
	if err != nil {
		var mismatch updater.ChecksumMismatchError
		if errors.As(err, &mismatch) {
			return s.failUpdateFromStatus(statusFromCheckResult(check), "sha256_mismatch", "安装包 SHA256 校验失败，已删除安装包，禁止安装。")
		}
		if updater.IsOfflineError(err) {
			status := UpdateStatus{
				Status:      "skipped",
				Message:     "当前无网络，已跳过更新下载。",
				Version:     check.LatestVersion,
				AssetName:   check.AssetName,
				Sha256:      check.Sha256,
				Source:      check.Source,
				ErrorReason: githubrelease.SkipReasonOffline,
				UpdatedAt:   nowRFC3339(),
			}
			s.setUpdateStatus(status)
			s.RecordLogWithSeverity("update", status.Message, "warning")
			return status
		}
		return s.failUpdateFromStatus(statusFromCheckResult(check), "download_failed", localisedErrorMessage("下载安装包", err))
	}

	verified := UpdateStatus{
		Status:          "verified",
		Message:         "安装包已下载并通过 SHA256 校验，等待用户选择安装时机。",
		Version:         result.Version,
		AssetName:       result.AssetName,
		FilePath:        result.FilePath,
		DownloadedBytes: result.SizeBytes,
		TotalBytes:      result.SizeBytes,
		ProgressPercent: 100,
		Sha256:          result.Sha256,
		Verified:        true,
		Source:          check.Source,
		UpdatedAt:       result.CompletedAt,
	}
	s.setUpdateStatus(verified)
	if err := s.saveVerifiedUpdate(verified); err != nil {
		s.RecordLogWithSeverity("update", fmt.Sprintf("保存已校验更新状态失败：%s", err), "warning")
	}
	s.RecordLog("update", "安装包已通过 SHA256 校验，等待用户选择安装时机")
	return verified
}

// ScheduleDownloadedUpdateOnStartup API 方法，把已校验更新包标记为下次启动安装。
func (api *API) ScheduleDownloadedUpdateOnStartup() (status UpdateStatus, err error) {
	defer api.recoverError("安排下次启动安装更新", &err)
	if err := api.requireAuthorized(); err != nil {
		return UpdateStatus{}, err
	}
	return api.runtime.ScheduleDownloadedUpdateOnStartup(), nil
}

// ScheduleDownloadedUpdateOnStartup 把当前已校验更新包标记为下次启动安装。
func (s *Runtime) ScheduleDownloadedUpdateOnStartup() UpdateStatus {
	s.updateOperationLock.Lock()
	defer s.updateOperationLock.Unlock()

	state := s.GetUpdateStatus()
	if !state.Verified || strings.TrimSpace(state.FilePath) == "" {
		return s.failUpdate("not_verified", "没有可安排下次启动安装的已校验更新包。")
	}
	pending := state
	pending.Status = "pending_install"
	pending.Message = "安装包已校验，将在下次启动时自动更新。"
	pending.UpdatedAt = nowRFC3339()
	if err := s.savePendingUpdate(pending); err != nil {
		return s.failUpdateFromStatus(state, "pending_save_failed", localisedErrorMessage("保存待安装更新状态", err))
	}
	s.setUpdateStatus(pending)
	s.RecordLog("update", "用户选择下次启动时更新")
	return pending
}

// InstallDownloadedUpdate API 方法，安装当前已校验更新包。
func (api *API) InstallDownloadedUpdate() (status UpdateStatus, err error) {
	defer api.recoverError("安装已下载更新", &err)
	if err := api.requireAuthorized(); err != nil {
		return UpdateStatus{}, err
	}
	return api.runtime.InstallDownloadedUpdate(), nil
}

// InstallDownloadedUpdate 启动静默安装器。
func (s *Runtime) InstallDownloadedUpdate() UpdateStatus {
	s.updateOperationLock.Lock()
	defer s.updateOperationLock.Unlock()

	state := s.GetUpdateStatus()
	if !state.Verified || strings.TrimSpace(state.FilePath) == "" {
		return s.failUpdate("not_verified", "没有可安装的已校验更新包。")
	}
	actual, err := updater.FileSHA256(state.FilePath)
	if err != nil {
		s.clearPendingUpdate()
		s.clearVerifiedUpdate()
		return s.failUpdateFromStatus(state, "verified_file_missing", localisedErrorMessage("读取已校验安装包", err))
	}
	if !strings.EqualFold(actual, state.Sha256) {
		_ = os.Remove(state.FilePath)
		s.clearPendingUpdate()
		s.clearVerifiedUpdate()
		return s.failUpdateFromStatus(state, "sha256_mismatch", "已校验安装包的 SHA256 发生变化，已删除该文件。")
	}

	installing := state
	installing.Status = "installing"
	installing.Message = "正在启动静默安装器..."
	installing.UpdatedAt = nowRFC3339()
	s.setUpdateStatus(installing)
	s.RecordLog("update", "正在启动静默安装器")

	if err := s.updateManager.Install(context.Background(), state.FilePath); err != nil {
		s.clearPendingUpdate()
		s.clearVerifiedUpdate()
		return s.failUpdateFromStatus(state, "install_failed", localisedErrorMessage("启动静默安装器", err))
	}
	s.clearPendingUpdate()
	s.clearVerifiedUpdate()

	started := installing
	started.Status = "install_started"
	started.Message = "静默安装器已启动，当前应用即将退出。"
	started.UpdatedAt = nowRFC3339()
	s.setUpdateStatus(started)
	s.RecordLog("update", "静默安装器已启动")
	s.QuitApp()
	return started
}

// InstallPendingUpdateOnStartup 在启动期安装当前进程已标记的待安装更新。
func (s *Runtime) InstallPendingUpdateOnStartup() (status UpdateStatus) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.recordRecoveredPanic("启动期自动安装更新", recovered)
			message := recoveredError("启动期自动安装更新", recovered).Error()
			if s == nil {
				status = UpdateStatus{Status: "error", Message: message, ErrorReason: "panic", UpdatedAt: nowRFC3339()}
				return
			}
			status = s.failUpdate("panic", message)
		}
	}()
	state := s.GetUpdateStatus()
	if state.Status == "pending_install" && state.Verified && strings.TrimSpace(state.FilePath) != "" {
		return s.InstallDownloadedUpdate()
	}
	pending, found, err := s.loadPendingUpdate()
	if err != nil {
		if found {
			s.clearPendingUpdate()
		}
		return s.failUpdate("pending_load_failed", localisedErrorMessage("读取待安装更新状态", err))
	}
	if !found {
		return state
	}
	s.setUpdateStatus(pending)
	return s.InstallDownloadedUpdate()
}

func idleUpdateStatus() UpdateStatus {
	return UpdateStatus{
		Status:    "idle",
		Message:   "尚未检查更新。",
		UpdatedAt: nowRFC3339(),
	}
}

func shouldReturnMemoryUpdateStatus(status UpdateStatus) bool {
	switch status.Status {
	case "downloading", "verifying", "verified", "pending_install", "installing", "install_started":
		return true
	default:
		return false
	}
}

func (s *Runtime) setUpdateStatus(status UpdateStatus) {
	if status.UpdatedAt == "" {
		status.UpdatedAt = nowRFC3339()
	}
	s.lock.Lock()
	s.updateState = status
	window := s.mainWindow
	s.lock.Unlock()
	if window != nil {
		func() {
			defer s.RecoverPanic("发送更新状态事件")
			window.EmitEvent("update:status:changed", status)
		}()
	}
}

func (s *Runtime) failUpdate(reason, message string) UpdateStatus {
	return s.failUpdateFromStatus(UpdateStatus{}, reason, message)
}

func (s *Runtime) failUpdateFromStatus(status UpdateStatus, reason, message string) UpdateStatus {
	status.Status = "error"
	status.Message = message
	status.ErrorReason = reason
	status.UpdatedAt = nowRFC3339()
	s.setUpdateStatus(status)
	s.RecordLogWithSeverity("update", message, "error")
	return status
}

func (s *Runtime) latestCheckResult() (githubrelease.CheckResult, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if !s.hasUpdateCheck {
		return githubrelease.CheckResult{}, false
	}
	return s.latestUpdateCheck, true
}

func statusFromCheckResult(result githubrelease.CheckResult) UpdateStatus {
	status := UpdateStatus{
		Status:      "idle",
		Message:     result.Message,
		Version:     result.LatestVersion,
		AssetName:   result.AssetName,
		Sha256:      result.Sha256,
		Source:      result.Source,
		ErrorReason: result.ErrorReason,
		UpdatedAt:   result.CheckedAt,
	}
	if status.UpdatedAt == "" {
		status.UpdatedAt = nowRFC3339()
	}
	switch result.Status {
	case githubrelease.StatusUpdateAvailable:
		status.Status = "update_available"
		if result.Sha256 == "" {
			status.Status = "error"
			status.ErrorReason = "sha256_missing"
		}
	case githubrelease.StatusNoUpdate:
		status.Status = "no_update"
	case githubrelease.StatusIgnored:
		status.Status = "skipped"
		if result.SkipReason != "" {
			status.ErrorReason = result.SkipReason
		}
	case githubrelease.StatusError:
		status.Status = "error"
	default:
		status.Status = "idle"
	}
	return status
}

func progressPercent(downloaded, total int64) float64 {
	if total <= 0 || downloaded <= 0 {
		return 0
	}
	value := float64(downloaded) / float64(total) * 100
	if value > 100 {
		return 100
	}
	return value
}

func localisedErrorMessage(action string, err error) string {
	action = strings.TrimSpace(action)
	if action == "" {
		action = "操作"
	}
	if err == nil {
		return action + "失败。"
	}
	if updater.IsOfflineError(err) {
		return action + "失败：当前无网络或连接不稳定。"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "createprocess"):
		return action + "失败：无法启动安装器进程。"
	case strings.Contains(message, "permission denied"), strings.Contains(message, "access is denied"):
		return action + "失败：权限不足。"
	case strings.Contains(message, "no such file"), strings.Contains(message, "cannot find the file"), strings.Contains(message, "file does not exist"):
		return action + "失败：目标文件不存在。"
	case strings.Contains(message, "unexpected eof"), strings.Contains(message, "short write"):
		return action + "失败：连接中断或文件写入不完整。"
	default:
		return action + "失败，请稍后重试。"
	}
}
