package uCrypto

import "golang.org/x/crypto/bcrypt"

// PasswordBcrypt 使用 bcrypt 对明文密码进行哈希加密，cost 固定 12（安全基线）
// 每次调用结果不同（bcrypt 内置随机 salt），不可反向解密
//
// 使用示例：
//
//	hash, err := uCrypto.PasswordBcrypt("myPassword123")
//	// hash = "$2a$12$xxxx..."（每次不同）
func PasswordBcrypt(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ValidatePassword 校验明文密码与 bcrypt 哈希是否匹配
// 匹配返回 true，不匹配或 hash 格式错误返回 false
//
// 使用示例：
//
//	ok := uCrypto.ValidatePassword(hash, "myPassword123") // true
//	ok := uCrypto.ValidatePassword(hash, "wrongPassword") // false
func ValidatePassword(hash, pwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)) == nil
}
