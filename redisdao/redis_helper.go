package redisdao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 快捷操作封装，简化常用 Redis 操作
type Redis struct {
	client *redis.Client
}

// NewRedis 创建 Redis 快捷操作实例（使用指定实例名）
//
// @param instance: Redis 实例名，对应配置文件中的 [Redis] 下配置的实例名，如 "redis", "cache", "session" 等
//
// @return *Redis: Redis 快捷操作实例
//
// 使用示例：
//
//	// 获取名为 "cache" 的 Redis 实例
//	rdb := redisdao.NewRedis("cache")
//	val, err := rdb.Get(ctx, "key")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewRedis(instance string) *Redis {
	return &Redis{
		client: NewSimpleRedis(instance),
	}
}

// DefaultRedis 创建 Redis 快捷操作实例（使用默认 "redis" 实例）
// 这是大多数项目的常用写法，简化调用
//
// @return *Redis: Redis 快捷操作实例（使用默认 "redis" 实例）
//
// 使用示例：
//
//	// 获取默认 Redis 实例（最常见用法）
//	rdb := redisdao.DefaultRedis()
//	err := rdb.Set(ctx, "user:1", "zhangsan", 60*time.Second)
//	val, err := rdb.Get(ctx, "user:1")
func DefaultRedis() *Redis {
	return &Redis{
		client: NewSimpleRedis("redis"),
	}
}

// IsNotFound 检查错误是否为 key 不存在（redis.Nil）
// 用于需要区分 key 不存在和其他错误的情况
// 使用 errors.Is 支持错误包装场景
//
// @param err: 错误对象
//
// @return bool: 如果是 redis.Nil 返回 true，否则返回 false
//
// 使用示例：
//
//	val, err := rdb.GetE(ctx, "key")
//	if redisdao.IsNotFound(err) {
//	    // key 不存在，使用默认值
//	    val = "default_value"
//	} else if err != nil {
//	    // 其他错误（网络故障、超时等）
//	    return err
//	}
func IsNotFound(err error) bool {
	return errors.Is(err, redis.Nil)
}

// ==================== String 操作 ====================

