# Logger 日志组件

Logger 是基于 Zap 封装的高性能日志组件，支持文件输出、控制台输出、日志轮转和结构化日志。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/logger
```

## 特性

- 支持 5 级日志：DEBUG、INFO、WARN、ERROR、PANIC
- 文件自动轮转（按大小/时间）
- 支持控制台同时输出
- 结构化日志（JSON 格式）
- 自动日志压缩
- 链路追踪字段注入（trace_id、rpc_id 等）
- 自定义字段支持

## 配置

```ini
[Log]
; 日志输出文件路径
fileName = /home/logs/app.log
; 日志级别：DEBUG、INFO、WARN、ERROR、PANIC
level = INFO
; 单个日志文件最大大小（MB）
maxSize = 200
; 旧日志保留份数
maxBackups = 5
; 旧日志保留天数（与 maxBackups 二选一）
MaxAge = 0
; 是否输出到控制台
console = true
; 是否启用压缩
compress = true
; 环境标识（写入日志字段）
suffixEnv = dev
```

## 使用示例

```go
package main

import (
    "context"
    
    "github.com/wanglelecc/gokitbox/config"
    "github.com/wanglelecc/gokitbox/logger"
)

func main() {
    // 方法1：通过配置文件初始化
    cfgMap := config.GetConfStringMap("Log")
    logConfig := logger.NewConfig().SetConfigMap(cfgMap)
    
    // 设置基础信息
    logger.SetEnv("dev")
    logger.SetName("myApp")
    logger.SetDepartment("myDept")
    logger.SetVersion("v1.0.0")
    
    // 初始化
    logger.InitWithConfig(logConfig)
    defer logger.Sync()
    
    // 方法2：快捷初始化（通过配置）
    // logger.InitWithConf("Log", "dev", "myApp", "myDept", "v1.0.0")
    
    ctx := context.Background()
    
    // 基础日志（无前缀 X 方法）
    logger.Debug("调试信息")
    logger.Info("普通信息")
    logger.Warn("警告信息")
    logger.Error("错误信息")
    
    // 结构化日志（带 X 前缀方法）
    // 参数：上下文、标签、消息、键值对...
    logger.Dx(ctx, "db", "查询用户信息", "user_id", 123, "cost_ms", 50)
    logger.Ix(ctx, "http", "请求处理完成", "path", "/api/user", "status", 200)
    logger.Wx(ctx, "cache", "缓存未命中", "key", "user:123")
    logger.Ex(ctx, "order", "订单创建失败", "order_id", "ORD001", "error", "库存不足")
}
```

## 日志字段说明

```json
{
  "x_level": "info",
  "@timestamp": "2024-01-15T10:30:00.000+0800",
  "x_caller": "main.go:25",
  "x_msg": "请求处理完成",
  "x_env": "dev",
  "x_name": "myApp",
  "x_version": "v1.0.0",
  "x_department": "myDept",
  "x_server_ip": "192.168.1.100",
  "x_host_name": "server-01",
  "x_rpc_id": "1.1",
  "x_trace_id": "abc123def456",
  "x_timestamp": 1705287000,
  "x_duration": 0.045,
  "x_tag": "http",
  "path": "/api/user",
  "status": 200
}
```

## 链路追踪

Logger 自动从 context 中提取以下字段：
- `trace_id` - 全局追踪 ID
- `rpc_id` - RPC 层级 ID（用于追踪调用链）
- `log_id` - 日志 ID（每次请求唯一）

## API 说明

```go
// 基础日志
func Debug(msg string, fields ...interface{})
func Info(msg string, fields ...interface{})
func Warn(msg string, fields ...interface{})
func Error(msg string, fields ...interface{})
func Panic(msg string, fields ...interface{})

// 结构化日志（推荐）
func Dx(ctx context.Context, tag, msg string, fields ...interface{})
func Ix(ctx context.Context, tag, msg string, fields ...interface{})
func Wx(ctx context.Context, tag, msg string, fields ...interface{})
func Ex(ctx context.Context, tag, msg string, fields ...interface{})
func Px(ctx context.Context, tag, msg string, fields ...interface{})

// 初始化
func InitWithConfig(config *Config)
func InitWithConf(section, env, name, department, version string)
func Sync() error

// 设置基础信息
func SetEnv(env string)
func SetName(name string)
func SetDepartment(dept string)
func SetVersion(version string)

// 生成 ID
func GenTraceId() string
func GenLoggerId() int64
```
