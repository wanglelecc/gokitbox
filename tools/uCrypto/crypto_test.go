package uCrypto

import (
	"testing"
)

func TestPasswordBcrypt(t *testing.T) {
	password := "myPassword123"
	hash, err := PasswordBcrypt(password)
	if err != nil {
		t.Errorf("PasswordBcrypt() error = %v", err)
		return
	}
	// 验证hash格式（bcrypt格式以$2a$开头）
	if len(hash) < 7 || hash[:4] != "$2a$" {
		t.Errorf("PasswordBcrypt() hash format invalid: %s", hash)
	}
	// 每次生成的hash应该不同（因为有随机salt）
	hash2, _ := PasswordBcrypt(password)
	if hash == hash2 {
		t.Error("PasswordBcrypt() should generate different hashes each time")
	}
}

func TestValidatePassword(t *testing.T) {
	password := "myPassword123"
	hash, _ := PasswordBcrypt(password)
	// 正确密码应该验证通过
	if !ValidatePassword(hash, password) {
		t.Error("ValidatePassword() returned false for correct password")
	}
	// 错误密码应该验证失败
	if ValidatePassword(hash, "wrongPassword") {
		t.Error("ValidatePassword() returned true for wrong password")
	}
}
