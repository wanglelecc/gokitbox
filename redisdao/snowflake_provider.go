package redisdao

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/redis/go-redis/v9"
)

// RedisNodeProvider 基于 Redis ZSet 的节点号分配器
// 支持分布式环境下多实例自动分配唯一节点号
type RedisNodeProvider struct {
	client *redis.Client
	// key 前缀，可配置避免冲突
	keyPrefix string
}

// NewRedisNodeProvider 创建 Redis 节点分配器
// 使用默认 "redis" 实例
func NewRedisNodeProvider() *RedisNodeProvider {
	return &RedisNodeProvider{
		client:    NewSimpleRedis("redis"),
		keyPrefix: "gokitbox_snowflake_node",
	}
}

// NewRedisNodeProviderWithClient 创建指定 Redis 客户端的分配器（便于测试）
func NewRedisNodeProviderWithClient(client *redis.Client) *RedisNodeProvider {
	return &RedisNodeProvider{
		client:    client,
		keyPrefix: "gokitbox_snowflake_node",
	}
}

// MakeNode 通过 Redis ZSet 自增获取节点号
// key: tools_snowflake_node:{project}
// member: {service}
// score: 当前节点号（自增）
func (p *RedisNodeProvider) MakeNode(ctx context.Context, project, service string) (int64, error) {
	key := fmt.Sprintf("%s_%s", p.keyPrefix, project)

	// 获取当前 score
	score, err := p.client.ZScore(ctx, key, service).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("获取节点号失败: %w", err)
	}
	if errors.Is(err, redis.Nil) {
		score = 0
	}

	// 自增
	_, err = p.client.ZIncrBy(ctx, key, 1, service).Result()
	if err != nil {
		return 0, fmt.Errorf("自增节点号失败: %w", err)
	}

	return int64(score), nil
}

// RandNode Redis 不可用时返回随机节点（兜底）
// 使用 crypto/rand 确保密码学安全
func (p *RedisNodeProvider) RandNode() int64 {
	max := big.NewInt(1024)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return n.Int64()
}
