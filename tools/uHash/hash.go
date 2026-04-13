package uHash

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"sort"
	"strings"
)

// BucketHash BKDR 字符串哈希，对 m 取模，常用于分桶路由/分片选择
//
// 使用示例：
//
//	bucket := uHash.BucketHash("user_123", 16)
//	// bucket = 0~15 中的某个固定值（相同输入输出相同）
func BucketHash(s string, m int) int {
	seed := 131
	hash := 0
	for i := 0; i < len(s); i++ {
		hash = hash*seed + int(s[i])
	}
	return (hash & 0x7FFFFFFF) % m
}

// MD5 计算字符串的 MD5 哈希，返回 32 位小写十六进制字符串
//
// 使用示例：
//
//	h := uHash.MD5("hello")
//	// h = "5d41402abc4b2a76b9719d911017c592"
func MD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// MD5Bytes 计算字节数组的 MD5 哈希，返回 32 位小写十六进制字符串
//
// 使用示例：
//
//	h := uHash.MD5Bytes([]byte{104, 101, 108, 108, 111})
//	// h = "5d41402abc4b2a76b9719d911017c592"
func MD5Bytes(buf []byte) string {
	h := md5.New()
	h.Write(buf)
	return hex.EncodeToString(h.Sum(nil))
}

// MD5File 计算文件的 MD5 哈希，常用于文件完整性校验
//
// 使用示例：
//
//	h, err := uHash.MD5File("/path/to/file.zip")
//	// h = "d8e8fca2dc0f896fd7cb4cb0031ba249"
func MD5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA1 计算字符串的 SHA1 哈希，返回 40 位小写十六进制字符串
//
// 使用示例：
//
//	h := uHash.SHA1("hello")
//	// h = "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
func SHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// SHA256 计算字符串的 SHA256 哈希，返回 64 位小写十六进制字符串
//
// 使用示例：
//
//	h := uHash.SHA256("hello")
//	// h = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
func SHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// HmacSHA1 计算 HMAC-SHA1 签名，返回小写十六进制字符串
//
// 使用示例：
//
//	sig := uHash.HmacSHA1("message", "secret_key")
func HmacSHA1(s, key string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(s))
	return hex.EncodeToString(mac.Sum(nil))
}

// HmacSHA256 计算 HMAC-SHA256 签名，返回小写十六进制字符串
// 常用于 API 签名校验、Webhook 验签
//
// 使用示例：
//
//	sig := uHash.HmacSHA256("timestamp=1706745600&nonce=abc", "my_secret_key")
func HmacSHA256(s, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(s))
	return hex.EncodeToString(mac.Sum(nil))
}

// Guid 生成随机 GUID 字符串（基于 crypto/rand + MD5）
// 每次调用结果唯一，不依赖时间戳
//
// 使用示例：
//
//	id := uHash.Guid()
//	// id = "550e8400e29b41d4a716446655440000"（示例）
func Guid() string {
	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return MD5(base64.URLEncoding.EncodeToString(b))
}

// SignByAscii 将 map 的 key 按 ASCII 升序排列后拼接为签名字符串，使用 HMAC-SHA256 签名
// 常用于开放平台 API 签名场景
//
// 使用示例：
//
//	params := map[string]string{"b": "2", "a": "1", "sign": "xxx"}
//	sig := uHash.SignByAscii(params, "my_secret", "sign")
//	// sig = HMAC-SHA256("a=1&b=2", "my_secret")，返回 64 位十六进制字符串
func SignByAscii(params map[string]string, secret string, exceptKeys ...string) string {
	excluded := make(map[string]struct{}, len(exceptKeys))
	for _, k := range exceptKeys {
		excluded[k] = struct{}{}
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		if _, ok := excluded[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params[k])
	}

	return HmacSHA256(sb.String(), secret)
}
