package uString

import (
	"html/template"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// IsEmpty 判断字符串是否为空（含纯空白字符）
//
// 使用示例：
//
//	uString.IsEmpty("")       // true
//	uString.IsEmpty("  \t")  // true
//	uString.IsEmpty("hello") // false
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotEmpty 判断字符串是否不为空
//
// 使用示例：
//
//	uString.IsNotEmpty("hello") // true
//	uString.IsNotEmpty("  ")   // false
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// IsNumeric 判断字符串是否全部由数字组成
//
// 使用示例：
//
//	uString.IsNumeric("123456") // true
//	uString.IsNumeric("12.34")  // false
//	uString.IsNumeric("abc")    // false
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	reg := regexp.MustCompile(`^[0-9]+$`)
	return reg.MatchString(s)
}

// ContainsDigit 判断字符串是否包含至少一个数字
//
// 使用示例：
//
//	uString.ContainsDigit("abc123") // true
//	uString.ContainsDigit("abc")    // false
func ContainsDigit(s string) bool {
	reg := regexp.MustCompile(`\d`)
	return reg.MatchString(s)
}

// PadLeft 左填充字符串到指定长度，超出目标长度则原样返回
//
// 使用示例：
//
//	uString.PadLeft("42", "0", 6)    // "000042"
//	uString.PadLeft("hello", "-", 3) // "hello"（已超出不截断）
func PadLeft(s, pad string, length int) string {
	for len([]rune(s)) < length {
		s = pad + s
	}
	return s
}

// PadRight 右填充字符串到指定长度，超出目标长度则原样返回
//
// 使用示例：
//
//	uString.PadRight("hi", "-", 6)    // "hi----"
//	uString.PadRight("hello", "x", 3) // "hello"（已超出不截断）
func PadRight(s, pad string, length int) string {
	for len([]rune(s)) < length {
		s = s + pad
	}
	return s
}

// SubStr 按 rune 安全截取字符串（支持中文），start 为起始位置，length 为截取长度，越界自动截断
//
// 使用示例：
//
//	uString.SubStr("hello world", 0, 5) // "hello"
//	uString.SubStr("你好世界", 0, 2)      // "你好"
//	uString.SubStr("hello", 10, 5)      // ""（start 越界返回空）
func SubStr(s string, start, length int) string {
	runes := []rune(s)
	total := len(runes)
	if start >= total {
		return ""
	}
	end := start + length
	if end > total {
		end = total
	}
	return string(runes[start:end])
}

// SubStrByPos 按 rune 安全截取字符串（支持中文），sPos 为起始位置，ePos 为结束位置（不含）
//
// 使用示例：
//
//	uString.SubStrByPos("hello world", 0, 5) // "hello"
//	uString.SubStrByPos("你好世界", 1, 3)      // "好世"
func SubStrByPos(s string, sPos, ePos int) string {
	runes := []rune(s)
	total := len(runes)
	if sPos >= total {
		return ""
	}
	if ePos > total {
		ePos = total
	}
	return string(runes[sPos:ePos])
}

// Truncate 截断字符串到指定长度（按 rune），超出时在末尾追加省略符
//
// 使用示例：
//
//	uString.Truncate("hello world", 5, "...") // "hello..."
//	uString.Truncate("你好世界", 2, "…")        // "你好…"
//	uString.Truncate("hi", 10, "...")         // "hi"（未超出不追加省略符）
func Truncate(s, ellipsis string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + ellipsis
}

// ContainsAny 判断字符串是否包含任意一个子串（短路求值）
//
// 使用示例：
//
//	uString.ContainsAny("hello world", "world", "go") // true
//	uString.ContainsAny("hello world", "foo", "bar")  // false
func ContainsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// StrReplace 批量替换字符串中的多个目标为同一值
//
// 使用示例：
//
//	uString.StrReplace("hello world", []string{"hello", "world"}, "go")
//	// "go go"
func StrReplace(s string, searches []string, replace string) string {
	for _, search := range searches {
		s = strings.ReplaceAll(s, search, replace)
	}
	return s
}

// FirstUpper 将字符串首字母转为大写，非字母开头则原样返回
//
// 使用示例：
//
//	uString.FirstUpper("hello") // "Hello"
//	uString.FirstUpper("world") // "World"
//	uString.FirstUpper("123")   // "123"
func FirstUpper(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] -= 32
		return string(runes)
	}
	return s
}

// FirstLower 将字符串首字母转为小写，非字母开头则原样返回
//
// 使用示例：
//
//	uString.FirstLower("Hello") // "hello"
//	uString.FirstLower("World") // "world"
//	uString.FirstLower("123")   // "123"
func FirstLower(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	if runes[0] >= 'A' && runes[0] <= 'Z' {
		runes[0] += 32
		return string(runes)
	}
	return s
}

