// 文件职责：验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chencn/go-desktop/app"
	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	updater "github.com/chencn/go-desktop/internal/desktopapp/update"
)

func testAssetNames(version string) []string {
	return []string{
		"go-desktop-v" + version + "-windows-amd64.exe",
		"go-desktop-" + version + "-windows-amd64.exe",
		"go-desktop-setup-v" + version + ".exe",
		"go-desktop-setup-" + version + ".exe",
	}
}

// TestDownloadUpdateVerifiesAndWaitsForUserInstall 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDownloadUpdateVerifiesAndWaitsForUserInstall(t *testing.T) {
	payload := []byte("installer")

	var installedPath string
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   updateHTTPClient(http.StatusOK, payload),
		Runner: func(ctx context.Context, installerPath string) error {
			installedPath = installerPath
			return nil
		},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.DownloadUpdate()
	if status.Status != "verified" || !status.Verified {
		t.Fatalf("expected verified status, got %#v", status)
	}
	if installedPath != "" {
		t.Fatalf("expected installer runner not to be called before explicit install, got %s", installedPath)
	}
	if runtime.GetUpdateStatus().Status != "verified" {
		t.Fatalf("expected persisted update status, got %#v", runtime.GetUpdateStatus())
	}
}

func TestGetUpdateStatusRestoresVerifiedDownloadFromDisk(t *testing.T) {
	payload := []byte("installer")
	cacheDir := t.TempDir()
	firstRuntime := app.NewRuntime(app.ServiceOptions{
		Version:   "1.0.0",
		CachePath: cacheDir,
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
			Client:   updateHTTPClient(http.StatusOK, payload),
		}),
	})
	firstRuntime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		Source:           "github",
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})
	if status := firstRuntime.DownloadUpdate(); status.Status != "verified" || !status.Verified {
		t.Fatalf("expected verified download before restart, got %#v", status)
	}
	firstRuntime.Shutdown()

	restarted := app.NewRuntime(app.ServiceOptions{
		Version:   "1.0.0",
		CachePath: cacheDir,
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
			Runner: func(ctx context.Context, installerPath string) error {
				t.Fatalf("installer should not run while restoring status: %s", installerPath)
				return nil
			},
		}),
	})
	defer restarted.Shutdown()

	status := restarted.GetUpdateStatus()
	if status.Status != "verified" || !status.Verified || status.Source != "github" {
		t.Fatalf("expected verified status restored from disk, got %#v", status)
	}
	if status.FilePath == "" || status.Sha256 != sha256Text(payload) {
		t.Fatalf("expected restored file path and sha256, got %#v", status)
	}
}

func TestGetUpdateStatusClearsStaleVerifiedDownloadForCurrentVersion(t *testing.T) {
	payload := []byte("installer")
	cacheDir := t.TempDir()
	firstRuntime := app.NewRuntime(app.ServiceOptions{
		Version:   "1.0.0",
		CachePath: cacheDir,
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
			Client:   updateHTTPClient(http.StatusOK, payload),
		}),
	})
	firstRuntime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		Source:           "github",
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})
	if status := firstRuntime.DownloadUpdate(); status.Status != "verified" {
		t.Fatalf("expected verified download before restart, got %#v", status)
	}
	firstRuntime.Shutdown()

	restarted := app.NewRuntime(app.ServiceOptions{
		Version:   "1.2.3",
		CachePath: cacheDir,
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
		}),
	})
	defer restarted.Shutdown()

	status := restarted.GetUpdateStatus()
	if status.Status == "verified" || status.Verified {
		t.Fatalf("expected stale verified package to be ignored for current version, got %#v", status)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "verified.json")); !os.IsNotExist(err) {
		t.Fatalf("expected stale verified cache to be cleared, stat err=%v", err)
	}
}

