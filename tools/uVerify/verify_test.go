package uVerify

import (
	"testing"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"有效邮箱", "test@example.com", true},
		{"有效邮箱-子域", "test@mail.example.com", true},
		{"无效邮箱-无@", "testexample.com", false},
		{"无效邮箱-无域", "test@", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Email(tt.email)
			if got != tt.expected {
				t.Errorf("Email(%q) = %v, want %v", tt.email, got, tt.expected)
			}
		})
	}
}

func TestMobileCN(t *testing.T) {
	tests := []struct {
		name     string
		mobile   string
		expected bool
	}{
		{"有效手机号-13x", "13800138000", true},
		{"有效手机号-15x", "15800158000", true},
		{"有效手机号-18x", "18800188000", true},
		{"有效手机号-19x", "19800198000", true},
		{"无效手机号-短", "1380013800", false},
		{"无效手机号-长", "138001380000", false},
		{"无效手机号-非1开头", "23800138000", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MobileCN(tt.mobile)
			if got != tt.expected {
				t.Errorf("MobileCN(%q) = %v, want %v", tt.mobile, got, tt.expected)
			}
		})
	}
}

func TestMobileHK(t *testing.T) {
	tests := []struct {
		name     string
		mobile   string
		expected bool
	}{
		{"有效手机号", "85298765432", true},
		{"无效手机号-大陆号", "13800138000", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MobileHK(tt.mobile)
			if got != tt.expected {
				t.Errorf("MobileHK(%q) = %v, want %v", tt.mobile, got, tt.expected)
			}
		})
	}
}

func TestMobileHKNoCode(t *testing.T) {
	tests := []struct {
		name     string
		mobile   string
		expected bool
	}{
		{"有效手机号", "98765432", true},
		{"无效手机号-短", "1234567", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MobileHKNoCode(tt.mobile)
			if got != tt.expected {
				t.Errorf("MobileHKNoCode(%q) = %v, want %v", tt.mobile, got, tt.expected)
			}
		})
	}
}

