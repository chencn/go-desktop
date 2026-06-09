package machineid_test

import (
	"testing"

	"github.com/chencn/go-desktop/internal/platform/machineid"
)

func TestDeviceCodeIsStableAndShort(t *testing.T) {
	code := machineid.CodeFromParts([]string{
		" 4c4c4544-0038-5910-8058-c7c04f385633 ",
		"To Be Filled By O.E.M.",
		"board-123",
	})

	again := machineid.CodeFromParts([]string{
		"4C4C4544-0038-5910-8058-C7C04F385633",
		"board-123",
	})

	if code != again {
		t.Fatal("设备码归一化后必须稳定")
	}
	if len(code) != len("GD-XXXX-XXXX-XXXX") {
		t.Fatalf("设备码格式错误：%q", code)
	}
}

func TestCodeFromPartsRejectsEmptyFingerprint(t *testing.T) {
	code := machineid.CodeFromParts([]string{
		"",
		"To Be Filled By O.E.M.",
		"00000000-0000-0000-0000-000000000000",
	})
	if code != "" {
		t.Fatalf("expected empty code when no stable identifier exists, got %q", code)
	}
}
