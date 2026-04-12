package uOs

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// DirExists 判断目录是否存在
//
// 使用示例：
//
//	ok, err := uOs.DirExists("/tmp/logs")
//	if err != nil { ... }
//	// ok = true / false
func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// FileExists 判断文件是否存在（非目录）
//
// 使用示例：
//
//	ok, err := uOs.FileExists("/tmp/app.log")
//	if err != nil { ... }
//	// ok = true / false
func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsWritable 判断文件或目录是否可写
//
// 使用示例：
//
//	ok, err := uOs.IsWritable("/tmp/logs")
func IsWritable(path string) (bool, error) {
	if err := syscall.Access(path, syscall.O_RDWR); err != nil {
		return false, err
	}
	return true, nil
}

// MkdirIfNotExist 目录不存在时自动创建（含多级目录），已存在时不报错
//
// 使用示例：
//
//	err := uOs.MkdirIfNotExist("/tmp/app/logs/2024")
func MkdirIfNotExist(path string) error {
	return os.MkdirAll(path, 0755)
}

// ReadFile 读取文件全部内容
//
// 使用示例：
//
//	data, err := uOs.ReadFile("/etc/config.json")
//	// data = []byte("{...}")
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile 写入文件内容（覆盖），父目录不存在时自动创建
//
// 使用示例：
//
//	err := uOs.WriteFile("/tmp/logs/app.log", []byte("hello\n"))
func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// AppendFile 向文件追加内容，文件不存在时自动创建，父目录不存在时自动创建
//
// 使用示例：
//
//	err := uOs.AppendFile("/tmp/logs/app.log", []byte("new line\n"))
func AppendFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// CopyFile 复制文件，目标父目录不存在时自动创建
//
// 使用示例：
//
//	err := uOs.CopyFile("/tmp/src.txt", "/backup/dst.txt")
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// SafeRemove 安全删除文件，文件不存在时不返回错误
//
// 使用示例：
//
//	err := uOs.SafeRemove("/tmp/app.pid")
func SafeRemove(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// GetFileExt 获取文件名后缀，不含 "."
//
// 使用示例：
//
//	uOs.GetFileExt("avatar.jpg")       // "jpg"
//	uOs.GetFileExt("archive.tar.gz")   // "gz"
//	uOs.GetFileExt("noextfile")        // ""
func GetFileExt(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// Hostname 获取当前主机名，失败时返回 "localhost"
//
// 使用示例：
//
//	name := uOs.Hostname() // "server-01"
func Hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return name
}

// FullStack 获取所有 goroutine 的完整调用栈，常用于排查死锁/panic
//
// 使用示例：
//
//	stack := uOs.FullStack()
//	log.Println(stack)
func FullStack() string {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	return string(buf[:n])
}
