package uHash

import (
	"testing"
)

func TestBucketHash(t *testing.T) {
	// 相同输入应该产生相同输出
	s := "user_123"
	got1 := BucketHash(s, 16)
	got2 := BucketHash(s, 16)
	if got1 != got2 {
		t.Errorf("BucketHash() not consistent: %d vs %d", got1, got2)
	}
	// 应该在范围内
	if got1 < 0 || got1 >= 16 {
		t.Errorf("BucketHash() = %d, not in range [0, 16)", got1)
	}
}

func TestMD5(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"空字符串", "", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", "hello", "5d41402abc4b2a76b9719d911017c592"},
		{"中文", "你好", "7eca689f0d3389d9dea66ae112e5cfd7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MD5(tt.input)
			if got != tt.expected {
				t.Errorf("MD5(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMD5Bytes(t *testing.T) {
	input := []byte("hello")
	expected := "5d41402abc4b2a76b9719d911017c592"
	got := MD5Bytes(input)
	if got != expected {
		t.Errorf("MD5Bytes() = %q, want %q", got, expected)
	}
}

func TestSHA1(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", "hello", "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA1(tt.input)
			if got != tt.expected {
				t.Errorf("SHA1(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", "hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA256(tt.input)
			if got != tt.expected {
				t.Errorf("SHA256(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestHmacSHA1(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		key      string
		expected int // 长度
	}{
		{"正常", "message", "secret_key", 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HmacSHA1(tt.data, tt.key)
			if len(got) != tt.expected {
				t.Errorf("HmacSHA1() length = %d, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestHmacSHA256(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		key      string
		expected int // 长度
	}{
		{"正常", "message", "secret_key", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HmacSHA256(tt.data, tt.key)
			if len(got) != tt.expected {
				t.Errorf("HmacSHA256() length = %d, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestGuid(t *testing.T) {
	// 生成多个GUID，验证不重复
	guids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := Guid()
		if id == "" {
			t.Error("Guid() returned empty string")
		}
		if guids[id] {
			t.Errorf("Duplicate GUID generated: %s", id)
		}
		guids[id] = true
		// 验证是32位MD5格式
		if len(id) != 32 {
			t.Errorf("Guid() length = %d, want 32", len(id))
		}
	}
}

func TestSignByAscii(t *testing.T) {
	params := map[string]string{"b": "2", "a": "1", "c": "3"}
	secret := "my_secret"
	// 验证不 panic
	got := SignByAscii(params, secret)
	if got == "" {
		t.Error("SignByAscii() returned empty string")
	}
	// HMAC-SHA256 输出 64 位十六进制字符串
	if len(got) != 64 {
		t.Errorf("SignByAscii() length = %d, want 64", len(got))
	}
	// 相同输入结果确定
	got2 := SignByAscii(params, secret)
	if got != got2 {
		t.Error("SignByAscii() not deterministic")
	}
	// 排除 key 后结果不同
	gotExcluded := SignByAscii(params, secret, "c")
	if gotExcluded == got {
		t.Error("SignByAscii() with excluded key should differ")
	}
}
