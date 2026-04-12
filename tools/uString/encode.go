package uString

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

// Base64Encode 将字节数组做标准 Base64 编码
//
// 使用示例：
//
//	s := uString.Base64Encode([]byte("hello world"))
//	// s = "aGVsbG8gd29ybGQ="
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode 将标准 Base64 字符串解码为字节数组
//
// 使用示例：
//
//	b, err := uString.Base64Decode("aGVsbG8gd29ybGQ=")
//	// b = []byte("hello world")
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// URLBase64Encode 将字节数组做 URL-safe Base64 编码（将 `=` 替换为 `.`）
//
// 使用示例：
//
//	s := uString.URLBase64Encode([]byte("hello+world/test"))
//	// s = "aGVsbG8rd29ybGQvdGVzdA.."（无 +/ = 特殊字符）
func URLBase64Encode(data []byte) string {
	s := base64.URLEncoding.EncodeToString(data)
	return strings.ReplaceAll(s, "=", ".")
}

// URLBase64Decode 将 URL-safe Base64 字符串（. 替代 =）解码为字节数组
//
// 使用示例：
//
//	b, err := uString.URLBase64Decode("aGVsbG8rd29ybGQvdGVzdA..")
func URLBase64Decode(data string) ([]byte, error) {
	data = strings.ReplaceAll(data, ".", "=")
	return base64.URLEncoding.DecodeString(data)
}

// UrlEncode 对字符串做 URL Query 编码（空格编码为 +）
//
// 使用示例：
//
//	uString.UrlEncode("hello world&a=1") // "hello+world%26a%3D1"
func UrlEncode(data string) string {
	return url.QueryEscape(data)
}

// UrlDecode 对 URL Query 编码的字符串做解码
//
// 使用示例：
//
//	s, err := uString.UrlDecode("hello+world%26a%3D1") // "hello world&a=1"
func UrlDecode(data string) (string, error) {
	return url.QueryUnescape(data)
}

// RawUrlEncode 对字符串做 URL 编码，空格编码为 %20（而非 +）
//
// 使用示例：
//
//	uString.RawUrlEncode("hello world") // "hello%20world"
func RawUrlEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// JSONMarshal 将任意对象序列化为 JSON 字节数组
// 与标准 json.Marshal 的区别：禁用 HTML 转义，<、>、& 等字符不会被转义为 unicode
//
// 使用示例：
//
//	b, err := uString.JSONMarshal(map[string]string{"url": "a=1&b=2"})
//	// b = `{"url":"a=1&b=2"}`（标准库会输出 `{"url":"a=1\u0026b=2"}`）
func JSONMarshal(v any) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// json.Encoder.Encode 末尾会多一个 \n，去掉
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// JSONMarshalStr 与 JSONMarshal 相同，直接返回字符串，序列化失败返回空字符串
//
// 使用示例：
//
//	s := uString.JSONMarshalStr(map[string]int{"a": 1})
//	// s = `{"a":1}`
func JSONMarshalStr(v any) string {
	b, _ := JSONMarshal(v)
	return string(b)
}
