package httpserver

import (
	"fmt"
	"os"
	"time"
)

// Duration 自定义 Duration 类型，支持从 ini 文件解析字符串格式（如 "5s", "500ms"）和数字格式
type Duration time.Duration

// UnmarshalText 实现从文本解析 Duration
func (d *Duration) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "" || s == "0" {
		*d = 0
		return nil
	}

	// 尝试按 time.ParseDuration 解析（支持 "5s", "500ms" 等格式）
	parsed, err := time.ParseDuration(s)
	if err == nil {
		*d = Duration(parsed)
		return nil
	}

	// 如果解析失败，尝试作为纯数字（秒）解析，保持向后兼容
	var seconds int64
	_, parseErr := fmt.Sscanf(s, "%d", &seconds)
	if parseErr == nil && seconds > 0 {
		fmt.Fprintf(os.Stderr, "[httpserver] WARNING: timeout value %q is a plain number, treating as seconds. Consider using duration format like %ds.\n", s, seconds)
		*d = Duration(time.Duration(seconds) * time.Second)
		return nil
	}

	return fmt.Errorf("invalid duration format: %s (expected format like '5s', '500ms', or number of seconds)", s)
}

// ToDuration 转换为 time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// parseDuration 解析超时字符串，s 为空时返回 defaultVal
func parseDuration(s string, defaultVal time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultVal, nil
	}
	var d Duration
	if err := d.UnmarshalText([]byte(s)); err != nil {
		return 0, err
	}
	return d.ToDuration(), nil
}

// ServerOptions http server options
type ServerOptions struct {
	// Mode run mode 可选 debug/release/test
	Mode string `ini:"mode"`
	// Addr TCP address to listen on, ":http" if empty
	Addr string `ini:"addr"`

	// ReadHeaderTimeout is the maximum duration allowed to read request headers.
	// 防御 Slowloris 慢速头攻击的关键字段，建议设为 ReadTimeout 的 1/2。
	// 配置格式支持: "5s", "500ms" 或纯数字（秒）
	ReadHeaderTimeout string `ini:"readHeaderTimeout"`

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	// 配置格式支持: "10s", "500ms", "1m" 或纯数字（秒）
	ReadTimeout string `ini:"readTimeout"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response.
	// 配置格式支持: "30s", "1m", "500ms" 或纯数字（秒）
	WriteTimeout string `ini:"writeTimeout"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled.
	// 连接空闲超过此时长后关闭，keep-alive 处于启用状态。
	// 配置格式支持: "80s", "1m" 或纯数字（秒）
	IdleTimeout string `ini:"idleTimeout"`

	// ShutdownTimeout 优雅关闭时等待活跃连接完成的最长时间，超时后强制关闭。
	// 配置格式支持: "30s", "1m" 或纯数字（秒）
	ShutdownTimeout string `ini:"shutdownTimeout"`
}

// GetReadHeaderTimeout 获取解析后的 ReadHeaderTimeout，默认 5s
func (o *ServerOptions) GetReadHeaderTimeout() (time.Duration, error) {
	return parseDuration(o.ReadHeaderTimeout, 5*time.Second)
}

// GetReadTimeout 获取解析后的 ReadTimeout，默认 10s
func (o *ServerOptions) GetReadTimeout() (time.Duration, error) {
	return parseDuration(o.ReadTimeout, 10*time.Second)
}

// GetWriteTimeout 获取解析后的 WriteTimeout，默认 30s
func (o *ServerOptions) GetWriteTimeout() (time.Duration, error) {
	return parseDuration(o.WriteTimeout, 30*time.Second)
}

// GetIdleTimeout 获取解析后的 IdleTimeout，默认 80s
func (o *ServerOptions) GetIdleTimeout() (time.Duration, error) {
	return parseDuration(o.IdleTimeout, 80*time.Second)
}

// GetShutdownTimeout 获取解析后的 ShutdownTimeout，默认 30s
func (o *ServerOptions) GetShutdownTimeout() (time.Duration, error) {
	return parseDuration(o.ShutdownTimeout, 30*time.Second)
}

func DefaultOptions() ServerOptions {
	return ServerOptions{
		Mode:              "release",
		Addr:              ":8088",
		ReadHeaderTimeout: "5s",
		ReadTimeout:       "10s",
		WriteTimeout:      "30s",
		IdleTimeout:       "80s",
		ShutdownTimeout:   "30s",
	}
}
