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

### 基础接口（连接管理）

```go
// 初始化 Redis 连接池（通常在 bootstrap 中调用）
func Init()

// 获取指定实例的 Redis 客户端
func NewSimpleRedis(instance string) *redis.Client

// 关闭所有 Redis 连接
func Close()
```

### 快捷操作接口（推荐）

```go
// 创建 Redis 快捷操作实例（方式一：默认实例）
rdb := redisdao.DefaultRedis()

// 创建 Redis 快捷操作实例（方式二：指定实例）
rdb := redisdao.NewRedis("cache")

// String 操作
val, err := rdb.Get(ctx, "key")
err = rdb.Set(ctx, "key", "value", 60*time.Second)
ok, err := rdb.SetNX(ctx, "lock:key", "1", 30*time.Second) // 分布式锁

// Hash 操作
err = rdb.HSet(ctx, "user:1", "name", "张三", "age", 25)
profile, err := rdb.HGetAll(ctx, "user:1")

// JSON 操作（自动序列化/反序列化）
err = rdb.SetJSON(ctx, "user:1", user, 60*time.Second)
var user User
err = rdb.GetJSON(ctx, "user:1", &user)

// 分布式锁
ok, err := rdb.TryLock(ctx, "lock:order:123", 30*time.Second)
if ok {
    defer rdb.Unlock(ctx, "lock:order:123")
    // 执行业务逻辑
}
```

### redis.Nil 处理

快捷操作默认将 `redis.Nil`（key 不存在）转换为返回空值，简化错误处理：

```go
// 简洁方式（推荐）：key 不存在返回空字符串，无需判断 redis.Nil
val, err := rdb.Get(ctx, "key")
// err 只包含真正的错误（如网络故障）

// 如需区分 key 不存在，使用 E 后缀方法
val, err := rdb.GetE(ctx, "key")
if redisdao.IsNotFound(err) {
    // key 不存在
} else if err != nil {
    // 其他错误
}
```

支持的方法对照：

| 简洁方法 | 完整方法 | 说明 |
|---------|---------|------|
| `Get` | `GetE` | 字符串获取 |
| `GetBytes` | `GetBytesE` | 字节获取 |
| `HGet` | `HGetE` | Hash 字段获取 |
| `LPop` | `LPopE` | 左侧弹出 |
| `RPop` | `RPopE` | 右侧弹出 |
| `ZScore` | `ZScoreE` | 有序集合分数 |

**完整快捷操作列表**：

| 类别 | 方法 | 说明 |
|------|------|------|
| **String** |||
| | `Get/GetE` | 获取字符串值（简洁/完整） |
| | `GetBytes/GetBytesE` | 获取字节值 |
| | `Set/SetNX` | 设置/不存在才设置 |
| | `Del/Exists` | 删除/检查存在 |
| | `Expire/TTL` | 设置/获取过期时间 |
| | `Incr/IncrBy/Decr/DecrBy` | 原子自增自减 |
| | `StrLen/Append` | 长度/追加 |
| | `GetSet/GetRange/SetRange` | 获取并设置/子串操作 |
| | `MGet/MSet/MSetNX` | 批量操作 |
| **Hash** |||
| | `HGet/HGetE/HGetAll` | 获取字段/全部 |
| | `HSet/HDel` | 设置/删除字段 |
| | `HExists` | 检查字段存在 |
| | `HIncrBy/HIncrByFloat` | 字段自增（整数/浮点） |
| | `HLen` | 字段数量 |
| | `HMGet/HMSet` | 批量获取/设置 |
| | `HKeys/HVals` | 获取所有字段名/值 |
| **List** |||
| | `LPush/RPush/LPushX/RPushX` | 推入元素（列表存在时才推入） |
| | `LPop/LPopE/RPop/RPopE` | 弹出元素（简洁/完整） |
| | `BLPop` | 阻塞式弹出 |
| | `LLen/LRange` | 长度/范围获取 |
| | `LIndex/LInsert` | 索引获取/插入 |
| | `LRem/LSet/LTrim` | 移除/设置/修剪 |
| | `RPopLPush/BRPopLPush` | 弹出并推入 |
| **Set** |||
| | `SAdd/SRem` | 添加/移除元素 |
| | `SMembers/SCard` | 获取全部/数量 |
| | `SIsMember` | 检查元素存在 |
| | `SPop/SPopOne` | 随机移除并返回 |
| | `SRandMember/SRandMemberOne` | 随机返回（不移除） |
| | `SInter/SUnion/SDiff` | 交集/并集/差集 |
| | `SInterStore/SUnionStore/SDiffStore` | 集合运算并存储 |
| **ZSet** |||
| | `ZAdd/ZRem/ZCard` | 添加/移除/数量 |
| | `ZRange/ZRevRange` | 范围获取（低到高/高到低） |
| | `ZRangeWithScores/ZRevRangeWithScores` | 带分数范围获取 |
| | `ZRangeByScore/ZRevRangeByScore` | 按分数范围获取 |
| | `ZRangeByScoreWithScores/ZRevRangeByScoreWithScores` | 按分数范围带分数获取 |
| | `ZScore/ZScoreE` | 获取分数（简洁/完整） |
| | `ZRank/ZRevRank` | 获取排名 |
| | `ZCount` | 分数范围内数量 |
| | `ZIncrBy` | 分数自增 |
| | `ZRemRangeByRank/ZRemRangeByScore` | 按排名/分数移除 |
| | `ZPopMin/ZPopMax` | 弹出并返回最小/最大成员 |
| | `BZPopMin/BZPopMax` | 阻塞式弹出最小/最大成员 |
| **Key** |||
| | `Keys/Scan` | 查找 key（Scan 推荐用于大数据量） |
| | `Type` | 获取 key 类型 |
| | `Rename/RenameNX` | 重命名 |
| | `Persist/PExpire/PTTL` | 持久化/毫秒级过期时间 |
| **Server** |||
| | `DBSize` | key 数量 |
| | `FlushDB/FlushAll` | 清空数据库 |
| | `Ping` | 检查连接 |
| **JSON** |||
| | `GetJSON/SetJSON` | JSON 自动序列化/反序列化 |
| **Lock** |||
| | `TryLock/Unlock` | 分布式锁（简单实现） |
| **Batch** |||
| | `MGet/MSet/MSetNX` | 批量操作 |
| | `Pipeline/TxPipeline` | 管道/事务管道 |
| **Client** |||
| | `Client()` | 获取原始 *redis.Client |

**快捷操作特点**：
- 自动处理 `redis.Nil` 错误（返回空值而非 error）
- 内置 JSON 序列化/反序列化
- 简化分布式锁使用
- 保留原始客户端访问能力

## 更多文档

- [go-redis 文档](https://redis.uptrace.dev/)
- [Redis 命令参考](https://redis.io/commands/)
