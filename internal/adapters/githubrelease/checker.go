// ============================================================================
// 文件: internal/adapters/githubrelease/checker.go
// 描述: GitHub Release 版本检查器
//
// 功能概述:
// - 检查 GitHub Release 是否有新版本
// - 支持代理、离线检测、速率限制处理
// - 解析 SHA256 校验信息（从 digest 或 .sha256 文件）
// ============================================================================

package githubrelease

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/chencn/go-desktop/internal/common/neterr"
	"github.com/chencn/go-desktop/internal/common/semver"
)

// ============================================================================
// 状态常量
// ============================================================================

const (
	// 无更新
	StatusNoUpdate = "no_update"
	// 有可用更新
	StatusUpdateAvailable = "update_available"
	// 已跳过（离线或速率限制）
	StatusIgnored = "ignored"
	// 检查出错
	StatusError = "error"

	// 跳过原因：离线
	SkipReasonOffline = "offline"
	// 跳过原因：速率限制
	SkipReasonRateLimit = "rate_limited"
	// SHA256 来源：GitHub digest
	Sha256SourceDigest = "github_digest"
	// SHA256 来源：.sha256 文件
	Sha256SourceAsset = "sha256_asset"
)

// ============================================================================
// 配置和结构体
// ============================================================================

type AssetNamePolicy func(version string) []string

// 检查器配置
type Config struct {
	// GitHub 仓库所有者
	Owner string // Owner 保存 Owner 对应的数据，供当前实体的调用方读取或持久化。
	// GitHub 仓库名称
	Repo string // Repo 保存 Repo 对应的数据，供当前实体的调用方读取或持久化。
	// 当前版本号
	CurrentVersion string // CurrentVersion 保存 CurrentVersion 对应的数据，供当前实体的调用方读取或持久化。
	// GitHub API 基础地址
	APIBase string // APIBase 保存 APIBase 对应的数据，供当前实体的调用方读取或持久化。
	// 本地静态 manifest 地址；非空时直接读取该 GitHub 兼容 JSON 数组。
	ManifestURL string
	// 更新源标识，用于向前端和日志说明本次检查来自 github 还是 local。
	Source string
	// 代理地址
	ProxyBase string // ProxyBase 保存 ProxyBase 对应的数据，供当前实体的调用方读取或持久化。
	// 请求 User-Agent
	UserAgent string // UserAgent 保存 UserAgent 对应的数据，供当前实体的调用方读取或持久化。
	// GitHub API 版本头。
	APIVersion string
	// AssetNames 返回指定版本允许匹配的安装资产名，顺序代表优先级。
	AssetNames AssetNamePolicy
	// HTTP 客户端（可选，用于测试）
	HTTPClient *http.Client // HTTPClient 保存 HTTPClient 对应的数据，供当前实体的调用方读取或持久化。
	// 当前时间函数（可选，用于测试）
	Now func() time.Time // Now 保存 Now 对应的数据，供当前实体的调用方读取或持久化。
}

// 版本检查器
type Checker struct {
	config Config       // config 保存 config 对应的数据，供当前实体的调用方读取或持久化。
	client *http.Client // client 保存 client 对应的数据，供当前实体的调用方读取或持久化。
}

