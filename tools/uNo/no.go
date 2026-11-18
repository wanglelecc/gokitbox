package uNo

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Option controls presentation; Length cannot remove uniqueness bits.
type Option struct {
	Prefix string
	Length int
}

// NewNumber returns a 128-bit identifier, normally 28 ASCII characters including the business prefix.
func NewNumber(bizType string, opt Option) (string, error) {
	if len(bizType) != 2 || len(opt.Prefix) > 32 {
		return "", errors.New("uNo: business prefix must have two characters; custom prefix <=32")
	}
	for _, c := range bizType {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return "", errors.New("uNo: business prefix must be uppercase ASCII alphanumeric")
		}
	}
	for _, c := range opt.Prefix {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return "", errors.New("uNo: invalid custom prefix")
		}
	}
	length := opt.Length
	if length == 0 {
		length = 28
	}
	if length < 28 || length > 128 {
		return "", errors.New("uNo: length must be 28..128, without truncating entropy")
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	body := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return opt.Prefix + bizType + strings.Repeat("0", length-28) + body, nil
}
func uniqueNumber(prefix string, opt Option) string {
	v, err := NewNumber(prefix, opt)
	if err != nil {
		panic(err)
	}
	return v
}
func generate(bizType string, id int64, opt Option) string { return uniqueNumber(bizType, opt) }
func generatePay(uid int64, opt Option) string             { return uniqueNumber("YP", opt) }

// normBizType 将业务类型标识规范化为 2 位大写，不足右补空格，超出截断
func normBizType(s string) string {
	s = strings.ToUpper(s)
	if len(s) >= 2 {
		return s[:2]
	}
	return fmt.Sprintf("%-2s", s)
}

// GenNo returns a business prefix plus 26 Base32 characters carrying 128 random bits.
// IDs are probabilistically unique; persistence must enforce a unique constraint.
// The legacy string API panics on entropy failure or unsafe truncation.
func GenNo(bizType string, id int64, opts ...Option) string {
	var opt Option
	if len(opts) > 0 {
		opt = opts[0]
	}
	return generate(normBizType(bizType), id, opt)
}

// GenPayNo returns a YP-prefixed 128-bit identifier (28 characters by default).
func GenPayNo(uid int64, opts ...Option) string {
	var opt Option
	if len(opts) > 0 {
		opt = opts[0]
	}
	return generatePay(uid, opt)
}

// GenOrderNo returns a YD-prefixed identifier; uid/orderId is not a uniqueness source.
func GenOrderNo(uid int64, opts ...Option) string {
	return GenNo("YD", uid, opts...)
}

// GenRefundNo returns a YR-prefixed identifier; uid/orderId is not a uniqueness source.
func GenRefundNo(orderId int64, opts ...Option) string {
	return GenNo("YR", orderId, opts...)
}

// GenWithdrawNo returns a YW-prefixed identifier; uid/orderId is not a uniqueness source.
func GenWithdrawNo(uid int64, opts ...Option) string {
	return GenNo("YW", uid, opts...)
}

// GenYearWeekNo 生成当前年份的后两位与 ISO 周数组成的编号字符串
//
// 常用于按周归档、批次号生成等场景
//
// 使用示例：
//
//	no := uNo.GenYearWeekNo()
//	// no = "2515"（表示 2025 年第 15 周）
func GenYearWeekNo() string {
	now := time.Now()
	isoYear, isoWeek := now.ISOWeek()
	return fmt.Sprintf("%02d%02d", isoYear%100, isoWeek)
}
