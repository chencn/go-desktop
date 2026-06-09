package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	updater "github.com/chencn/go-desktop/internal/desktopapp/update"
)

const (
	// pending.json 只表示用户已选择“下次启动安装”，由启动期消费。
	pendingUpdateFileName = "pending.json"
	// verified.json 表示安装包已下载且 SHA256 校验通过，可在重启后恢复到“可安装”状态。
	verifiedUpdateFileName = "verified.json"
)

// pendingUpdateState 是 pending.json 和 verified.json 共用的磁盘格式。
// 两者语义由文件名和 Status 区分；安装包路径必须指向更新缓存目录内的文件。
type pendingUpdateState struct {
	Status          string  `json:"status"`
	Message         string  `json:"message"`
	Version         string  `json:"version,omitempty"`
	AssetName       string  `json:"assetName,omitempty"`
	FilePath        string  `json:"filePath"`
	DownloadedBytes int64   `json:"downloadedBytes,omitempty"`
	TotalBytes      int64   `json:"totalBytes,omitempty"`
	ProgressPercent float64 `json:"progressPercent,omitempty"`
	Sha256          string  `json:"sha256"`
	Verified        bool    `json:"verified"`
	Source          string  `json:"source,omitempty"`
	UpdatedAt       string  `json:"updatedAt"`
}

// pendingUpdatePath 返回下次启动安装状态文件路径。
func (s *Runtime) pendingUpdatePath() string {
	return s.updateCacheStatePath(pendingUpdateFileName)
}

// verifiedUpdatePath 返回已校验安装包状态文件路径。
func (s *Runtime) verifiedUpdatePath() string {
	return s.updateCacheStatePath(verifiedUpdateFileName)
}

// updateCacheStatePath 在 Runtime 缓存目录下定位更新状态文件；缓存目录缺失时返回空路径。
func (s *Runtime) updateCacheStatePath(fileName string) string {
	if s == nil {
		return ""
	}
	s.lock.RLock()
	cachePath := strings.TrimSpace(s.cachePath)
	s.lock.RUnlock()
	if cachePath == "" {
		return ""
	}
	return filepath.Join(cachePath, fileName)
}