// CheckResult 表示一次更新检查结果。
type CheckResult struct {
	// 更新源：github 或 local
	Source string `json:"source,omitempty"`
	// 检查状态
	Status string `json:"status"` // Status 保存 status 对应的数据，供当前实体的调用方读取或持久化。
	// 当前版本
	CurrentVersion string `json:"currentVersion"` // CurrentVersion 保存 currentVersion 对应的数据，供当前实体的调用方读取或持久化。
	// 请求 URL
	RequestURL string `json:"requestUrl,omitempty"` // RequestURL 保存 requestUrl 对应的数据，供当前实体的调用方读取或持久化。
	// HTTP 状态码
	HTTPStatus int `json:"httpStatus,omitempty"` // HTTPStatus 保存 httpStatus 对应的数据，供当前实体的调用方读取或持久化。
	// 最新版本号
	LatestVersion string `json:"latestVersion,omitempty"` // LatestVersion 保存 latestVersion 对应的数据，供当前实体的调用方读取或持久化。
	// 版本标签
	TagName string `json:"tagName,omitempty"` // TagName 保存 tagName 对应的数据，供当前实体的调用方读取或持久化。
	// 发布页面 URL
	ReleaseURL string `json:"releaseUrl,omitempty"` // ReleaseURL 保存 releaseUrl 对应的数据，供当前实体的调用方读取或持久化。
	// 发布说明
	ReleaseNotes string `json:"releaseNotes,omitempty"` // ReleaseNotes 保存 releaseNotes 对应的数据，供当前实体的调用方读取或持久化。
	// 资源名称
	AssetName string `json:"assetName,omitempty"` // AssetName 保存 assetName 对应的数据，供当前实体的调用方读取或持久化。
	// 资源大小（字节）
	AssetSizeBytes int64 `json:"assetSizeBytes,omitempty"` // AssetSizeBytes 保存 assetSizeBytes 对应的数据，供当前实体的调用方读取或持久化。
	// 资源下载 URL
	AssetDownloadURL string `json:"assetDownloadUrl,omitempty"` // AssetDownloadURL 保存 assetDownloadUrl 对应的数据，供当前实体的调用方读取或持久化。
	// SHA256 校验值
	Sha256 string `json:"sha256,omitempty"` // Sha256 保存 sha256 对应的数据，供当前实体的调用方读取或持久化。
	// SHA256 来源
	Sha256Source string `json:"sha256Source,omitempty"` // Sha256Source 保存 sha256Source 对应的数据，供当前实体的调用方读取或持久化。
	// 跳过原因
	SkipReason string `json:"skipReason,omitempty"` // SkipReason 保存 skipReason 对应的数据，供当前实体的调用方读取或持久化。
	// 错误原因
	ErrorReason string `json:"errorReason,omitempty"` // ErrorReason 保存 errorReason 对应的数据，供当前实体的调用方读取或持久化。
	// 检查时间
	CheckedAt string `json:"checkedAt"` // CheckedAt 保存 checkedAt 对应的数据，供当前实体的调用方读取或持久化。
	// 消息
	Message string `json:"message"` // Message 保存 message 对应的数据，供当前实体的调用方读取或持久化。
}

// GitHub Release API 响应结构
type githubRelease struct {
	TagName    string        `json:"tag_name"`   // TagName 保存 tag_name 对应的数据，供当前实体的调用方读取或持久化。
	Name       string        `json:"name"`       // Name 保存 name 对应的数据，供当前实体的调用方读取或持久化。
	HTMLURL    string        `json:"html_url"`   // HTMLURL 保存 html_url 对应的数据，供当前实体的调用方读取或持久化。
	Body       string        `json:"body"`       // Body 保存 body 对应的数据，供当前实体的调用方读取或持久化。
	Draft      bool          `json:"draft"`      // Draft 保存 draft 对应的数据，供当前实体的调用方读取或持久化。
	Prerelease bool          `json:"prerelease"` // Prerelease 保存 prerelease 对应的数据，供当前实体的调用方读取或持久化。
	Assets     []githubAsset `json:"assets"`     // Assets 保存 assets 对应的数据，供当前实体的调用方读取或持久化。
}

// GitHub Release 资产
type githubAsset struct {
	Name               string `json:"name"`                 // Name 保存 name 对应的数据，供当前实体的调用方读取或持久化。
	Size               int64  `json:"size"`                 // Size 保存 size 对应的数据，供当前实体的调用方读取或持久化。
	Digest             string `json:"digest"`               // Digest 保存 digest 对应的数据，供当前实体的调用方读取或持久化。
	BrowserDownloadURL string `json:"browser_download_url"` // BrowserDownloadURL 保存 browser_download_url 对应的数据，供当前实体的调用方读取或持久化。
}

// ============================================================================
// 构造函数
// ============================================================================

