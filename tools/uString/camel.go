package uString

import "unicode"

// CamelCaseToColon 将驼峰命名转换为冒号分隔命名
//
// 常用于路由参数、配置键名等场景
//
// 使用示例：
//
//	s := uString.CamelCaseToColon("UserName")
//	// s = "user:name"
func CamelCaseToColon(s string) string {
	var output []rune
	for i, r := range s {
		if i == 0 {
			output = append(output, unicode.ToLower(r))
			continue
		}
		if unicode.IsUpper(r) {
			output = append(output, ':')
		}
		output = append(output, unicode.ToLower(r))
	}
	return string(output)
}

// CalculateTabs 计算 field 按 width 对齐所需的制表符个数
//
// 基于每个制表符占 8 个字符宽度计算
//
// 使用示例：
//
//	tabs := uString.CalculateTabs("name", 16)
//	// tabs = 1
func CalculateTabs(field string, width int) int {
	fieldLen := len(field)
	tabs := (width - fieldLen + 7) / 8
	return tabs
}

// Format400 将 10 位数字格式化为 400-XXX-XXXX 电话格式
//
// 如果长度不是 10 位，则原样返回
//
// 使用示例：
//
//	phone := uString.Format400("4001234567")
//	// phone = "400-123-4567"
func Format400(phone string) string {
	if len(phone) != 10 {
		return phone
	}
	return phone[:3] + "-" + phone[3:6] + "-" + phone[6:]
}
