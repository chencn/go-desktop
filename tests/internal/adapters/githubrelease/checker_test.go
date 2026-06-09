// 文件职责：验证 GitHub/local Release 清单解析、安装资产匹配和 SHA256 来源选择。

package githubrelease_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func testAssetNames(version string) []string {
	return []string{
		"go-desktop-v" + version + "-windows-amd64.exe",
		"go-desktop-" + version + "-windows-amd64.exe",
		"go-desktop-setup-v" + version + ".exe",
		"go-desktop-setup-" + version + ".exe",
	}
}

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// TestCheckStaticSelectsNewestStableVersion 验证 draft、prerelease 和非法 tag 不参与最新稳定版本选择。
func TestCheckStaticSelectsNewestStableVersion(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "0.9.9",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v2.0.0",
			"draft": true,
			"assets": []
		},
		{
			"tag_name": "v1.9.0",
			"prerelease": true,
			"assets": []
		},
		{
			"tag_name": "nightly",
			"assets": []
		},
		{
			"tag_name": "v1.10.0",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-v1.10.0-windows-amd64.exe",
					"digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"browser_download_url": "https://example.test/primary.exe"
				}
			]
		},
		{
			"tag_name": "v1.2.0",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-v1.2.0-windows-amd64.exe",
					"digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"browser_download_url": "https://example.test/old.exe"
				}
			]
		}
	]`))

	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available, got %s: %s", result.Status, result.Message)
	}
	if result.LatestVersion != "1.10.0" || result.TagName != "v1.10.0" {
		t.Fatalf("expected v1.10.0 to be selected, got latest=%q tag=%q", result.LatestVersion, result.TagName)
	}
}

func TestCheckStaticNormalizesShortReleaseTags(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "0.9.9",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v1.2",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-v1.2.0-windows-amd64.exe",
					"digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"browser_download_url": "https://example.test/primary.exe"
				}
			]
		}
	]`))

	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available, got %s: %s", result.Status, result.Message)
	}
	if result.LatestVersion != "1.2.0" {
		t.Fatalf("expected normalized latest version, got %q", result.LatestVersion)
	}
}

// TestCheckStaticRejectsInvalidReleaseTags 验证没有可解析稳定版本时返回受控错误。
func TestCheckStaticRejectsInvalidReleaseTags(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "0.9.9",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{"tag_name": "release-1.0.0", "draft": false, "prerelease": false, "assets": []},
		{"tag_name": "1.2.3.4", "draft": false, "prerelease": false, "assets": []},
		{"tag_name": "v1.-2.3", "draft": false, "prerelease": false, "assets": []}
	]`))

	if result.Status != githubrelease.StatusError {
		t.Fatalf("expected error for invalid release tags, got %s", result.Status)
	}
	if result.ErrorReason != "no_available_release" {
		t.Fatalf("unexpected error reason: %s", result.ErrorReason)
	}
}

// TestCheckStaticTreatsVPrefixAsSameVersion 验证版本比较忽略 tag 的大小写 v 前缀。
func TestCheckStaticTreatsVPrefixAsSameVersion(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "V1.0.0",
			"draft": false,
			"prerelease": false,
			"assets": []
		}
	]`))

	if result.Status != githubrelease.StatusNoUpdate {
		t.Fatalf("expected no_update for V1.0.0 vs 1.0.0, got %s", result.Status)
	}
}

// TestCheckStaticSelectsPrimaryWindowsAssetAndDigest 验证资产名优先级选择主安装包，并优先使用 GitHub digest。
func TestCheckStaticSelectsPrimaryWindowsAssetAndDigest(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "0.1.0",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v1.2.3",
			"html_url": "https://github.com/chencn/go-desktop/releases/tag/v1.2.3",
			"body": "中文发布说明",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-setup-v1.2.3.exe",
					"size": 12,
					"browser_download_url": "https://example.test/fallback.exe"
				},
				{
					"name": "go-desktop-v1.2.3-windows-amd64.exe",
					"size": 34,
					"digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"browser_download_url": "https://example.test/primary.exe"
				}
			]
		}
	]`))

	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available, got %s: %s", result.Status, result.Message)
	}
	if result.AssetName != "go-desktop-v1.2.3-windows-amd64.exe" {
		t.Fatalf("unexpected asset: %s", result.AssetName)
	}
	if result.Sha256Source != githubrelease.Sha256SourceDigest {
		t.Fatalf("expected github digest, got %s", result.Sha256Source)
	}
	if result.Sha256 != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected sha256: %s", result.Sha256)
	}
}

// TestCheckStaticNoUpdate 验证最新稳定版本不高于当前版本时返回 no_update。
func TestCheckStaticNoUpdate(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.2.3",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v1.2.3",
			"draft": false,
			"prerelease": false,
			"assets": []
		}
	]`))

	if result.Status != githubrelease.StatusNoUpdate {
		t.Fatalf("expected no_update, got %s", result.Status)
	}
}