// 创建新版本检查器
// 参数:
//   - config: 检查器配置
//
// 返回:
//   - *Checker: 检查器实例
func NewChecker(config Config) *Checker {
	// 填充默认值
	if config.APIBase == "" {
		config.APIBase = "https://api.github.com"
	}
	if config.Source == "" {
		if strings.TrimSpace(config.ManifestURL) != "" {
			config.Source = "local"
		} else {
			config.Source = "github"
		}
	}
	if config.UserAgent == "" {
		config.UserAgent = "go-desktop-updater"
	}
	if config.APIVersion == "" {
		config.APIVersion = "2026-03-10"
	}
	if config.AssetNames == nil {
		config.AssetNames = func(string) []string { return nil }
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 6 * time.Second}
	}
	return &Checker{config: config, client: client}
}

// ============================================================================
// 公开方法
// ============================================================================

// 检查 GitHub Release
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - CheckResult: 检查结果
func (c *Checker) Check(ctx context.Context) CheckResult {
	checkedAt := c.config.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(c.config.ManifestURL) != "" {
		return c.checkManifest(ctx, checkedAt)
	}
	if strings.TrimSpace(c.config.Owner) == "" || strings.TrimSpace(c.config.Repo) == "" {
		return c.errorResult(checkedAt, "", 0, "repository_missing", "GitHub Release 仓库配置为空。")
	}
	apiURL := c.releaseAPIURL()
	requestURL := proxiedURL(apiURL, c.config.ProxyBase)

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return c.errorResult(checkedAt, requestURL, 0, "request_create_failed", fmt.Sprintf("GitHub Release 请求创建失败：%s", err))
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", c.config.APIVersion)
	req.Header.Set("User-Agent", c.config.UserAgent)

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		if neterr.IsOfflineError(err) {
			return c.ignoredOfflineResult(checkedAt, requestURL)
		}
		return c.errorResult(checkedAt, requestURL, 0, "request_failed", fmt.Sprintf("GitHub Release 请求失败：%s", err))
	}
	defer resp.Body.Close()

	// 处理速率限制
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return CheckResult{
			Source:         c.config.Source,
			Status:         StatusIgnored,
			CurrentVersion: c.config.CurrentVersion,
			RequestURL:     requestURL,
			HTTPStatus:     resp.StatusCode,
			SkipReason:     SkipReasonRateLimit,
			ErrorReason:    SkipReasonRateLimit,
			CheckedAt:      checkedAt,
			Message:        "GitHub API 暂时受限，已跳过本次更新检查。",
		}
	}
	// 处理其他 HTTP 错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.errorResult(checkedAt, requestURL, resp.StatusCode, "http_error", fmt.Sprintf("GitHub Release API 返回 HTTP %d。", resp.StatusCode))
	}

	// 读取响应体
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return c.errorResult(checkedAt, requestURL, resp.StatusCode, "response_read_failed", fmt.Sprintf("读取 GitHub Release 响应失败：%s", err))
	}

	// 解析 JSON
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return c.errorResult(checkedAt, requestURL, resp.StatusCode, "response_parse_failed", fmt.Sprintf("解析 GitHub Release 响应失败：%s", err))
	}
	return c.checkReleases(ctx, releases, checkedAt, true, requestURL, resp.StatusCode)
}

// 从静态 JSON 数据检查版本（用于测试）
func (c *Checker) CheckStatic(releasesJSON []byte) CheckResult {
	checkedAt := c.config.Now().UTC().Format(time.RFC3339)
	var releases []githubRelease
	if err := json.Unmarshal(releasesJSON, &releases); err != nil {
		return c.errorResult(checkedAt, "", 0, "response_parse_failed", fmt.Sprintf("解析 GitHub Release 响应失败：%s", err))
	}
	return c.checkReleases(context.Background(), releases, checkedAt, false, "", 0)
}

// ============================================================================
// 内部方法
// ============================================================================

