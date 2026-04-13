package uNo

import (
	"strings"
	"sync"
	"testing"
)

// TestRandDigitsFormat 验证 randDigits 只返回数字字符
func TestRandDigitsFormat(t *testing.T) {
	tests := []struct{ n int }{{0}, {1}, {5}, {10}, {32}}
	for _, tt := range tests {
		s := randDigits(tt.n)
		if len(s) != tt.n {
			t.Errorf("randDigits(%d) len = %d, want %d", tt.n, len(s), tt.n)
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				t.Errorf("randDigits(%d) contains non-digit char %q in %q", tt.n, c, s)
			}
		}
	}
}

// TestRandDigitsUniqueness 验证 H-5 修复：高并发下生成结果不全部相同（crypto/rand）
func TestRandDigitsUniqueness(t *testing.T) {
	const total = 1000
	results := make([]string, total)
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		idx := i
		go func() {
			defer wg.Done()
			results[idx] = randDigits(8)
		}()
	}
	wg.Wait()

	seen := make(map[string]int)
	for _, s := range results {
		seen[s]++
	}
	// 8 位数字 10^8 = 1 亿种组合，1000 次碰撞率极低
	// 若全部相同说明随机源无效
	if len(seen) == 1 {
		t.Error("randDigits produced identical results for all goroutines, random source may be broken")
	}
}

// TestGenOrderNoFormat 验证 GenOrderNo 格式：YD 前缀 + 默认 18 位
func TestGenOrderNoFormat(t *testing.T) {
	uid := int64(10001)
	no := GenOrderNo(uid)

	if !strings.HasPrefix(no, "YD") {
		t.Errorf("GenOrderNo() = %q, want prefix YD", no)
	}
	if len(no) != 18 {
		t.Errorf("GenOrderNo() len = %d, want 18", len(no))
	}
	// 末尾 4 位固定为 uid % 10000
	tail := no[len(no)-4:]
	if tail != "0001" {
		t.Errorf("GenOrderNo() tail = %q, want 0001 (uid%%10000)", tail)
	}
}

// TestGenRefundNoFormat 验证 GenRefundNo 末尾为 orderId%10000
func TestGenRefundNoFormat(t *testing.T) {
	orderId := int64(20260410001234)
	no := GenRefundNo(orderId)

	if !strings.HasPrefix(no, "YR") {
		t.Errorf("GenRefundNo() = %q, want prefix YR", no)
	}
	tail := no[len(no)-4:]
	if tail != "1234" {
		t.Errorf("GenRefundNo() tail = %q, want 1234 (orderId%%10000)", tail)
	}
}

// TestGenPayNoFormat 验证 GenPayNo 格式：日期 + YP + 默认 28 位
func TestGenPayNoFormat(t *testing.T) {
	uid := int64(10001)
	no := GenPayNo(uid)

	if len(no) != 28 {
		t.Errorf("GenPayNo() len = %d, want 28", len(no))
	}
	// 末尾 6 位固定为 uid % 1000000
	tail := no[len(no)-6:]
	if tail != "010001" {
		t.Errorf("GenPayNo() tail = %q, want 010001 (uid%%1000000)", tail)
	}
}

// TestGenNoCustomLength 验证自定义长度选项
func TestGenNoCustomLength(t *testing.T) {
	no := GenOrderNo(100, Option{Length: 22})
	if len(no) != 22 {
		t.Errorf("GenOrderNo with Length=22, got len=%d", len(no))
	}
}

// TestGenNoWithPrefix 验证自定义前缀
func TestGenNoWithPrefix(t *testing.T) {
	no := GenOrderNo(100, Option{Prefix: "T", Length: 20})
	if !strings.HasPrefix(no, "TYD") {
		t.Errorf("GenOrderNo with Prefix=T, got %q, want prefix TYD", no)
	}
	// 总长度 = len("T") + Length = 1 + 20 = 21
	if len(no) != 21 {
		t.Errorf("GenOrderNo with Prefix=T, Length=20, got len=%d, want 21", len(no))
	}
}

// TestGenNoConcurrent 验证并发生成无 race condition
func TestGenNoConcurrent(t *testing.T) {
	const total = 500
	results := make([]string, total)
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		idx := i
		go func() {
			defer wg.Done()
			results[idx] = GenOrderNo(int64(idx))
		}()
	}
	wg.Wait()

	seen := make(map[string]bool)
	for _, no := range results {
		if no == "" {
			t.Error("GenOrderNo() returned empty string")
		}
		seen[no] = true
	}
}
