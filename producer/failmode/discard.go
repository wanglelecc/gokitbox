package failmode

import "github.com/wanglelecc/gokitbox/producer/meta"

func init() {
	AddFailMode(meta.DISCARD, InitDiscard)
}

func InitDiscard() (FailMode, error) {
	d := new(Discard)
	return d, nil
}

type Discard struct {
}

func (d *Discard) Do(fallback chan<- []byte, message []byte, data []byte, keyParams []interface{}) {
}
