package consumer

import (
	"math/rand"
	"time"
)

// backoff 指数退避，每次 Next() 返回带 ±25% jitter 的等待时长，并将当前值翻倍直至上限。
// 不是并发安全的，每个 goroutine 应持有独立实例。
type backoff struct {
	initial time.Duration
	current time.Duration
	max     time.Duration
}

func newBackoff(initial, max time.Duration) *backoff {
	return &backoff{initial: initial, current: initial, max: max}
}

// Next 返回当前退避时长（含 ±25% jitter），然后将 current 翻倍（上限 max）。
func (b *backoff) Next() time.Duration {
	d := b.current

	// ±25% jitter：在 [0.75d, 1.25d) 区间均匀随机
	jitter := time.Duration(rand.Int63n(int64(d)/2)) - d/4
	d += jitter
	if d < 0 {
		d = b.initial
	}

	// 翻倍，不超过 max
	b.current *= 2
	if b.current > b.max {
		b.current = b.max
	}

	return d
}

// Reset 将退避重置为初始值，连接成功后调用。
func (b *backoff) Reset() {
	b.current = b.initial
}
