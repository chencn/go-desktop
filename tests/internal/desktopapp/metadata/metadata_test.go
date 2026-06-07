// 文件职责：提供由项目元数据生成的产品、仓库、默认设置和安装器常量。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package metadata_test

import (
	"testing"

	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

// TestMetadataDefinesStableProductDefaults 验证 提供由项目元数据生成的产品、仓库、默认设置和安装器常量 的关键行为，避免后续重构破坏既有约束。
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

// TestReleaseAssetNameUsesProjectMetadata 验证 提供由项目元数据生成的产品、仓库、默认设置和安装器常量 的关键行为，避免后续重构破坏既有约束。
func TestReleaseAssetNameUsesProjectMetadata(t *testing.T) {
	if metadata.WindowsInstallerAssetName("1.2.3") != "go-desktop-v1.2.3-windows-amd64.exe" {
		t.Fatalf("unexpected primary asset name: %q", metadata.WindowsInstallerAssetName("1.2.3"))
	}
	if metadata.WindowsSetupAssetName("1.2.3") != "go-desktop-setup-v1.2.3.exe" {
		t.Fatalf("unexpected setup asset name: %q", metadata.WindowsSetupAssetName("1.2.3"))
	}
}
