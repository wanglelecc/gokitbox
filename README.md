# Gokitbox

Gokitbox 是一个基于 Go 语言的基础工具包集合，封装了 Web 服务开发中常用的组件，帮助快速构建高性能、可维护的微服务应用。

## 特性

- **模块化设计**：各组件独立，按需引用
- **统一配置管理**：支持 INI/YAML，一次加载全局使用
- **完整的服务治理**：HTTP/RPC 服务、服务注册发现、消息队列
- **丰富的工具函数**：字符串、日期、加密、校验等常用工具
- **生产就绪**：日志轮转、连接池、熔断、限流等生产级特性

## 安装

```shell
go get github.com/wanglelecc/gokitbox
```

## 模块概览

| 模块 | 路径 | 说明 | 状态 |
|------|------|------|------|
| `config` | `github.com/wanglelecc/gokitbox/config` | 配置管理（INI/YAML） | 可用 |
| `logger` | `github.com/wanglelecc/gokitbox/logger` | 结构化日志（Zap） | 可用 |
| `httpserver` | `github.com/wanglelecc/gokitbox/httpserver` | HTTP 服务（Gin） | 可用 |
| `rpcxserver` | `github.com/wanglelecc/gokitbox/rpcxserver` | RPC 服务（rpcx） | 可用 |
| `rpcxclient` | `github.com/wanglelecc/gokitbox/rpcxclient` | RPC 客户端 | 可用 |
| `dbdao` | `github.com/wanglelecc/gokitbox/dbdao` | 数据库访问（XORM） | 可用 |
| `redisdao` | `github.com/wanglelecc/gokitbox/redisdao` | Redis 客户端 | 可用 |
| `producer` | `github.com/wanglelecc/gokitbox/producer` | 消息生产者（Kafka） | 可用 |
| `worker` | `github.com/wanglelecc/gokitbox/worker` | 消息消费者 | 可用 |
| `event` | `github.com/wanglelecc/gokitbox/event` | 事件分发器 | 可用 |
| `bootstrap` | `github.com/wanglelecc/gokitbox/bootstrap` | 启动引导 | 可用 |
| `tools` | `github.com/wanglelecc/gokitbox/tools` | 工具函数集合 | 可用 |

## 快速开始

### 1. HTTP 服务

```go
package main

import (
    "github.com/wanglelecc/gokitbox/bootstrap"
    "github.com/wanglelecc/gokitbox/httpserver"
    "github.com/wanglelecc/gokitbox/httpserver/middleware"
)

func main() {
    s := httpserver.NewServer()
    
    // 注册中间件
    s.UseMiddleware(
        middleware.Logger(middleware.CheckCode0),
        middleware.Recovery(),
        middleware.Context(),
    )
    
    // 注册初始化函数
    s.AddBeforeServerStartFunc(
        bootstrap.InitLogger("dev", "myApp", "myDept", "v1.0.0"),
        bootstrap.InitConfig(),
    )
    
    // 注册路由
    s.GinEngine().GET("/hello", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "hello"})
    })
    
    // 启动服务
    s.Serve()
}
```

### 2. 配置读取

```go
import "github.com/wanglelecc/gokitbox/config"

// 获取配置
name := config.GetConf("app", "name")
port := config.GetConfDefault("server", "port", "8080")

// 获取数组
hosts := config.GetConfs("database", "hosts")

// 映射到结构体
var cfg ServerConfig
config.ConfMapToStruct("server", &cfg)
```

### 3. 数据库操作

```go
import "github.com/wanglelecc/gokitbox/dbdao"

// 初始化
dbdao.Init()

// 获取实例
db := dbdao.GetDbInstance("gokit")

// 主库写
master := db.Engine.Master()
affected, err := master.Insert(&user)

// 从库读
slave := db.Engine.Slave()
has, err := slave.Get(&user)
```

### 4. Redis 操作

```go
import (
    "context"
    "github.com/wanglelecc/gokitbox/redisdao"
)

ctx := context.Background()
rdb := redisdao.NewSimpleRedis("redis")

// 基本操作
rdb.Set(ctx, "key", "value", 0)
val, err := rdb.Get(ctx, "key").Result()
```

### 5. 日志记录

```go
import (
    "context"
    "github.com/wanglelecc/gokitbox/logger"
)

ctx := context.Background()

// 结构化日志
logger.Ix(ctx, "http", "请求处理完成", "path", "/api/user", "status", 200)
logger.Ex(ctx, "order", "订单处理失败", "order_id", "123", "error", err.Error())
```

### 6. 雪花 ID

```go
import (
    "context"
    "github.com/wanglelecc/gokitbox/tools/uSnowflake"
)

// 初始化
ctx := context.Background()
uSnowflake.InitSnowflake(ctx, "my_project", "order_service")

// 生成 ID
id := uSnowflake.NewIdInt64()   // int64
idStr := uSnowflake.NewIdString() // string
```

## 配置文件示例

```ini
; 应用配置
[App]
env = dev
name = myApp
version = v1.0.0

; HTTP 服务
[HttpServer]
addr = :8088
mode = release
readTimeout = 10s
writeTimeout = 30s

; 日志
[Log]
fileName = /home/logs/app.log
level = INFO
maxSize = 200
console = true

; 数据库
[MysqlConfig]
showSql = false
maxConn = 50
maxIdle = 30

[MysqlCluster]
gokit = gokit_rw:pass@tcp(localhost:3306)/gokit gokit_ro:pass@tcp(localhost:3306)/gokit

; Redis
[Redis]
redis = localhost:6379

[RedisConfig]
redis.db = 0
redis.poolsize = 100

; RPC 注册中心
[Registry]
addrs = localhost:2181
basePath = /myapp
```

## RPC 服务构建标签

`rpcxserver` 和 `rpcxclient` 支持多种注册中心，需要通过 build tags 指定：

```shell
# ZooKeeper
go build -tags zookeeper ./...

# Etcd
go build -tags etcd ./...

# Redis
go build -tags redis ./...

# Consul
go build -tags consul ./...
```

## 目录结构

```
gokitbox/
├── bootstrap/      # 启动引导组件
├── config/         # 配置管理
├── dbdao/          # 数据库访问
├── event/          # 事件分发器
├── httpserver/     # HTTP 服务
├── logger/         # 日志组件
├── producer/       # 消息生产者
├── redisdao/       # Redis 客户端
├── rpcxclient/     # RPC 客户端
├── rpcxserver/     # RPC 服务
├── tools/          # 工具函数集合
│   ├── uAddress/   # 网络地址
│   ├── uConvert/   # 类型转换
│   ├── uCrypto/    # 加密解密
│   ├── uDate/      # 日期时间
│   ├── uHash/      # 哈希算法
│   ├── uMath/      # 数学计算
│   ├── uOs/        # 文件操作
│   ├── uRand/      # 随机数
│   ├── uSlice/     # 切片操作
│   ├── uSnowflake/ # 雪花 ID
│   ├── uString/    # 字符串处理
│   └── uVerify/    # 数据校验
└── worker/         # 消息消费者
```

## 依赖要求

- Go 1.26+
- MySQL 5.7+（dbdao）
- Redis 5.0+（redisdao, uSnowflake）
- Kafka 2.0+（producer, worker）
- ZooKeeper 3.5+ / Etcd 3.0+ / Redis / Consul（服务注册发现）

## 许可证

MIT License

## 作者

wanglelecc
