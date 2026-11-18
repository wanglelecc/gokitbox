package middleware

import (
	"bytes"
	"time"

	"github.com/tidwall/gjson"
	"github.com/wanglelecc/gokitbox/logger"

	"github.com/gin-gonic/gin"
)

type CheckError func([]byte) bool

// CheckCode0 检查响应体顶层 code 字段是否为 0，非 0 视为错误
func CheckCode0(in []byte) bool {
	result := gjson.GetBytes(in, "code")
	if !result.Exists() {
		return false
	}
	return result.Int() != 0
}

func Logger(checkers ...CheckError) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var captured *bodyLogWriter
		if len(checkers) > 0 && checkers[0] != nil {
			captured = &bodyLogWriter{ResponseWriter: c.Writer, body: bytes.NewBuffer(nil)}
			c.Writer = captured
		}
		c.Next()
		if _, skip := c.Get("SKIPLOG"); skip {
			return
		}
		fields := []interface{}{"route", c.FullPath(), "method", c.Request.Method, "status", c.Writer.Status(), "duration", time.Since(start), "response_bytes", c.Writer.Size()}
		if captured != nil && checkers[0](captured.body.Bytes()) {
			logger.Ex(transferToContext(c), "gin", "http application failure", fields...)
		} else {
			logger.Dx(transferToContext(c), "gin", "http completed", fields...)
		}
	}
}
