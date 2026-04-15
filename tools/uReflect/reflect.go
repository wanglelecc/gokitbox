package uReflect

import "reflect"

// IsNil 判断接口值是否为 nil 或空指针
//
// 对于非指针类型（如 int、string、struct）始终返回 false
//
// 使用示例：
//
//	var p *int
//	uReflect.IsNil(p)     // true
//	uReflect.IsNil(nil)   // true
//	uReflect.IsNil(123)   // false
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return false
}