// Get 获取字符串值
// key 不存在时返回空字符串和 nil error（简化错误处理）
// 如需区分 key 不存在，使用 GetE 或 IsNotFound 判断
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return string: 键值，key 不存在时返回空字符串
// @return error: 错误信息，key 不存在时为 nil
//
// 使用示例：
//
//	val, err := rdb.Get(ctx, "user:name")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if val == "" {
//	    // key 不存在
//	}
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// GetE 获取字符串值（返回完整错误）
// key 不存在时返回 redis.Nil，调用方可通过 IsNotFound(err) 判断
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return string: 键值
// @return error: 错误信息，key 不存在时返回 redis.Nil
//
// 使用示例：
//
//	val, err := rdb.GetE(ctx, "user:name")
//	if redisdao.IsNotFound(err) {
//	    // key 不存在，初始化
//	    err = rdb.Set(ctx, "user:name", "default", 60*time.Second)
//	} else if err != nil {
//	    // 其他错误
//	    log.Fatal(err)
//	}
func (r *Redis) GetE(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// GetBytes 获取字节值
// key 不存在时返回 nil 和 nil error
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return []byte: 键值，key 不存在时返回 nil
// @return error: 错误信息，key 不存在时为 nil
//
// 使用示例：
//
//	data, err := rdb.GetBytes(ctx, "binary:data")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if data == nil {
//	    // key 不存在
//	}
func (r *Redis) GetBytes(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

// GetBytesE 获取字节值（返回完整错误）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return []byte: 键值
// @return error: 错误信息，key 不存在时返回 redis.Nil
//
// 使用示例：
//
//	data, err := rdb.GetBytesE(ctx, "binary:data")
//	if redisdao.IsNotFound(err) {
//	    // key 不存在
//	} else if err != nil {
//	    log.Fatal(err)
//	}
func (r *Redis) GetBytesE(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

// Set 设置字符串值
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param value: 键值，可以是字符串、数字、字节数组等
// @param expiration: 过期时间，0 表示永不过期
//
// @return error: 错误信息
//
// 使用示例：
//
//	// 设置永不过期的值
//	err := rdb.Set(ctx, "config:version", "v1.0.0", 0)
//
//	// 设置 60 秒后过期
//	err := rdb.Set(ctx, "session:123", "user_data", 60*time.Second)
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// SetNX 仅在 key 不存在时设置（分布式锁常用）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param value: 键值
// @param expiration: 过期时间，必须设置以防止死锁
//
// @return bool: 设置成功返回 true，key 已存在返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	// 尝试获取分布式锁
//	ok, err := rdb.SetNX(ctx, "lock:order:123", "1", 30*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if ok {
//	    // 获取锁成功，执行业务逻辑
//	    defer rdb.Del(ctx, "lock:order:123")
//	} else {
//	    // 锁已被占用
//	}
func (r *Redis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// Del 删除 key
//
// @param ctx: 上下文，用于控制超时和取消
// @param keys: 要删除的键名列表，支持删除多个
//
// @return error: 错误信息
//
// 使用示例：
//
//	// 删除单个 key
//	err := rdb.Del(ctx, "user:name")
//
//	// 删除多个 key
//	err := rdb.Del(ctx, "user:1", "user:2", "user:3")
func (r *Redis) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查 key 是否存在
//
// @param ctx: 上下文，用于控制超时和取消
// @param keys: 要检查的键名列表
//
// @return bool: 至少有一个 key 存在返回 true，都不存在返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	exists, err := rdb.Exists(ctx, "user:1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if exists {
//	    // key 存在
//	}
func (r *Redis) Exists(ctx context.Context, keys ...string) (bool, error) {
	n, err := r.client.Exists(ctx, keys...).Result()
	return n > 0, err
}

// Expire 设置过期时间
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param expiration: 过期时间
//
// @return error: 错误信息
//
// 使用示例：
//
//	// 设置 5 分钟后过期
//	err := rdb.Expire(ctx, "session:123", 5*time.Minute)
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL 获取剩余过期时间
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return time.Duration: 剩余过期时间，key 不存在返回 -2，永不过期返回 -1
// @return error: 错误信息
//
// 使用示例：
//
//	ttl, err := rdb.TTL(ctx, "session:123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if ttl > 0 {
//	    fmt.Printf("还有 %v 过期\n", ttl)
//	}
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Incr 原子自增
// 将 key 中储存的数字值增一。如果 key 不存在，那么 key 的值会先被初始化为 0，然后再执行 INCR 操作
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return int64: 自增后的值
// @return error: 错误信息
//
// 使用示例：
//
//	// 计数器 +1
//	newVal, err := rdb.Incr(ctx, "counter:visits")
//	// newVal = 自增后的值
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// IncrBy 原子自增指定值
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param value: 增量（可以是负数实现减法）
//
// @return int64: 自增后的值
// @return error: 错误信息
//
// 使用示例：
//
//	// 增加 100
//	newVal, err := rdb.IncrBy(ctx, "user:1:credits", 100)
//	// 减少 50（传入负数）
//	newVal, err := rdb.IncrBy(ctx, "user:1:credits", -50)
func (r *Redis) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

// Decr 原子自减
// 将 key 中储存的数字值减一。如果 key 不存在，那么 key 的值会先被初始化为 0，然后再执行 DECR 操作
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return int64: 自减后的值
// @return error: 错误信息
//
// 使用示例：
//
//	newVal, err := rdb.Decr(ctx, "inventory:sku123")
func (r *Redis) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

// DecrBy 原子自减指定值
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param decrement: 减量（可以是负数实现加法）
//
// @return int64: 自减后的值
// @return error: 错误信息
//
// 使用示例：
//
//	// 减少 10
//	newVal, err := rdb.DecrBy(ctx, "inventory:sku123", 10)
func (r *Redis) DecrBy(ctx context.Context, key string, decrement int64) (int64, error) {
	return r.client.DecrBy(ctx, key, decrement).Result()
}

// StrLen 获取字符串长度
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return int64: 字符串长度，key 不存在返回 0
// @return error: 错误信息
//
// 使用示例：
//
//	length, err := rdb.StrLen(ctx, "article:1:content")
func (r *Redis) StrLen(ctx context.Context, key string) (int64, error) {
	return r.client.StrLen(ctx, key).Result()
}

// Append 追加字符串
// 如果 key 已经存在并且是一个字符串，APPEND 命令将 value 追加到 key 原来的值的末尾
// 如果 key 不存在，APPEND 就简单地将给定 key 设为 value
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param value: 要追加的值
//
// @return int64: 追加后字符串的长度
// @return error: 错误信息
//
// 使用示例：
//
//	// 追加日志
//	length, err := rdb.Append(ctx, "log:20240101", "new log line\n")
func (r *Redis) Append(ctx context.Context, key, value string) (int64, error) {
	return r.client.Append(ctx, key, value).Result()
}

// GetSet 获取旧值并设置新值
// 将给定 key 的值设为 value，并返回 key 的旧值
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param value: 新值
//
// @return string: 旧值，key 不存在返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取旧值并更新
//	oldVal, err := rdb.GetSet(ctx, "config:version", "v2.0.0")
//	if oldVal == "" {
//	    fmt.Println("这是第一次设置")
//	} else {
//	    fmt.Printf("版本从 %s 更新到 v2.0.0\n", oldVal)
//	}
func (r *Redis) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	val, err := r.client.GetSet(ctx, key, value).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// GetRange 获取子字符串
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param start: 开始位置（包含，0 表示第一个字符）
// @param end: 结束位置（包含，-1 表示最后一个字符）
//
// @return string: 子字符串
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取前 100 个字符
//	prefix, err := rdb.GetRange(ctx, "article:1:content", 0, 99)
//	// 获取最后 50 个字符
//	suffix, err := rdb.GetRange(ctx, "article:1:content", -50, -1)
func (r *Redis) GetRange(ctx context.Context, key string, start, end int64) (string, error) {
	return r.client.GetRange(ctx, key, start, end).Result()
}

// SetRange 设置子字符串
// 用 value 参数覆写给定 key 所储存的字符串值，从偏移量 offset 开始
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param offset: 偏移量（从 0 开始）
// @param value: 要设置的值
//
// @return int64: 被修改后的字符串长度
// @return error: 错误信息
//
// 使用示例：
//
//	// 从第 10 个字符开始替换
//	length, err := rdb.SetRange(ctx, "template", 10, "REPLACED")
func (r *Redis) SetRange(ctx context.Context, key string, offset int64, value string) (int64, error) {
	return r.client.SetRange(ctx, key, offset, value).Result()
}

// MSetNX 批量设置（仅当所有 key 都不存在时才设置）
//
// @param ctx: 上下文，用于控制超时和取消
// @param values: 键值对列表，如 "key1", "value1", "key2", "value2"
//
// @return bool: 所有 key 都不存在并设置成功返回 true，否则返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	ok, err := rdb.MSetNX(ctx, "user:1:name", "张三", "user:1:age", "25")
//	if ok {
//	    // 所有 key 都是新设置的
//	} else {
//	    // 至少有一个 key 已存在，没有任何 key 被设置
//	}
func (r *Redis) MSetNX(ctx context.Context, values ...interface{}) (bool, error) {
	return r.client.MSetNX(ctx, values...).Result()
}

// ==================== Hash 操作 ====================

// HGet 获取 hash 字段值
// field 不存在时返回空字符串和 nil error
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param field: 字段名
//
// @return string: 字段值，field 不存在返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	name, err := rdb.HGet(ctx, "user:1", "name")
//	if name == "" {
//	    // field 不存在
//	}
func (r *Redis) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := r.client.HGet(ctx, key, field).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// HGetE 获取 hash 字段值（返回完整错误）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param field: 字段名
//
// @return string: 字段值
// @return error: 错误信息，field 不存在返回 redis.Nil
//
// 使用示例：
//
//	name, err := rdb.HGetE(ctx, "user:1", "name")
//	if redisdao.IsNotFound(err) {
//	    // field 不存在
//	}
func (r *Redis) HGetE(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, key, field).Result()
}

// HGetAll 获取整个 hash
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
//
// @return map[string]string: 所有字段和值，key 不存在返回空 map
// @return error: 错误信息
//
// 使用示例：
//
//	user, err := rdb.HGetAll(ctx, "user:1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(user["name"], user["age"])
func (r *Redis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// HSet 设置 hash 字段
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param values: 字段和值的列表，如 "name", "张三", "age", 25
//
// @return error: 错误信息
//
// 使用示例：
//
//	// 设置单个字段
//	err := rdb.HSet(ctx, "user:1", "name", "张三")
//
//	// 设置多个字段
//	err := rdb.HSet(ctx, "user:1", "name", "张三", "age", 25, "city", "北京")
func (r *Redis) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.HSet(ctx, key, values...).Err()
}

// HDel 删除 hash 字段
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param fields: 要删除的字段名列表
//
// @return error: 错误信息
//
// 使用示例：
//
//	err := rdb.HDel(ctx, "user:1", "temp_field", "old_field")
func (r *Redis) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

// HExists 检查 hash 字段是否存在
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param field: 字段名
//
// @return bool: 字段存在返回 true，否则返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	exists, err := rdb.HExists(ctx, "user:1", "name")
func (r *Redis) HExists(ctx context.Context, key, field string) (bool, error) {
	return r.client.HExists(ctx, key, field).Result()
}

// HIncrBy hash 字段原子自增
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param field: 字段名
// @param increment: 增量
//
// @return int64: 自增后的值
// @return error: 错误信息
//
// 使用示例：
//
//	newVal, err := rdb.HIncrBy(ctx, "user:1", "visit_count", 1)
func (r *Redis) HIncrBy(ctx context.Context, key, field string, increment int64) (int64, error) {
	return r.client.HIncrBy(ctx, key, field, increment).Result()
}

// HIncrByFloat hash 字段原子自增（浮点数）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param field: 字段名
// @param increment: 增量（浮点数）
//
// @return float64: 自增后的值
// @return error: 错误信息
//
// 使用示例：
//
//	newVal, err := rdb.HIncrByFloat(ctx, "product:1", "price", 9.99)
func (r *Redis) HIncrByFloat(ctx context.Context, key, field string, increment float64) (float64, error) {
	return r.client.HIncrByFloat(ctx, key, field, increment).Result()
}

// HLen 获取 hash 字段数量
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
//
// @return int64: 字段数量，key 不存在返回 0
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.HLen(ctx, "user:1")
func (r *Redis) HLen(ctx context.Context, key string) (int64, error) {
	return r.client.HLen(ctx, key).Result()
}

// HMGet 批量获取 hash 字段
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param fields: 字段名列表
//
// @return []interface{}: 字段值列表，field 不存在时对应位置为 nil
// @return error: 错误信息
//
// 使用示例：
//
//	values, err := rdb.HMGet(ctx, "user:1", "name", "age", "email")
//	// values[0] = name 的值
//	// values[1] = age 的值
//	// values[2] = email 的值（可能为 nil）
func (r *Redis) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	return r.client.HMGet(ctx, key, fields...).Result()
}

// HMSet 批量设置 hash 字段（已废弃，使用 HSet）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
// @param values: 字段和值的列表
//
// @return error: 错误信息
//
// 使用示例：
//
//	err := rdb.HMSet(ctx, "user:1", "name", "张三", "age", 25)
func (r *Redis) HMSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.HMSet(ctx, key, values...).Err()
}

// HKeys 获取所有字段名
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
//
// @return []string: 字段名列表
// @return error: 错误信息
//
// 使用示例：
//
//	fields, err := rdb.HKeys(ctx, "user:1")
//	// fields = ["name", "age", "city"]
func (r *Redis) HKeys(ctx context.Context, key string) ([]string, error) {
	return r.client.HKeys(ctx, key).Result()
}

// HVals 获取所有字段值
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: hash 键名
//
// @return []string: 字段值列表
// @return error: 错误信息
//
// 使用示例：
//
//	values, err := rdb.HVals(ctx, "user:1")
//	// values = ["张三", "25", "北京"]
func (r *Redis) HVals(ctx context.Context, key string) ([]string, error) {
	return r.client.HVals(ctx, key).Result()
}

// ==================== List 操作 ====================

// LPush 从左侧推入元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param values: 要推入的元素列表
//
// @return int64: 列表长度
// @return error: 错误信息
//
// 使用示例：
//
//	length, err := rdb.LPush(ctx, "queue:tasks", "task1", "task2", "task3")
func (r *Redis) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return r.client.LPush(ctx, key, values...).Result()
}

