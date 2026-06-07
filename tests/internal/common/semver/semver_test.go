package semver_test

import (
	"testing"

	"github.com/chencn/go-desktop/internal/common/semver"
)

func TestNormalizePadsOneAndTwoPartVersions(t *testing.T) {
	cases := map[string]string{
		"v1":     "1.0.0",
		"1":      "1.0.0",
		"v1.2":   "1.2.0",
		"1.2":    "1.2.0",
		"v1.2.3": "1.2.3",
	}
	for input, want := range cases {
		if got := semver.Normalize(input); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeRejectsInvalidVersions(t *testing.T) {
	for _, input := range []string{"", "v", "v1.2.3.4", "v1.-2.3", "v1.x.3", "release-1.0.0"} {
		if got := semver.Normalize(input); got != "" {
			t.Fatalf("Normalize(%q) = %q, want empty invalid version", input, got)
		}
	}
}

func TestCompareUsesNormalizedThreePartVersions(t *testing.T) {
	if semver.Compare("v1.2", "1.2.0") != 0 {
		t.Fatal("expected v1.2 and 1.2.0 to compare equal")
	}
	if semver.Compare("v1.10", "1.2.9") <= 0 {
		t.Fatal("expected 1.10.0 to be greater than 1.2.9")
	}
}
