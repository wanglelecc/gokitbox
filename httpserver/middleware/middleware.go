package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httputil"
	"runtime/debug"
	"strconv"

	"github.com/wanglelecc/gokitbox/logger"

	"github.com/gin-gonic/gin"
)

// maxBodySize 请求体/响应体日志截断上限，超出后截断记录，不影响实际请求转发
const maxBodySize = 1 << 20 // 1MB

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	if w.body.Len() < maxBodySize {
		remaining := maxBodySize - w.body.Len()
		if len(b) <= remaining {
			_, _ = w.body.Write(b) // bytes.Buffer.Write 永远返回 nil error
		} else {
			_, _ = w.body.Write(b[:remaining])
		}
	}
	return w.ResponseWriter.Write(b)
}

// Context 注入请求上下文：log_id、trace_id、rpc_id 等链路追踪字段。
// HTTP header 名称经 gin 规范化（textproto.CanonicalMIMEHeaderKey）后不区分大小写，
// "traceId" 与 "traceid" 规范化结果相同，因此只需两路匹配。
func Context() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("log_id", strconv.FormatInt(logger.GenLoggerId(), 10))
		ctx.Set("hostname", hostname)

		// traceId / trace_id 均兼容，"traceId"/"traceid" 规范化后相同
		traceId := ctx.GetHeader("traceId")
		if traceId == "" {
			traceId = ctx.GetHeader("trace_id")
		}
		if traceId == "" {
			traceId = logger.GenTraceId()
		}
		ctx.Set("trace_id", traceId)

		// rpcId / rpc_id 均兼容，"rpcId"/"rpcid" 规范化后相同
		rpcId := ctx.GetHeader("rpcId")
		if rpcId == "" {
			rpcId = ctx.GetHeader("rpc_id")
		}
		if rpcId == "" {
			rpcId = "1.0"
		}
		ctx.Set("rpc_id", rpcId)

		ctx.Next()
	}
}

// SkipLogInfo 设置跳过本次请求的日志记录
func SkipLogInfo() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("SKIPLOG", "1")
		ctx.Next()
	}
}

// Recovery panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				httpRequest, dumpErr := httputil.DumpRequest(c.Request, false)
				httpRequestStr := string(httpRequest)
				if dumpErr != nil {
					httpRequestStr = fmt.Sprintf("dump error: %v", dumpErr)
				}
				ctx := transferToContext(c)
				logger.Ex(ctx, "recovery", "http request panic recovered error",
					"request_url", c.Request.RequestURI,
					"http_request", httpRequestStr,
					"error", fmt.Sprintf("%v", err),
					"stacks", string(stack),
				)
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}

// ginCtx 将 gin.Keys 包装为标准 context，O(1) map 查找，
// 兼容 logger 包使用字符串 key 的 ctx.Value("trace_id") 调用。
type ginCtx struct {
	context.Context
	keys map[string]any
}

func (c *ginCtx) Value(key any) any {
	if k, ok := key.(string); ok {
		if v, exists := c.keys[k]; exists {
			return v
		}
	}
	return c.Context.Value(key)
}

func transferToContext(c *gin.Context) context.Context {
	if len(c.Keys) == 0 {
		return context.Background()
	}
	// 复制 keys，避免持有 gin.Context 引用导致 GC 延迟
	keys := make(map[string]any, len(c.Keys))
	for k, v := range c.Keys {
		keys[k.(string)] = v
	}
	return &ginCtx{Context: context.Background(), keys: keys}
}