// RPush 从右侧推入元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param values: 要推入的元素列表
//
// @return int64: 列表长度
// @return error: 错误信息
//
// 使用示例：
//
//	length, err := rdb.RPush(ctx, "queue:tasks", "task1", "task2")
func (r *Redis) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return r.client.RPush(ctx, key, values...).Result()
}

// LPop 从左侧弹出元素
// list 为空时返回空字符串和 nil error
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
//
// @return string: 弹出的元素，列表为空返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	task, err := rdb.LPop(ctx, "queue:tasks")
//	if task == "" {
//	    // 队列为空
//	}
func (r *Redis) LPop(ctx context.Context, key string) (string, error) {
	val, err := r.client.LPop(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// LPopE 从左侧弹出元素（返回完整错误）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
//
// @return string: 弹出的元素
// @return error: 错误信息，列表为空返回 redis.Nil
//
// 使用示例：
//
//	task, err := rdb.LPopE(ctx, "queue:tasks")
//	if redisdao.IsNotFound(err) {
//	    // 队列为空
//	}
func (r *Redis) LPopE(ctx context.Context, key string) (string, error) {
	return r.client.LPop(ctx, key).Result()
}

// RPop 从右侧弹出元素
// list 为空时返回空字符串和 nil error
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
//
// @return string: 弹出的元素，列表为空返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	task, err := rdb.RPop(ctx, "queue:tasks")
func (r *Redis) RPop(ctx context.Context, key string) (string, error) {
	val, err := r.client.RPop(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// RPopE 从右侧弹出元素（返回完整错误）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
//
// @return string: 弹出的元素
// @return error: 错误信息，列表为空返回 redis.Nil
func (r *Redis) RPopE(ctx context.Context, key string) (string, error) {
	return r.client.RPop(ctx, key).Result()
}

// BLPop 阻塞式左侧弹出（带超时）
// 如果列表为空，会阻塞等待直到有元素或超时
//
// @param ctx: 上下文，用于控制超时和取消
// @param timeout: 最大阻塞时间
// @param keys: 要监听的列表键名列表（支持多个列表）
//
// @return []string: [列表名, 弹出的元素]，超时返回空切片
// @return error: 错误信息
//
// 使用示例：
//
//	result, err := rdb.BLPop(ctx, 30*time.Second, "queue:tasks", "queue:backup")
//	if len(result) > 0 {
//	    listName := result[0]
//	    task := result[1]
//	}
func (r *Redis) BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return r.client.BLPop(ctx, timeout, keys...).Result()
}

// LLen 获取列表长度
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
//
// @return int64: 列表长度，key 不存在返回 0
// @return error: 错误信息
//
// 使用示例：
//
//	length, err := rdb.LLen(ctx, "queue:tasks")
func (r *Redis) LLen(ctx context.Context, key string) (int64, error) {
	return r.client.LLen(ctx, key).Result()
}

// LRange 获取列表范围元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param start: 开始索引（0 表示第一个，-1 表示最后一个）
// @param stop: 结束索引（包含，-1 表示到最后一个）
//
// @return []string: 范围内的元素列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取前 10 个元素
//	items, err := rdb.LRange(ctx, "queue:tasks", 0, 9)
//	// 获取全部元素
//	items, err := rdb.LRange(ctx, "queue:tasks", 0, -1)
func (r *Redis) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
}

// LIndex 获取指定索引元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param index: 索引（0 表示第一个，-1 表示最后一个）
//
// @return string: 索引位置的元素，索引越界返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取第一个元素
//	first, err := rdb.LIndex(ctx, "queue:tasks", 0)
//	// 获取最后一个元素
//	last, err := rdb.LIndex(ctx, "queue:tasks", -1)
func (r *Redis) LIndex(ctx context.Context, key string, index int64) (string, error) {
	val, err := r.client.LIndex(ctx, key, index).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// LInsert 在指定元素前后插入
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param op: 插入位置，"BEFORE" 或 "AFTER"
// @param pivot: 参考元素
// @param value: 要插入的值
//
// @return int64: 插入后列表长度，pivot 不存在返回 -1
// @return error: 错误信息
//
// 使用示例：
//
//	// 在 "task2" 前面插入 "task1.5"
//	length, err := rdb.LInsert(ctx, "queue:tasks", "BEFORE", "task2", "task1.5")
func (r *Redis) LInsert(ctx context.Context, key, op string, pivot, value interface{}) (int64, error) {
	return r.client.LInsert(ctx, key, op, pivot, value).Result()
}

// LRem 移除指定元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param count: 移除数量（0 表示移除所有，正数表示从左侧开始移除指定数量，负数表示从右侧）
// @param value: 要移除的元素值
//
// @return int64: 实际移除的元素数量
// @return error: 错误信息
//
// 使用示例：
//
//	// 移除所有 "task1"
//	count, err := rdb.LRem(ctx, "queue:tasks", 0, "task1")
//	// 从左侧开始移除 2 个 "task1"
//	count, err := rdb.LRem(ctx, "queue:tasks", 2, "task1")
func (r *Redis) LRem(ctx context.Context, key string, count int64, value interface{}) (int64, error) {
	return r.client.LRem(ctx, key, count, value).Result()
}

// LSet 设置指定索引值
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param index: 索引
// @param value: 新值
//
// @return error: 错误信息，索引越界返回错误
//
// 使用示例：
//
//	err := rdb.LSet(ctx, "queue:tasks", 0, "new_first_task")
func (r *Redis) LSet(ctx context.Context, key string, index int64, value interface{}) error {
	return r.client.LSet(ctx, key, index, value).Err()
}

// LTrim 修剪列表
// 只保留指定范围内的元素，其余删除
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 列表键名
// @param start: 开始索引
// @param stop: 结束索引（包含）
//
// @return error: 错误信息
//
// 使用示例：
//
//	// 只保留前 100 个元素
//	err := rdb.LTrim(ctx, "log:recent", 0, 99)
func (r *Redis) LTrim(ctx context.Context, key string, start, stop int64) error {
	return r.client.LTrim(ctx, key, start, stop).Err()
}

// RPopLPush 弹出并推入
// 从 source 列表右侧弹出元素，推入 destination 列表左侧
//
// @param ctx: 上下文，用于控制超时和取消
// @param source: 源列表
// @param destination: 目标列表
//
// @return string: 移动的元素，source 为空返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	// 将任务从待处理队列移到处理中队列
//	task, err := rdb.RPopLPush(ctx, "queue:pending", "queue:processing")
func (r *Redis) RPopLPush(ctx context.Context, source, destination string) (string, error) {
	val, err := r.client.RPopLPush(ctx, source, destination).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// BRPopLPush 阻塞式弹出并推入
//
// @param ctx: 上下文，用于控制超时和取消
// @param source: 源列表
// @param destination: 目标列表
// @param timeout: 最大阻塞时间
//
// @return string: 移动的元素，超时返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	task, err := rdb.BRPopLPush(ctx, "queue:pending", "queue:processing", 30*time.Second)
func (r *Redis) BRPopLPush(ctx context.Context, source, destination string, timeout time.Duration) (string, error) {
	val, err := r.client.BRPopLPush(ctx, source, destination, timeout).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// ==================== Set 操作 ====================

// SAdd 添加元素到集合
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
// @param members: 要添加的元素列表
//
// @return int64: 实际添加的元素数量（已存在的不会被计算）
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.SAdd(ctx, "tags:article:1", "go", "redis", "database")
func (r *Redis) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return r.client.SAdd(ctx, key, members...).Result()
}

// SMembers 获取集合所有成员
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
//
// @return []string: 所有成员
// @return error: 错误信息
//
// 使用示例：
//
//	tags, err := rdb.SMembers(ctx, "tags:article:1")
//	// tags = ["go", "redis", "database"]
func (r *Redis) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// SIsMember 检查元素是否在集合中
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
// @param member: 要检查的元素
//
// @return bool: 元素存在返回 true
// @return error: 错误信息
//
// 使用示例：
//
//	exists, err := rdb.SIsMember(ctx, "tags:article:1", "go")
func (r *Redis) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// SRem 从集合移除元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
// @param members: 要移除的元素列表
//
// @return int64: 实际移除的元素数量
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.SRem(ctx, "tags:article:1", "old_tag", "deprecated")
func (r *Redis) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return r.client.SRem(ctx, key, members...).Result()
}

// SCard 获取集合元素数量
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
//
// @return int64: 元素数量
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.SCard(ctx, "tags:article:1")
func (r *Redis) SCard(ctx context.Context, key string) (int64, error) {
	return r.client.SCard(ctx, key).Result()
}

// SPop 随机移除并返回指定数量元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
// @param count: 要移除的元素数量
//
// @return []string: 被移除的元素列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 随机抽取 3 个中奖者
//	winners, err := rdb.SPop(ctx, "lottery:participants", 3)
func (r *Redis) SPop(ctx context.Context, key string, count int64) ([]string, error) {
	return r.client.SPopN(ctx, key, count).Result()
}

// SRandMember 随机返回指定数量元素（不移除）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
// @param count: 要返回的元素数量
//
// @return []string: 随机选中的元素列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 随机推荐 5 篇文章
//	articles, err := rdb.SRandMember(ctx, "articles:hot", 5)
func (r *Redis) SRandMember(ctx context.Context, key string, count int64) ([]string, error) {
	return r.client.SRandMemberN(ctx, key, count).Result()
}

// SPopOne 随机移除并返回一个元素
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
//
// @return string: 被移除的元素，集合为空返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	member, err := rdb.SPopOne(ctx, "set:pool")
func (r *Redis) SPopOne(ctx context.Context, key string) (string, error) {
	val, err := r.client.SPop(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// SRandMemberOne 随机返回一个元素（不移除）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 集合键名
//
// @return string: 随机选中的元素，集合为空返回空字符串
// @return error: 错误信息
//
// 使用示例：
//
//	// 随机获取一个推荐标签
//	tag, err := rdb.SRandMemberOne(ctx, "tags:recommended")
func (r *Redis) SRandMemberOne(ctx context.Context, key string) (string, error) {
	val, err := r.client.SRandMember(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// SInter 交集
// 返回所有给定集合的交集
//
// @param ctx: 上下文，用于控制超时和取消
// @param keys: 集合键名列表
//
// @return []string: 交集结果
// @return error: 错误信息
//
// 使用示例：
//
//	// 查找同时关注 user1 和 user2 的用户
//	common, err := rdb.SInter(ctx, "followers:user:1", "followers:user:2")
func (r *Redis) SInter(ctx context.Context, keys ...string) ([]string, error) {
	return r.client.SInter(ctx, keys...).Result()
}

// SUnion 并集
// 返回所有给定集合的并集
//
// @param ctx: 上下文，用于控制超时和取消
// @param keys: 集合键名列表
//
// @return []string: 并集结果
// @return error: 错误信息
//
// 使用示例：
//
//	// 合并所有标签
//	allTags, err := rdb.SUnion(ctx, "tags:go", "tags:redis", "tags:database")
func (r *Redis) SUnion(ctx context.Context, keys ...string) ([]string, error) {
	return r.client.SUnion(ctx, keys...).Result()
}

// SDiff 差集
// 返回第一个集合与其他集合的差集
//
// @param ctx: 上下文，用于控制超时和取消
// @param keys: 集合键名列表，第一个是基准集合
//
// @return []string: 差集结果
// @return error: 错误信息
//
// 使用示例：
//
//	// 查找关注 user1 但没关注 user2 的用户
//	diff, err := rdb.SDiff(ctx, "followers:user:1", "followers:user:2")
func (r *Redis) SDiff(ctx context.Context, keys ...string) ([]string, error) {
	return r.client.SDiff(ctx, keys...).Result()
}

// SInterStore 交集并存储
// 将交集结果存储到 destination 集合
//
// @param ctx: 上下文，用于控制超时和取消
// @param destination: 目标集合键名
// @param keys: 源集合键名列表
//
// @return int64: 结果集合元素数量
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.SInterStore(ctx, "common:followers", "followers:user:1", "followers:user:2")
func (r *Redis) SInterStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	return r.client.SInterStore(ctx, destination, keys...).Result()
}

// SUnionStore 并集并存储
//
// @param ctx: 上下文，用于控制超时和取消
// @param destination: 目标集合键名
// @param keys: 源集合键名列表
//
// @return int64: 结果集合元素数量
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.SUnionStore(ctx, "all:tags", "tags:article:1", "tags:article:2")
func (r *Redis) SUnionStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	return r.client.SUnionStore(ctx, destination, keys...).Result()
}

// SDiffStore 差集并存储
//
// @param ctx: 上下文，用于控制超时和取消
// @param destination: 目标集合键名
// @param keys: 源集合键名列表
//
// @return int64: 结果集合元素数量
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.SDiffStore(ctx, "unique:to:user1", "followers:user:1", "followers:user:2")
func (r *Redis) SDiffStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	return r.client.SDiffStore(ctx, destination, keys...).Result()
}

// ==================== ZSet 操作 ====================

// ZAdd 添加到有序集合
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param members: 成员和分数，使用 redis.Z{Score: 100, Member: "player1"}
//
// @return int64: 实际添加的新成员数量
// @return error: 错误信息
//
// 使用示例：
//
//	// 添加单个成员
//	count, err := rdb.ZAdd(ctx, "rank:score", redis.Z{Score: 100, Member: "player1"})
//	// 添加多个成员
//	count, err := rdb.ZAdd(ctx, "rank:score",
//	    redis.Z{Score: 100, Member: "player1"},
//	    redis.Z{Score: 200, Member: "player2"},
//	)
func (r *Redis) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	return r.client.ZAdd(ctx, key, members...).Result()
}

// ZRange 按排名范围获取（低到高）
// 按分数从低到高排序，返回指定排名范围的成员
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param start: 开始排名（0 表示第一个）
// @param stop: 结束排名（包含，-1 表示到最后一个）
//
// @return []string: 成员列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取前 10 名
//	top10, err := rdb.ZRange(ctx, "rank:score", 0, 9)
func (r *Redis) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(ctx, key, start, stop).Result()
}

// ZRevRange 按排名范围获取（高到低）
// 按分数从高到低排序
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param start: 开始排名（0 表示分数最高的）
// @param stop: 结束排名（包含，-1 表示到最后一个）
//
// @return []string: 成员列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取前 10 名（分数最高的）
//	top10, err := rdb.ZRevRange(ctx, "rank:score", 0, 9)
func (r *Redis) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRevRange(ctx, key, start, stop).Result()
}

// ZRangeByScore 按分数范围获取
// 返回分数在 min 和 max 之间的成员（包含边界）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param opt: 范围选项，使用 &redis.ZRangeBy{Min: "100", Max: "200", Offset: 0, Count: 10}
//
// @return []string: 成员列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取分数 100-200 的成员
//	members, err := rdb.ZRangeByScore(ctx, "rank:score", &redis.ZRangeBy{
//	    Min: "100",
//	    Max: "200",
//	})
func (r *Redis) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return r.client.ZRangeByScore(ctx, key, opt).Result()
}