// TestCheckUpdateAutoDownloadsButDoesNotInstall 验证检查更新会自动下载校验但不会直接安装。
func TestCheckUpdateAutoDownloadsButDoesNotInstall(t *testing.T) {
	payload := []byte("installer")

	var installedPath string
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   updateHTTPClient(http.StatusOK, payload),
		Runner: func(ctx context.Context, installerPath string) error {
			installedPath = installerPath
			return nil
		},
	})
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		CurrentVersion: "1.0.0",
		AssetNames:     testAssetNames,
		HTTPClient: updateHTTPClient(http.StatusOK, []byte(`[
			{
				"tag_name": "v1.2.3",
				"draft": false,
				"prerelease": false,
				"assets": [
					{
						"name": "go-desktop-v1.2.3-windows-amd64.exe",
						"browser_download_url": "https://example.test/installer.exe",
						"digest": "sha256:`+sha256Text(payload)+`"
					}
				]
			}
		]`)),
	})
	runtime := app.NewRuntime(app.ServiceOptions{
		Version:        "1.0.0",
		CachePath:      cacheDir,
		ReleaseChecker: checker,
		UpdateManager:  manager,
	})

	result := runtime.CheckUpdate()
	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available check result, got %#v", result)
	}
	status := runtime.GetUpdateStatus()
	if status.Status != "verified" || !status.Verified {
		t.Fatalf("expected update check to download and wait for install, got %#v", status)
	}
	if installedPath != "" {
		t.Fatalf("expected installer runner not to be called before explicit install, got %s", installedPath)
	}
}