// TestCheckStaticRequiresChecksumForUpdate 验证可更新安装包必须带 digest 或同名 .sha256 资产。
func TestCheckStaticRequiresChecksumForUpdate(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v1.2.3",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-v1.2.3-windows-amd64.exe",
					"size": 34,
					"browser_download_url": "https://example.test/primary.exe"
				}
			]
		}
	]`))

	if result.Status != githubrelease.StatusError {
		t.Fatalf("expected error when checksum is missing, got %s", result.Status)
	}
}

// TestCheckStaticMarksSha256AssetFallback 验证静态解析能标记同名 .sha256 作为校验摘要来源。
func TestCheckStaticMarksSha256AssetFallback(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v1.2.3",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-v1.2.3-windows-amd64.exe",
					"size": 34,
					"browser_download_url": "https://example.test/primary.exe"
				},
				{
					"name": "go-desktop-v1.2.3-windows-amd64.exe.sha256",
					"browser_download_url": "https://example.test/primary.exe.sha256"
				}
			]
		}
	]`))

	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available, got %s", result.Status)
	}
	if result.Sha256Source != githubrelease.Sha256SourceAsset {
		t.Fatalf("expected sha256 asset fallback, got %s", result.Sha256Source)
	}
}

// TestCheckFetchesSha256AssetFallback 验证联网检查会读取同 Release 的 .sha256 文件并保留审计字段。
func TestCheckFetchesSha256AssetFallback(t *testing.T) {
	sha := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	var requested []string
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requested = append(requested, req.URL.String())
			if req.Header.Get("User-Agent") != "go-desktop-updater" {
				t.Fatalf("unexpected user agent: %s", req.Header.Get("User-Agent"))
			}
			if req.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Fatalf("unexpected api version: %s", req.Header.Get("X-GitHub-Api-Version"))
			}
			if strings.Contains(req.URL.Path, ".sha256") {
				return textResponse(sha + "  go-desktop.exe"), nil
			}
			return jsonResponse(`[
				{
					"tag_name": "v1.2.3",
					"draft": false,
					"prerelease": false,
					"assets": [
						{
							"name": "go-desktop-v1.2.3-windows-amd64.exe",
							"browser_download_url": "https://example.test/primary.exe"
						},
						{
							"name": "go-desktop-v1.2.3-windows-amd64.exe.sha256",
							"browser_download_url": "https://example.test/primary.exe.sha256"
						}
					]
				}
			]`), nil
		})},
	})

	result := checker.Check(context.Background())
	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available, got %s: %s", result.Status, result.Message)
	}
	if result.Sha256Source != githubrelease.Sha256SourceAsset {
		t.Fatalf("expected sha256 asset fallback, got %s", result.Sha256Source)
	}
	if result.Sha256 != sha {
		t.Fatalf("unexpected sha256: %s", result.Sha256)
	}
	if result.RequestURL != "https://api.github.com/repos/chencn/go-desktop/releases?per_page=30" {
		t.Fatalf("unexpected request url: %s", result.RequestURL)
	}
	if result.HTTPStatus != http.StatusOK {
		t.Fatalf("unexpected http status: %d", result.HTTPStatus)
	}
	if len(requested) != 2 {
		t.Fatalf("expected release and sha256 requests, got %d", len(requested))
	}
}

func TestCheckReadsLocalManifestURL(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	var requested []string
	checker := githubrelease.NewChecker(githubrelease.Config{
		ManifestURL:    "https://updates.example/releases/latest.json",
		Source:         "local",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requested = append(requested, req.URL.String())
			if req.Header.Get("Accept") != "application/json" {
				t.Fatalf("unexpected accept header: %s", req.Header.Get("Accept"))
			}
			return jsonResponse(`[
				{
					"tag_name": "v1.2.3",
					"html_url": "https://updates.example/releases/download/v1.2.3/",
					"draft": false,
					"prerelease": false,
					"assets": [
						{
							"name": "go-desktop-v1.2.3-windows-amd64.exe",
							"size": 123,
							"digest": "sha256:` + sha + `",
							"browser_download_url": "https://updates.example/releases/download/v1.2.3/go-desktop-v1.2.3-windows-amd64.exe"
						}
					]
				}
			]`), nil
		})},
	})

	result := checker.Check(context.Background())
	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available, got %s: %s", result.Status, result.Message)
	}
	if result.Source != "local" {
		t.Fatalf("expected local source, got %#v", result)
	}
	if result.RequestURL != "https://updates.example/releases/latest.json" || len(requested) != 1 {
		t.Fatalf("expected one manifest request, got url=%q requested=%v", result.RequestURL, requested)
	}
}

// TestParseSha256Text 验证兼容常见 “hash filename” 格式的 .sha256 文本。
func TestParseSha256Text(t *testing.T) {
	sha, ok := githubrelease.ParseSha256Text("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  go-desktop.exe")
	if !ok {
		t.Fatal("expected sha256 text to parse")
	}
	if sha != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected sha: %s", sha)
	}
}