// ZScore 获取成员分数
// member 不存在时返回 0 和 nil error
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param member: 成员
//
// @return float64: 成员分数，不存在返回 0
// @return error: 错误信息
//
// 使用示例：
//
//	score, err := rdb.ZScore(ctx, "rank:score", "player1")
//	if score == 0 {
//	    // 成员不存在或分数就是 0
//	}
func (r *Redis) ZScore(ctx context.Context, key, member string) (float64, error) {
	val, err := r.client.ZScore(ctx, key, member).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, err
}

// ZScoreE 获取成员分数（返回完整错误）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param member: 成员
//
// @return float64: 成员分数
// @return error: 错误信息，成员不存在返回 redis.Nil
//
// 使用示例：
//
//	score, err := rdb.ZScoreE(ctx, "rank:score", "player1")
//	if redisdao.IsNotFound(err) {
//	    // 成员不存在
//	}
func (r *Redis) ZScoreE(ctx context.Context, key, member string) (float64, error) {
	return r.client.ZScore(ctx, key, member).Result()
}

// ZRem 从有序集合移除成员
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param members: 要移除的成员列表
//
// @return int64: 实际移除的成员数量
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.ZRem(ctx, "rank:score", "player1", "player2")
func (r *Redis) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return r.client.ZRem(ctx, key, members...).Result()
}

