package uRand

import (
	"strconv"
	"strings"
)

// GenerateRandomStr 生成指定长度的随机字母数字字符串（兼容旧 API）
//
// 使用示例：
//
//	s := uRand.GenerateRandomStr(8)
//	// s = "a3F9kL2p"（随机结果，每次不同）
func GenerateRandomStr(length int) string {
	return NewRandString(length)
}

// GenerateRandom64 生成 [min, max) 范围内的随机 int64
//
// 如果 min >= max，则返回 max
//
// 使用示例：
//
//	n := uRand.GenerateRandom64(10, 100)
//	// n 为 [10, 100) 之间的随机整数
func GenerateRandom64(min, max int64) int64 {
	if min >= max {
		return max
	}
	return int64(NewRandIntRange(int(min), int(max-1)))
}

// GenerateMobileCaptcha 生成指定长度的手机验证码数字字符串
//
// 长度必须在 4 到 8 之间（含），否则返回空字符串
//
// 使用示例：
//
//	code := uRand.GenerateMobileCaptcha(6)
//	// code = "123456"（随机 6 位数字，首位非 0）
func GenerateMobileCaptcha(length int) string {
	if length < 4 || length > 8 {
		return ""
	}
	minStr := "1" + strings.Repeat("0", length-1)
	maxStr := "1" + strings.Repeat("0", length)
	min, _ := strconv.Atoi(minStr)
	max, _ := strconv.Atoi(maxStr)
	return strconv.Itoa(NewRandIntRange(min, max))
}
