package middleware

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

var hostname string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
		fmt.Fprintf(os.Stderr, "[httpserver] WARNING: failed to get hostname: %v\n", err)
	}
}

func RequestHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		benchmark := c.GetHeader("Request-Type")
		if benchmark == "performance-testing" {
			c.Set("IS_BENCHMARK", "1")
		}
		c.Next()
	}
}