// ToSnakeCase 驼峰命名转下划线命名（大写字母前插入下划线并转小写）
//
// 使用示例：
//
//	uString.ToSnakeCase("UserName")    // "user_name"
//	uString.ToSnakeCase("userID")      // "user_i_d"
//	uString.ToSnakeCase("HTTPSClient") // "h_t_t_p_s_client"
func ToSnakeCase(s string) string {
	var sb strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				sb.WriteByte('_')
			}
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ToCamelCase 下划线命名转小驼峰命名
//
// 使用示例：
//
//	uString.ToCamelCase("user_name")   // "userName"
//	uString.ToCamelCase("user_id")     // "userId"
//	uString.ToCamelCase("hello_world") // "helloWorld"
func ToCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// ToPascalCase 下划线命名转大驼峰（Pascal）命名
//
// 使用示例：
//
//	uString.ToPascalCase("user_name") // "UserName"
//	uString.ToPascalCase("order_id")  // "OrderId"
func ToPascalCase(s string) string {
	return FirstUpper(ToCamelCase(s))
}

// HTMLEscape 对字符串做 HTML 转义（防 XSS）
//
// 使用示例：
//
//	uString.HTMLEscape("<script>alert(1)</script>")
//	// "&lt;script&gt;alert(1)&lt;/script&gt;"
func HTMLEscape(s string) string {
	return template.HTMLEscapeString(s)
}

// AddSlashes 在字符串中的单引号、双引号、反斜杠前添加反斜杠
//
// 使用示例：
//
//	uString.AddSlashes(`it's a "test"`) // `it\'s a \"test\"`
func AddSlashes(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "'", `\'`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// StripSlashes 移除字符串中 AddSlashes 添加的反斜杠
//
// 使用示例：
//
//	uString.StripSlashes(`it\'s a \"test\"`) // `it's a "test"`
func StripSlashes(s string) string {
	s = strings.ReplaceAll(s, `\'`, "'")
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// ExtractURLs 从字符串中提取所有 http/https 链接
//
// 使用示例：
//
//	urls, err := uString.ExtractURLs("visit https://example.com and http://test.org")
//	// urls = ["https://example.com", "http://test.org"]
func ExtractURLs(s string) ([]string, error) {
	re := regexp.MustCompile(`https?://\S+`)
	matches := re.FindAllString(s, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	return matches, nil
}

// SplitByLength 将字符串按字节长度分割为固定大小的片段
//
// 使用示例：
//
//	uString.SplitByLength("abcdefg", 3) // ["abc", "def", "g"]
func SplitByLength(s string, n int) []string {
	if n <= 0 {
		return nil
	}
	b := []byte(s)
	result := make([]string, 0, (len(b)+n-1)/n)
	for len(b) > 0 {
		size := n
		if len(b) < size {
			size = len(b)
		}
		result = append(result, string(b[:size]))
		b = b[size:]
	}
	return result
}

// JoinMapByAscii 将 map 按 key 的 ASCII 码升序排列后拼接
// sep 为键值对之间的分隔符，onlyValues 为 true 时只拼接 value，includeEmpty 控制是否包含空值，exceptKeys 为排除的 key
//
// 使用示例：
//
//	// 生成 API 签名字符串
//	data := map[string]string{"b": "2", "a": "1", "c": ""}
//	s := uString.JoinMapByAscii(data, "&", false, false) // "a=1&b=2"
//	s := uString.JoinMapByAscii(data, "&", false, true)  // "a=1&b=2&c="
func JoinMapByAscii(data map[string]string, sep string, onlyValues, includeEmpty bool, exceptKeys ...string) string {
	excluded := make(map[string]struct{}, len(exceptKeys))
	for _, k := range exceptKeys {
		excluded[k] = struct{}{}
	}

	var parts []string
	var keys []string

	for k, v := range data {
		if _, ok := excluded[k]; ok {
			continue
		}
		if !includeEmpty && v == "" {
			continue
		}
		if onlyValues {
			keys = append(keys, k)
		} else {
			parts = append(parts, k+"="+v)
		}
	}

	if onlyValues {
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, data[k])
		}
	} else {
		sort.Strings(parts)
	}

	return strings.Join(parts, sep)
}

// Reverse 反转字符串（按 rune，支持中文）
//
// 使用示例：
//
//	uString.Reverse("hello") // "olleh"
//	uString.Reverse("你好世界") // "界世好你"
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// RuneCount 统计字符串的 Unicode 字符数（支持中文）
//
// 使用示例：
//
//	uString.RuneCount("hello")  // 5
//	uString.RuneCount("你好世界") // 4
func RuneCount(s string) int {
	return len([]rune(s))
}

// TrimSpace 去除字符串首尾的空白字符（空格、制表符、换行符等）
// 常用于表单提交时预处理用户输入
//
// 使用示例：
//
//	uString.TrimSpace("  hello world  ") // "hello world"
//	uString.TrimSpace("\t\n text \n")    // "text"
func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// TrimAll 移除字符串中的所有空白字符（包括内部空格、换行、制表符等）
//
// 使用示例：
//
//	uString.TrimAll("hello world")  // "helloworld"
//	uString.TrimAll("  a  b  c  ") // "abc"
//	uString.TrimAll("你 好 世 界")   // "你好世界"
func TrimAll(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
