package failmode

import (
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

}
