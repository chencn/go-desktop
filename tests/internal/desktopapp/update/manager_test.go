// 文件职责：验证更新包管理器的下载、校验、缓存边界和安装器启动约束。

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

// TestDownloadAndVerifySuccess 验证成功路径会落盘安装包、上报下载/校验进度并返回 verified 结果。
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

// TestDownloadAndVerifyRejectsHTTPError 验证 HTTP 非 2xx 响应不会被当成可校验安装包。
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

// TestDownloadAndVerifyDeletesTempFileOnInterruptedDownload 验证下载中断会清理目标文件和 .download 临时文件。
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

// TestDownloadAndVerifyKeepsExistingTargetOnChecksumMismatch 验证 SHA256 不匹配时只清理本次临时下载。
func TestDownloadAndVerifyKeepsExistingTargetOnChecksumMismatch(t *testing.T) {
	previousPayload := []byte("verified installer")
	payload := []byte("corrupt installer")
	cacheDir := t.TempDir()
	target := filepath.Join(cacheDir, "1.2.3", "go-desktop-v1.2.3-windows-amd64.exe")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	if err := os.WriteFile(target, previousPayload, 0o600); err != nil {
		t.Fatalf("write previous verified target: %v", err)
	}

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
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected previous verified target to remain: %v", err)
	}
	if string(data) != string(previousPayload) {
		t.Fatalf("expected previous verified target to remain unchanged, got %q", string(data))
	}
	if _, err := os.Stat(target + ".download"); !os.IsNotExist(err) {
		t.Fatalf("expected failed temp download to be deleted, stat err=%v", err)
	}
}

func TestDownloadAndVerifyRejectsOversizedResponse(t *testing.T) {
	payload := []byte("installer payload is larger than declared")
	cacheDir := t.TempDir()

	_, err := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   httpClient(http.StatusOK, payload),
	}).DownloadAndVerify(context.Background(), updater.ReleaseAsset{
		LatestVersion:    "1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetSizeBytes:   4,
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Hex(payload),
	}, nil)
	if err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
	target := filepath.Join(cacheDir, "1.2.3", "go-desktop-v1.2.3-windows-amd64.exe")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected oversized download target to be absent, stat err=%v", err)
	}
	if _, err := os.Stat(target + ".download"); !os.IsNotExist(err) {
		t.Fatalf("expected oversized download temp file to be deleted, stat err=%v", err)
	}
}

// TestInstallRequiresVerifiedFilePath 验证安装前必须能读取已校验文件。
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

// TestInstallRejectsFileOutsideUpdateCache 验证安装器路径必须留在更新缓存目录内。
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

// TestIsOfflineError 验证 DNS 失败会归类为离线错误，供上层返回 skipped 状态。
func TestIsOfflineError(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "example.invalid"}
	if !updater.IsOfflineError(err) {
		t.Fatal("expected DNS error to be offline")
	}
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// httpClient 构造固定响应下载客户端，避免触发真实网络。
func httpClient(status int, body []byte) *http.Client {
	return httpClientWithBody(status, io.NopCloser(bytes.NewReader(body)), int64(len(body)))
}

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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// interruptedReader 首次读取返回部分内容，第二次读取模拟下载连接中断。
type interruptedReader struct {
	readOnce bool
}

func (r *interruptedReader) Read(p []byte) (int, error) {
	if r.readOnce {
		return 0, errors.New("download interrupted")
	}
	r.readOnce = true
	return copy(p, []byte("partial")), nil
}
