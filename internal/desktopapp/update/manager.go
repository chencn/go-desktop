// ============================================================================
// 文件: internal/desktopapp/update/manager.go
// 描述: 更新管理器
//
// 功能概述:
// - 下载安装包并校验 SHA256
// - 启动静默安装器
// - 支持下载进度回调
// - 检测网络离线状态
// ============================================================================

package update

import (
	"context"       // 上下文包
	"errors"        // 错误处理
	"fmt"           // 格式化字符串
	"io"            // 输入输出
	"net/http"      // HTTP 客户端
	"os"            // 操作系统接口
	"path/filepath" // 路径处理
	"strings"       // 字符串处理
	"time"          // 时间包

	"github.com/chencn/go-desktop/internal/common/checksum"
	"github.com/chencn/go-desktop/internal/common/neterr"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
	"github.com/chencn/go-desktop/internal/platform/installer"
)

// ============================================================================
// 数据结构定义
// ============================================================================

// Config 是更新管理器的配置
type Config struct {
	CacheDir string           // 安装包缓存目录
	Client   *http.Client     // HTTP 客户端（可选）
	Runner   InstallerRunner  // 安装器运行函数（可选）
	Now      func() time.Time // 当前时间函数（可选）
}

// Manager 是更新管理器
type Manager struct {
	cacheDir string           // cacheDir 保存 cacheDir 对应的数据，供当前实体的调用方读取或持久化。
	client   *http.Client     // client 保存 client 对应的数据，供当前实体的调用方读取或持久化。
	runner   InstallerRunner  // runner 保存 runner 对应的数据，供当前实体的调用方读取或持久化。
	now      func() time.Time // now 保存 now 对应的数据，供当前实体的调用方读取或持久化。
}

// ReleaseAsset 是发布资产信息
type ReleaseAsset struct {
	LatestVersion    string // 最新版本号
	TagName          string // Git 标签名
	AssetName        string // 安装包文件名
	AssetSizeBytes   int64  // 安装包大小（字节）
	AssetDownloadURL string // 下载链接
	Sha256           string // SHA256 哈希值
}

// DownloadResult 是下载结果
type DownloadResult struct {
	Version     string `json:"version"`     // 版本号
	AssetName   string `json:"assetName"`   // 安装包名称
	FilePath    string `json:"filePath"`    // 本地文件路径
	SizeBytes   int64  `json:"sizeBytes"`   // 文件大小
	Sha256      string `json:"sha256"`      // SHA256 哈希值
	Verified    bool   `json:"verified"`    // 是否通过校验
	CompletedAt string `json:"completedAt"` // 完成时间
}

// Progress 是下载进度
type Progress struct {
	Stage           string // 当前阶段
	DownloadedBytes int64  // 已下载字节数
	TotalBytes      int64  // 总字节数
}

// ProgressFunc 是进度回调函数类型
type ProgressFunc func(Progress)

// InstallerRunner 是安装器运行函数类型
type InstallerRunner = installer.Runner

const maxUpdateDownloadBytes int64 = 2 * 1024 * 1024 * 1024

// ChecksumMismatchError 是校验和不匹配错误
type ChecksumMismatchError struct {
	Expected string // 期望的 SHA256
	Actual   string // 实际的 SHA256
}

// Error 封装 管理更新包下载、校验、待安装状态和安装器启动 中的一段独立逻辑，调用方通过它复用同一业务规则。
func (e ChecksumMismatchError) Error() string {
	return fmt.Sprintf("SHA256 校验失败，期望 %s，实际 %s", e.Expected, e.Actual)
}

// ============================================================================
// 构造函数
// ============================================================================

// NewManager 创建更新管理器实例
// 参数:
//   - config: 配置
//
// 返回:
//   - *Manager: 更新管理器
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

// ============================================================================
// 下载和校验
// ============================================================================

// DownloadAndVerify 下载安装包并校验 SHA256
// 参数:
//   - ctx: 上下文
//   - asset: 发布资产信息
//   - progress: 进度回调函数（可选）
//
// 返回:
//   - DownloadResult: 下载结果
//   - error: 错误信息
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

	// 解析版本号
	version := asset.LatestVersion
	if version == "" {
		version = strings.TrimPrefix(strings.TrimPrefix(asset.TagName, "v"), "V")
	}
	version = safePathPart(version)
	if version == "" {
		version = "unknown"
	}

	// 创建缓存目录
	targetDir := filepath.Join(m.cacheDir, version)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return DownloadResult{}, err
	}
	targetPath := filepath.Join(targetDir, asset.AssetName)
	tempPath := targetPath + ".download"
	_ = os.Remove(tempPath)

	// 发送 HTTP 请求
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

	// 写入临时文件
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

	// 校验 SHA256
	if progress != nil {
		progress(Progress{Stage: "verifying", DownloadedBytes: written, TotalBytes: resp.ContentLength})
	}
	actual, err := checksum.FileSHA256(tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return DownloadResult{}, err
	}
	if !strings.EqualFold(actual, asset.Sha256) {
		_ = os.Remove(tempPath)
		_ = os.Remove(targetPath)
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

// ============================================================================
// 安装
// ============================================================================

// Install 启动安装器
// 参数:
//   - ctx: 上下文
//   - installerPath: 安装包路径
//
// 返回:
//   - error: 错误信息
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

// requireUpdateCacheFile 验证安装包是否在更新缓存目录内
// 防止安装非缓存目录下的恶意文件
// 参数:
//   - installerPath: 安装包路径
//
// 返回:
//   - error: 错误信息
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

// ============================================================================
// 工具函数
// ============================================================================

// FileSHA256 计算文件的 SHA256 哈希值
// 参数:
//   - path: 文件路径
//
// 返回:
//   - string: SHA256 哈希值（十六进制字符串）
//   - error: 错误信息
func FileSHA256(path string) (string, error) {
	return checksum.FileSHA256(path)
}

// IsOfflineError 判断错误是否为网络离线错误
// 参数:
//   - err: 错误
//
// 返回:
//   - bool: 是否为离线错误
func IsOfflineError(err error) bool {
	return neterr.IsOfflineError(err)
}

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

// copyWithProgress 带进度回调的文件复制
// 参数:
//   - dst: 目标写入器
//   - src: 源读取器
//   - total: 总字节数（可能为 -1 表示未知）
//   - maxBytes: 最大允许写入字节数
//   - progress: 进度回调函数
//
// 返回:
//   - int64: 实际写入的字节数
//   - error: 错误信息
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

// safePathPart 清理路径部分（移除非法字符）
// 参数:
//   - value: 原始值
//
// 返回:
//   - string: 清理后的值
func safePathPart(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, ". ")
	return value
}
