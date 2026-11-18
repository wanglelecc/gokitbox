package rpcxserver

import (
	"github.com/soheilhy/cmux"
	"time"
)

type muxBudget struct{ timeout time.Duration }

func (p muxBudget) MuxMatch(m cmux.CMux) {
	timeout := p.timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	m.SetReadTimeout(timeout)
}
