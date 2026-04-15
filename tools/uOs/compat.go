package uOs

import "os"

// Exists 判断所给路径的文件或文件夹是否存在（兼容旧 API）
//
// 仅返回 bool，不区分文件和目录
//
// 使用示例：
//
//	ok := uOs.Exists("/tmp/test.txt")
//	// ok = true 或 false
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
