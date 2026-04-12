package middleware

import (
	"context"
	"time"

	"github.com/wanglelecc/gokitbox/logger"

	"github.com/smallnest/rpcx/share"
)

// 私有类型作为 context key，防止命名冲突
type contextKey string

const (
	traceIDKey contextKey = "trace_id"
	rpcIDKey   contextKey = "rpc_id"
	logIDKey   contextKey = "log_id"
	startKey   contextKey = "start"
)

type ContextMiddleware struct {
	closed bool
}

func (m *ContextMiddleware) WrapCall(w *WrapContext) error {
	if m.closed {
		return nil
	}

	ctx := w.GetCtx()
	reqMeta, ok := ctx.Value(share.ReqMetaDataKey).(map[string]string)
	if ok {
		for k, v := range reqMeta {
			ctx = context.WithValue(ctx, contextKey(k), v)
		}
	}

	if ctx.Value(traceIDKey) == nil {
		ctx = context.WithValue(ctx, traceIDKey, logger.GenTraceId())
	}

	if rpcId := ctx.Value(rpcIDKey); rpcId != nil {
		if rpcIdStr, ok := rpcId.(string); ok {
			ctx = context.WithValue(ctx, rpcIDKey, rpcIdStr+".0")
		} else {
			ctx = context.WithValue(ctx, rpcIDKey, "1.0")
		}
	} else {
		ctx = context.WithValue(ctx, rpcIDKey, "1.0")
	}

	ctx = context.WithValue(ctx, logIDKey, logger.GenLoggerId())
	ctx = context.WithValue(ctx, startKey, time.Now())

	w.SetCtx(ctx)

	// 继续执行后续中间件链
	return w.Next()
}

func initContext() *ContextMiddleware {
	middleware := new(ContextMiddleware)
	middleware.closed = false

	return middleware
}

func ContextOption() Options {
	return func() (Middleware, Point) {
		ctx := initContext()
		return ctx, ctx.WrapCall
	}
}
