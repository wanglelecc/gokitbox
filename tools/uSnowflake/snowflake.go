package uSnowflake

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/bwmarrin/snowflake"
)

var (
	snowflakeNode *snowflake.Node
	nodeProvider  NodeProvider // 保留 provider 引用，便于后续扩展
)

// InitSnowflake 初始化雪花 ID 节点
// 节点号通过 NodeProvider 分配（推荐 Redis 实现，支持分布式协调）
// 初始化失败直接 panic，确保问题早发现
//
// 使用示例：
//
//	// bootstrap 中组装
//	provider := redisdao.NewRedisNodeProvider()
//	uSnowflake.InitSnowflake(ctx, provider, "my_project", "order_service")
func InitSnowflake(ctx context.Context, provider NodeProvider, project string, service string) {
	if provider == nil {
		panic("NodeProvider 不能为 nil")
	}
	nodeProvider = provider

	node, err := provider.MakeNode(ctx, project, service)
	if err != nil {
		// Redis 失败时降级为随机节点（保证可用性）
		node = randSnowflakeNode()
	}

	// 确保节点号在有效范围（0~1023）
	node = node % 1024
	fmt.Println("snowflakeNode:", node)

	snowflakeNode, err = snowflake.NewNode(node)
	if err != nil {
		panic(err)
	}
}

// randSnowflakeNode 随机生成节点号（降级方案）
// 使用 crypto/rand 确保密码学安全，避免节点号冲突
func randSnowflakeNode() int64 {
	max := big.NewInt(1024)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// 极端情况下随机数生成失败，返回 0 作为兜底
		return 0
	}
	return n.Int64()
}

// NewIdInt64 生成 int64 类型的雪花 ID，调用前必须先执行 InitSnowflake
//
// 使用示例：
//
//	id := uSnowflake.NewIdInt64()
//	// id = 1750123456789012345
func NewIdInt64() int64 {
	if snowflakeNode == nil {
		panic("snowflake 未初始化，请先调用 InitSnowflake")
	}
	return snowflakeNode.Generate().Int64()
}

// NewIdString 生成字符串类型的雪花 ID，调用前必须先执行 InitSnowflake
//
// 使用示例：
//
//	id := uSnowflake.NewIdString()
//	// id = "1750123456789012345"
func NewIdString() string {
	if snowflakeNode == nil {
		panic("snowflake 未初始化，请先调用 InitSnowflake")
	}
	return snowflakeNode.Generate().String()
}
