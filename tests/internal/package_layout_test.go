package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInternalUsesLayeredTwoLevelLayout(t *testing.T) {
	root := rootPath("internal")
	allowedFirstLevel := map[string]bool{
		"common":     true,
		"platform":   true,
		"adapters":   true,
		"desktopapp": true,
	}
	allowedSecondLevel := map[string]map[string]bool{
		"common": {
			"checksum": true,
			"neterr":   true,
			"semver":   true,
			"timefmt":  true,
		},
		"platform": {
			"installer": true,
			"paths":     true,
			"process":   true,
			"shortcut":  true,
		},
		"adapters": {
			"configstore":   true,
			"filelog":       true,
			"githubrelease": true,
		},
		"desktopapp": {
			"crash":    true,
			"display":  true,
			"logging":  true,
			"metadata": true,
			"runtime":  true,
			"settings": true,
			"update":   true,
		},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read internal: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		first := entry.Name()
		if !allowedFirstLevel[first] {
			t.Fatalf("internal first-level directory %q is not allowed; use common/platform/adapters/desktopapp", first)
		}
		children, err := os.ReadDir(filepath.Join(root, first))
		if err != nil {
			t.Fatalf("read internal/%s: %v", first, err)
		}
		for _, child := range children {
			if child.IsDir() && !allowedSecondLevel[first][child.Name()] {
				t.Fatalf("internal/%s/%s is not an allowed functional package", first, child.Name())
			}
		}
	}
}

func TestInternalDependencyDirection(t *testing.T) {
	files := goFiles(t, rootPath("internal"))
	for _, file := range files {
		source := readFile(t, file)
		rel := repoRelativePath(t, file)
		layer := internalLayer(rel)
		for _, imported := range internalImports(source) {
			importLayer := importInternalLayer(imported)
			if layerRank(importLayer) > layerRank(layer) {
				t.Fatalf("%s imports higher-level package %s; dependency direction must stay common <- platform <- adapters <- desktopapp", rel, imported)
			}
			if isOldFlatInternalImport(imported) {
				t.Fatalf("%s imports old flat internal package %s", rel, imported)
			}
		}
	}
}

func TestCommonPackagesDoNotUseProjectOrExternalSideEffects(t *testing.T) {
	files := goFiles(t, rootPath("internal", "common"))
	for _, file := range files {
		source := readFile(t, file)
		for _, forbidden := range []string{
			"github.com/chencn/go-desktop/internal/",
			"github.com/wailsapp/wails",
			"modernc.org/sqlite",
			"github.com/go-ole/go-ole",
			"os/exec",
		} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s common package must stay pure; found %q", repoRelativePath(t, file), forbidden)
			}
		}
	}
}

func goFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return files
}

func internalImports(source string) []string {
	var imports []string
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"github.com/chencn/go-desktop/internal/`) {
			imports = append(imports, strings.Trim(line, `"`))
		}
	}
	return imports
}

func internalLayer(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 || parts[0] != "internal" {
		return ""
	}
	return parts[1]
}

func importInternalLayer(importPath string) string {
	const marker = "/internal/"
	index := strings.Index(importPath, marker)
	if index < 0 {
		return ""
	}
	rest := importPath[index+len(marker):]
	return strings.Split(rest, "/")[0]
}

func layerRank(layer string) int {
	switch layer {
	case "common":
		return 1
	case "platform":
		return 2
	case "adapters":
		return 3
	case "desktopapp":
		return 4
	default:
		return 0
	}
}

func isOldFlatInternalImport(importPath string) bool {
	const marker = "/internal/"
	index := strings.Index(importPath, marker)
	if index < 0 {
		return false
	}
	rest := importPath[index+len(marker):]
	first := strings.Split(rest, "/")[0]
	switch first {
	case "desktop", "githubrelease", "netutil", "project", "storage", "update":
		return true
	default:
		return false
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func repoRelativePath(t *testing.T, path string) string {
	t.Helper()
	rel, err := filepath.Rel(rootPath(), path)
	if err != nil {
		t.Fatalf("relative path for %s: %v", path, err)
	}
	return filepath.ToSlash(rel)
}

func rootPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}
