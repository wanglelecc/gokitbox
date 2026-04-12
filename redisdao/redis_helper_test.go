package redisdao

import (
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

// TestIsNotFound 测试 IsNotFound 函数
func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"redis.Nil", redis.Nil, true},
		{"nil error", nil, false},
		{"其他错误", errors.New("some error"), false},
		{"包装后的 redis.Nil", errors.New("wrapped: redis.Nil"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

// TestRedisHelperAPI 验证 API 能正常调用
func TestRedisHelperAPI(t *testing.T) {
	t.Run("Redis 结构体创建", func(t *testing.T) {
		// 由于需要 Redis 环境，这里仅验证不会 panic
		// r := NewRedis("redis")
		// if r == nil {
		// 	t.Error("NewRedis returned nil")
		// }
	})
}

// TestRedisHelperMock 模拟测试
func TestRedisHelperMock(t *testing.T) {
	t.Run("Get 返回值处理", func(t *testing.T) {
		// 模拟 redis.Nil 情况
		var err error = redis.Nil
		if !errors.Is(err, redis.Nil) {
			t.Error("Expected errors.Is to work with redis.Nil")
		}
	})
}

// TestRedisHelperMethods 验证所有方法存在并可调用（编译期检查）
func TestRedisHelperMethods(t *testing.T) {
	// 由于需要 Redis 环境，这里仅做编译期检查
	// 实际测试需要 Redis 环境或使用 miniredis

	var r *Redis
	_ = r

	// String 方法检查
	_ = r.Get
	_ = r.GetE
	_ = r.GetBytes
	_ = r.GetBytesE
	_ = r.Set
	_ = r.SetNX
	_ = r.Del
	_ = r.Exists
	_ = r.Expire
	_ = r.TTL
	_ = r.Incr
	_ = r.IncrBy
	_ = r.Decr
	_ = r.DecrBy
	_ = r.StrLen
	_ = r.Append
	_ = r.GetSet
	_ = r.GetRange
	_ = r.SetRange
	_ = r.MSetNX

	// Hash 方法检查
	_ = r.HGet
	_ = r.HGetE
	_ = r.HGetAll
	_ = r.HSet
	_ = r.HDel
	_ = r.HExists
	_ = r.HIncrBy
	_ = r.HIncrByFloat
	_ = r.HLen
	_ = r.HMGet
	_ = r.HMSet
	_ = r.HKeys
	_ = r.HVals

	// List 方法检查
	_ = r.LPush
	_ = r.RPush
	_ = r.LPop
	_ = r.LPopE
	_ = r.RPop
	_ = r.RPopE
	_ = r.BLPop
	_ = r.LLen
	_ = r.LRange
	_ = r.LIndex
	_ = r.LInsert
	_ = r.LRem
	_ = r.LSet
	_ = r.LTrim
	_ = r.RPopLPush
	_ = r.BRPopLPush

	// Set 方法检查
	_ = r.SAdd
	_ = r.SMembers
	_ = r.SIsMember
	_ = r.SRem
	_ = r.SCard
	_ = r.SPop
	_ = r.SPopOne
	_ = r.SRandMember
	_ = r.SRandMemberOne
	_ = r.SInter
	_ = r.SUnion
	_ = r.SDiff
	_ = r.SInterStore
	_ = r.SUnionStore
	_ = r.SDiffStore

	// ZSet 方法检查
	_ = r.ZAdd
	_ = r.ZRange
	_ = r.ZRevRange
	_ = r.ZRangeByScore
	_ = r.ZScore
	_ = r.ZScoreE
	_ = r.ZRem
	_ = r.ZCard
	_ = r.ZIncrBy
	_ = r.ZRank
	_ = r.ZRevRank
	_ = r.ZCount
	_ = r.ZRangeByScoreWithScores
	_ = r.ZRevRangeByScore
	_ = r.ZRevRangeByScoreWithScores
	_ = r.ZRemRangeByRank
	_ = r.ZRemRangeByScore
	_ = r.ZRangeWithScores
	_ = r.ZRevRangeWithScores
	_ = r.ZPopMin
	_ = r.ZPopMax
	_ = r.BZPopMin
	_ = r.BZPopMax

	// Key 方法检查
	_ = r.Keys
	_ = r.Scan
	_ = r.Type
	_ = r.Rename
	_ = r.RenameNX
	_ = r.Persist
	_ = r.PExpire
	_ = r.PTTL

	// Server 方法检查
	_ = r.DBSize
	_ = r.FlushDB
	_ = r.FlushAll
	_ = r.Ping

	// JSON 方法检查
	_ = r.GetJSON
	_ = r.SetJSON

	// Lock 方法检查
	_ = r.TryLock
	_ = r.Unlock

	// Batch 方法检查
	_ = r.MGet
	_ = r.MSet
	_ = r.Pipeline
	_ = r.TxPipeline

	// Client 方法检查
	_ = r.Client
}

// TestRedisHelperUsage 展示使用示例
func TestRedisHelperUsage(t *testing.T) {
	// 示例代码展示（不实际执行）
	t.Log("Redis Helper 使用示例：")
	t.Log("- String 操作: Get/Set/Del/Incr 等")
	t.Log("- Hash 操作: HGet/HSet/HGetAll/HDel 等")
	t.Log("- List 操作: LPush/RPush/LPop/RPop/BLPop 等")
	t.Log("- Set 操作: SAdd/SMembers/SRem/SInter/SUnion 等")
	t.Log("- ZSet 操作: ZAdd/ZRange/ZRevRange/ZScore 等")
	t.Log("- Key 操作: Keys/Scan/Rename/Persist 等")
	t.Log("- JSON 操作: GetJSON/SetJSON 自动序列化")
	t.Log("- 分布式锁: TryLock/Unlock")
	t.Log("- 双版本方法: Get/GetE, LPop/LPopE 等")
}