// TestCheckStaticRejectsDigestWithTrailingText 验证 GitHub digest 必须是完整 sha256:<64 hex>，不能接受尾随文本。
func TestCheckStaticRejectsDigestWithTrailingText(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v1.2.3",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-v1.2.3-windows-amd64.exe",
					"digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef trailing",
					"browser_download_url": "https://example.test/primary.exe"
				}
			]
		}
	]`))

	if result.Status != githubrelease.StatusError {
		t.Fatalf("expected invalid digest to be ignored and checksum missing error, got %s", result.Status)
	}
	if result.ErrorReason != "sha256_missing" {
		t.Fatalf("unexpected error reason: %s", result.ErrorReason)
	}
}

// TestProxiedURLIsIdempotent 验证已带同一代理前缀的下载 URL 不会被重复包一层代理。
func TestProxiedURLIsIdempotent(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		ProxyBase:      "https://gh-proxy.example",
		Now:            fixedNow,
	})
	result := checker.CheckStatic([]byte(`[
		{
			"tag_name": "v1.2.3",
			"draft": false,
			"prerelease": false,
			"assets": [
				{
					"name": "go-desktop-v1.2.3-windows-amd64.exe",
					"digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"browser_download_url": "https://gh-proxy.example/https://github.com/chencn/go-desktop/releases/download/v1.0.0/app.exe"
				}
			]
		}
	]`))

	if result.Status != githubrelease.StatusUpdateAvailable {
		t.Fatalf("expected update_available, got %s", result.Status)
	}
	if result.AssetDownloadURL != "https://gh-proxy.example/https://github.com/chencn/go-desktop/releases/download/v1.0.0/app.exe" {
		t.Fatalf("expected proxied URL to stay idempotent, got %s", result.AssetDownloadURL)
	}
}

// TestCheckReturnsIgnoredOfflineForNetworkFailure 验证离线检查返回 ignored/offline，避免误报成更新失败。
func TestCheckReturnsIgnoredOfflineForNetworkFailure(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: no such host")
		})},
	})

	result := checker.Check(context.Background())
	if result.Status != githubrelease.StatusIgnored {
		t.Fatalf("expected ignored, got %s", result.Status)
	}
	if result.SkipReason != githubrelease.SkipReasonOffline {
		t.Fatalf("expected offline skip reason, got %s", result.SkipReason)
	}
	if result.ErrorReason != githubrelease.SkipReasonOffline {
		t.Fatalf("expected offline error reason, got %s", result.ErrorReason)
	}
	if result.RequestURL != "https://api.github.com/repos/chencn/go-desktop/releases?per_page=30" {
		t.Fatalf("unexpected request url: %s", result.RequestURL)
	}
}

// TestCheckRecordsHTTPStatusAndErrorReason 验证 HTTP 错误状态会进入结果，方便日志和 UI 排障。
func TestCheckRecordsHTTPStatusAndErrorReason(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("server error")),
			}, nil
		})},
	})

	result := checker.Check(context.Background())
	if result.Status != githubrelease.StatusError {
		t.Fatalf("expected error, got %s", result.Status)
	}
	if result.HTTPStatus != http.StatusInternalServerError {
		t.Fatalf("unexpected http status: %d", result.HTTPStatus)
	}
	if result.ErrorReason != "http_error" {
		t.Fatalf("unexpected error reason: %s", result.ErrorReason)
	}
}

// TestCheckKeepsReleaseContextWhenSha256AssetIsOffline 验证 .sha256 下载离线时仍保留已解析的版本和资产上下文。
func TestCheckKeepsReleaseContextWhenSha256AssetIsOffline(t *testing.T) {
	checker := githubrelease.NewChecker(githubrelease.Config{
		Owner:          "chencn",
		Repo:           "go-desktop",
		AssetNames:     testAssetNames,
		CurrentVersion: "1.0.0",
		Now:            fixedNow,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, ".sha256") {
				return nil, errors.New("dial tcp: network is unreachable")
			}
			return jsonResponse(`[
				{
					"tag_name": "v1.2.3",
					"html_url": "https://github.com/chencn/go-desktop/releases/tag/v1.2.3",
					"draft": false,
					"prerelease": false,
					"assets": [
						{
							"name": "go-desktop-v1.2.3-windows-amd64.exe",
							"browser_download_url": "https://example.test/primary.exe"
						},
						{
							"name": "go-desktop-v1.2.3-windows-amd64.exe.sha256",
							"browser_download_url": "https://example.test/primary.exe.sha256"
						}
					]
				}
			]`), nil
		})},
	})

	result := checker.Check(context.Background())
	if result.Status != githubrelease.StatusIgnored {
		t.Fatalf("expected ignored, got %s", result.Status)
	}
	if result.SkipReason != githubrelease.SkipReasonOffline {
		t.Fatalf("expected offline skip reason, got %s", result.SkipReason)
	}
	if result.ErrorReason != "sha256_asset_offline" {
		t.Fatalf("unexpected error reason: %s", result.ErrorReason)
	}
	if result.HTTPStatus != http.StatusOK {
		t.Fatalf("unexpected http status: %d", result.HTTPStatus)
	}
	if result.TagName != "v1.2.3" || result.AssetName != "go-desktop-v1.2.3-windows-amd64.exe" {
		t.Fatalf("expected release context to be preserved, got tag=%q asset=%q", result.TagName, result.AssetName)
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 2, 3, 0, 0, 0, time.UTC)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func textResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