// ZCard 获取有序集合元素数量
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
//
// @return int64: 元素数量
// @return error: 错误信息
//
// 使用示例：
//
//	count, err := rdb.ZCard(ctx, "rank:score")
func (r *Redis) ZCard(ctx context.Context, key string) (int64, error) {
	return r.client.ZCard(ctx, key).Result()
}

// ZIncrBy 有序集合成员分数自增
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param increment: 增量
// @param member: 成员
//
// @return float64: 自增后的分数
// @return error: 错误信息
//
// 使用示例：
//
//	newScore, err := rdb.ZIncrBy(ctx, "rank:score", 10.5, "player1")
func (r *Redis) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	return r.client.ZIncrBy(ctx, key, increment, member).Result()
}

// ZRank 获取成员排名（低到高，从0开始）
// 分数最低的排名为 0
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param member: 成员
//
// @return int64: 排名，成员不存在返回 -1
// @return error: 错误信息
//
// 使用示例：
//
//	rank, err := rdb.ZRank(ctx, "rank:score", "player1")
//	if rank == -1 {
//	    // 成员不存在
//	} else {
//	    fmt.Printf("排名第 %d\n", rank+1) // 转换为 1-based
//	}
func (r *Redis) ZRank(ctx context.Context, key, member string) (int64, error) {
	val, err := r.client.ZRank(ctx, key, member).Result()
	if errors.Is(err, redis.Nil) {
		return -1, nil
	}
	return val, err
}

