package uString

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// MobileMask 手机号脱敏，中间 4 位替换为 ****
// 支持 11 位（大陆）和 12 位（含区号）格式，其他格式返回空字符串
//
// 使用示例：
//
//	uString.MobileMask("13812345678")  // "138****5678"
//	uString.MobileMask("85213812345678") // "8521****5678" （不太对，应该是12位）
//	uString.MobileMask("138123456789") // "138****6789"（12位）
func MobileMask(mobile string) string {
	switch len(mobile) {
	case 11:
		return mobile[:3] + strings.Repeat("*", 4) + mobile[7:]
	case 12:
		return mobile[:4] + strings.Repeat("*", 4) + mobile[8:]
	}
	return ""
}

// RealNameMask 真实姓名脱敏：2字显示首字 + *；3字及以上显示首尾字，中间全为 *
//
// 使用示例：
//
//	uString.RealNameMask("张三")   // "张*"
//	uString.RealNameMask("张三丰")  // "张*丰"
//	uString.RealNameMask("欧阳修远") // "欧**远"
func RealNameMask(name string) string {
	runes := []rune(name)
	n := len(runes)
	switch {
	case n <= 1:
		return name
	case n == 2:
		return string(runes[0:1]) + "*"
	default:
		stars := strings.Repeat("*", n-2)
		return string(runes[0:1]) + stars + string(runes[n-1:])
	}
}

// NicknameMask 昵称脱敏：保留首尾字符，中间替换为 *
//
// 使用示例：
//
//	uString.NicknameMask("小王")      // "小*"
//	uString.NicknameMask("超级用户名") // "超***名"
//	uString.NicknameMask("a")        // "a*"
func NicknameMask(nickname string) string {
	runes := []rune(nickname)
	n := len(runes)
	switch {
	case n == 0:
		return nickname
	case n < 2:
		return string(runes[0:1]) + "*"
	case n <= 4:
		return string(runes[0:1]) + strings.Repeat("*", n-2) + string(runes[n-1:])
	default:
		return string(runes[0:1]) + strings.Repeat("*", n-3) + string(runes[n-2:])
	}
}

// SecretKeyMask 密钥/Token 脱敏：超过 9 位时显示前 3 后 3，中间替换为 ***
//
// 使用示例：
//
//	uString.SecretKeyMask("sk-abcdefghijk") // "sk-***ijk"
//	uString.SecretKeyMask("abc")            // "abc"（不足 9 位原样返回）
func SecretKeyMask(key string) string {
	if len(key) > 9 {
		return key[:3] + "***" + key[len(key)-3:]
	}
	return key
}

// CentsToYuan 将分（int64）转换为元字符串，保留两位小数
//
// 使用示例：
//
//	uString.CentsToYuan(9900) // "99.00"
//	uString.CentsToYuan(100)  // "1.00"
//	uString.CentsToYuan(1)    // "0.01"
func CentsToYuan(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

// CentsToYuanHuman 将分（int64）转换为带千位分隔符的元字符串
//
// 使用示例：
//
//	uString.CentsToYuanHuman(123456789) // "1,234,567.89"
//	uString.CentsToYuanHuman(9900)      // "99.00"
func CentsToYuanHuman(cents int64) string {
	s := CentsToYuan(cents)
	parts := strings.Split(s, ".")
	intPart := parts[0]
	decPart := parts[1]

	// 负数处理
	neg := strings.HasPrefix(intPart, "-")
	if neg {
		intPart = intPart[1:]
	}

	// 添加千位分隔符
	n := len(intPart)
	for i := n - 3; i > 0; i -= 3 {
		intPart = intPart[:i] + "," + intPart[i:]
	}

	if neg {
		return "-" + intPart + "." + decPart
	}
	return intPart + "." + decPart
}

// YuanToCents 将元字符串转换为分（int64），字符串解析避免浮点精度丢失
// 支持整数、一/两位小数、千位分隔符、负数；小数超过 2 位时返回错误
//
// 使用示例：
//
//	cents, err := uString.YuanToCents("99.99")   // 9999, nil
//	cents, err := uString.YuanToCents("-1.5")    // -150, nil
//	cents, err := uString.YuanToCents("1,234")   // 123400, nil
//	cents, err := uString.YuanToCents("99.999")  // 0, error（小数位超过 2 位）
func YuanToCents(yuan string) (int64, error) {
	yuan = strings.ReplaceAll(yuan, ",", "")
	yuan = strings.TrimSpace(yuan)
	if yuan == "" {
		return 0, fmt.Errorf("YuanToCents: 输入不能为空")
	}

	neg := false
	if strings.HasPrefix(yuan, "-") {
		neg = true
		yuan = yuan[1:]
	}

	parts := strings.SplitN(yuan, ".", 2)
	intVal, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("YuanToCents: 整数部分非法 %q", parts[0])
	}

	var decVal int64
	if len(parts) == 2 {
		dec := parts[1]
		if len(dec) > 2 {
			return 0, fmt.Errorf("YuanToCents: 小数位超过 2 位 %q", parts[1])
		}
		if len(dec) == 1 {
			dec += "0"
		}
		decVal, err = strconv.ParseInt(dec, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("YuanToCents: 小数部分非法 %q", parts[1])
		}
	}

	result := intVal*100 + decVal
	if neg {
		result = -result
	}
	return result, nil
}

// GenerateCaptcha 生成指定位数的纯数字验证码（length 范围 4~8）
// 不符合范围时返回空字符串
//
// 使用 crypto/rand 生成，密码学安全，适用于验证码场景
//
// 使用示例：
//
//	uString.GenerateCaptcha(6) // "385024"（6位随机数字）
//	uString.GenerateCaptcha(4) // "7193"
func GenerateCaptcha(length int) string {
	if length < 4 || length > 8 {
		return ""
	}
	// 生成 length 位的随机数
	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return ""
	}
	// 格式化为指定位数，不足补前导零
	format := "%0" + strconv.Itoa(length) + "d"
	return fmt.Sprintf(format, n.Int64())
}
