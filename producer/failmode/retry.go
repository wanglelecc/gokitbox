package failmode

import (
	"errors"
	"github.com/wanglelecc/gokitbox/producer/meta"
	"log"
)

var ErrRetryFull = errors.New("producer: legacy retry queue full")

func init()                        { AddFailMode(meta.RETRY, InitRetry) }
func InitRetry() (FailMode, error) { return &Retry{}, nil }

type Retry struct{}

// TryDo provides an explicit result for the historical retry adapter. Native
// Kafka uses its own bounded retry scheduler and does not traverse this channel.
func (r *Retry) TryDo(fallback chan<- []byte, message []byte) error {
	select {
	case fallback <- message:
		return nil
	default:
		return ErrRetryFull
	}
}
func (r *Retry) Do(fallback chan<- []byte, message, data []byte, keyParams []interface{}) {
	if err := r.TryDo(fallback, message); err != nil {
		log.Printf("producer legacy retry failed: %v", err)
	}
}