// ZRevRank 获取成员排名（高到低，从0开始）
// 分数最高的排名为 0
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param member: 成员
//
// @return int64: 排名，成员不存在返回 -1
// @return error: 错误信息
//
// 使用示例：
//
//	rank, err := rdb.ZRevRank(ctx, "rank:score", "player1")
func (r *Redis) ZRevRank(ctx context.Context, key, member string) (int64, error) {
	val, err := r.client.ZRevRank(ctx, key, member).Result()
	if errors.Is(err, redis.Nil) {
		return -1, nil
	}
	return val, err
}

// ZCount 获取分数范围内成员数量
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param min: 最小分数（"-inf" 表示负无穷，"(" 表示不包含，如 "(100"）
// @param max: 最大分数（"+inf" 表示正无穷）
//
// @return int64: 成员数量
// @return error: 错误信息
//
// 使用示例：
//
//	// 分数 100-200 的成员数量
//	count, err := rdb.ZCount(ctx, "rank:score", "100", "200")
//	// 分数大于 100 的成员数量（不包含 100）
//	count, err := rdb.ZCount(ctx, "rank:score", "(100", "+inf")
func (r *Redis) ZCount(ctx context.Context, key, min, max string) (int64, error) {
	return r.client.ZCount(ctx, key, min, max).Result()
}

// ZRangeByScoreWithScores 按分数范围获取成员及分数
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param opt: 范围选项
//
// @return []redis.Z: 成员和分数列表
// @return error: 错误信息
//
// 使用示例：
//
//	results, err := rdb.ZRangeByScoreWithScores(ctx, "rank:score", &redis.ZRangeBy{
//	    Min: "100",
//	    Max: "200",
//	    Offset: 0,
//	    Count: 10,
//	})
//	for _, z := range results {
//	    fmt.Println(z.Member, z.Score)
//	}
func (r *Redis) ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error) {
	return r.client.ZRangeByScoreWithScores(ctx, key, opt).Result()
}

// ZRevRangeByScore 按分数范围获取（高到低）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param opt: 范围选项
//
// @return []string: 成员列表
// @return error: 错误信息
//
// 使用示例：
//
//	members, err := rdb.ZRevRangeByScore(ctx, "rank:score", &redis.ZRangeBy{
//	    Min: "100",
//	    Max: "200",
//	})
func (r *Redis) ZRevRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return r.client.ZRevRangeByScore(ctx, key, opt).Result()
}

// ZRevRangeByScoreWithScores 按分数范围获取成员及分数（高到低）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param opt: 范围选项
//
// @return []redis.Z: 成员和分数列表
// @return error: 错误信息
//
// 使用示例：
//
//	results, err := rdb.ZRevRangeByScoreWithScores(ctx, "rank:score", &redis.ZRangeBy{
//	    Min: "100",
//	    Max: "200",
//	})
func (r *Redis) ZRevRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error) {
	return r.client.ZRevRangeByScoreWithScores(ctx, key, opt).Result()
}

// ZRemRangeByRank 按排名范围移除
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param start: 开始排名
// @param stop: 结束排名（包含）
//
// @return int64: 移除的成员数量
// @return error: 错误信息
//
// 使用示例：
//
//	// 只保留前 100 名
//	count, err := rdb.ZRemRangeByRank(ctx, "rank:score", 0, 99)
func (r *Redis) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) (int64, error) {
	return r.client.ZRemRangeByRank(ctx, key, start, stop).Result()
}

// ZRemRangeByScore 按分数范围移除
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param min: 最小分数
// @param max: 最大分数
//
// @return int64: 移除的成员数量
// @return error: 错误信息
//
// 使用示例：
//
//	// 移除分数小于 100 的成员
//	count, err := rdb.ZRemRangeByScore(ctx, "rank:score", "-inf", "(100")
func (r *Redis) ZRemRangeByScore(ctx context.Context, key, min, max string) (int64, error) {
	return r.client.ZRemRangeByScore(ctx, key, min, max).Result()
}

// ZPopMin 弹出并返回分数最小的成员
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param count: 要弹出的数量
//
// @return []redis.Z: 弹出的成员和分数列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 弹出分数最低的 3 个成员
//	members, err := rdb.ZPopMin(ctx, "queue:priority", 3)
//	for _, z := range members {
//	    fmt.Println(z.Member, z.Score)
//	}
func (r *Redis) ZPopMin(ctx context.Context, key string, count int64) ([]redis.Z, error) {
	return r.client.ZPopMin(ctx, key, count).Result()
}

// ZPopMax 弹出并返回分数最大的成员
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param count: 要弹出的数量
//
// @return []redis.Z: 弹出的成员和分数列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 弹出分数最高的前 3 名
//	top3, err := rdb.ZPopMax(ctx, "rank:score", 3)
func (r *Redis) ZPopMax(ctx context.Context, key string, count int64) ([]redis.Z, error) {
	return r.client.ZPopMax(ctx, key, count).Result()
}

