package uRand

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

// 字符集常量
const (
	// letterBytes 字母数字字符集
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	letterLen   = 62
)

// NewRandInt 返回 [0, n) 范围内的随机整数
//
// 使用 crypto/rand 生成，适用于验证码、随机密钥等安全场景
//
// 使用示例：
//
//	n := uRand.NewRandInt(100)
//	// n = 0~99 中的某个值
func NewRandInt(n int) int {
	if n <= 0 {
		return 0
	}
	max := big.NewInt(int64(n))
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		// crypto/rand 失败时回退（极少发生）
		return 0
	}
	return int(v.Int64())
}

// NewRandIntRange 返回 [min, max] 范围内的随机整数（含两端）
//
// 使用 crypto/rand 生成，适用于验证码、随机密钥等安全场景
//
// 使用示例：
//
//	n := uRand.NewRandIntRange(10, 20)
//	// n = 10~20 中的某个值
func NewRandIntRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + NewRandInt(max-min+1)
}

// NewRandString 生成指定长度的随机字母数字字符串，常用于验证码、临时 key
//
// 使用 crypto/rand 生成，密码学安全，适用于验证码场景
//
// 使用示例：
//
//	s := uRand.NewRandString(8)
//	// s = "aB3xK9mZ"（每次不同）
func NewRandString(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	max := big.NewInt(int64(letterLen))
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			b[i] = letterBytes[0]
			continue
		}
		b[i] = letterBytes[v.Int64()]
	}
	return string(b)
}

// NewRandHex 生成指定长度的随机十六进制字符串，常用于 token、nonce
//
// 使用 crypto/rand 生成，密码学安全
//
// 使用示例：
//
//	s := uRand.NewRandHex(32)
//	// s = "3f8a2b1c4e9d7f0a..."（32 个十六进制字符）
func NewRandHex(n int) string {
	if n <= 0 {
		return ""
	}
	// 每字节生成 2 个十六进制字符
	bytes := make([]byte, (n+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:n]
}
