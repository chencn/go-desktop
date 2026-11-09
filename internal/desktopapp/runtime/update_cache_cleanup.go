// 文件职责：启动时清理已经安装或过期的更新缓存。

package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chencn/go-desktop/internal/common/semver"
)

// cleanupInstalledUpdateCache 删除版本小于等于当前应用版本的更新状态文件和版本目录。
func (s *Runtime) cleanupInstalledUpdateCache() {
	currentVersion := strings.TrimSpace(s.options.Version)
	if _, ok := semver.Parse(currentVersion); !ok {
		return
	}

	s.lock.RLock()
	cachePath := strings.TrimSpace(s.cachePath)
	s.lock.RUnlock()
	if cachePath == "" {
		return
	}

	entries, err := os.ReadDir(cachePath)
	if err != nil {
		if !os.IsNotExist(err) {
			s.RecordLogWithSeverity("update", fmt.Sprintf("读取更新缓存目录失败：%s", err), "warning")
		}
		return
	}

	for _, fileName := range []string{pendingUpdateFileName, verifiedUpdateFileName} {
		s.cleanupInstalledUpdateStateFile(fileName, currentVersion)
	}
	for _, entry := range entries {
		if !entry.IsDir() || !shouldCleanupUpdateCacheVersion(entry.Name(), currentVersion) {
			continue
		}
		path := filepath.Join(cachePath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			s.RecordLogWithSeverity("update", fmt.Sprintf("清理旧版本更新缓存目录失败：%s", err), "warning")
		}
	}
}

// cleanupInstalledUpdateStateFile 删除指向小于等于当前版本安装包的更新状态文件。
func (s *Runtime) cleanupInstalledUpdateStateFile(fileName string, currentVersion string) {
	path := s.updateCacheStatePath(fileName)
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			s.RecordLogWithSeverity("update", fmt.Sprintf("读取更新状态文件失败：%s", err), "warning")
		}
		return
	}

	var state pendingUpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	version := updateStateVersion(state)
	if !shouldCleanupUpdateCacheVersion(version, currentVersion) {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		s.RecordLogWithSeverity("update", fmt.Sprintf("清理旧版本更新状态文件失败：%s", err), "warning")
	}
}

// updateStateVersion 从状态文件取目标版本；Version 缺失或非法时回退到安装包路径的版本目录名。
func updateStateVersion(state pendingUpdateState) string {
	version := strings.TrimSpace(state.Version)
	if _, ok := semver.Parse(version); ok {
		return version
	}
	filePath := strings.TrimSpace(state.FilePath)
	if filePath == "" {
		return ""
	}
	return filepath.Base(filepath.Dir(filePath))
}

// shouldCleanupUpdateCacheVersion 判断版本是否小于等于当前版本（已安装或已过期）；任一侧无法解析时不清理。
func shouldCleanupUpdateCacheVersion(version string, currentVersion string) bool {
	if _, ok := semver.Parse(version); !ok {
		return false
	}
	if _, ok := semver.Parse(currentVersion); !ok {
		return false
	}
	return semver.Compare(version, currentVersion) <= 0
}
