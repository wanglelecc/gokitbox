package uString

import (
	"testing"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"空字符串", "", true},
		{"纯空格", "   ", true},
		{"制表符", "\t\n", true},
		{"非空", "hello", false},
		{"空格中间", " hello ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEmpty(tt.input)
			if got != tt.expected {
				t.Errorf("IsEmpty(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"空字符串", "", false},
		{"非空", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotEmpty(tt.input)
			if got != tt.expected {
				t.Errorf("IsNotEmpty(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"纯数字", "123456", true},
		{"含小数", "12.34", false},
		{"含字母", "abc123", false},
		{"空字符串", "", false},
		{"负数", "-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumeric(tt.input)
			if got != tt.expected {
				t.Errorf("IsNumeric(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		pad      string
		length   int
		expected string
	}{
		{"左填充0", "42", "0", 6, "000042"},
		{"长度超出", "hello", "-", 3, "hello"},
		{"刚好长度", "abc", "x", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadLeft(tt.s, tt.pad, tt.length)
			if got != tt.expected {
				t.Errorf("PadLeft(%q, %q, %d) = %q, want %q", tt.s, tt.pad, tt.length, got, tt.expected)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		pad      string
		length   int
		expected string
	}{
		{"右填充", "hi", "-", 6, "hi----"},
		{"长度超出", "hello", "x", 3, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadRight(tt.s, tt.pad, tt.length)
			if got != tt.expected {
				t.Errorf("PadRight(%q, %q, %d) = %q, want %q", tt.s, tt.pad, tt.length, got, tt.expected)
			}
		})
	}
}

func TestSubStr(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		start    int
		length   int
		expected string
	}{
		{"正常截取", "hello world", 0, 5, "hello"},
		{"中文截取", "你好世界", 0, 2, "你好"},
		{"起始位置", "hello world", 6, 5, "world"},
		{"越界", "hello", 10, 5, ""},
		{"负数长度", "hello", 0, 100, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubStr(tt.s, tt.start, tt.length)
			if got != tt.expected {
				t.Errorf("SubStr(%q, %d, %d) = %q, want %q", tt.s, tt.start, tt.length, got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		ellipsis string
		maxLen   int
		expected string
	}{
		{"正常截断", "hello world", "...", 5, "hello..."},
		{"中文截断", "你好世界", "…", 2, "你好…"},
		{"未超出", "hi", "...", 10, "hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.ellipsis, tt.maxLen)
			if got != tt.expected {
				t.Errorf("Truncate(%q, %q, %d) = %q, want %q", tt.s, tt.ellipsis, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"驼峰", "UserName", "user_name"},
		{"连续大写", "UserID", "user_i_d"},
		{"全大写", "HTTPSClient", "h_t_t_p_s_client"},
		{"小写", "user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSnakeCase(tt.input)
			if got != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"下划线", "user_name", "userName"},
		{"多个下划线", "user_id_name", "userIdName"},
		{"无下划线", "user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToCamelCase(tt.input)
			if got != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"下划线", "user_name", "UserName"},
		{"驼峰", "userName", "UserName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToPascalCase(tt.input)
			if got != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFirstUpper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"小写首字母", "hello", "Hello"},
		{"大写首字母", "Hello", "Hello"},
		{"数字开头", "123abc", "123abc"},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FirstUpper(tt.input)
			if got != tt.expected {
				t.Errorf("FirstUpper(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFirstLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"大写首字母", "Hello", "hello"},
		{"小写首字母", "hello", "hello"},
		{"数字开头", "123abc", "123abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FirstLower(tt.input)
			if got != tt.expected {
				t.Errorf("FirstLower(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"英文", "hello", "olleh"},
		{"中文", "你好世界", "界世好你"},
		{"空字符串", "", ""},
		{"单字符", "a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reverse(tt.input)
			if got != tt.expected {
				t.Errorf("Reverse(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRuneCount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"英文", "hello", 5},
		{"中文", "你好世界", 4},
		{"混合", "hello世界", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RuneCount(tt.input)
			if got != tt.expected {
				t.Errorf("RuneCount(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		subs     []string
		expected bool
	}{
		{"包含一个", "hello world", []string{"world", "foo"}, true},
		{"不包含", "hello world", []string{"foo", "bar"}, false},
		{"空子串", "hello", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsAny(tt.s, tt.subs...)
			if got != tt.expected {
				t.Errorf("ContainsAny(%q, %v) = %v, want %v", tt.s, tt.subs, got, tt.expected)
			}
		})
	}
}

func TestHTMLEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"script标签", "<script>alert(1)</script>", "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{"普通文本", "hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTMLEscape(tt.input)
			if got != tt.expected {
				t.Errorf("HTMLEscape(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTrimAll(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"含空格", "hello world", "helloworld"},
		{"含制表符", "a\tb\tc", "abc"},
		{"中文", "你 好 世 界", "你好世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimAll(tt.input)
			if got != tt.expected {
				t.Errorf("TrimAll(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
