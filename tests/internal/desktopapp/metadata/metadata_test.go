// 文件职责：验证由 project.metadata.json 生成的后端 metadata 常量。

package metadata_test

import (
	"testing"

	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

// TestMetadataDefinesStableProductDefaults 验证产品身份、仓库地址和运行时默认设置没有偏离元数据真源。
func TestMetadataDefinesStableProductDefaults(t *testing.T) {
	if metadata.AppName != "go-desktop" {
		t.Fatalf("unexpected app name: %q", metadata.AppName)
	}
	if metadata.GitHubOwner != "chencn" || metadata.GitHubRepo != "go-desktop" {
		t.Fatalf("unexpected github repo: %s/%s", metadata.GitHubOwner, metadata.GitHubRepo)
	}
	if metadata.RepositoryURL != "https://github.com/chencn/go-desktop" {
		t.Fatalf("unexpected repository URL: %q", metadata.RepositoryURL)
	}
	if metadata.UserAgent != "go-desktop-updater" {
		t.Fatalf("unexpected updater user agent: %q", metadata.UserAgent)
	}
	if metadata.WindowsWindowClass != "com.github.chencn.go-desktop-window" {
		t.Fatalf("unexpected window class: %q", metadata.WindowsWindowClass)
	}
	if metadata.WindowsSingleInstanceID == "" {
		t.Fatal("single instance ID must be configured")
	}
	if metadata.DefaultUpdateCheckIntervalHours != 3 {
		t.Fatalf("unexpected update interval default: %d", metadata.DefaultUpdateCheckIntervalHours)
	}
	if !metadata.DefaultMinimizeToTray {
		t.Fatal("minimize-to-tray should default to true")
	}
	if metadata.DefaultLogRetentionDays != 30 {
		t.Fatalf("unexpected log retention default: %d", metadata.DefaultLogRetentionDays)
	}
}

// TestReleaseAssetNameUsesProjectMetadata 验证 Windows 安装包命名策略仍来自生成的 metadata helper。
func TestReleaseAssetNameUsesProjectMetadata(t *testing.T) {
	if metadata.WindowsInstallerAssetName("1.2.3") != "go-desktop-v1.2.3-windows-amd64.exe" {
		t.Fatalf("unexpected primary asset name: %q", metadata.WindowsInstallerAssetName("1.2.3"))
	}
	if metadata.WindowsSetupAssetName("1.2.3") != "go-desktop-setup-v1.2.3.exe" {
		t.Fatalf("unexpected setup asset name: %q", metadata.WindowsSetupAssetName("1.2.3"))
	}
}
