package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const pendingUpdateFileName = "pending.json"

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
	UpdatedAt       string  `json:"updatedAt"`
}

func (s *Runtime) pendingUpdatePath() string {
	if s == nil {
		return ""
	}
	s.lock.RLock()
	cachePath := strings.TrimSpace(s.cachePath)
	s.lock.RUnlock()
	if cachePath == "" {
		return ""
	}
	return filepath.Join(cachePath, pendingUpdateFileName)
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

func (s *Runtime) validatePendingUpdateFile(status UpdateStatus) error {
	s.lock.RLock()
	cachePath := strings.TrimSpace(s.cachePath)
	s.lock.RUnlock()
	if cachePath == "" {
		return errors.New("更新缓存目录未配置")
	}
	filePath := strings.TrimSpace(status.FilePath)
	if filePath == "" {
		return errors.New("待安装更新状态不完整")
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
		return fmt.Errorf("待安装包不在更新缓存目录内：%s", filePath)
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
