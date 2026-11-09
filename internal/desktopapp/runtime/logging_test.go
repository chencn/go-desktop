package runtime

import (
	"testing"
)

// TestInitRuntimeLoggerKeepsExistingLogger 验证并发或重复初始化 logger 时不会覆盖已打开的
// 文件 writer。回归背景：懒初始化被两个调用方同时触发时，后到者覆盖会泄漏先者的文件句柄。
func TestInitRuntimeLoggerKeepsExistingLogger(t *testing.T) {
	runtime := &Runtime{
		options:    ServiceOptions{AppName: "go-desktop"},
		logDirPath: t.TempDir(),
		settings:   defaultSettings(),
	}

	runtime.initRuntimeLogger()
	firstLogger := runtime.logger
	firstWriter := runtime.logWriter
	if firstLogger == nil || firstWriter == nil {
		t.Fatalf("expected first logger initialization to open file writer, logger=%v writer=%v", firstLogger, firstWriter)
	}

	runtime.initRuntimeLogger()
	if runtime.logger != firstLogger || runtime.logWriter != firstWriter {
		t.Fatal("expected repeated logger initialization to keep the existing logger and writer")
	}

	runtime.closeRuntimeLogger()
}
