# Bootstrap 启动引导

Bootstrap 提供应用启动和关闭时的初始化/清理功能，统一管理各组件的生命周期。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/bootstrap
```

## 功能特性

- 配置初始化
- 日志初始化
- 数据库连接初始化
- Redis 连接初始化
- 消息队列初始化
- 雪花 ID 初始化
- 优雅关闭清理

## 使用示例

### 前置初始化（Before）

```go
package main

import (
    "github.com/wanglelecc/gokitbox/bootstrap"
    "github.com/wanglelecc/gokitbox/config"
    "github.com/wanglelecc/gokitbox/httpserver"
)

func main() {
    s := httpserver.NewServer()
    
    // 注册前置初始化函数
    s.AddBeforeServerStartFunc(
        bootstrap.InitLogger("dev", "myApp", "myDepartment", "v1.0.0"),
        bootstrap.InitConfig(),      // 初始化配置
        bootstrap.InitDB(),          // 初始化数据库
        bootstrap.InitRedis(),       // 初始化 Redis
        bootstrap.InitProducer(),    // 初始化消息生产者
    )
    
    // 启动服务
    s.Serve()
}
```

### 后置清理（After）

```go
s.AddAfterServerStopFunc(
    bootstrap.CloseLogger(),      // 关闭日志
    bootstrap.CloseProducer(),    // 关闭消息生产者
)
```

## 可用函数

### 初始化函数（Before）

| 函数 | 说明 | 依赖配置 |
|------|------|----------|
| `InitConfig()` | 初始化配置系统 | - |
| `InitLogger(env, name, department, version)` | 初始化日志 | Log |
| `InitDB()` | 初始化数据库连接池 | MysqlCluster, MysqlConfig |
| `InitRedis()` | 初始化 Redis 连接池 | Redis, RedisConfig |
| `InitProducer()` | 初始化消息生产者 | MQProxy, KafkaProxy |
| `InitSnowflake(project, service)` | 初始化雪花 ID | - |

### 清理函数（After）

| 函数 | 说明 |
|------|------|
| `CloseLogger()` | 刷新并关闭日志 |
| `CloseProducer()` | 关闭消息生产者 |

## 配置示例

```ini
; 日志配置
[Log]
fileName = /home/logs/app.log
level = INFO
maxSize = 200
maxBackups = 5
console = true

; 数据库配置
[MysqlConfig]
showSql = false
showExecTime = false
slowDuration = 500
maxConn = 50
maxIdle = 30

[MysqlCluster]
gokit = gokit_rw:password@tcp(localhost:3306)/gokit gokit_ro:password@tcp(localhost:3306)/gokit

; Redis 配置
[Redis]
redis = localhost:6379

[RedisConfig]
redis.password =
redis.db = 0
redis.poolsize = 100
redis.minidleconns = 60
```
