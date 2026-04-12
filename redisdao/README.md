# RedisDao Redis 客户端

RedisDao 基于 `github.com/redis/go-redis/v9` 封装，提供 Redis 连接池管理和多实例支持。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/redisdao
```

## 特性

- 支持多实例管理
- 连接池自动维护
- 自动重连机制
- TLS 支持
- 完整的 Redis 命令支持（通过 go-redis）

## 配置

```ini
[Redis]
; 实例名 = 地址列表（空格分隔）
redis = 127.0.0.1:6379
; 支持多实例
; cache = 10.0.0.1:6379 10.0.0.2:6379
; session = 10.0.0.3:6379

[RedisConfig]
; 默认数据库
redis.db = 0
; 连接密码
redis.password =
; 连接池大小（默认 100）
redis.poolsize = 100
; 最小空闲连接数（默认 50）
redis.minidleconns = 60
; 读超时（秒，默认 5）
redis.readtimeout = 5
; 写超时（秒，默认 5）
redis.writetimeout = 5
; 最大重试次数（默认 0）
redis.maxretries = 3
; TLS 跳过证书验证（true/false）
redis.tlsinsecureskip = false
; 用户名（Redis 6.0+ ACL 认证）
redis.username =
```

## 使用示例

```go
package main

import (
    "context"
    "log"
    
    "github.com/redis/go-redis/v9"
    "github.com/wanglelecc/gokitbox/redisdao"
)

func main() {
    ctx := context.Background()
    
    // 获取 Redis 实例（参数为实例名）
    rdb := redisdao.NewSimpleRedis("redis")
    
    // 字符串操作
    err := rdb.Set(ctx, "key", "value", 0).Err()
    if err != nil {
        log.Fatal(err)
    }
    
    val, err := rdb.Get(ctx, "key").Result()
    if err != nil {
        log.Fatal(err)
    }
    log.Println("key:", val)
    
    // 判断 key 是否存在
    val2, err := rdb.Get(ctx, "key2").Result()
    if err == redis.Nil {
        log.Println("key2 does not exist")
    } else if err != nil {
        log.Fatal(err)
    } else {
        log.Println("key2:", val2)
    }
    
    // Hash 操作
    err = rdb.HSet(ctx, "user:1", "name", "张三", "age", 25).Err()
    if err != nil {
        log.Fatal(err)
    }
    
    user, err := rdb.HGetAll(ctx, "user:1").Result()
    if err != nil {
        log.Fatal(err)
    }
    log.Println("user:", user)
    
    // List 操作
    err = rdb.LPush(ctx, "queue", "task1", "task2").Err()
    if err != nil {
        log.Fatal(err)
    }
    
    task, err := rdb.RPop(ctx, "queue").Result()
    if err != nil {
        log.Fatal(err)
    }
    log.Println("task:", task)
    
    // Set 操作
    err = rdb.SAdd(ctx, "tags", "go", "redis", "database").Err()
    if err != nil {
        log.Fatal(err)
    }
    
    tags, err := rdb.SMembers(ctx, "tags").Result()
    if err != nil {
        log.Fatal(err)
    }
    log.Println("tags:", tags)
    
    // ZSet 操作（有序集合）
    err = rdb.ZAdd(ctx, "rank", redis.Z{Score: 100, Member: "player1"}).Err()
    if err != nil {
        log.Fatal(err)
    }
    
    rank, err := rdb.ZRevRangeWithScores(ctx, "rank", 0, 9).Result()
    if err != nil {
        log.Fatal(err)
    }
    log.Println("rank:", rank)
    
    // 设置过期时间
    err = rdb.Expire(ctx, "key", 60*time.Second).Err()
    if err != nil {
        log.Fatal(err)
    }
    
    // 关闭连接
    redisdao.Close()
}
```

## API 说明

```go
// 初始化 Redis 连接池（通常在 bootstrap 中调用）
func Init()

// 获取指定实例的 Redis 客户端
func NewSimpleRedis(instance string) *redis.Client

// 关闭所有 Redis 连接
func Close()
```

## 更多文档

- [go-redis 文档](https://redis.uptrace.dev/)
- [Redis 命令参考](https://redis.io/commands/)
