// 文件职责：下载安装包、校验 SHA256，并只允许从更新缓存目录启动静默安装器。

package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/common/checksum"
	"github.com/chencn/go-desktop/internal/common/neterr"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
	"github.com/chencn/go-desktop/internal/platform/installer"
)

// Config 是更新管理器的配置。
type Config struct {
	// CacheDir 是安装包缓存目录；下载、校验、安装路径边界都以它为准。
	CacheDir string
	// Client 是下载 HTTP 客户端；为空时使用带超时的默认客户端。
	Client *http.Client
	// Runner 启动静默安装器；为空时使用平台默认实现。
	Runner InstallerRunner
	// Now 生成完成时间；为空时使用 UTC 当前时间。
	Now func() time.Time
}

// Manager 持有更新下载和安装所需的依赖。
type Manager struct {
	cacheDir string
	client   *http.Client
	runner   InstallerRunner
	now      func() time.Time
}

// ReleaseAsset 是检查器选中的安装资产。
type ReleaseAsset struct {
	LatestVersion    string
	TagName          string
	AssetName        string
	AssetSizeBytes   int64
	AssetDownloadURL string
	Sha256           string
}

// DownloadResult 描述已下载且通过 SHA256 校验的安装包。
type DownloadResult struct {
	Version     string `json:"version"`
	AssetName   string `json:"assetName"`
	FilePath    string `json:"filePath"`
	SizeBytes   int64  `json:"sizeBytes"`
	Sha256      string `json:"sha256"`
	Verified    bool   `json:"verified"`
	CompletedAt string `json:"completedAt"`
}

// Progress 是下载或校验阶段的进度快照。
type Progress struct {
	Stage           string
	DownloadedBytes int64
	TotalBytes      int64
}

// ProgressFunc 在下载写入和校验开始时回调；调用方不得在回调里阻塞太久。
type ProgressFunc func(Progress)

// InstallerRunner 是平台安装器启动函数。
type InstallerRunner = installer.Runner

// maxUpdateDownloadBytes 是未知 Content-Length 时仍允许写入的硬上限。
const maxUpdateDownloadBytes int64 = 2 * 1024 * 1024 * 1024

// ChecksumMismatchError 表示下载完成后实际 SHA256 与期望值不一致。
type ChecksumMismatchError struct {
	Expected string
	Actual   string
}

// Error 返回用于日志和错误分支识别的 SHA256 不匹配说明。
func (e ChecksumMismatchError) Error() string {
	return fmt.Sprintf("SHA256 校验失败，期望 %s，实际 %s", e.Expected, e.Actual)
}

// NewManager 创建更新管理器实例，并补齐可选依赖的默认实现。
func NewManager(config Config) *Manager {
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	runner := config.Runner
	if runner == nil {
		runner = installer.RunSilent
	}
	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Manager{
		cacheDir: strings.TrimSpace(config.CacheDir),
		client:   client,
		runner:   runner,
		now:      now,
	}
}

// DownloadAndVerify 下载指定安装资产到版本子目录，并在重命名前校验 SHA256。
// 临时文件固定为目标路径加 ".download"，调用方需要自行串行化同一缓存目录的并发下载。
func (m *Manager) DownloadAndVerify(ctx context.Context, asset ReleaseAsset, progress ProgressFunc) (DownloadResult, error) {
	if m == nil {
		return DownloadResult{}, errors.New("更新管理器未初始化")
	}
	asset.AssetDownloadURL = strings.TrimSpace(asset.AssetDownloadURL)
	asset.AssetName = filepath.Base(strings.TrimSpace(asset.AssetName))
	asset.Sha256 = strings.ToLower(strings.TrimSpace(asset.Sha256))
	if asset.AssetDownloadURL == "" {
		return DownloadResult{}, errors.New("缺少安装包下载地址")
	}
	if asset.AssetName == "" || asset.AssetName == "." || asset.AssetName == string(filepath.Separator) {
		return DownloadResult{}, errors.New("缺少安装包文件名")
	}
	if asset.Sha256 == "" {
		return DownloadResult{}, errors.New("缺少 SHA256 校验信息，禁止下载")
	}
	if len(asset.Sha256) != 64 {
		return DownloadResult{}, fmt.Errorf("SHA256 格式无效：%s", asset.Sha256)
	}
	downloadLimit, err := downloadSizeLimit(asset.AssetSizeBytes)
	if err != nil {
		return DownloadResult{}, err
	}

	// 版本号参与目录名，必须先清理为安全路径片段。
	version := asset.LatestVersion
	if version == "" {
		version = strings.TrimPrefix(strings.TrimPrefix(asset.TagName, "v"), "V")
	}
	version = safePathPart(version)
	if version == "" {
		version = "unknown"
	}

	// 先写入临时文件，校验通过后再替换目标文件，避免暴露半成品安装包。
	targetDir := filepath.Join(m.cacheDir, version)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return DownloadResult{}, err
	}
	targetPath := filepath.Join(targetDir, asset.AssetName)
	tempPath := targetPath + ".download"
	_ = os.Remove(tempPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.AssetDownloadURL, nil)
	if err != nil {
		return DownloadResult{}, err
	}
	req.Header.Set("User-Agent", metadata.UserAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		return DownloadResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DownloadResult{}, fmt.Errorf("下载安装包返回 HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > downloadLimit {
		return DownloadResult{}, fmt.Errorf("安装包大小超过允许上限：%d > %d", resp.ContentLength, downloadLimit)
	}

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return DownloadResult{}, err
	}
	written, copyErr := copyWithProgress(file, resp.Body, resp.ContentLength, downloadLimit, progress)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return DownloadResult{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return DownloadResult{}, closeErr
	}

	if progress != nil {
		progress(Progress{Stage: "verifying", DownloadedBytes: written, TotalBytes: resp.ContentLength})
	}
	actual, err := checksum.FileSHA256(tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return DownloadResult{}, err
	}
	if !strings.EqualFold(actual, asset.Sha256) {
		// 只清理本次下载的临时文件；targetPath 可能是上一轮已验证、verified.json 仍指向的完好安装包。
		_ = os.Remove(tempPath)
		return DownloadResult{}, ChecksumMismatchError{Expected: asset.Sha256, Actual: actual}
	}
	_ = os.Remove(targetPath)
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return DownloadResult{}, err
	}

	return DownloadResult{
		Version:     version,
		AssetName:   asset.AssetName,
		FilePath:    targetPath,
		SizeBytes:   written,
		Sha256:      strings.ToLower(actual),
		Verified:    true,
		CompletedAt: m.now().UTC().Format(time.RFC3339),
	}, nil
}

