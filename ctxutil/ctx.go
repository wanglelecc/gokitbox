package ctxutil

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wanglelecc/gokitbox/logger"
)

// TransferToContext 将 gin.Context 中存储的 Keys 转移到标准 context.Context
//
// 常用于在请求入口处将 gin 的上下文数据传递给后续依赖标准 context 的函数
//
// 使用示例：
//
//	ctx := ctxutil.TransferToContext(c)
//	// ctx 包含 c.Keys 中的所有键值对
func TransferToContext(c *gin.Context) context.Context {
	ctx := context.Background()
	for k, v := range c.Keys {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

// WithContext 确保传入的 context 包含 trace_id、log_id、start 三个基础追踪字段
//
// 如果字段已存在则跳过，避免覆盖
//
// 使用示例：
//
//	ctx = ctxutil.WithContext(ctx)
//	// ctx.Value("trace_id") 一定不为 nil
func WithContext(ctx context.Context) context.Context {
	if ctx.Value("trace_id") == nil {
		ctx = context.WithValue(ctx, "trace_id", logger.GenTraceId())
	}
	if ctx.Value("log_id") == nil {
		ctx = context.WithValue(ctx, "log_id", logger.GenLoggerId())
	}
	if ctx.Value("start") == nil {
		ctx = context.WithValue(ctx, "start", time.Now())
	}
	return ctx
}

// MakeContext 创建一个新的 context，并自动注入 trace_id、log_id、start
//
// 适合在后台任务、定时器等没有上游 context 的场景使用
//
// 使用示例：
//
//	ctx := ctxutil.MakeContext()
//	// 可直接用于日志记录和链路追踪
func MakeContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", logger.GenTraceId())
	ctx = context.WithValue(ctx, "log_id", logger.GenLoggerId())
	ctx = context.WithValue(ctx, "start", time.Now())
	return ctx
}