// BZPopMin 阻塞式弹出最小成员
// 如果集合为空，会阻塞等待直到有元素或超时
//
// @param ctx: 上下文，用于控制超时和取消
// @param timeout: 最大阻塞时间
// @param keys: 要监听的有序集合键名列表
//
// @return *redis.ZWithKey: 弹出的信息，包含 Key（集合名）、Member、Score
// @return error: 错误信息，超时返回 redis.Nil
//
// 使用示例：
//
//	result, err := rdb.BZPopMin(ctx, 30*time.Second, "queue:priority", "queue:backup")
//	if err == nil {
//	    fmt.Printf("从 %s 弹出 %s，分数 %f\n", result.Key, result.Member, result.Score)
//	}
func (r *Redis) BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) (*redis.ZWithKey, error) {
	return r.client.BZPopMin(ctx, timeout, keys...).Result()
}

// BZPopMax 阻塞式弹出最大成员
//
// @param ctx: 上下文，用于控制超时和取消
// @param timeout: 最大阻塞时间
// @param keys: 要监听的有序集合键名列表
//
// @return *redis.ZWithKey: 弹出的信息
// @return error: 错误信息，超时返回 redis.Nil
//
// 使用示例：
//
//	result, err := rdb.BZPopMax(ctx, 30*time.Second, "rank:daily")
func (r *Redis) BZPopMax(ctx context.Context, timeout time.Duration, keys ...string) (*redis.ZWithKey, error) {
	return r.client.BZPopMax(ctx, timeout, keys...).Result()
}

// ZRangeWithScores 按排名范围获取成员及分数
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param start: 开始排名
// @param stop: 结束排名（包含）
//
// @return []redis.Z: 成员和分数列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取前 10 名及分数
//	results, err := rdb.ZRangeWithScores(ctx, "rank:score", 0, 9)
func (r *Redis) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.client.ZRangeWithScores(ctx, key, start, stop).Result()
}

// ZRevRangeWithScores 按排名范围获取成员及分数（高到低）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 有序集合键名
// @param start: 开始排名
// @param stop: 结束排名（包含）
//
// @return []redis.Z: 成员和分数列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 获取前 10 名（分数最高的）及分数
//	results, err := rdb.ZRevRangeWithScores(ctx, "rank:score", 0, 9)
func (r *Redis) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

// ==================== Key 操作 ====================

// Keys 查找 key（慎用，大数据量时性能差）
// 时间复杂度 O(N)，N 为数据库中 key 的总数，生产环境建议使用 Scan
//
// @param ctx: 上下文，用于控制超时和取消
// @param pattern: 匹配模式，如 "user:*"、"*session*"
//
// @return []string: 匹配的 key 列表
// @return error: 错误信息
//
// 使用示例：
//
//	// 查找所有 user: 开头的 key（注意：大数据量时很慢）
//	keys, err := rdb.Keys(ctx, "user:*")
//
//	// 查找所有 key
//	keys, err := rdb.Keys(ctx, "*")
func (r *Redis) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

// Scan 迭代 key（推荐替代 Keys）
// 基于游标的迭代器，不会阻塞服务器，适合大数据量场景
//
// @param ctx: 上下文，用于控制超时和取消
// @param cursor: 游标，第一次传 0，后续传上次返回的游标
// @param match: 匹配模式
// @param count: 每次返回的大约数量
//
// @return []string: 本次迭代的 key 列表
// @return uint64: 下次迭代的游标，为 0 表示迭代结束
// @return error: 错误信息
//
// 使用示例：
//
//	var cursor uint64 = 0
//	for {
//	    keys, nextCursor, err := rdb.Scan(ctx, cursor, "user:*", 100)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    for _, key := range keys {
//	        fmt.Println(key)
//	    }
//	    cursor = nextCursor
//	    if cursor == 0 {
//	        break
//	    }
//	}
func (r *Redis) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return r.client.Scan(ctx, cursor, match, count).Result()
}

// Type 获取 key 类型
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return string: key 类型，可能的值：string、list、set、zset、hash、none（不存在）
// @return error: 错误信息
//
// 使用示例：
//
//	keyType, err := rdb.Type(ctx, "user:1")
//	// keyType = "hash"
func (r *Redis) Type(ctx context.Context, key string) (string, error) {
	return r.client.Type(ctx, key).Result()
}

// Rename 重命名 key
// 如果 newKey 已存在，会被覆盖
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 原键名
// @param newKey: 新键名
//
// @return error: 错误信息
//
// 使用示例：
//
//	err := rdb.Rename(ctx, "temp:user:1", "user:1")
func (r *Redis) Rename(ctx context.Context, key, newKey string) error {
	return r.client.Rename(ctx, key, newKey).Err()
}

// RenameNX 重命名 key（仅当 newKey 不存在）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 原键名
// @param newKey: 新键名
//
// @return bool: 重命名成功返回 true，newKey 已存在返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	ok, err := rdb.RenameNX(ctx, "temp:user:1", "user:1")
//	if ok {
//	    // 重命名成功
//	} else {
//	    // user:1 已存在
//	}
func (r *Redis) RenameNX(ctx context.Context, key, newKey string) (bool, error) {
	return r.client.RenameNX(ctx, key, newKey).Result()
}

// Persist 移除过期时间（持久化 key）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return bool: 成功移除返回 true，key 不存在或没有过期时间返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	ok, err := rdb.Persist(ctx, "session:123")
//	if ok {
//	    // 已设为永不过期
//	}
func (r *Redis) Persist(ctx context.Context, key string) (bool, error) {
	return r.client.Persist(ctx, key).Result()
}

// PExpire 设置过期时间（毫秒）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param expiration: 过期时间（毫秒）
//
// @return bool: 设置成功返回 true，key 不存在返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	// 设置 500 毫秒后过期
//	ok, err := rdb.PExpire(ctx, "temp:data", 500*time.Millisecond)
func (r *Redis) PExpire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return r.client.PExpire(ctx, key, expiration).Result()
}

// PTTL 获取剩余过期时间（毫秒）
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
//
// @return time.Duration: 剩余毫秒数，key 不存在返回 -2，永不过期返回 -1
// @return error: 错误信息
//
// 使用示例：
//
//	pttl, err := rdb.PTTL(ctx, "session:123")
func (r *Redis) PTTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.PTTL(ctx, key).Result()
}

// ==================== Server 操作 ====================

// DBSize 获取当前数据库 key 数量
//
// @param ctx: 上下文，用于控制超时和取消
//
// @return int64: key 数量
// @return error: 错误信息
//
// 使用示例：
//
//	size, err := rdb.DBSize(ctx)
//	fmt.Printf("当前数据库有 %d 个 key\n", size)
func (r *Redis) DBSize(ctx context.Context) (int64, error) {
	return r.client.DBSize(ctx).Result()
}