func TestBirthday(t *testing.T) {
	tests := []struct {
		name     string
		birthday string
		expected bool
	}{
		{"有效生日", "1990-01-15", true},
		{"无效格式-斜杠", "1990/01/15", false},
		{"无效格式-无分隔", "19900115", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Birthday(tt.birthday)
			if got != tt.expected {
				t.Errorf("Birthday(%q) = %v, want %v", tt.birthday, got, tt.expected)
			}
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"有效URL-http", "http://example.com", true},
		{"有效URL-https", "https://example.com", true},
		{"有效URL-带路径", "https://example.com/path?q=1", true},
		{"无效URL-无协议", "example.com", false},
		{"无效URL-ftp", "ftp://example.com", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := URL(tt.url)
			if got != tt.expected {
				t.Errorf("URL(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestIDCard(t *testing.T) {
	tests := []struct {
		name     string
		idcard   string
		expected bool
	}{
		{"有效身份证-18位", "110101199001011234", true},
		{"有效身份证-带X", "11010119900101121X", true},
		{"无效身份证-短", "11010119900101123", false},
		{"无效身份证-长", "1101011990010112345", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IDCard(tt.idcard)
			if got != tt.expected {
				t.Errorf("IDCard(%q) = %v, want %v", tt.idcard, got, tt.expected)
			}
		})
	}
}

func TestRegex(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		s        string
		expected bool
	}{
		{"匹配", `^\d{6}$`, "123456", true},
		{"不匹配", `^[a-z]+$`, "ABC", false},
		{"无效正则", `[`, "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Regex(tt.pattern, tt.s)
			if got != tt.expected {
				t.Errorf("Regex(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.expected)
			}
		})
	}
}

// ==================== 网络相关测试 ====================

func TestIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"有效IP", "192.168.1.1", true},
		{"有效IP-255", "255.255.255.255", true},
		{"有效IP-0", "0.0.0.0", true},
		{"无效IP-256", "256.1.1.1", false},
		{"无效IP-四段", "192.168.1", false},
		{"无效IP-五段", "192.168.1.1.1", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IPv4(tt.ip)
			if got != tt.expected {
				t.Errorf("IPv4(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"有效IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"有效IPv6-简写", "::1", true},
		{"IPv4不是IPv6", "192.168.1.1", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IPv6(tt.ip)
			if got != tt.expected {
				t.Errorf("IPv6(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"有效IPv4", "192.168.1.1", true},
		{"有效IPv6", "::1", true},
		{"无效IP", "invalid", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IP(tt.ip)
			if got != tt.expected {
				t.Errorf("IP(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestPort(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		expected bool
	}{
		{"有效端口-80", "80", true},
		{"有效端口-8080", "8080", true},
		{"有效端口-65535", "65535", true},
		{"无效端口-0", "0", false},
		{"无效端口-65536", "65536", false},
		{"无效端口-负数", "-1", false},
		{"无效端口-非数字", "abc", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Port(tt.port)
			if got != tt.expected {
				t.Errorf("Port(%q) = %v, want %v", tt.port, got, tt.expected)
			}
		})
	}
}

func TestPortInt(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected bool
	}{
		{"有效端口-80", 80, true},
		{"有效端口-8080", 8080, true},
		{"有效端口-65535", 65535, true},
		{"无效端口-0", 0, false},
		{"无效端口-65536", 65536, false},
		{"无效端口-负数", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PortInt(tt.port)
			if got != tt.expected {
				t.Errorf("PortInt(%d) = %v, want %v", tt.port, got, tt.expected)
			}
		})
	}
}

// ==================== 日期时间测试 ====================

func TestDateTime(t *testing.T) {
	tests := []struct {
		name     string
		dt       string
		expected bool
	}{
		{"有效时间", "2024-01-15 10:30:00", true},
		{"无效格式-无秒", "2024-01-15 10:30", false},
		{"无效格式-斜杠", "2024/01/15 10:30:00", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DateTime(tt.dt)
			if got != tt.expected {
				t.Errorf("DateTime(%q) = %v, want %v", tt.dt, got, tt.expected)
			}
		})
	}
}

// ==================== 字符类型测试 ====================

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"纯数字", "123456", true},
		{"含字母", "123abc", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumeric(tt.s)
			if got != tt.expected {
				t.Errorf("IsNumeric(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsAlpha(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"纯字母", "abcdef", true},
		{"含数字", "abc123", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAlpha(tt.s)
			if got != tt.expected {
				t.Errorf("IsAlpha(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"字母数字", "abc123", true},
		{"含特殊字符", "abc-123", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAlphanumeric(tt.s)
			if got != tt.expected {
				t.Errorf("IsAlphanumeric(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsChinese(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"纯中文", "你好世界", true},
		{"含中文", "hello世界", true},
		{"无中文", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsChinese(tt.s)
			if got != tt.expected {
				t.Errorf("IsChinese(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsLowercase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"全小写", "abcdef", true},
		{"含大写", "Abcdef", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLowercase(tt.s)
			if got != tt.expected {
				t.Errorf("IsLowercase(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsUppercase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"全大写", "ABCDEF", true},
		{"含小写", "Abcdef", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUppercase(tt.s)
			if got != tt.expected {
				t.Errorf("IsUppercase(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsHex(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"有效十六进制", "abc123def", true},
		{"有效十六进制-大写", "ABC123DEF", true},
		{"无效-含G", "abcg123", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHex(tt.s)
			if got != tt.expected {
				t.Errorf("IsHex(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsBase64(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"有效Base64", "SGVsbG8gV29ybGQ=", true},
		{"无效Base64", "SGVsbG8gV29ybGQ", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBase64(tt.s)
			if got != tt.expected {
				t.Errorf("IsBase64(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"有效UUID", "550e8400-e29b-41d4-a716-446655440000", true},
		{"无效格式", "550e8400e29b41d4a716446655440000", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUUID(tt.s)
			if got != tt.expected {
				t.Errorf("IsUUID(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"有效JSON对象", `{"name":"test"}`, true},
		{"有效JSON数组", `[1,2,3]`, true},
		{"无效JSON", `{"name":}`, false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsJSON(tt.s)
			if got != tt.expected {
				t.Errorf("IsJSON(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

// ==================== 密码强度测试 ====================

func TestPasswordStrong(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"强密码", "Abc123!@#", true},
		{"缺特殊字符", "Abc123def", false},
		{"缺大写", "abc123!@#", false},
		{"缺小写", "ABC123!@#", false},
		{"缺数字", "Abcdef!@#", false},
		{"太短", "Ab1!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PasswordStrong(tt.password)
			if got != tt.expected {
				t.Errorf("PasswordStrong(%q) = %v, want %v", tt.password, got, tt.expected)
			}
		})
	}
}

func TestPasswordMedium(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"中等密码", "Abc123", true},
		{"缺字母", "123456", false},
		{"缺数字", "abcdef", false},
		{"太短", "Ab1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PasswordMedium(tt.password)
			if got != tt.expected {
				t.Errorf("PasswordMedium(%q) = %v, want %v", tt.password, got, tt.expected)
			}
		})
	}
}

func TestPasswordLength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		min      int
		max      int
		expected bool
	}{
		{"在范围内", "abcdef", 4, 10, true},
		{"太短", "abc", 4, 10, false},
		{"太长", "abcdefghijk", 4, 10, false},
		{"边界-最小", "abcd", 4, 10, true},
		{"边界-最大", "abcdefghij", 4, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PasswordLength(tt.password, tt.min, tt.max)
			if got != tt.expected {
				t.Errorf("PasswordLength(%q, %d, %d) = %v, want %v", tt.password, tt.min, tt.max, got, tt.expected)
			}
		})
	}
}

// ==================== 数值验证测试 ====================

func TestIsPositiveInt(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"正整数", "123", true},
		{"零", "0", false},
		{"负数", "-1", false},
		{"非数字", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPositiveInt(tt.s)
			if got != tt.expected {
				t.Errorf("IsPositiveInt(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestInRange(t *testing.T) {
	tests := []struct {
		name     string
		val      int
		min      int
		max      int
		expected bool
	}{
		{"在范围内", 5, 1, 10, true},
		{"边界-最小", 1, 1, 10, true},
		{"边界-最大", 10, 1, 10, true},
		{"小于最小", 0, 1, 10, false},
		{"大于最大", 11, 1, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InRange(tt.val, tt.min, tt.max)
			if got != tt.expected {
				t.Errorf("InRange(%d, %d, %d) = %v, want %v", tt.val, tt.min, tt.max, got, tt.expected)
			}
		})
	}
}

// ==================== 字符串验证测试 ====================

func TestNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"非空", "hello", true},
		{"仅空格", "   ", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NotEmpty(tt.s)
			if got != tt.expected {
				t.Errorf("NotEmpty(%q) = %v, want %v", tt.s, got, tt.expected)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		min      int
		expected bool
	}{
		{"满足最小", "hello", 3, true},
		{"不满足", "hi", 3, false},
		{"刚好满足", "abc", 3, true},
		{"中文字符", "你好", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MinLength(tt.s, tt.min)
			if got != tt.expected {
				t.Errorf("MinLength(%q, %d) = %v, want %v", tt.s, tt.min, got, tt.expected)
			}
		})
	}
}

func TestStartsWith(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefix   string
		expected bool
	}{
		{"匹配前缀", "hello world", "hello", true},
		{"不匹配", "hello world", "world", false},
		{"空字符串", "", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StartsWith(tt.s, tt.prefix)
			if got != tt.expected {
				t.Errorf("StartsWith(%q, %q) = %v, want %v", tt.s, tt.prefix, got, tt.expected)
			}
		})
	}
}

func TestEndsWith(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		suffix   string
		expected bool
	}{
		{"匹配后缀", "hello world", "world", true},
		{"不匹配", "hello world", "hello", false},
		{"空字符串", "", "world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EndsWith(tt.s, tt.suffix)
			if got != tt.expected {
				t.Errorf("EndsWith(%q, %q) = %v, want %v", tt.s, tt.suffix, got, tt.expected)
			}
		})
	}
}

// ==================== 其他测试 ====================

func TestPostalCodeCN(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"有效邮编", "100000", true},
		{"无效-短", "10000", false},
		{"无效-长", "1000000", false},
		{"无效-含字母", "10000a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PostalCodeCN(tt.code)
			if got != tt.expected {
				t.Errorf("PostalCodeCN(%q) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestCreditCard(t *testing.T) {
	tests := []struct {
		name     string
		card     string
		expected bool
	}{
		{"有效信用卡号", "4532015112830366", true},
		{"无效-位数不足", "123456789", false},
		{"无效-Luhn校验失败", "1234567890123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreditCard(tt.card)
			if got != tt.expected {
				t.Errorf("CreditCard(%q) = %v, want %v", tt.card, got, tt.expected)
			}
		})
	}
}
