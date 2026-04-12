package middleware

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/wanglelecc/gokitbox/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Config represents all available options for the middleware.
type Config struct {
	AllowAllOrigins bool `ini:"allowAllOrigins"`

	// AllowOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// Default value is []
	AllowOrigins []string `ini:"allowOrigins"`

	// AllowOriginFunc is a custom function to validate the origin. It take the origin
	// as argument and returns true if allowed or false otherwise. If this option is
	// set, the content of AllowOrigins is ignored.
	AllowOriginFunc func(origin string) bool

	// AllowMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (GET and POST)
	AllowMethods []string `ini:"allowMethods"`

	// AllowHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	AllowHeaders []string `ini:"allowHeaders"`

	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool `ini:"allowCredentials"`

	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposeHeaders []string `ini:"exposeHeaders"`

	// MaxAge indicates how long the results of a preflight request can be cached.
	// 配置格式同其他超时字段，如 "1h"、"30m"。空值表示不设置（由浏览器决定）。
	MaxAge string `ini:"maxAge"`

	// Allows to add origins like http://some-domain/*, https://api.* or http://some.*.subdomain.com
	AllowWildcard bool `ini:"allowWildcard"`

	// Allows usage of popular browser extensions schemas
	AllowBrowserExtensions bool `ini:"allowBrowserExtensions"`

	// Allows usage of WebSocket protocol
	AllowWebSockets bool `ini:"allowWebSockets"`

	// Allows usage of file:// schema (dangerous!) use it only when you 100% sure it's needed
	AllowFiles bool `ini:"allowFiles"`
}

// Cors 返回 CORS 中间件，配置在首次请求时从 [HttpCors] 段懒加载，
// 保证在 InitConfig BeforeServerStartFunc 执行后才读取配置。
// 配置加载失败时返回 HTTP 503，不 panic；错误日志只打印一次，避免高并发下日志爆炸。
func Cors() gin.HandlerFunc {
	var (
		once    sync.Once
		logOnce sync.Once
		handler gin.HandlerFunc
		initErr error
	)
	return func(c *gin.Context) {
		once.Do(func() {
			var cfg Config
			if err := config.ConfMapToStruct("HttpCors", &cfg); err != nil {
				initErr = fmt.Errorf("cors: failed to load [HttpCors] config: %w", err)
				return
			}
			var maxAge time.Duration
			if cfg.MaxAge != "" {
				var parseErr error
				maxAge, parseErr = time.ParseDuration(cfg.MaxAge)
				if parseErr != nil {
					initErr = fmt.Errorf("cors: invalid MaxAge %q: %w", cfg.MaxAge, parseErr)
					return
				}
			}
			corsConfig := cors.Config{
				AllowAllOrigins:        cfg.AllowAllOrigins,
				AllowOrigins:           cfg.AllowOrigins,
				AllowOriginFunc:        cfg.AllowOriginFunc,
				AllowMethods:           cfg.AllowMethods,
				AllowHeaders:           cfg.AllowHeaders,
				AllowCredentials:       cfg.AllowCredentials,
				ExposeHeaders:          cfg.ExposeHeaders,
				MaxAge:                 maxAge,
				AllowWildcard:          cfg.AllowWildcard,
				AllowBrowserExtensions: cfg.AllowBrowserExtensions,
				AllowWebSockets:        cfg.AllowWebSockets,
				AllowFiles:             cfg.AllowFiles,
			}
			handler = cors.New(corsConfig)
		})
		if initErr != nil {
			// 只打印一次，防止高并发时 stderr 被淹没
			logOnce.Do(func() {
				fmt.Fprintf(os.Stderr, "[cors] config error: %v\n", initErr)
			})
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		handler(c)
	}
}
