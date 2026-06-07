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
	pendingUpdateFileName  = "pending.json"
	verifiedUpdateFileName = "verified.json"
)

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

func (s *Runtime) pendingUpdatePath() string {
	return s.updateCacheStatePath(pendingUpdateFileName)
}

func (s *Runtime) verifiedUpdatePath() string {
	return s.updateCacheStatePath(verifiedUpdateFileName)
}

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

func (s *Runtime) validatePendingUpdateFile(status UpdateStatus) error {
	return s.validateUpdateCacheFile(status, "待安装包")
}

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

func (s *Runtime) clearPendingUpdate() {
	path := s.pendingUpdatePath()
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		s.RecordLogWithSeverity("update", fmt.Sprintf("清理待安装更新状态失败：%s", err), "warning")
	}
}

func (s *Runtime) clearVerifiedUpdate() {
	path := s.verifiedUpdatePath()
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		s.RecordLogWithSeverity("update", fmt.Sprintf("清理已校验更新状态失败：%s", err), "warning")
	}
}
