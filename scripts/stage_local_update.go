//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/chencn/go-desktop/internal/common/semver"
)

type projectMetadata struct {
	AppName string `json:"appName"`
	Update  struct {
		LocalBaseURL      string `json:"localBaseUrl"`
		LocalManifestPath string `json:"localManifestPath"`
	} `json:"update"`
}

// localRelease 按 GitHub List releases 响应的最小字段集输出，供现有 GitHub updater 直接复用。
type localRelease struct {
	TagName    string       `json:"tag_name"`
	Name       string       `json:"name"`
	HTMLURL    string       `json:"html_url"`
	Body       string       `json:"body"`
	Draft      bool         `json:"draft"`
	Prerelease bool         `json:"prerelease"`
	Assets     []localAsset `json:"assets"`
}

// localAsset 描述本地静态服务中的安装器和校验文件。
// BrowserDownloadURL 必须是 updater 可以直接 GET 的绝对 URL。
type localAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest,omitempty"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// main 生成本地静态更新目录。
// 输入为已构建安装器和 project.metadata.json；输出为 releases/download/vX.Y.Z 下的安装器、sha256 和 latest.json。
func main() {
	versionFlag := flag.String("version", "", "installer version")
	installerFlag := flag.String("installer", "", "installer path")
	outFlag := flag.String("out", "", "local update staging root")
	archFlag := flag.String("arch", "amd64", "windows architecture")
	metadataFlag := flag.String("metadata", "project.metadata.json", "project metadata path")
	flag.Parse()

	meta := readProjectMetadata(*metadataFlag)
	version := semver.Normalize(*versionFlag)
	if version == "" {
		exitf("版本号无效：%s", *versionFlag)
	}
	arch := strings.TrimSpace(*archFlag)
	if arch == "" {
		arch = "amd64"
	}
	assetName := fmt.Sprintf("%s-v%s-windows-%s.exe", meta.AppName, version, arch)
	installerPath := strings.TrimSpace(*installerFlag)
	if installerPath == "" {
		installerPath = filepath.Join("bin", assetName)
	}
	stagingRoot := strings.TrimSpace(*outFlag)
	if stagingRoot == "" {
		stagingRoot = filepath.Join("bin", meta.AppName)
	}
	manifestRelPath := cleanManifestPath(meta.Update.LocalManifestPath)

	tagName := "v" + version
	downloadDir := filepath.Join(stagingRoot, "releases", "download", tagName)
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		exitf("创建本地更新目录失败：%v", err)
	}
	targetInstaller := filepath.Join(downloadDir, assetName)
	size := copyFile(installerPath, targetInstaller)
	sha := fileSHA256(targetInstaller)
	shaName := assetName + ".sha256"
	if err := os.WriteFile(filepath.Join(downloadDir, shaName), []byte(sha+"  "+assetName), 0o644); err != nil {
		exitf("写入 SHA256 文件失败：%v", err)
	}

	baseURL := strings.TrimRight(meta.Update.LocalBaseURL, "/")
	releaseURL := baseURL + "/releases/download/" + tagName + "/"
	manifest := []localRelease{{
		TagName:    tagName,
		Name:       meta.AppName + " " + tagName,
		HTMLURL:    releaseURL,
		Body:       "本地静态服务更新。",
		Draft:      false,
		Prerelease: false,
		Assets: []localAsset{
			{
				Name:               assetName,
				Size:               size,
				Digest:             "sha256:" + sha,
				BrowserDownloadURL: releaseURL + assetName,
			},
			{
				Name:               shaName,
				Size:               int64(len(sha) + 2 + len(assetName)),
				BrowserDownloadURL: releaseURL + shaName,
			},
		},
	}}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		exitf("生成 latest.json 失败：%v", err)
	}
	manifestPath := filepath.Join(stagingRoot, filepath.FromSlash(manifestRelPath))
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		exitf("创建 manifest 目录失败：%v", err)
	}
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o644); err != nil {
		exitf("写入 latest.json 失败：%v", err)
	}
	fmt.Printf("本地更新 staging 已生成：%s\n", manifestPath)
}

// readProjectMetadata 只读取 staging 必需字段；缺失任一字段都会终止，避免生成 updater 无法消费的 manifest。
func readProjectMetadata(path string) projectMetadata {
	data, err := os.ReadFile(path)
	if err != nil {
		exitf("读取 project.metadata.json 失败：%v", err)
	}
	var meta projectMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		exitf("解析 project.metadata.json 失败：%v", err)
	}
	if strings.TrimSpace(meta.AppName) == "" ||
		strings.TrimSpace(meta.Update.LocalBaseURL) == "" ||
		strings.TrimSpace(meta.Update.LocalManifestPath) == "" {
		exitf("project.metadata.json 缺少 appName、update.localBaseUrl 或 update.localManifestPath")
	}
	return meta
}

// cleanManifestPath 规范化 update.localManifestPath。
// 只允许 staging 根目录内的相对路径，拒绝绝对路径、卷名、URL 和 ../ 跳出。
func cleanManifestPath(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		exitf("update.localManifestPath 不能为空")
	}
	if filepath.IsAbs(value) {
		exitf("update.localManifestPath 必须是相对路径：%s", raw)
	}
	value = strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/")
	if strings.Contains(value, ":") {
		exitf("update.localManifestPath 不能包含卷名或 URL：%s", raw)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		exitf("update.localManifestPath 不能跳出 staging 目录：%s", raw)
	}
	return cleaned
}

// copyFile 复制安装器到 staging 目录，并返回实际写入字节数。
func copyFile(source, target string) int64 {
	input, err := os.Open(source)
	if err != nil {
		exitf("读取安装器失败：%v", err)
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		exitf("创建 staging 安装器失败：%v", err)
	}
	written, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		exitf("复制安装器失败：%v", copyErr)
	}
	if closeErr != nil {
		exitf("写入安装器失败：%v", closeErr)
	}
	return written
}

// fileSHA256 返回文件内容的十六进制小写 SHA256。
func fileSHA256(path string) string {
	file, err := os.Open(path)
	if err != nil {
		exitf("读取安装器 SHA256 失败：%v", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		exitf("计算安装器 SHA256 失败：%v", err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// exitf 输出失败原因并以非零状态退出，供 Taskfile package 链路捕获。
func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
