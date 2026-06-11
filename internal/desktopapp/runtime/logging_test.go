package runtime

import (
	"testing"
)

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
