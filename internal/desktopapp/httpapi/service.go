// Package httpapi 把 app.API 的方法集暴露成本地 HTTP 门面，供开发期浏览器预览调用。
//
// 路由约定：挂载方在进入 ServeHTTP 前剥掉 /api 前缀（Wails 由 ServiceOptions.Route 负责，
// scripts/devapi 自己负责），所以这里看到的路径就是方法名本身，例如 POST /api/GetSettings → /GetSettings。
// 请求体是按位置传参的 JSON 数组（无参写 []）；成功返回 {"ok":true,"data":...}，
// 失败返回 {"ok":false,"error":"..."} 并带 4xx/5xx 状态码，让前端能区分「后端报错」和「门面没起来」。
package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

// maxRequestBytes 限制单次请求体大小，避免异常请求把门面读进大对象。
const maxRequestBytes = 1 << 20

// errorType 用于识别方法签名里的 error 返回值。
var errorType = reflect.TypeOf((*error)(nil)).Elem()

// apiMethod 是登记好的可调用方法。
type apiMethod struct {
	// call 是绑定好接收者的方法值。
	call reflect.Value
	// argType 是唯一的入参类型；nil 表示无参方法。
	argType reflect.Type
	// returnsValue 标记方法是否返回数据值（区分 error-only 签名）。
	returnsValue bool
}

// Options 配置本地 HTTP 门面。
type Options struct {
	// Target 是待暴露的服务对象，通常是 app.Runtime.API()。
	Target any
}

// Service 是挂在 /api 下的 HTTP 入口。
type Service struct {
	// methods 以方法名为键，只登记签名受支持的导出方法。
	methods map[string]*apiMethod
}

// New 反射 Target 的导出方法集创建门面。
// 只登记「0 或 1 个入参，返回 error 或 (值, error)」的方法，其余签名直接忽略。
func New(options Options) *Service {
	service := &Service{methods: map[string]*apiMethod{}}
	value := reflect.ValueOf(options.Target)
	if !value.IsValid() {
		return service
	}
	for index := 0; index < value.NumMethod(); index++ {
		method := value.Type().Method(index)
		if !method.IsExported() {
			continue
		}
		bound := value.Method(index)
		signature := bound.Type()
		if signature.IsVariadic() || signature.NumIn() > 1 || signature.NumOut() == 0 || signature.NumOut() > 2 {
			continue
		}
		if signature.NumOut() == 0 || signature.NumOut() > 2 || signature.Out(signature.NumOut()-1) != errorType {
			continue
		}
		entry := &apiMethod{call: bound, returnsValue: signature.NumOut() == 2}
		if signature.NumIn() == 1 {
			argType := signature.In(0)
			if argType.Kind() == reflect.Slice || argType.Kind() == reflect.Map {
				// 切片/字典入参的方法（如批量接口）不在当前 API 契约里，跳过以免误调。
				continue
			}
			entry.argType = argType
		}
		service.methods[method.Name] = entry
	}
	return service
}

// Methods 返回已登记的可调用方法名，供启动日志和契约测试核对前端清单。
func (s *Service) Methods() []string {
	names := make([]string, 0, len(s.methods))
	for name := range s.methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ServeHTTP 分发单个方法名的 POST 调用。
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recovered := recover(); recovered != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("HTTP 门面异常：%v", recovered))
		}
	}()
	writeCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "本地门面只接受 POST 调用")
		return
	}

	name := strings.Trim(strings.TrimSpace(r.URL.Path), "/")
	if name == "" || strings.Contains(name, "/") {
		writeError(w, http.StatusNotFound, "接口不存在")
		return
	}
	method, ok := s.methods[name]
	if !ok {
		writeError(w, http.StatusNotFound, "接口不存在："+name)
		return
	}

	argv, err := readArguments(r.Body, method.argType)
	if err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无法解析："+err.Error())
		return
	}
	results := method.call.Call(argv)
	if failure := results[len(results)-1]; !failure.IsNil() {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("%v", failure.Interface()))
		return
	}
	if !method.returnsValue {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": results[0].Interface()})
}

// readArguments 把 JSON 数组按位置解成反射调用参数。
// 无参方法同样要求 body 为 []，让前端漏传参数在本地就能被发现。
func readArguments(body io.Reader, argType reflect.Type) ([]reflect.Value, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxRequestBytes))
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		trimmed = "[]"
	}
	var arguments []json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &arguments); err != nil {
		return nil, fmt.Errorf("请求体必须是按位置传参的 JSON 数组：%w", err)
	}
	expected := 0
	if argType != nil {
		expected = 1
	}
	if len(arguments) != expected {
		return nil, fmt.Errorf("需要 %d 个参数，收到 %d 个", expected, len(arguments))
	}
	if argType == nil {
		return nil, nil
	}
	value := reflect.New(argType)
	if err := json.Unmarshal(arguments[0], value.Interface()); err != nil {
		return nil, fmt.Errorf("参数类型不匹配：%w", err)
	}
	return []reflect.Value{value.Elem()}, nil
}

// writeJSON 写出 JSON 响应并设置 utf-8 媒体类型。
func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(`{"ok":false,"error":"响应序列化失败"}`)
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeError 写出 {"ok":false,"error":"..."}，供前端区分后端错误与门面缺失。
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": message})
}

// writeCORS 只对本地开发源放开跨域，且不允许携带凭据（门面自身不使用 cookie）。
func writeCORS(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if !isAllowedLocalOrigin(origin) {
		return
	}
	header := w.Header()
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	header.Set("Access-Control-Allow-Headers", "Content-Type,Accept")
	header.Add("Vary", "Origin")
}

// isAllowedLocalOrigin 判断 Origin 是否为本地开发主机。
func isAllowedLocalOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
