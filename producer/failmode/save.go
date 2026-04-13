package failmode

import (
	"log"

	"github.com/wanglelecc/gokitbox/producer/meta"
)

func init() {
	AddFailMode(meta.SAVE, InitSave)
}

func InitSave() (FailMode, error) {
	s := new(Save)
	return s, nil
}

type Save struct {
}

func (s *Save) Do(fallback chan<- []byte, message []byte, data []byte, keyParams []interface{}) {
	// Save 模式尚未实现持久化，消息将被丢弃，记录告警避免静默丢失
	log.Printf("[WARN] failmode save: message dropped (not implemented), message=%s", string(message))
}