// Install 启动已校验安装包。
// 这里不重新计算 SHA256；调用方必须在进入前完成校验，Manager 只负责路径边界和进程启动。
func (m *Manager) Install(ctx context.Context, installerPath string) error {
	if m == nil {
		return errors.New("更新管理器未初始化")
	}
	installerPath = strings.TrimSpace(installerPath)
	if installerPath == "" {
		return errors.New("缺少已校验安装包路径")
	}
	info, err := os.Stat(installerPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("安装包路径不是文件：%s", installerPath)
	}
	if err := m.requireUpdateCacheFile(installerPath); err != nil {
		return err
	}
	return m.runner(ctx, installerPath)
}

// requireUpdateCacheFile 验证安装包是否位于更新缓存目录内，防止安装任意本地文件。
func (m *Manager) requireUpdateCacheFile(installerPath string) error {
	if strings.TrimSpace(m.cacheDir) == "" {
		return errors.New("更新缓存目录未配置，拒绝安装")
	}
	installerAbs, err := filepath.Abs(installerPath)
	if err != nil {
		return err
	}
	updatesAbs, err := filepath.Abs(m.cacheDir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(updatesAbs, installerAbs)
	if err != nil {
		return err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return fmt.Errorf("安装包不在更新缓存目录内：%s", installerPath)
	}
	return nil
}

// FileSHA256 计算文件的 SHA256 十六进制摘要。
func FileSHA256(path string) (string, error) {
	return checksum.FileSHA256(path)
}

// IsOfflineError 判断错误是否属于可跳过的离线网络错误。
func IsOfflineError(err error) bool {
	return neterr.IsOfflineError(err)
}

// downloadSizeLimit 计算本次下载允许写入的最大字节数。
// Release 已声明大小时以声明值为上限；未知大小时使用全局硬上限。
func downloadSizeLimit(assetSizeBytes int64) (int64, error) {
	if assetSizeBytes < 0 {
		return 0, fmt.Errorf("安装包大小无效：%d", assetSizeBytes)
	}
	if assetSizeBytes > maxUpdateDownloadBytes {
		return 0, fmt.Errorf("安装包大小超过允许上限：%d > %d", assetSizeBytes, maxUpdateDownloadBytes)
	}
	if assetSizeBytes > 0 {
		return assetSizeBytes, nil
	}
	return maxUpdateDownloadBytes, nil
}

// copyWithProgress 复制响应体并持续上报下载进度，同时强制执行最大写入字节数。
func copyWithProgress(dst io.Writer, src io.Reader, total int64, maxBytes int64, progress ProgressFunc) (int64, error) {
	buffer := make([]byte, 64*1024)
	var written int64
	for {
		n, readErr := src.Read(buffer)
		if n > 0 {
			if maxBytes > 0 && written+int64(n) > maxBytes {
				remaining := maxBytes - written
				if remaining > 0 {
					w, writeErr := dst.Write(buffer[:int(remaining)])
					written += int64(w)
					if progress != nil {
						progress(Progress{Stage: "downloading", DownloadedBytes: written, TotalBytes: total})
					}
					if writeErr != nil {
						return written, writeErr
					}
					if int64(w) != remaining {
						return written, io.ErrShortWrite
					}
				}
				return written, fmt.Errorf("下载安装包超过允许大小：%d 字节", maxBytes)
			}
			w, writeErr := dst.Write(buffer[:n])
			written += int64(w)
			if progress != nil {
				progress(Progress{Stage: "downloading", DownloadedBytes: written, TotalBytes: total})
			}
			if writeErr != nil {
				return written, writeErr
			}
			if w != n {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return written, nil
			}
			return written, readErr
		}
	}
}

// safePathPart 清理版本目录名，避免把 Release tag 中的路径分隔符写入文件系统。
func safePathPart(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, ". ")
	return value
}
