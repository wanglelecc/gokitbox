package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

func init() {
	stdBuilder.serverIp, _ = Extract("")
	stdBuilder.hostName, _ = os.Hostname()
}

func NewConfig() *Config {
	config := new(Config)
	//存储路径
	config.FileName = "./logs/app.log"
	//日志级别
	config.Level = "INFO"
	//日志标签 多日志时使用
	config.Tag = "default"
	//日志格式
	config.Format = ""
	//旧日志保留5个备份
	config.MaxBackups = "5"
	//日志最大保存MB
	config.MaxSize = "100"
	//最多保留30天日志 和MaxBackups参数配置1个就可以
	config.MaxAge = "0"
	// gzip包 默认false
	config.Compress = false
	//Console输出
	config.Console = false

	return config
}

// 自定义config Init
func InitWithConfig(config *Config) {
	if config.FileName == "" {
		fmt.Fprintf(os.Stderr, "InitLoggerConfig: Error: config could not found logpath %s\n", config.FileName)
		os.Exit(1)
	}

	stdBuilder.LoadConfig(config)
}

func NewContext() context.Context {
	return WithContext(context.Background())
}

func WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "request_time", time.Now().UnixNano()/1e6)
}

// Initialize is the error-returning alternative for application lifecycle owners.
func Initialize(config *Config) error {
	if config == nil || config.FileName == "" {
		return errors.New("logger: fileName is required")
	}
	copyConfig := *config
	copyConfig.Level = strings.ToUpper(config.Level)
	if copyConfig.Level == "WARN" {
		copyConfig.Level = "WARNING"
	}
	switch copyConfig.Level {
	case "DEBUG", "INFO", "WARNING", "ERROR", "DPANIC", "PANIC", "FATAL":
	default:
		return errors.New("logger: unsupported level")
	}
	stdBuilder.LoadConfig(&copyConfig)
	return nil
}