func TestCheckUpdateUsesLocalManifestWhenSourceIsLocal(t *testing.T) {
	payload := []byte("installer")
	sha := sha256Text(payload)
	var manifestRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/exe/go-desktop/releases/latest.json":
			manifestRequests++
			downloadURL := "http://" + r.Host + "/exe/go-desktop/releases/download/v1.2.3/go-desktop-v1.2.3-windows-amd64.exe"
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"tag_name": "v1.2.3",
					"draft": false,
					"prerelease": false,
					"assets": [
						{
							"name": "go-desktop-v1.2.3-windows-amd64.exe",
							"size": 9,
							"digest": "sha256:` + sha + `",
							"browser_download_url": "` + downloadURL + `"
						}
					]
				}
			]`))
		case "/exe/go-desktop/releases/download/v1.2.3/go-desktop-v1.2.3-windows-amd64.exe":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cacheDir := t.TempDir()

	runtime := app.NewRuntime(app.ServiceOptions{
		Version:                 "1.0.0",
		CachePath:               cacheDir,
		DatabasePath:            filepath.Join(t.TempDir(), "go-desktop.db"),
		LocalUpdateBaseURL:      server.URL + "/exe/go-desktop",
		LocalUpdateManifestPath: "releases/latest.json",
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
			Runner: func(ctx context.Context, installerPath string) error {
				t.Fatalf("installer should not run during local check: %s", installerPath)
				return nil
			},
		}),
	})
	defer runtime.Shutdown()
	if _, err := runtime.SaveSettings(app.Settings{
		UpdateSource:             "local",
		GitHubOwner:              "chencn",
		GitHubRepo:               "go-desktop",
		UpdateCheckIntervalHours: 3,
		MinimizeToTray:           true,
		LogRetentionDays:         30,
		CreateDesktopShortcut:    true,
	}); err != nil {
		t.Fatalf("save local update source: %v", err)
	}

	result := runtime.CheckUpdate()
	if result.Status != githubrelease.StatusUpdateAvailable || result.Source != "local" {
		t.Fatalf("expected local update_available, got %#v", result)
	}
	if manifestRequests != 1 {
		t.Fatalf("expected one local manifest request, got %d", manifestRequests)
	}
	status := runtime.GetUpdateStatus()
	if status.Status != "verified" || status.Source != "local" {
		t.Fatalf("expected local verified status after auto download, got %#v", status)
	}
}

// TestInstallDownloadedUpdateStartsInstallerAfterVerifiedDownload 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestInstallDownloadedUpdateStartsInstallerAfterVerifiedDownload(t *testing.T) {
	payload := []byte("installer")

	var installedPath string
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   updateHTTPClient(http.StatusOK, payload),
		Runner: func(ctx context.Context, installerPath string) error {
			installedPath = installerPath
			return nil
		},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	if status := runtime.DownloadUpdate(); status.Status != "verified" || !status.Verified {
		t.Fatalf("expected verified pending install before explicit install, got %#v", status)
	}
	status := runtime.InstallDownloadedUpdate()
	if status.Status != "install_started" || !status.Verified {
		t.Fatalf("expected explicit install to start installer, got %#v", status)
	}
	if installedPath == "" {
		t.Fatal("expected installer runner to be called after explicit install")
	}
}

// TestInstallDownloadedUpdateRequiresVerifiedDownload 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestInstallDownloadedUpdateRequiresVerifiedDownload(t *testing.T) {
	payload := []byte("installer")

	var installedPath string
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   updateHTTPClient(http.StatusOK, payload),
		Runner: func(ctx context.Context, installerPath string) error {
			installedPath = installerPath
			return nil
		},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.InstallDownloadedUpdate()
	if status.Status != "error" || status.ErrorReason != "not_verified" {
		t.Fatalf("expected not_verified error before download, got %#v", status)
	}
	if installedPath != "" {
		t.Fatalf("expected installer runner not to be called, got %s", installedPath)
	}
}

// TestDownloadUpdateRejectsMissingSha256 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDownloadUpdateRejectsMissingSha256(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/app.exe",
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.DownloadUpdate()
	if status.Status != "error" || status.ErrorReason != "sha256_missing" {
		t.Fatalf("expected sha256_missing error, got %#v", status)
	}
}

// TestDownloadUpdateReportsChecksumMismatchAndDoesNotInstall 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDownloadUpdateReportsChecksumMismatchAndDoesNotInstall(t *testing.T) {
	payload := []byte("installer")

	var installedPath string
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   updateHTTPClient(http.StatusOK, payload),
		Runner: func(ctx context.Context, installerPath string) error {
			installedPath = installerPath
			return nil
		},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text([]byte("different")),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.DownloadUpdate()
	if status.Status != "error" || status.ErrorReason != "sha256_mismatch" {
		t.Fatalf("expected sha256_mismatch error, got %#v", status)
	}
	if installedPath != "" {
		t.Fatalf("expected installer runner not to be called, got %s", installedPath)
	}
}

// TestDownloadUpdateSkipsOfflineDownload 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDownloadUpdateSkipsOfflineDownload(t *testing.T) {
	var installedPath string
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client: &http.Client{Transport: updateRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: no such host")
		})},
		Runner: func(ctx context.Context, installerPath string) error {
			installedPath = installerPath
			return nil
		},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text([]byte("installer")),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.DownloadUpdate()
	if status.Status != "skipped" || status.ErrorReason != githubrelease.SkipReasonOffline {
		t.Fatalf("expected offline skipped status, got %#v", status)
	}
	if installedPath != "" {
		t.Fatalf("expected installer runner not to be called, got %s", installedPath)
	}
}

func TestDownloadUpdateSerialisesConcurrentRequests(t *testing.T) {
	payload := []byte("installer")
	var activeRequests int32
	var maxActiveRequests int32
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client: &http.Client{Transport: updateRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			active := atomic.AddInt32(&activeRequests, 1)
			for {
				maximum := atomic.LoadInt32(&maxActiveRequests)
				if active <= maximum || atomic.CompareAndSwapInt32(&maxActiveRequests, maximum, active) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&activeRequests, -1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Length": []string{strconv.Itoa(len(payload))}},
				Body:       io.NopCloser(bytes.NewReader(payload)),
				Request:    req,
			}, nil
		})},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	var wait sync.WaitGroup
	start := make(chan struct{})
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			<-start
			if status := runtime.DownloadUpdate(); status.Status != "verified" {
				t.Errorf("expected verified status, got %#v", status)
			}
		}()
	}
	close(start)
	wait.Wait()

	if maxActiveRequests > 1 {
		t.Fatalf("expected download requests to be serialised, max active requests=%d", maxActiveRequests)
	}
}

// TestDownloadUpdateDoesNotExposeEnglishTransportError 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestDownloadUpdateDoesNotExposeEnglishTransportError(t *testing.T) {
	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client: &http.Client{Transport: updateRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected EOF")
		})},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text([]byte("installer")),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.DownloadUpdate()
	if strings.Contains(status.Message, "unexpected EOF") {
		t.Fatalf("expected user-facing message to be Chinese, got %q", status.Message)
	}
}

// TestInstallDownloadedUpdateDoesNotExposeEnglishRunnerError 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestInstallDownloadedUpdateDoesNotExposeEnglishRunnerError(t *testing.T) {
	payload := []byte("installer")

	cacheDir := t.TempDir()
	manager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   updateHTTPClient(http.StatusOK, payload),
		Runner: func(ctx context.Context, installerPath string) error {
			return errors.New("CreateProcess failed")
		},
	})
	runtime := app.NewRuntime(app.ServiceOptions{CachePath: cacheDir, UpdateManager: manager})
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.DownloadUpdate()
	if status.Status != "verified" || !status.Verified {
		t.Fatalf("expected pending_install before explicit install, got %#v", status)
	}
	status = runtime.InstallDownloadedUpdate()
	if status.Status != "error" || status.ErrorReason != "install_failed" {
		t.Fatalf("expected install_failed error after explicit install, got %#v", status)
	}
	if strings.Contains(status.Message, "CreateProcess failed") {
		t.Fatalf("expected installer error to be localised, got %q", status.Message)
	}
}

// TestDownloadUpdateRequiresCurrentProcessCheckAfterRestart 验证重启后不会从数据库恢复旧更新检查结果。
func TestDownloadUpdateRequiresCurrentProcessCheckAfterRestart(t *testing.T) {
	payload := []byte("installer")

	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	firstRuntime := app.NewRuntime(app.ServiceOptions{DatabasePath: dbPath})
	firstRuntime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})
	firstRuntime.Shutdown()

	manager := updater.NewManager(updater.Config{
		CacheDir: t.TempDir(),
		Client:   updateHTTPClient(http.StatusOK, payload),
	})
	restarted := app.NewRuntime(app.ServiceOptions{
		DatabasePath:  dbPath,
		UpdateManager: manager,
		ReleaseChecker: githubrelease.NewChecker(githubrelease.Config{
			Owner:          "chencn",
			Repo:           "go-desktop",
			CurrentVersion: "1.0.0",
			AssetNames:     testAssetNames,
		}),
	})
	defer restarted.Shutdown()

	status := restarted.DownloadUpdate()
	if status.Status != "error" || status.ErrorReason != "missing_update_check" {
		t.Fatalf("expected missing current process check, got %#v", status)
	}
}

// TestPendingStartupInstallPersistsAcrossRestart 验证下次启动安装状态通过 updates/pending.json 跨重启恢复。
func TestPendingStartupInstallPersistsAcrossRestart(t *testing.T) {
	payload := []byte("installer")
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	cacheDir := filepath.Join(t.TempDir(), "data", "updates")
	var startupInstalledPath string

	firstManager := updater.NewManager(updater.Config{
		CacheDir: cacheDir,
		Client:   updateHTTPClient(http.StatusOK, payload),
		Runner: func(ctx context.Context, installerPath string) error {
			return nil
		},
	})
	firstRuntime := app.NewRuntime(app.ServiceOptions{
		DatabasePath:  dbPath,
		CachePath:     cacheDir,
		UpdateManager: firstManager,
	})
	firstRuntime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})
	if status := firstRuntime.DownloadUpdate(); status.Status != "verified" {
		t.Fatalf("expected download session to wait for install, got %#v", status)
	}
	if status := firstRuntime.ScheduleDownloadedUpdateOnStartup(); status.Status != "pending_install" {
		t.Fatalf("expected explicit startup schedule, got %#v", status)
	}
	pendingPath := filepath.Join(cacheDir, "pending.json")
	if _, err := os.Stat(pendingPath); err != nil {
		t.Fatalf("expected pending install file to be persisted: %v", err)
	}
	firstRuntime.Shutdown()

	restarted := app.NewRuntime(app.ServiceOptions{
		DatabasePath: dbPath,
		CachePath:    cacheDir,
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
			Runner: func(ctx context.Context, installerPath string) error {
				startupInstalledPath = installerPath
				return nil
			},
		}),
	})
	defer restarted.Shutdown()

	status := restarted.InstallPendingUpdateOnStartup()
	if status.Status != "install_started" {
		t.Fatalf("expected restart to install persisted pending update, got %#v", status)
	}
	if startupInstalledPath == "" {
		t.Fatal("expected restarted runtime to run installer")
	}
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Fatalf("expected pending install file to be cleared after install start, stat err=%v", err)
	}
}

// TestPendingStartupInstallRejectsInstallerOutsideCache 验证 pending.json 不能指向更新缓存目录外的安装包。
func TestPendingStartupInstallRejectsInstallerOutsideCache(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	cacheDir := filepath.Join(t.TempDir(), "data", "updates")
	outsideInstaller := filepath.Join(t.TempDir(), "installer.exe")
	if err := os.WriteFile(outsideInstaller, []byte("installer"), 0o600); err != nil {
		t.Fatalf("write outside installer: %v", err)
	}

	pendingPath := filepath.Join(cacheDir, "pending.json")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("create cache dir: %v", err)
	}
	pending, err := json.Marshal(map[string]any{
		"status":    "pending_install",
		"message":   "安装包已校验，将在本次启动时自动更新。",
		"version":   "1.2.3",
		"assetName": "go-desktop-v1.2.3-windows-amd64.exe",
		"filePath":  outsideInstaller,
		"sha256":    strings.Repeat("a", 64),
		"verified":  true,
		"updatedAt": "2026-06-03T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal pending update: %v", err)
	}
	if err := os.WriteFile(pendingPath, pending, 0o600); err != nil {
		t.Fatalf("write pending update: %v", err)
	}

	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: dbPath,
		CachePath:    cacheDir,
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
			Runner: func(ctx context.Context, installerPath string) error {
				t.Fatalf("installer should not run for path outside cache: %s", installerPath)
				return nil
			},
		}),
	})
	defer runtime.Shutdown()

	status := runtime.InstallPendingUpdateOnStartup()
	if status.Status != "error" || status.ErrorReason != "pending_load_failed" {
		t.Fatalf("expected pending_load_failed for installer outside cache, got %#v", status)
	}
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Fatalf("expected invalid pending install file to be cleared, stat err=%v", err)
	}
}

// TestScheduleDownloadedUpdateOnStartupMarksCurrentStatus 验证安排下次启动只更新当前状态。
func TestScheduleDownloadedUpdateOnStartupMarksCurrentStatus(t *testing.T) {
	payload := []byte("installer")
	dbPath := filepath.Join(t.TempDir(), "go-desktop.db")
	cacheDir := t.TempDir()

	runtime := app.NewRuntime(app.ServiceOptions{
		DatabasePath: dbPath,
		CachePath:    cacheDir,
		UpdateManager: updater.NewManager(updater.Config{
			CacheDir: cacheDir,
			Client:   updateHTTPClient(http.StatusOK, payload),
			Runner: func(ctx context.Context, installerPath string) error {
				return nil
			},
		}),
	})
	defer runtime.Shutdown()
	runtime.RecordUpdateCheckResult(githubrelease.CheckResult{
		Status:           githubrelease.StatusUpdateAvailable,
		CurrentVersion:   "1.0.0",
		LatestVersion:    "1.2.3",
		TagName:          "v1.2.3",
		AssetName:        "go-desktop-v1.2.3-windows-amd64.exe",
		AssetDownloadURL: "https://example.test/installer.exe",
		Sha256:           sha256Text(payload),
		CheckedAt:        "2026-06-03T00:00:00Z",
		Message:          "发现新版本。",
	})

	status := runtime.DownloadUpdate()
	if status.Status != "verified" {
		t.Fatalf("expected verified, got %#v", status)
	}
	status = runtime.ScheduleDownloadedUpdateOnStartup()
	if status.Status != "pending_install" {
		t.Fatalf("expected pending_install after explicit scheduling, got %#v", status)
	}
	if current := runtime.GetUpdateStatus(); current.Status != "pending_install" || !current.Verified {
		t.Fatalf("expected current status to be pending_install, got %#v", current)
	}
}

// TestScheduleDownloadedUpdateOnStartupRequiresVerifiedDownload 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 的关键行为，避免后续重构破坏既有约束。
func TestScheduleDownloadedUpdateOnStartupRequiresVerifiedDownload(t *testing.T) {
	runtime := app.NewRuntime(app.ServiceOptions{})

	status := runtime.ScheduleDownloadedUpdateOnStartup()
	if status.Status != "error" || status.ErrorReason != "not_verified" {
		t.Fatalf("expected not_verified error before download, got %#v", status)
	}
}

// sha256Text 封装 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 中的一段独立逻辑，调用方通过它复用同一业务规则。
func sha256Text(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// updateHTTPClient 修改 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 管理的状态、文件或外部副作用，并把失败原因向上返回。
func updateHTTPClient(status int, body []byte) *http.Client {
	return &http.Client{Transport: updateRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Length": []string{strconv.Itoa(len(body))}},
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    req,
		}, nil
	})}
}

// updateRoundTripFunc 定义 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 使用的数据实体，字段会直接参与校验、渲染、持久化或平台适配。
type updateRoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip 封装 验证 update_service_test.go 覆盖的生产行为、结构约束或构建脚本约束 中的一段独立逻辑，调用方通过它复用同一业务规则。
func (fn updateRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
