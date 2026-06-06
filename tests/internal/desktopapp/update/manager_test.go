// 文件职责：管理更新包下载、校验、待安装状态和安装器启动。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package update_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	updater "github.com/chencn/go-desktop/internal/desktopapp/update"
)

// TestDownloadAndVerifySuccess 验证 管理更新包下载、校验、待安装状态和安装器启动 的关键行为，避免后续重构破坏既有约束。
func TestDownloadAndVerifySuccess(t *testing.T) {
	payload := []byte("installer")

	manager := updater.NewManager(updater.Config{
		CacheDir: t.TempDir(),
		Client:   httpClient(http.StatusOK, payload),
	})
	var sawDownload bool
	var sawVerify bool
	result, err := manager.DownloadAndVerify(context.Background(), updater.ReleaseAsset{
		LatestVersion:    "1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Hex(payload),
	}, func(progress updater.Progress) {
		if progress.Stage == "downloading" {
			sawDownload = true
		}
		if progress.Stage == "verifying" {
			sawVerify = true
		}
	})
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if !result.Verified || result.Sha256 != sha256Hex(payload) {
		t.Fatalf("expected verified download, got %#v", result)
	}
	if !sawDownload || !sawVerify {
		t.Fatalf("expected download and verify progress, got download=%v verify=%v", sawDownload, sawVerify)
	}
	if _, err := os.Stat(result.FilePath); err != nil {
		t.Fatalf("expected downloaded file to exist: %v", err)
	}
}

// TestDownloadAndVerifyRejectsHTTPError 验证 管理更新包下载、校验、待安装状态和安装器启动 的关键行为，避免后续重构破坏既有约束。
func TestDownloadAndVerifyRejectsHTTPError(t *testing.T) {
	_, err := updater.NewManager(updater.Config{
		CacheDir: t.TempDir(),
		Client:   httpClient(http.StatusInternalServerError, []byte("boom")),
	}).DownloadAndVerify(context.Background(), updater.ReleaseAsset{
		LatestVersion:    "1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Hex([]byte("installer")),
	}, nil)
	if err == nil {
		t.Fatal("expected http error")
	}
}

// TestDownloadAndVerifyDeletesTempFileOnInterruptedDownload 验证 管理更新包下载、校验、待安装状态和安装器启动 的关键行为，避免后续重构破坏既有约束。
func TestDownloadAndVerifyDeletesTempFileOnInterruptedDownload(t *testing.T) {
	cacheDir := t.TempDir()
	_, err := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   httpClientWithBody(http.StatusOK, io.NopCloser(&interruptedReader{}), 16),
	}).DownloadAndVerify(context.Background(), updater.ReleaseAsset{
		LatestVersion:    "1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Hex([]byte("installer")),
	}, nil)
	if err == nil {
		t.Fatal("expected interrupted download error")
	}
	target := filepath.Join(cacheDir, "1.2.3", "go-desktop-v1.2.3-windows-amd64.exe")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected interrupted download target to be absent, stat err=%v", err)
	}
	if _, err := os.Stat(target + ".download"); !os.IsNotExist(err) {
		t.Fatalf("expected interrupted download temp file to be deleted, stat err=%v", err)
	}
}

// TestDownloadAndVerifyDeletesFileOnChecksumMismatch 验证 管理更新包下载、校验、待安装状态和安装器启动 的关键行为，避免后续重构破坏既有约束。
func TestDownloadAndVerifyDeletesFileOnChecksumMismatch(t *testing.T) {
	payload := []byte("installer")

	cacheDir := t.TempDir()
	_, err := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   httpClient(http.StatusOK, payload),
	}).DownloadAndVerify(context.Background(), updater.ReleaseAsset{
		LatestVersion:    "1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Hex([]byte("different")),
	}, nil)
	var mismatch updater.ChecksumMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	target := filepath.Join(cacheDir, "1.2.3", "go-desktop-v1.2.3-windows-amd64.exe")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected failed download to be deleted, stat err=%v", err)
	}
}

