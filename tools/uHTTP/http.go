package uHTTP

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config HTTP 客户端超时配置（单位：秒）
type Config struct {
	DialTimeout           int // TCP 连接超时，默认 10s
	DialKeepAlive         int // 长连接保活时间，默认 30s
	TLSHandshakeTimeout   int // TLS 握手超时，默认 10s
	ResponseHeaderTimeout int // 等待响应头超时，默认 30s
	ExpectContinueTimeout int // Expect: 100-continue 等待超时，默认 5s
	Timeout               int // 整个请求总超时，默认 60s
}

// DefaultConfig 返回推荐的默认超时配置
//
// 使用示例：
//
//	cfg := uHTTP.DefaultConfig()
func DefaultConfig() Config {
	return Config{
		DialTimeout:           10,
		DialKeepAlive:         30,
		TLSHandshakeTimeout:   10,
		ResponseHeaderTimeout: 30,
		ExpectContinueTimeout: 5,
		Timeout:               60,
	}
}

// newClient 根据配置创建 http.Client
func newClient(cfg Config) *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.DialTimeout) * time.Second,
			KeepAlive: time.Duration(cfg.DialKeepAlive) * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   time.Duration(cfg.TLSHandshakeTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.ResponseHeaderTimeout) * time.Second,
		ExpectContinueTimeout: time.Duration(cfg.ExpectContinueTimeout) * time.Second,
	}
	return &http.Client{
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
		Transport: transport,
	}
}

// Request 发起 HTTP 请求，支持 context 取消与超时
// 返回：HTTP 状态码、响应头、响应体、错误
//
// 使用示例：
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	status, header, body, err := uHTTP.Request(ctx, "POST", "https://api.example.com/v1/data",
//	    map[string]string{"Content-Type": "application/json"},
//	    `{"key":"value"}`,
//	    uHTTP.DefaultConfig(),
//	)
func Request(ctx context.Context, method, url string, headers map[string]string, body string, cfg Config) (statusCode int, respHeader http.Header, respBody []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	if err != nil {
		return
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := newClient(cfg).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	respHeader = resp.Header
	statusCode = resp.StatusCode
	return
}

// Get 发起 GET 请求，非 200 状态码时返回 error
//
// 使用示例：
//
//	body, header, err := uHTTP.Get(ctx, "https://api.example.com/user/1",
//	    map[string]string{"Authorization": "Bearer token"},
//	)
func Get(ctx context.Context, url string, headers map[string]string) (body []byte, respHeader http.Header, err error) {
	code, respHeader, body, err := Request(ctx, http.MethodGet, url, headers, "", DefaultConfig())
	if err != nil {
		return
	}
	if code != http.StatusOK {
		err = fmt.Errorf("GET %s returned status %d", url, code)
	}
	return
}

// Post 发起表单 POST 请求（application/x-www-form-urlencoded），非 200 时返回 error
//
// 使用示例：
//
//	body, err := uHTTP.Post(ctx, "https://api.example.com/login",
//	    map[string]string{"username": "admin", "password": "123456"},
//	    nil,
//	)
func Post(ctx context.Context, url string, params map[string]string, headers map[string]string) (body []byte, err error) {
	formData := neturl.Values{}
	for k, v := range params {
		formData.Add(k, v)
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	code, _, body, err := Request(ctx, http.MethodPost, url, headers, formData.Encode(), DefaultConfig())
	if err != nil {
		return
	}
	if code != http.StatusOK {
		err = fmt.Errorf("POST %s returned status %d", url, code)
	}
	return
}

// PostJSON 发起 JSON POST 请求，body 为任意可序列化对象，非 200 时返回 error
//
// 使用示例：
//
//	type Req struct { Name string `json:"name"` }
//	body, err := uHTTP.PostJSON(ctx, "https://api.example.com/user",
//	    Req{Name: "张三"},
//	    map[string]string{"Authorization": "Bearer token"},
//	)
func PostJSON(ctx context.Context, url string, payload any, headers map[string]string) (body []byte, err error) {
	b, err := marshalJSON(payload)
	if err != nil {
		return
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/json; charset=utf-8"

	code, _, body, err := Request(ctx, http.MethodPost, url, headers, string(b), DefaultConfig())
	if err != nil {
		return
	}
	if code != http.StatusOK {
		err = fmt.Errorf("POST %s returned status %d", url, code)
	}
	return
}

// Head 发起 HEAD 请求，返回响应头
//
// 使用示例：
//
//	header, err := uHTTP.Head(ctx, "https://example.com/file.zip")
//	size := header.Get("Content-Length")
func Head(ctx context.Context, url string) (respHeader http.Header, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return
	}
	resp, err := newClient(DefaultConfig()).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	respHeader = resp.Header
	return
}

// Upload 发起 multipart 文件上传请求，支持同时携带表单字段
// fields 为普通表单字段，files 为 map[表单字段名]本地文件路径
//
// 使用示例：
//
//	status, header, body, err := uHTTP.Upload(ctx, "https://api.example.com/upload",
//	    map[string]string{"remark": "头像"},
//	    map[string]string{"avatar": "/tmp/avatar.jpg"},
//	    nil,
//	    uHTTP.DefaultConfig(),
//	)
func Upload(ctx context.Context, url string, fields map[string]string, files map[string]string, headers map[string]string, cfg Config) (statusCode int, respHeader http.Header, respBody []byte, err error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	for fieldName, filePath := range files {
		fw, e := w.CreateFormFile(fieldName, filepath.Base(filePath))
		if e != nil {
			err = fmt.Errorf("create form file %s: %w", fieldName, e)
			return
		}
		f, e := os.Open(filePath)
		if e != nil {
			err = fmt.Errorf("open file %s: %w", filePath, e)
			return
		}
		_, e = io.Copy(fw, f)
		f.Close()
		if e != nil {
			err = fmt.Errorf("copy file %s: %w", filePath, e)
			return
		}
	}

	for k, v := range fields {
		_ = w.WriteField(k, v)
	}

	contentType := w.FormDataContentType()
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := newClient(cfg).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	respHeader = resp.Header
	statusCode = resp.StatusCode
	return
}

// BasicAuth 生成 HTTP Basic Auth 认证头的值（Base64 编码）
//
// 使用示例：
//
//	auth := uHTTP.BasicAuth("admin", "password")
//	// auth = "YWRtaW46cGFzc3dvcmQ="
//	headers["Authorization"] = "Basic " + auth
func BasicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// ParseCookie 从 Cookie 字符串中解析指定 key 的值
//
// 使用示例：
//
//	val, err := uHTTP.ParseCookie("session=abc123; user=admin", "user")
//	// val = "admin"
func ParseCookie(cookie, key string) (string, error) {
	for _, item := range strings.Split(cookie, "; ") {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("key %q not found in cookie", key)
}

// marshalJSON 内部用：序列化为 JSON，禁用 HTML 转义
func marshalJSON(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}
