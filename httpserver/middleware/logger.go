package middleware

import (
	"bytes"
	"io"
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

func Logger(fns ...CheckError) gin.HandlerFunc {
	var errCheck CheckError
	if len(fns) > 0 {
		errCheck = fns[0]
	}

	return func(c *gin.Context) {
		blw := &bodyLogWriter{body: bytes.NewBuffer(nil), ResponseWriter: c.Writer}
		c.Writer = blw

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		var (
			body        []byte // 用于日志（截断至 maxBodySize）
			bodyReadErr error
		)
		if c.Request.Body != nil {
			// 读取至多 maxBodySize+1 字节：+1 用于探测是否超限，避免全量读入大文件导致 OOM
			limited, readErr := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize+1))
			bodyReadErr = readErr
			if len(limited) > maxBodySize {
				// body 超过上限：日志截断，用 MultiReader 将已读部分+剩余流拼接还原完整 body
				body = limited[:maxBodySize]
				c.Request.Body = io.NopCloser(io.MultiReader(
					bytes.NewBuffer(limited),
					c.Request.Body,
				))
			} else {
				body = limited
				c.Request.Body = io.NopCloser(bytes.NewBuffer(limited))
			}
		}

		c.Next()

		_, skip := c.Get("SKIPLOG")
		if skip {
			return
		}

		end := time.Now()
		latency := end.Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		comment := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		buf := blw.body.Bytes()
		ctx := transferToContext(c)

		fields := []any{
			"request_time", start.Format("2006/01/02 - 15:04:05"),
			"status_code", statusCode,
			"latency", latency.String(),
			"client_ip", clientIP,
			"method", method,
			"path", path,
			"comment", comment,
			"request_body", string(body),
			"response_body", string(buf),
		}
		if bodyReadErr != nil {
			fields = append(fields, "body_read_error", bodyReadErr.Error())
		}

		if errCheck != nil && errCheck(buf) {
			logger.Ex(ctx, "gin", "", fields...)
		} else {
			logger.Dx(ctx, "gin", "", fields...)
		}
	}
}