// FlushDB 清空当前数据库
// 危险操作！会删除当前选中的数据库中的所有 key
//
// @param ctx: 上下文，用于控制超时和取消
//
// @return error: 错误信息
//
// 使用示例：
//
//	// 清空当前数据库（谨慎使用！）
//	err := rdb.FlushDB(ctx)
func (r *Redis) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// FlushAll 清空所有数据库
// 危险操作！会删除所有数据库中的 key
//
// @param ctx: 上下文，用于控制超时和取消
//
// @return error: 错误信息
//
// 使用示例：
//
//	// 清空所有数据库（极其危险！）
//	err := rdb.FlushAll(ctx)
func (r *Redis) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}

// Ping 检查连接
//
// @param ctx: 上下文，用于控制超时和取消
//
// @return string: "PONG" 表示连接正常
// @return error: 错误信息
//
// 使用示例：
//
//	pong, err := rdb.Ping(ctx)
//	if err != nil {
//	    // 连接失败
//	}
func (r *Redis) Ping(ctx context.Context) (string, error) {
	return r.client.Ping(ctx).Result()
}

// ==================== JSON 操作 ====================

// GetJSON 获取 JSON 并反序列化到对象
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param dest: 目标对象指针，用于接收反序列化结果
//
// @return error: 错误信息，key 不存在返回错误
//
// 使用示例：
//
//	type User struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	}
//	var user User
//	err := rdb.GetJSON(ctx, "user:1", &user)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(user.Name, user.Age)
func (r *Redis) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.GetBytes(ctx, key)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	return json.Unmarshal(data, dest)
}

// SetJSON 将对象序列化为 JSON 存储
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 键名
// @param value: 要序列化的对象
// @param expiration: 过期时间
//
// @return error: 错误信息
//
// 使用示例：
//
//	user := User{Name: "张三", Age: 25}
//	err := rdb.SetJSON(ctx, "user:1", user, 60*time.Second)
func (r *Redis) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Set(ctx, key, data, expiration)
}

// ==================== 分布式锁 ====================

// TryLock 尝试获取分布式锁
// 使用 SetNX 实现，锁的 value 包含时间戳信息
// 注意：这是简单实现，生产环境建议使用 Redlock 或类似方案
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 锁的键名
// @param expiration: 锁的过期时间，必须设置以防止死锁
//
// @return bool: 获取成功返回 true，锁已被占用返回 false
// @return error: 错误信息
//
// 使用示例：
//
//	// 尝试获取锁，30 秒后自动释放
//	ok, err := rdb.TryLock(ctx, "lock:order:123", 30*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if ok {
//	    // 获取锁成功
//	    defer rdb.Unlock(ctx, "lock:order:123") // 确保释放锁
//	    // 执行业务逻辑
//	} else {
//	    // 锁已被占用
//	}
func (r *Redis) TryLock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	// 使用唯一标识作为 value，便于后续释放时验证
	value := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
	ok, err := r.client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Unlock 释放锁（简单实现，不验证 value）
// 警告：此方法会强制删除 key，如果锁已过期被其他客户端获取，会导致误删
// 生产环境建议使用 Lua 脚本验证 value 后再删除
//
// @param ctx: 上下文，用于控制超时和取消
// @param key: 锁的键名
//
// @return error: 错误信息
//
// 使用示例：
//
//	err := rdb.Unlock(ctx, "lock:order:123")
func (r *Redis) Unlock(ctx context.Context, key string) error {
	return r.Del(ctx, key)
}

// ==================== 批量操作 ====================

// MGet 批量获取
// 返回的结果顺序与传入的 keys 顺序一致
//
// @param ctx: 上下文，用于控制超时和取消
// @param keys: 键名列表
//
// @return []interface{}: 值列表，key 不存在时对应位置为 nil
// @return error: 错误信息
//
// 使用示例：
//
//	values, err := rdb.MGet(ctx, "user:1", "user:2", "user:3")
//	for i, val := range values {
//	    if val != nil {
//	        fmt.Printf("user:%d = %s\n", i+1, val.(string))
//	    }
//	}
func (r *Redis) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return r.client.MGet(ctx, keys...).Result()
}

// MSet 批量设置
//
// @param ctx: 上下文，用于控制超时和取消
// @param values: 键值对列表，如 "key1", "value1", "key2", "value2"
//
// @return error: 错误信息
//
// 使用示例：
//
//	err := rdb.MSet(ctx, "key1", "value1", "key2", "value2", "key3", "value3")
func (r *Redis) MSet(ctx context.Context, values ...interface{}) error {
	return r.client.MSet(ctx, values...).Err()
}

// Pipeline 获取 Pipeline 用于批量操作
// Pipeline 将多个命令打包发送，减少网络往返，但不保证原子性
//
// @return redis.Pipeliner: Pipeline 对象
//
// 使用示例：
//
//	pipe := rdb.Pipeline()
//	pipe.Set(ctx, "key1", "value1", 0)
//	pipe.Get(ctx, "key2")
//	pipe.Incr(ctx, "counter")
//	cmders, err := pipe.Exec(ctx)
//	// 获取结果
//	getCmd := cmders[1].(*redis.StringCmd)
//	val, err := getCmd.Result()
func (r *Redis) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// TxPipeline 获取事务 Pipeline
// 事务 Pipeline 会将所有命令打包到一个事务中执行（MULTI/EXEC），保证原子性
//
// @return redis.Pipeliner: 事务 Pipeline 对象
//
// 使用示例：
//
//	pipe := rdb.TxPipeline()
//	pipe.Get(ctx, "key1")
//	pipe.Set(ctx, "key2", "value2", 0)
//	cmders, err := pipe.Exec(ctx)
func (r *Redis) TxPipeline() redis.Pipeliner {
	return r.client.TxPipeline()
}

// ==================== 原始客户端访问 ====================

// Client 获取原始 redis.Client，用于高级操作
// 当快捷方法不满足需求时，可以直接使用原始客户端
//
// @return *redis.Client: 原始的 go-redis 客户端
//
// 使用示例：
//
//	// 使用原始客户端执行 Geo 操作（快捷方法未封装）
//	client := rdb.Client()
//	client.GeoAdd(ctx, "cities", &redis.GeoLocation{
//	    Name:      "Beijing",
//	    Longitude: 116.4074,
//	    Latitude:  39.9042,
//	})
func (r *Redis) Client() *redis.Client {
	return r.client
}
