//go:build ignore

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/chencn/go-desktop/internal/common/semver"
)

// main 根据运行场景输出规范化版本号。
// local 模式会比较 build/config.yml、显式版本和当前 HEAD 标签；github 模式只信任显式版本或发布标签。
func main() {
	mode := flag.String("mode", "local", "version resolution mode: local or github")
	configPath := flag.String("config", "build/config.yml", "Wails build config path")
	explicitVersion := flag.String("version", "", "explicit app version")
	tag := flag.String("tag", "", "git tag version")
	flag.Parse()

	switch strings.ToLower(strings.TrimSpace(*mode)) {
	case "", "local":
		fmt.Print(resolveLocalVersion(*configPath, *explicitVersion, *tag))
	case "github":
		fmt.Print(resolveGitHubVersion(*explicitVersion, *tag))
	default:
		exitf("未知版本解析模式：%s", *mode)
	}
}

// resolveLocalVersion 选择本地打包可用的最大版本。
// 来源优先收集 build/config.yml 的 info.version、APP_VERSION、GITHUB_REF_NAME 或当前 HEAD 标签，再统一规范化比较。
func resolveLocalVersion(configPath, explicitVersion, tag string) string {
	candidates := []string{}
	if value := strings.TrimSpace(readInfoVersion(configPath)); value != "" {
		candidates = append(candidates, value)
	}
	if value := strings.TrimSpace(explicitVersion); value != "" {
		candidates = append(candidates, value)
	}
	if value := strings.TrimSpace(tag); value != "" {
		candidates = append(candidates, value)
	} else {
		candidates = append(candidates, currentHeadVersionTags()...)
	}
	if len(candidates) == 0 {
		exitf("没有可用版本来源")
	}

	selected := ""
	for _, candidate := range candidates {
		version := semver.Normalize(candidate)
		if version == "" {
			exitf("版本号无效：%s", candidate)
		}
		if selected == "" || semver.Compare(version, selected) > 0 {
			selected = version
		}
	}
	return selected
}

// currentHeadVersionTags 返回当前 HEAD 上可解析的版本标签；git 不可用或查询失败时静默返回空列表。
func currentHeadVersionTags() []string {
	cmd := exec.Command("git", "tag", "--points-at", "HEAD")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var tags []string
	for _, line := range strings.Split(string(output), "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" || semver.Normalize(tag) == "" {
			continue
		}
		tags = append(tags, tag)
	}
	return tags
}

// resolveGitHubVersion 解析发布版本；发布模式不读取 build/config.yml，避免 workflow 使用旧兜底版本。
func resolveGitHubVersion(explicitVersion, tag string) string {
	candidate := strings.TrimSpace(explicitVersion)
	if candidate == "" {
		candidate = strings.TrimSpace(tag)
	}
	if candidate == "" {
		exitf("GitHub 打包缺少 tag 版本")
	}
	version := semver.Normalize(candidate)
	if version == "" {
		exitf("版本号无效：%s", candidate)
	}
	return version
}

var infoVersionPattern = regexp.MustCompile(`^\s{2}version:\s*["']?([^"'\s#]+)`)

// readInfoVersion 从 Wails build/config.yml 的 info 块读取 version。
// 这是轻量解析器，只匹配缩进两格的 version 字段；读取失败会终止版本解析。
func readInfoVersion(path string) string {
	file, err := os.Open(path)
	if err != nil {
		exitf("读取构建配置失败：%v", err)
	}
	defer file.Close()

	inInfo := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "info:" {
			inInfo = true
			continue
		}
		if inInfo && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "  ") {
			break
		}
		if !inInfo {
			continue
		}
		matches := infoVersionPattern.FindStringSubmatch(line)
		if len(matches) == 2 {
			return strings.TrimSpace(matches[1])
		}
	}
	if err := scanner.Err(); err != nil {
		exitf("读取构建配置失败：%v", err)
	}
	return ""
}

// exitf 输出失败原因并以非零状态退出，供 Taskfile 和 release workflow 捕获。
func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