// checkManifest 从本地静态 HTTP manifest 读取 GitHub 兼容 Release 数组。
func (c *Checker) checkManifest(ctx context.Context, checkedAt string) CheckResult {
	requestURL := strings.TrimSpace(c.config.ManifestURL)
	if requestURL == "" {
		return c.errorResult(checkedAt, "", 0, "manifest_missing", "本地更新 manifest 地址为空。")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return c.errorResult(checkedAt, requestURL, 0, "request_create_failed", fmt.Sprintf("本地更新 manifest 请求创建失败：%s", err))
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		if neterr.IsOfflineError(err) {
			return c.ignoredOfflineResult(checkedAt, requestURL)
		}
		return c.errorResult(checkedAt, requestURL, 0, "request_failed", fmt.Sprintf("本地更新 manifest 请求失败：%s", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.errorResult(checkedAt, requestURL, resp.StatusCode, "http_error", fmt.Sprintf("本地更新 manifest 返回 HTTP %d。", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return c.errorResult(checkedAt, requestURL, resp.StatusCode, "response_read_failed", fmt.Sprintf("读取本地更新 manifest 失败：%s", err))
	}
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return c.errorResult(checkedAt, requestURL, resp.StatusCode, "response_parse_failed", fmt.Sprintf("解析本地更新 manifest 失败：%s", err))
	}
	return c.checkReleases(ctx, releases, checkedAt, true, requestURL, resp.StatusCode)
}

// checkReleases 检查 Release 列表并返回当前检查结果。
func (c *Checker) checkReleases(ctx context.Context, releases []githubRelease, checkedAt string, resolveShaAsset bool, requestURL string, httpStatus int) CheckResult {
	// 选择最新 Release
	release, ok := selectLatestRelease(releases)
	if !ok {
		return c.errorResult(checkedAt, requestURL, httpStatus, "no_available_release", "没有找到可用的更新发布。")
	}

	latestVersion := semver.Normalize(release.TagName)
	base := CheckResult{
		Source:         c.config.Source,
		CurrentVersion: c.config.CurrentVersion,
		RequestURL:     requestURL,
		HTTPStatus:     httpStatus,
		LatestVersion:  latestVersion,
		TagName:        release.TagName,
		ReleaseURL:     release.HTMLURL,
		ReleaseNotes:   release.Body,
		CheckedAt:      checkedAt,
	}

	// 版本比较
	if semver.Compare(latestVersion, c.config.CurrentVersion) <= 0 {
		base.Status = StatusNoUpdate
		base.Message = "当前已经是最新版本。"
		return base
	}

	// 选择 Windows 资产
	asset, shaAsset, ok := selectAsset(release, c.config.AssetNames(latestVersion))
	if !ok {
		base.Status = StatusError
		base.ErrorReason = "asset_missing"
		base.Message = "发现新版本，但没有找到匹配的安装资产。"
		return base
	}
	base.Status = StatusUpdateAvailable
	base.AssetName = asset.Name
	base.AssetSizeBytes = asset.Size
	base.AssetDownloadURL = proxiedURL(asset.BrowserDownloadURL, c.config.ProxyBase)

	// 解析 SHA256
	if sha256, ok := parseDigestSha256(asset.Digest); ok {
		base.Sha256 = sha256
		base.Sha256Source = Sha256SourceDigest
		base.Message = "发现新版本，校验摘要来自 GitHub asset digest。"
		return base
	}
	if shaAsset != nil && shaAsset.BrowserDownloadURL != "" {
		if resolveShaAsset {
			sha256, err := c.fetchSha256Asset(ctx, proxiedURL(shaAsset.BrowserDownloadURL, c.config.ProxyBase))
			if err != nil {
				if neterr.IsOfflineError(err) {
					base.Status = StatusIgnored
					base.SkipReason = SkipReasonOffline
					base.ErrorReason = "sha256_asset_offline"
					base.Message = "已发现新版本，但当前无网络读取 .sha256 文件，已跳过本次更新检查。"
					return base
				}
				base.Status = StatusError
				base.ErrorReason = "sha256_asset_read_failed"
				base.Message = fmt.Sprintf("发现新版本，但读取 .sha256 文件失败：%s", err)
				return base
			}
			base.Sha256 = sha256
		}
		base.Sha256Source = Sha256SourceAsset
		base.Message = "发现新版本，校验摘要来自 .sha256 文件。"
		return base
	}

	base.Status = StatusError
	base.ErrorReason = "sha256_missing"
	base.Message = "发现新版本，但缺少 SHA256 校验信息。"
	return base
}

// 获取 .sha256 文件内容
func (c *Checker) fetchSha256Asset(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("X-GitHub-Api-Version", c.config.APIVersion)
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf(".sha256 文件返回 HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	sha256, ok := ParseSha256Text(string(body))
	if !ok {
		return "", fmt.Errorf(".sha256 文件未包含有效 SHA256")
	}
	return sha256, nil
}

// 返回 Release API URL
func (c *Checker) releaseAPIURL() string {
	base := strings.TrimRight(c.config.APIBase, "/")
	return fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", base, c.config.Owner, c.config.Repo)
}

// 返回离线跳过结果
func (c *Checker) ignoredOfflineResult(checkedAt, requestURL string) CheckResult {
	return CheckResult{
		Source:         c.config.Source,
		Status:         StatusIgnored,
		CurrentVersion: c.config.CurrentVersion,
		RequestURL:     requestURL,
		SkipReason:     SkipReasonOffline,
		ErrorReason:    SkipReasonOffline,
		CheckedAt:      checkedAt,
		Message:        "当前无网络，已跳过更新检查。",
	}
}

// 返回错误结果
func (c *Checker) errorResult(checkedAt, requestURL string, httpStatus int, reason, message string) CheckResult {
	return CheckResult{
		Source:         c.config.Source,
		Status:         StatusError,
		CurrentVersion: c.config.CurrentVersion,
		RequestURL:     requestURL,
		HTTPStatus:     httpStatus,
		ErrorReason:    reason,
		CheckedAt:      checkedAt,
		Message:        message,
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

// 选择最新 Release
func selectLatestRelease(releases []githubRelease) (githubRelease, bool) {
	candidates := make([]githubRelease, 0, len(releases))
	for _, release := range releases {
		// 跳过草稿和预发布版本
		if release.Draft || release.Prerelease {
			continue
		}
		// 跳过无效版本号
		if _, ok := semver.Parse(release.TagName); !ok {
			continue
		}
		candidates = append(candidates, release)
	}
	if len(candidates) == 0 {
		return githubRelease{}, false
	}
	// 按版本号排序
	sort.SliceStable(candidates, func(i, j int) bool {
		return semver.Compare(candidates[i].TagName, candidates[j].TagName) > 0
	})
	return candidates[0], true
}

// selectAsset 按调用方提供的资产名优先级匹配 Release 资产。
func selectAsset(release githubRelease, names []string) (githubAsset, *githubAsset, bool) {
	var selected *githubAsset
	for _, name := range names {
		if asset := findAsset(release.Assets, name); asset != nil {
			selected = asset
			break
		}
	}
	if selected == nil {
		return githubAsset{}, nil, false
	}

	// 查找对应的 .sha256 文件
	shaNames := []string{
		selected.Name + ".sha256",
		strings.TrimSuffix(selected.Name, ".exe") + ".sha256",
	}
	for _, name := range shaNames {
		if asset := findAsset(release.Assets, name); asset != nil {
			return *selected, asset, true
		}
	}
	return *selected, nil, true
}

// 查找资产
func findAsset(assets []githubAsset, name string) *githubAsset {
	for i := range assets {
		if strings.EqualFold(assets[i].Name, name) {
			return &assets[i]
		}
	}
	return nil
}

// SHA256 正则表达式
var sha256Pattern = regexp.MustCompile(`(?i)\b[0-9a-f]{64}\b`)

// sha256ExactPattern 保存 SHA256 精确匹配规则。
var sha256ExactPattern = regexp.MustCompile(`(?i)^[0-9a-f]{64}$`)

// 从 digest 解析 SHA256
func parseDigestSha256(digest string) (string, bool) {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(digest), "sha256:") {
		value := strings.TrimSpace(digest[len("sha256:"):])
		if sha256ExactPattern.MatchString(value) {
			return strings.ToLower(value), true
		}
	}
	return "", false
}

// 从文本解析 SHA256
func ParseSha256Text(text string) (string, bool) {
	value := sha256Pattern.FindString(text)
	if value == "" {
		return "", false
	}
	return strings.ToLower(value), true
}

// 返回代理后的 URL
func proxiedURL(rawURL, proxyBase string) string {
	rawURL = strings.TrimSpace(rawURL)
	proxyBase = strings.TrimRight(strings.TrimSpace(proxyBase), "/")
	if rawURL == "" || proxyBase == "" {
		return rawURL
	}
	if strings.HasPrefix(rawURL, proxyBase+"/") {
		return rawURL
	}
	return proxyBase + "/" + rawURL
}
