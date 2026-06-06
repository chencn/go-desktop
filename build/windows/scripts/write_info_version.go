//go:build ignore

// 文件职责：把 Windows 版本资源模板写入构建产物。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// main 是命令入口，负责解析启动上下文、装配依赖并启动核心流程。
func main() {
	version := flag.String("version", "", "version to inject")
	input := flag.String("in", "", "source info.json path")
	output := flag.String("out", "", "target info.json path")
	flag.Parse()

	if strings.TrimSpace(*version) == "" {
		exitf("version is required")
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
	fixed["file_version"] = *version

	info := objectAt(document, "info")
	lang := objectAt(info, "0000")
	lang["ProductVersion"] = *version
	lang["FileVersion"] = *version

	rendered, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		exitf("render output json: %v", err)
	}
	rendered = append(rendered, '\n')
	if err := os.WriteFile(*output, rendered, 0o644); err != nil {
		exitf("write output: %v", err)
	}
}

// objectAt 封装 把 Windows 版本资源模板写入构建产物 中的一段独立逻辑，调用方通过它复用同一业务规则。
func objectAt(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	value := map[string]any{}
	parent[key] = value
	return value
}

// exitf 封装 把 Windows 版本资源模板写入构建产物 中的一段独立逻辑，调用方通过它复用同一业务规则。
func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