// TestInstallRequiresVerifiedFilePath 验证 管理更新包下载、校验、待安装状态和安装器启动 的关键行为，避免后续重构破坏既有约束。
func TestInstallRequiresVerifiedFilePath(t *testing.T) {
	manager := updater.NewManager(updater.Config{
		CacheDir: t.TempDir(),
		Runner: func(ctx context.Context, installerPath string) error {
			t.Fatalf("runner should not be called for missing file")
			return nil
		},
	})
	if err := manager.Install(context.Background(), filepath.Join(t.TempDir(), "missing.exe")); err == nil {
		t.Fatal("expected missing file error")
	}
}

// TestInstallRejectsFileOutsideUpdateCache 验证 管理更新包下载、校验、待安装状态和安装器启动 的关键行为，避免后续重构破坏既有约束。
func TestInstallRejectsFileOutsideUpdateCache(t *testing.T) {
	cacheDir := t.TempDir()
	outsideDir := t.TempDir()
	outsideInstaller := filepath.Join(outsideDir, "go-desktop-v1.2.3-windows-amd64.exe")
	if err := os.WriteFile(outsideInstaller, []byte("installer"), 0o600); err != nil {
		t.Fatalf("write installer fixture: %v", err)
	}

	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Runner: func(ctx context.Context, installerPath string) error {
			t.Fatalf("runner should not be called for file outside cache: %s", installerPath)
			return nil
		},
	})
	if err := manager.Install(context.Background(), outsideInstaller); err == nil {
		t.Fatal("expected file outside cache to be rejected")
	}
}

// TestIsOfflineError 验证 管理更新包下载、校验、待安装状态和安装器启动 的关键行为，避免后续重构破坏既有约束。
func TestIsOfflineError(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "example.invalid"}
	if !updater.IsOfflineError(err) {
		t.Fatal("expected DNS error to be offline")
	}
}

// sha256Hex 封装 管理更新包下载、校验、待安装状态和安装器启动 中的一段独立逻辑，调用方通过它复用同一业务规则。
func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// httpClient 封装 管理更新包下载、校验、待安装状态和安装器启动 中的一段独立逻辑，调用方通过它复用同一业务规则。
func httpClient(status int, body []byte) *http.Client {
	return httpClientWithBody(status, io.NopCloser(bytes.NewReader(body)), int64(len(body)))
}

// httpClientWithBody 封装 管理更新包下载、校验、待安装状态和安装器启动 中的一段独立逻辑，调用方通过它复用同一业务规则。
func httpClientWithBody(status int, body io.ReadCloser, contentLength int64) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    status,
			Header:        http.Header{"Content-Length": []string{strconv.FormatInt(contentLength, 10)}},
			Body:          body,
			ContentLength: contentLength,
			Request:       req,
		}, nil
	})}
}

// roundTripFunc 定义 管理更新包下载、校验、待安装状态和安装器启动 使用的数据实体，字段会直接参与校验、渲染、持久化或平台适配。
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip 封装 管理更新包下载、校验、待安装状态和安装器启动 中的一段独立逻辑，调用方通过它复用同一业务规则。
func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// interruptedReader 定义 管理更新包下载、校验、待安装状态和安装器启动 使用的数据实体，字段会直接参与校验、渲染、持久化或平台适配。
type interruptedReader struct {
	readOnce bool // readOnce 保存 readOnce 对应的数据，供当前实体的调用方读取或持久化。
}

// Read 读取、解析或归一化 管理更新包下载、校验、待安装状态和安装器启动 需要的数据，并把结果返回给调用方。
func (r *interruptedReader) Read(p []byte) (int, error) {
	if r.readOnce {
		return 0, errors.New("download interrupted")
	}
	r.readOnce = true
	return copy(p, []byte("partial")), nil
}
