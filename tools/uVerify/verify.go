package uVerify

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// 预编译正则表达式，避免每次调用重复编译
var (
	emailRegex       = regexp.MustCompile(`^[0-9a-z][_.0-9a-z-]{0,31}@([0-9a-z][0-9a-z-]{0,30}[0-9a-z]\.){1,4}[a-z]{2,4}$`)
	mobileCNRegex    = regexp.MustCompile(`^1[3456789]\d{9}$`)
	mobileHKRegex    = regexp.MustCompile(`^(852)\d{8}$`)
	mobileHKNoRegex  = regexp.MustCompile(`^\d{8}$`)
	birthdayRegex    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	urlRegex         = regexp.MustCompile(`^(http(s?))://(.+)$`)
	idCardRegex      = regexp.MustCompile(`(^\d{15}$)|(^\d{17}([0-9]|X|x)$)`)
	dateTimeRegex    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}$`)
	uuidRegex        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	ipv4Regex        = regexp.MustCompile(`^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$`)
	hexRegex         = regexp.MustCompile(`^[0-9a-fA-F]+$`)
	creditCardRegex  = regexp.MustCompile(`^\d{13,19}$`)
	postalCodeCNRegex = regexp.MustCompile(`^\d{6}$`)
)

// ==================== 基础格式验证 ====================

// Email 校验邮箱格式
func Email(email string) bool {
	return emailRegex.MatchString(email)
}

// MobileCN 校验中国大陆手机号（1[3-9] 开头，共 11 位）
func MobileCN(mobile string) bool {
	return mobileCNRegex.MatchString(mobile)
}

// MobileHK 校验香港手机号（含 852 区号，共 11 位）
func MobileHK(mobile string) bool {
	return mobileHKRegex.MatchString(mobile)
}

// MobileHKNoCode 校验香港手机号（不含区号，共 8 位）
func MobileHKNoCode(mobile string) bool {
	return mobileHKNoRegex.MatchString(mobile)
}

// IDCard 校验中国大陆居民身份证号（15 位或 18 位）
func IDCard(id string) bool {
	return idCardRegex.MatchString(id)
}

// PostalCodeCN 校验中国大陆邮政编码（6 位数字）
func PostalCodeCN(code string) bool {
	return postalCodeCNRegex.MatchString(code)
}

// ==================== 日期时间验证 ====================

// Birthday 校验日期格式（YYYY-MM-DD）
func Birthday(birthday string) bool {
	return birthdayRegex.MatchString(birthday)
}

// DateTime 校验日期时间格式（YYYY-MM-DD HH:MM:SS）
func DateTime(dt string) bool {
	return dateTimeRegex.MatchString(dt)
}

// ==================== 网络相关验证 ====================

// URL 校验 http/https URL 格式
func URL(rawURL string) bool {
	return urlRegex.MatchString(rawURL)
}

// IPv4 校验 IPv4 地址格式
func IPv4(ip string) bool {
	return ipv4Regex.MatchString(ip)
}

// IPv6 校验 IPv6 地址格式
func IPv6(ip string) bool {
	return net.ParseIP(ip) != nil && strings.Contains(ip, ":")
}

// IP 校验 IP 地址（支持 IPv4 和 IPv6）
func IP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// Port 校验端口号（1-65535）
func Port(port string) bool {
	p, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return p >= 1 && p <= 65535
}

// PortInt 校验端口号（int 类型）
func PortInt(port int) bool {
	return port >= 1 && port <= 65535
}

// ==================== 字符类型验证 ====================

// IsNumeric 校验是否为纯数字
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsAlpha 校验是否为纯字母（a-zA-Z）
func IsAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsAlphanumeric 校验是否为字母和数字组合
func IsAlphanumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsChinese 校验是否包含中文字符
func IsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// IsAllChinese 校验是否全为中文字符
func IsAllChinese(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	return true
}

// IsLowercase 校验是否全为小写
func IsLowercase(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLower(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsUppercase 校验是否全为大写
func IsUppercase(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsHex 校验是否为十六进制字符串
func IsHex(s string) bool {
	return hexRegex.MatchString(s)
}

// IsBase64 校验是否为 Base64 编码字符串
func IsBase64(s string) bool {
	if s == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// IsUUID 校验 UUID 格式
func IsUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// IsJSON 校验是否为合法 JSON 字符串
func IsJSON(s string) bool {
	if s == "" {
		return false
	}
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// ==================== 密码强度验证 ====================

// PasswordStrong 强密码验证：8-32 位，包含大小写字母、数字和特殊字符
func PasswordStrong(password string) bool {
	if len(password) < 8 || len(password) > 32 {
		return false
	}
	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

// PasswordMedium 中等密码验证：6-32 位，包含字母和数字
func PasswordMedium(password string) bool {
	if len(password) < 6 || len(password) > 32 {
		return false
	}
	var (
		hasLetter bool
		hasDigit  bool
	)
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

// PasswordLength 密码长度验证（自定义最小和最大长度）
func PasswordLength(password string, min, max int) bool {
	length := len(password)
	return length >= min && length <= max
}

// ==================== 数值验证 ====================

// IsPositiveInt 校验正整数
func IsPositiveInt(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0
}

// IsNonNegativeInt 校验非负整数（包含 0）
func IsNonNegativeInt(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 0
}

// IsInt 校验整数（支持负数）
func IsInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// IsFloat 校验浮点数
func IsFloat(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// InRange 校验数值是否在指定范围内（包含边界）
func InRange(val int, min, max int) bool {
	return val >= min && val <= max
}

// InRangeFloat 校验浮点数是否在指定范围内（包含边界）
func InRangeFloat(val float64, min, max float64) bool {
	return val >= min && val <= max
}

// ==================== 字符串验证 ====================

// NotEmpty 校验字符串非空（去除空白后）
func NotEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

// MinLength 校验字符串最小长度
func MinLength(s string, min int) bool {
	return len([]rune(s)) >= min
}

// MaxLength 校验字符串最大长度
func MaxLength(s string, max int) bool {
	return len([]rune(s)) <= max
}

// LengthInRange 校验字符串长度在指定范围内（包含边界）
func LengthInRange(s string, min, max int) bool {
	length := len([]rune(s))
	return length >= min && length <= max
}

// Contains 校验字符串是否包含子串
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// StartsWith 校验字符串是否以指定前缀开头
func StartsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// EndsWith 校验字符串是否以指定后缀结尾
func EndsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// ==================== 其他验证 ====================

// CreditCard 校验信用卡号（13-19 位数字，简单校验）
func CreditCard(card string) bool {
	if !creditCardRegex.MatchString(card) {
		return false
	}
	// Luhn 算法校验
	return luhnCheck(card)
}

// luhnCheck Luhn 算法校验信用卡号
func luhnCheck(card string) bool {
	var sum int
	var alternate bool
	for i := len(card) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(card[i]))
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alternate = !alternate
	}
	return sum%10 == 0
}

// Regex 使用自定义正则表达式校验字符串
//
// 注意：频繁使用的正则建议预编译，此函数适合一次性校验
func Regex(pattern, s string) bool {
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

// IsNil 校验接口是否为 nil
func IsNil(i interface{}) bool {
	return i == nil
}
