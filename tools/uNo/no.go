package uNo

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// customEpoch 自定义纪元：2026-01-01 00:00:00 UTC（Unix 时间戳 1767196800）
// 纪元秒数当前约 1.66 亿（9位），%09d 格式有效至约 2056 年
const customEpoch = int64(1767196800)

// Option 编号生成选项
type Option struct {
	Prefix string // 自定义前缀，可为空
	Length int    // 编号总长度（不含 Prefix），默认因类型而异
}

// generate 普通业务编号生成（订单号 / 退款单号 / 提现单号）
//
// 格式：[Prefix][bizType(2)][epochSec(9)][random(N)][idFrag(4)]
//   - epochSec: 自定义纪元秒数，%09d 固定 9 位，有效至约 2051 年
//   - random:   随机补位，填充至目标长度
//   - idFrag:   abs(id)%10000，末尾固定 4 位，%04d 补零，支持 LIKE '%0001' 后缀模糊查询
//
// 默认 length=18：2+9+3+4=18；后 4 位始终为 idFrag
func generate(bizType string, id int64, opt Option) string {
	length := opt.Length
	if length <= 0 {
		length = 18
	}

	if id < 0 {
		id = -id
	}
	epochSec := time.Now().Unix() - customEpoch
	idFrag := id % 10000

	head := fmt.Sprintf("%s%09d", bizType, epochSec) // 11 chars
	tail := fmt.Sprintf("%04d", idFrag)              // 4 chars，末尾固定

	randLen := length - len(head) - len(tail)
	if randLen < 0 {
		randLen = 0
	}

	return opt.Prefix + head + randDigits(randLen) + tail
}

// generatePay 支付流水号生成，参考支付宝交易号风格
//
// 格式：[Prefix][YYYYMMDD(8)][YP(2)][milliOfDay(8)][random(N)][uidFrag(6)]
//   - YYYYMMDD:   日期开头，高辨识度，便于按日对账归档
//   - milliOfDay: 当日已过毫秒数（0~86399999），%08d 固定 8 位，精确到毫秒
//   - random:     随机补位
//   - uidFrag:    abs(uid)%1000000，末尾固定 6 位，%06d 补零，支持后缀模糊查询
//
// 默认 length=28：8+2+8+4+6=28；后 6 位始终为 uidFrag
func generatePay(uid int64, opt Option) string {
	length := opt.Length
	if length <= 0 {
		length = 28
	}

	now := time.Now()
	date := now.Format("20060102")
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	milliOfDay := now.UnixMilli() - dayStart.UnixMilli() // 0~86399999

	if uid < 0 {
		uid = -uid
	}
	uidFrag := uid % 1000000

	head := fmt.Sprintf("%sYP%08d", date, milliOfDay) // 18 chars
	tail := fmt.Sprintf("%06d", uidFrag)                          // 6 chars，末尾固定

	randLen := length - len(head) - len(tail)
	if randLen < 0 {
		randLen = 0
	}

	return opt.Prefix + head + randDigits(randLen) + tail
}

// randDigits 生成 n 位随机数字字符串（使用 crypto/rand 保证安全性）
func randDigits(n int) string {
	b := make([]byte, n)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(10))
		b[i] = byte('0') + byte(num.Int64())
	}
	return string(b)
}

// normBizType 将业务类型标识规范化为 2 位大写，不足右补空格，超出截断
func normBizType(s string) string {
	s = strings.ToUpper(s)
	if len(s) >= 2 {
		return s[:2]
	}
	return fmt.Sprintf("%-2s", s)
}

// GenNo 生成通用业务编号
//
// 格式：[Prefix][bizType(2)][epochSec(9)][random(N)][idFrag(4)]
// 末尾 4 位固定为 abs(id)%10000（零补位），支持 LIKE '%0001' 后缀模糊查询
// 默认长度 18（不含前缀）
//
// 使用示例：
//
//	uNo.GenNo("XX", 10001)
//	// → "XX166377600xyz0001"（18位，末尾 0001 固定）
//	uNo.GenNo("XX", 10001, uNo.Option{Prefix: "T", Length: 22})
func GenNo(bizType string, id int64, opts ...Option) string {
	var opt Option
	if len(opts) > 0 {
		opt = opts[0]
	}
	return generate(normBizType(bizType), id, opt)
}

// GenPayNo 生成支付请求流水号，掺入 uid
//
// 参考支付宝交易号风格，日期开头 + 当日毫秒 + 末尾 uidFrag
// 格式：[Prefix][YYYYMMDD][YP][milliOfDay(8)][random(4)][uidFrag(6)]
// 末尾 6 位固定为 abs(uid)%1000000（零补位），支持后缀模糊查询
// 默认长度 28（不含前缀）
//
// 使用示例：
//
//	uNo.GenPayNo(10001)
//	// → "20260410YP02345678904010001"（28位，末尾 010001 固定）
//	uNo.GenPayNo(10001, uNo.Option{Prefix: "AP", Length: 32})
func GenPayNo(uid int64, opts ...Option) string {
	var opt Option
	if len(opts) > 0 {
		opt = opts[0]
	}
	return generatePay(uid, opt)
}

// GenOrderNo 生成订单号，掺入 uid
//
// 格式：[Prefix][YD][epochSec(9)][random(3)][uidFrag(4)]
// 末尾 4 位固定为 abs(uid)%10000，默认 18 位（不含前缀）
//
// 使用示例：
//
//	uNo.GenOrderNo(10001)
//	// → "YD166377600xyz0001"（18位）
func GenOrderNo(uid int64, opts ...Option) string {
	return GenNo("YD", uid, opts...)
}

// GenRefundNo 生成退款单号，掺入 orderId
//
// 格式：[Prefix][YR][epochSec(9)][random(3)][orderIdFrag(4)]
// 末尾 4 位固定为 abs(orderId)%10000，默认 18 位（不含前缀）
//
// 使用示例：
//
//	uNo.GenRefundNo(20260410001234)
//	// → "YR166377600xyz1234"（18位，末尾 1234 为 orderId%10000）
func GenRefundNo(orderId int64, opts ...Option) string {
	return GenNo("YR", orderId, opts...)
}

// GenWithdrawNo 生成提现单号，掺入 uid
//
// 格式：[Prefix][YW][epochSec(9)][random(3)][uidFrag(4)]
// 末尾 4 位固定为 abs(uid)%10000，默认 18 位（不含前缀）
//
// 使用示例：
//
//	uNo.GenWithdrawNo(10001)
//	// → "YW166377600xyz0001"（18位）
func GenWithdrawNo(uid int64, opts ...Option) string {
	return GenNo("YW", uid, opts...)
}
