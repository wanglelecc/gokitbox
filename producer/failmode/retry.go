package failmode

import (
	"github.com/wanglelecc/gokitbox/producer/meta"
)

func init() {
	AddFailMode(meta.RETRY, InitRetry)
}

func InitRetry() (FailMode, error) {
	r := new(Retry)
	return r, nil
}

type Retry struct {
}

func (r *Retry) Do(fallback chan<- []byte, message []byte, data []byte, keyParams []interface{}) {
	// 使用 select + default 避免阻塞，如果 channel 已满则丢弃消息
	select {
	case fallback <- message:
	default:
		// channel 已满，避免阻塞，消息丢失
	}
}
