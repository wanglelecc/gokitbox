package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/wanglelecc/gokitbox/logger"
)

type RecoveryMiddleware struct {
	closed bool
}

// WrapCall 包装调用，捕获 panic 并恢复
func (m *RecoveryMiddleware) WrapCall(w *WrapContext) (err error) {
	if m.closed {
		return nil
	}

	ctx := w.GetCtx()
	defer func() {
		if r := recover(); r != nil {
			logger.Ex(ctx, "recovery", "call rpc service recovery error", "method", w.MethodTag, "error", fmt.Sprintf("%v", r), "stacks", string(debug.Stack()))
			err = fmt.Errorf("[service internal error]: %v", r)
		}
	}()

	return w.Next()
}

func initRecovery() *RecoveryMiddleware {
	middleware := new(RecoveryMiddleware)
	middleware.closed = false

	return middleware
}

func RecoveryOption() Options {
	return func() (Middleware, Point) {
		ctx := initRecovery()
		return ctx, ctx.WrapCall
	}
}
