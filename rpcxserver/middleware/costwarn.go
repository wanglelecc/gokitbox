package middleware

import (
	"time"

	"github.com/wanglelecc/gokitbox/logger"
)

// 默认耗时阈值配置
const (
	DefaultCostThreshold = 200 * time.Millisecond
)

type CostwarnMiddleware struct {
	closed        bool
	costThreshold time.Duration
}

func (m *CostwarnMiddleware) WrapCall(w *WrapContext) error {
	if m.closed {
		return nil
	}

	start := time.Now()
	e := w.Next()
	end := time.Now()
	cost := end.Sub(start)
	ctx := w.GetCtx()
	if e != nil {
		logger.Ex(ctx, "costLog", "call rpc service error", "method", w.MethodTag, "cost", cost, "error", e.Error())
	} else if cost > m.costThreshold {
		logger.Wx(ctx, "costLog", "call rpc service timeout", "method", w.MethodTag, "cost", cost, "cost_threshold", m.costThreshold)
	} else {
		logger.Ix(ctx, "costLog", "call rpc service success", "method", w.MethodTag, "cost", cost, "cost_threshold", m.costThreshold)
	}

	return e
}

func InitCostwarn(threshold time.Duration) *CostwarnMiddleware {
	middleware := new(CostwarnMiddleware)
	middleware.closed = false
	if threshold <= 0 {
		middleware.costThreshold = DefaultCostThreshold
	} else {
		middleware.costThreshold = threshold
	}

	return middleware
}

func CostwarnOption() Options {
	return func() (Middleware, Point) {
		ctx := InitCostwarn(DefaultCostThreshold)
		return ctx, ctx.WrapCall
	}
}

// CostwarnOptionWithThreshold 允许自定义阈值的选项
func CostwarnOptionWithThreshold(threshold time.Duration) Options {
	return func() (Middleware, Point) {
		ctx := InitCostwarn(threshold)
		return ctx, ctx.WrapCall
	}
}
