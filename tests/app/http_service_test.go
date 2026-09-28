// 文件职责：验证 /api HTTP 门面覆盖前端 ServiceBinding 的全部方法，并能真实读写 Runtime 状态。

package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/chencn/go-desktop/app"
)

var frontendServiceBindingPattern = regexp.MustCompile(`(?s)export type ServiceBinding = \{(.*?)\n\}`)

var frontendMethodNamePattern = regexp.MustCompile(`(?m)^\s*([A-Z]\w*)\?:`)

// TestHTTPFacadeCoversFrontendServiceBinding 锁住前端绑定清单与门面方法集的对应关系：
// 新增或改名 app.API 方法时，浏览器预览会缺接口，这里必须先失败。
func TestHTTPFacadeCoversFrontendServiceBinding(t *testing.T) {
	source := readRootFile(t, "frontend", "src", "api", "wails.ts")
	block := frontendServiceBindingPattern.FindStringSubmatch(source)
	if len(block) != 2 {
		t.Fatalf("frontend/src/api/wails.ts should declare an `export type ServiceBinding = {...}` block")
	}
	names := frontendMethodNamePattern.FindAllStringSubmatch(block[1], -1)
	if len(names) < 15 {
		t.Fatalf("frontend ServiceBinding parse looks wrong: got %d methods", len(names))
	}

	runtimeService := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   filepath.Join(t.TempDir(), "logs"),
	})
	defer runtimeService.Shutdown()

	facade := app.NewHTTPService(app.HTTPServiceOptions{API: runtimeService.API()})
	available := map[string]bool{}
	for _, name := range facade.Methods() {
		available[name] = true
	}
	for _, match := range names {
		if !available[match[1]] {
			t.Fatalf("前端 ServiceBinding 里的 %s 没有被 /api 门面登记（app.API 签名不受支持或已改名）", match[1])
		}
	}
}

// TestHTTPFacadeReadsAndWritesRuntime 用真实 Runtime 走一遍读取和保存，确认门面向后兼容前端契约。
func TestHTTPFacadeReadsAndWritesRuntime(t *testing.T) {
	runtimeService := app.NewRuntime(app.ServiceOptions{
		DatabasePath: filepath.Join(t.TempDir(), "go-desktop.db"),
		LogDirPath:   filepath.Join(t.TempDir(), "logs"),
	})
	defer runtimeService.Shutdown()

	facade := app.NewHTTPService(app.HTTPServiceOptions{API: runtimeService.API()})

	read := callFacade(t, facade, "GetSettings", `[]`)
	if read.status != http.StatusOK || read.envelope["ok"] != true {
		t.Fatalf("GetSettings 应成功，收到 status=%d body=%v", read.status, read.raw)
	}
	data, _ := read.envelope["data"].(map[string]any)
	settings, _ := data["updateSource"].(string)
	if settings == "" {
		t.Fatalf("GetSettings 返回缺少 updateSource: %s", read.raw)
	}
	data["updateSource"] = "local"
	body, err := json.Marshal([]any{data})
	if err != nil {
		t.Fatal(err)
	}

	saved := callFacade(t, facade, "SaveSettings", string(body))
	if saved.status != http.StatusOK {
		t.Fatalf("SaveSettings 应成功，收到 %s", saved.raw)
	}
	if got := runtimeService.SettingsSnapshot().UpdateSource; got != "local" {
		t.Fatalf("门面保存后 Runtime 状态未更新: %q", got)
	}

	unknown := callFacade(t, facade, "NotAMethod", `[]`)
	if unknown.status != http.StatusNotFound {
		t.Fatalf("未登记的方法应返回 404，收到 %d", unknown.status)
	}
}

type facadeResponse struct {
	status   int
	raw      string
	envelope map[string]any
}

func callFacade(t *testing.T, facade *app.HTTPService, method string, args string) facadeResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	facade.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/"+method, strings.NewReader(args)))
	payload := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &payload)
	return facadeResponse{status: recorder.Code, raw: recorder.Body.String(), envelope: payload}
}
