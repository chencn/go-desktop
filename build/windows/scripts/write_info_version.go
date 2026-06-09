//go:build ignore

// 文件职责：把已解析的应用版本写入 Windows version resource JSON。

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chencn/go-desktop/internal/common/semver"
)

// main 读取生成的 info.json 模板，更新 fixed.file_version 和 info.0000 的版本字段后写入目标文件。
// version 只接受 semver.Normalize 支持的数字版本，避免 Windows 资源里混入 tag 前缀或预发布文本。
func main() {
	version := flag.String("version", "", "version to inject")
	input := flag.String("in", "", "source info.json path")
	output := flag.String("out", "", "target info.json path")
	flag.Parse()

	if strings.TrimSpace(*version) == "" {
		exitf("version is required")
	}
	normalisedVersion := semver.Normalize(*version)
	if normalisedVersion == "" {
		exitf("invalid version: %s", *version)
	}
	if strings.TrimSpace(*input) == "" || strings.TrimSpace(*output) == "" {
		exitf("input and output paths are required")
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		exitf("read input: %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		exitf("parse input json: %v", err)
	}

	fixed := objectAt(document, "fixed")
	fixed["file_version"] = normalisedVersion

	info := objectAt(document, "info")
	lang := objectAt(info, "0000")
	lang["ProductVersion"] = normalisedVersion
	lang["FileVersion"] = normalisedVersion

	rendered, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		exitf("render output json: %v", err)
	}
	rendered = append(rendered, '\n')
	if err := os.WriteFile(*output, rendered, 0o644); err != nil {
		exitf("write output: %v", err)
	}
}

// objectAt 返回指定子对象；缺失或类型不匹配时创建空对象，让脚本可以修复最小模板。
func objectAt(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	value := map[string]any{}
	parent[key] = value
	return value
}

// exitf 输出失败原因并以非零状态退出，供构建任务捕获。
func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