// savePendingUpdate 写入 pending.json，并在落盘前确认安装包仍位于缓存目录内。
func (s *Runtime) savePendingUpdate(status UpdateStatus) error {
	path := s.pendingUpdatePath()
	if path == "" {
		return errors.New("更新缓存目录未配置")
	}
	if !status.Verified || strings.TrimSpace(status.FilePath) == "" || strings.TrimSpace(status.Sha256) == "" {
		return errors.New("待安装更新状态不完整")
	}
	if err := s.validatePendingUpdateFile(status); err != nil {
		return err
	}
	pending := pendingUpdateState{
		Status:          "pending_install",
		Message:         status.Message,
		Version:         status.Version,
		AssetName:       status.AssetName,
		FilePath:        strings.TrimSpace(status.FilePath),
		DownloadedBytes: status.DownloadedBytes,
		TotalBytes:      status.TotalBytes,
		ProgressPercent: status.ProgressPercent,
		Sha256:          strings.ToLower(strings.TrimSpace(status.Sha256)),
		Verified:        true,
		Source:          strings.TrimSpace(status.Source),
		UpdatedAt:       status.UpdatedAt,
	}
	data, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// saveVerifiedUpdate 写入 verified.json，并在落盘前复验安装包 SHA256。
func (s *Runtime) saveVerifiedUpdate(status UpdateStatus) error {
	path := s.verifiedUpdatePath()
	if path == "" {
		return errors.New("更新缓存目录未配置")
	}
	if !status.Verified || strings.TrimSpace(status.FilePath) == "" || strings.TrimSpace(status.Sha256) == "" {
		return errors.New("已校验更新状态不完整")
	}
	if err := s.validateVerifiedUpdateFile(status); err != nil {
		return err
	}
	verified := pendingUpdateState{
		Status:          "verified",
		Message:         status.Message,
		Version:         status.Version,
		AssetName:       status.AssetName,
		FilePath:        strings.TrimSpace(status.FilePath),
		DownloadedBytes: status.DownloadedBytes,
		TotalBytes:      status.TotalBytes,
		ProgressPercent: status.ProgressPercent,
		Sha256:          strings.ToLower(strings.TrimSpace(status.Sha256)),
		Verified:        true,
		Source:          strings.TrimSpace(status.Source),
		UpdatedAt:       status.UpdatedAt,
	}
	data, err := json.MarshalIndent(verified, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// loadPendingUpdate 读取 pending.json；返回 found=true 的错误表示文件存在但内容不可用，调用方应清理。
func (s *Runtime) loadPendingUpdate() (UpdateStatus, bool, error) {
	path := s.pendingUpdatePath()
	if path == "" {
		return UpdateStatus{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return UpdateStatus{}, false, nil
		}
		return UpdateStatus{}, false, err
	}
	var pending pendingUpdateState
	if err := json.Unmarshal(data, &pending); err != nil {
		return UpdateStatus{}, true, err
	}
	status := UpdateStatus{
		Status:          "pending_install",
		Message:         strings.TrimSpace(pending.Message),
		Version:         pending.Version,
		AssetName:       pending.AssetName,
		FilePath:        strings.TrimSpace(pending.FilePath),
		DownloadedBytes: pending.DownloadedBytes,
		TotalBytes:      pending.TotalBytes,
		ProgressPercent: pending.ProgressPercent,
		Sha256:          strings.ToLower(strings.TrimSpace(pending.Sha256)),
		Verified:        pending.Verified,
		Source:          strings.TrimSpace(pending.Source),
		UpdatedAt:       pending.UpdatedAt,
	}
	if status.Message == "" {
		status.Message = "安装包已校验，将在本次启动时自动更新。"
	}
	if status.UpdatedAt == "" {
		status.UpdatedAt = nowRFC3339()
	}
	if !status.Verified || strings.TrimSpace(status.FilePath) == "" || strings.TrimSpace(status.Sha256) == "" {
		return UpdateStatus{}, true, fmt.Errorf("待安装更新状态不完整")
	}
	if err := s.validatePendingUpdateFile(status); err != nil {
		return UpdateStatus{}, true, err
	}
	return status, true, nil
}

// loadVerifiedUpdate 读取 verified.json；恢复前会复验安装包路径和 SHA256。
func (s *Runtime) loadVerifiedUpdate() (UpdateStatus, bool, error) {
	path := s.verifiedUpdatePath()
	if path == "" {
		return UpdateStatus{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return UpdateStatus{}, false, nil
		}
		return UpdateStatus{}, false, err
	}
	var verified pendingUpdateState
	if err := json.Unmarshal(data, &verified); err != nil {
		return UpdateStatus{}, true, err
	}
	status := UpdateStatus{
		Status:          "verified",
		Message:         strings.TrimSpace(verified.Message),
		Version:         verified.Version,
		AssetName:       verified.AssetName,
		FilePath:        strings.TrimSpace(verified.FilePath),
		DownloadedBytes: verified.DownloadedBytes,
		TotalBytes:      verified.TotalBytes,
		ProgressPercent: verified.ProgressPercent,
		Sha256:          strings.ToLower(strings.TrimSpace(verified.Sha256)),
		Verified:        verified.Verified,
		Source:          strings.TrimSpace(verified.Source),
		UpdatedAt:       verified.UpdatedAt,
	}
	if status.Message == "" {
		status.Message = "安装包已下载并通过 SHA256 校验，等待用户选择安装时机。"
	}
	if status.UpdatedAt == "" {
		status.UpdatedAt = nowRFC3339()
	}
	if !status.Verified || strings.TrimSpace(status.FilePath) == "" || strings.TrimSpace(status.Sha256) == "" {
		return UpdateStatus{}, true, fmt.Errorf("已校验更新状态不完整")
	}
	if err := s.validateVerifiedUpdateFile(status); err != nil {
		return UpdateStatus{}, true, err
	}
	return status, true, nil
}

// validatePendingUpdateFile 只检查路径边界；pending.json 消费时还会走 InstallDownloadedUpdate 复验 SHA256。
func (s *Runtime) validatePendingUpdateFile(status UpdateStatus) error {
	return s.validateUpdateCacheFile(status, "待安装包")
}

// validateVerifiedUpdateFile 检查路径边界并立即复算 SHA256，保证 verified.json 可作为“已校验”状态恢复。
func (s *Runtime) validateVerifiedUpdateFile(status UpdateStatus) error {
	if err := s.validateUpdateCacheFile(status, "已校验安装包"); err != nil {
		return err
	}
	actual, err := updater.FileSHA256(status.FilePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, status.Sha256) {
		return fmt.Errorf("已校验安装包 SHA256 不匹配")
	}
	return nil
}

// validateUpdateCacheFile 拒绝缓存目录外的安装包路径，防止 pending/verified 文件被篡改后安装任意文件。
func (s *Runtime) validateUpdateCacheFile(status UpdateStatus, label string) error {
	s.lock.RLock()
	cachePath := strings.TrimSpace(s.cachePath)
	s.lock.RUnlock()
	if cachePath == "" {
		return errors.New("更新缓存目录未配置")
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = "安装包"
	}
	filePath := strings.TrimSpace(status.FilePath)
	if filePath == "" {
		return fmt.Errorf("%s路径为空", label)
	}
	fileAbs, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	cacheAbs, err := filepath.Abs(cachePath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(cacheAbs, fileAbs)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%s不在更新缓存目录内：%s", label, filePath)
	}
	return nil
}

// clearPendingUpdate 删除 pending.json；删除失败只记录警告，不覆盖主流程状态。
func (s *Runtime) clearPendingUpdate() {
	path := s.pendingUpdatePath()
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		s.RecordLogWithSeverity("update", fmt.Sprintf("清理待安装更新状态失败：%s", err), "warning")
	}
}

// clearVerifiedUpdate 删除 verified.json；删除失败只记录警告，不覆盖主流程状态。
func (s *Runtime) clearVerifiedUpdate() {
	path := s.verifiedUpdatePath()
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		s.RecordLogWithSeverity("update", fmt.Sprintf("清理已校验更新状态失败：%s", err), "warning")
	}
}
