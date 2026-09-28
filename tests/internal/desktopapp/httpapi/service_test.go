// 文件职责：验证 /api 门面的反射方法登记、参数解码、错误包与本地来源限制。

package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chencn/go-desktop/internal/desktopapp/httpapi"
)

// snapshot 是门面试验里被调用的假服务对象，覆盖契约中出现的三种方法签名。
type snapshot struct {
	// saved 记录最后一次 Save 收到的参数，用于验证按位置解码。
	saved map[string]any
	// failWith 非空时让 Read 返回错误，模拟后端已应答的失败。
	failWith error
}

type payload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Read 是无参、返回 (值, error) 的方法。
func (s *snapshot) Read() (map[string]any, error) {
	if s.failWith != nil {
		return nil, s.failWith
	}
	return map[string]any{"name": "go-desktop", "count": 3}, nil
}

// Save 是带单个结构体参数的方法。
func (s *snapshot) Save(value payload) (payload, error) {
	s.saved = map[string]any{"name": value.Name, "count": value.Count}
	return value, nil
}

// Reset 是只返回 error 的方法。
func (s *snapshot) Reset() error {
	return nil
}

// Bulk 的入参是切片，不属于门面契约，应当完全不被登记。
func (s *snapshot) Bulk(values []string) error {
	return errors.New("不应被调用")
}

func newTestService(target any) *httpapi.Service {
	return httpapi.New(httpapi.Options{Target: target})
}

func doJSON(t *testing.T, service *httpapi.Service, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	service.ServeHTTP(recorder, request)
	return recorder
}

func TestFacadeCallsZeroArgMethod(t *testing.T) {
	service := newTestService(&snapshot{})

	recorder := doJSON(t, service, http.MethodPost, "/Read", "[]")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		OK   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, recorder.Body.String())
	}
	if !envelope.OK || envelope.Data["name"] != "go-desktop" {
		t.Fatalf("unexpected envelope: %s", recorder.Body.String())
	}
}

func TestFacadeDecodesPositionalArgument(t *testing.T) {
	target := &snapshot{}
	service := newTestService(target)

	recorder := doJSON(t, service, http.MethodPost, "/Save", `[{"name":"settings","count":7}]`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if target.saved["name"] != "settings" || target.saved["count"] != 7 {
		t.Fatalf("Save 未收到解码后的参数: %#v", target.saved)
	}
}

func TestFacadeReportsErrorOnlyMethodAsNullData(t *testing.T) {
	service := newTestService(&snapshot{})

	recorder := doJSON(t, service, http.MethodPost, "/Reset", `[]`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"data":null`) {
		t.Fatalf("无返回值的方法应写出 data:null，收到 %s", body)
	}
}

func TestFacadeSurfacesBackendError(t *testing.T) {
	service := newTestService(&snapshot{failWith: errors.New("数据库未就绪")})

	recorder := doJSON(t, service, http.MethodPost, "/Read", "[]")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "数据库未就绪") {
		t.Fatalf("后端错误必须原样返回，收到 %s", body)
	}
}

func TestFacadeRejectsUnsupportedRequests(t *testing.T) {
	service := newTestService(&snapshot{})

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "未知方法", method: http.MethodPost, path: "/Nope", body: "[]", status: http.StatusNotFound},
		{name: "未登记的切片入参", method: http.MethodPost, path: "/Bulk", body: `[["a"]]`, status: http.StatusNotFound},
		{name: "非 POST", method: http.MethodGet, path: "/Read", body: "", status: http.StatusMethodNotAllowed},
		{name: "请求体不是数组", method: http.MethodPost, path: "/Read", body: `{"name":"x"}`, status: http.StatusBadRequest},
		{name: "参数个数不符", method: http.MethodPost, path: "/Save", body: "[]", status: http.StatusBadRequest},
		{name: "路径带额外层级", method: http.MethodPost, path: "/Read/Extra", body: "[]", status: http.StatusNotFound},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := doJSON(t, service, testCase.method, testCase.path, testCase.body)
			if recorder.Code != testCase.status {
				t.Fatalf("%s: status = %d, want %d, body = %s", testCase.name, recorder.Code, testCase.status, recorder.Body.String())
			}
		})
	}
}

func TestFacadeAllowsOnlyLocalOrigin(t *testing.T) {
	service := newTestService(&snapshot{})

	local := doJSON(t, service, http.MethodOptions, "/Read", "")
	if got := local.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("无 Origin 的请求不应得到 CORS 头，收到 %q", got)
	}

	request := httptest.NewRequest(http.MethodOptions, "/Read", nil)
	request.Header.Set("Origin", "https://evil.example")
	remote := httptest.NewRecorder()
	service.ServeHTTP(remote, request)
	if got := remote.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("非本地来源不应放行跨域，收到 %q", got)
	}

	request = httptest.NewRequest(http.MethodOptions, "/Read", nil)
	request.Header.Set("Origin", "http://127.0.0.1:9245")
	allowed := httptest.NewRecorder()
	service.ServeHTTP(allowed, request)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:9245" {
		t.Fatalf("Vite 开发源应被放行，收到 %q", got)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("门面不使用 cookie，不应允许携带凭据，收到 %q", got)
	}
}
