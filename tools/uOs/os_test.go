package uOs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirExists(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		path         string
		wantExist    bool
		wantErr      bool
	}{
		{"存在的目录", tmpDir, true, false},
		{"不存在的目录", "/tmp/not_exist_dir_12345", false, false},
		{"文件不是目录", filepath.Join(tmpDir, "testfile"), false, false},
	}

	// 创建测试文件
	os.WriteFile(filepath.Join(tmpDir, "testfile"), []byte("test"), 0644)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DirExists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("DirExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantExist {
				t.Errorf("DirExists() = %v, want %v", got, tt.wantExist)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile")
	os.WriteFile(testFile, []byte("test"), 0644)

	tests := []struct {
		name         string
		path         string
		wantExist    bool
		wantErr      bool
	}{
		{"存在的文件", testFile, true, false},
		{"不存在的文件", "/tmp/not_exist_file_12345", false, false},
		{"目录不是文件", tmpDir, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FileExists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("FileExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantExist {
				t.Errorf("FileExists() = %v, want %v", got, tt.wantExist)
			}
		})
	}
}

func TestMkdirIfNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "new", "nested", "dir")

	err := MkdirIfNotExist(testDir)
	if err != nil {
		t.Errorf("MkdirIfNotExist() error = %v", err)
	}

	// 验证目录已创建
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Errorf("Dir was not created")
	}

	// 再次创建不应报错
	err = MkdirIfNotExist(testDir)
	if err != nil {
		t.Errorf("MkdirIfNotExist() on existing dir error = %v", err)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "test.txt")
	data := []byte("hello world")

	err := WriteFile(testFile, data)
	if err != nil {
		t.Errorf("WriteFile() error = %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %s, want %s", content, data)
	}
}

func TestAppendFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "append.txt")

	// 第一次追加
	err := AppendFile(testFile, []byte("hello"))
	if err != nil {
		t.Errorf("AppendFile() error = %v", err)
	}

	// 第二次追加
	err = AppendFile(testFile, []byte(" world"))
	if err != nil {
		t.Errorf("AppendFile() error = %v", err)
	}

	// 验证内容
	content, _ := os.ReadFile(testFile)
	if string(content) != "hello world" {
		t.Errorf("File content = %s, want 'hello world'", content)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstFile := filepath.Join(tmpDir, "dst.txt")

	// 创建源文件
	os.WriteFile(srcFile, []byte("test content"), 0644)

	// 复制
	err := CopyFile(srcFile, dstFile)
	if err != nil {
		t.Errorf("CopyFile() error = %v", err)
	}

	// 验证目标文件内容
	content, _ := os.ReadFile(dstFile)
	if string(content) != "test content" {
		t.Errorf("Copied content = %s, want 'test content'", content)
	}
}

func TestSafeRemove(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "remove.txt")

	// 删除存在的文件
	os.WriteFile(testFile, []byte("test"), 0644)
	err := SafeRemove(testFile)
	if err != nil {
		t.Errorf("SafeRemove() error = %v", err)
	}

	// 删除不存在的文件不应报错
	err = SafeRemove(filepath.Join(tmpDir, "not_exist.txt"))
	if err != nil {
		t.Errorf("SafeRemove() on non-existent file error = %v", err)
	}
}

func TestGetFileExt(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"jpg文件", "avatar.jpg", "jpg"},
		{"多级后缀", "archive.tar.gz", "gz"},
		{"无后缀", "noextfile", ""},
		{"点开头", ".htaccess", "htaccess"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFileExt(tt.filename)
			if got != tt.want {
				t.Errorf("GetFileExt(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestHostname(t *testing.T) {
	// 简单测试不返回空
	got := Hostname()
	if got == "" {
		t.Error("Hostname() returned empty string")
	}
}
