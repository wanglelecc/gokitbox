package uConvert

import (
	"fmt"
	"strconv"
)

// ToString 将任意类型转为字符串，nil 返回空字符串
//
// 使用示例：
//
//	uConvert.ToString(123)    // "123"
//	uConvert.ToString(3.14)   // "3.14"
//	uConvert.ToString(true)   // "true"
//	uConvert.ToString(nil)    // ""
func ToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ToInt 将字符串或数字类型转为 int，转换失败返回 0
//
// 使用示例：
//
//	uConvert.ToInt("42")    // 42
//	uConvert.ToInt("abc")   // 0
//	uConvert.ToInt(3.7)     // 3（截断）
//	uConvert.ToInt(true)    // 1
func ToInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	case bool:
		if val {
			return 1
		}
		return 0
	}
	return 0
}

// ToInt64 将字符串或数字类型转为 int64，转换失败返回 0
//
// 使用示例：
//
//	uConvert.ToInt64("9999999999") // 9999999999
//	uConvert.ToInt64(42)           // 42
//	uConvert.ToInt64("3.7")        // 0（非整数字符串转换失败返回 0）
func ToInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	case bool:
		if val {
			return 1
		}
		return 0
	}
	return 0
}

// ToFloat64 将字符串或数字类型转为 float64，转换失败返回 0
//
// 使用示例：
//
//	uConvert.ToFloat64("3.14")  // 3.14
//	uConvert.ToFloat64(42)      // 42.0
//	uConvert.ToFloat64("abc")   // 0
func ToFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case bool:
		if val {
			return 1
		}
		return 0
	}
	return 0
}

// ToBool 将字符串或数字类型转为 bool
// 字符串 "1"、"t"、"T"、"true"、"TRUE"、"True" 返回 true，其余返回 false
//
// 使用示例：
//
//	uConvert.ToBool("true") // true
//	uConvert.ToBool("1")    // true
//	uConvert.ToBool(1)      // true
//	uConvert.ToBool("no")   // false（strconv.ParseBool 不识别 yes/no）
//	uConvert.ToBool(0)      // false
func ToBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		b, _ := strconv.ParseBool(val)
		return b
	default:
		return ToInt64(val) != 0
	}
}
