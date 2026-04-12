package redisdao

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wanglelecc/gokitbox/config"
)

var testStrKey = "redisdao_test_name"

func TestNewSimpleRedis(t *testing.T) {
	// load local conf.ini
	dir, _ := os.Getwd()
	config.SetConfigPath(dir + "/conf/conf.ini")

	ctx := context.Background()
	client := NewSimpleRedis("redis")

	// 检查客户端是否初始化成功
	if client == nil {
		t.Skip("Redis 客户端初始化失败（配置不存在），跳过集成测试")
	}

	// 检查是否能连接到 Redis，如果不能则跳过测试
	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis 连接失败，跳过集成测试:", err)
	}

	err = client.Set(ctx, testStrKey, "wanglele", time.Duration(30*time.Second)).Err()
	if err != nil {
		t.Error("fail")
	} else {
		t.Log("pass")
	}

	v := client.Get(context.Background(), testStrKey).Val()
	if v == "wanglele" {
		t.Log("pass")
	} else {
		t.Error("fail")
	}

	delayClient := NewSimpleRedis("delayRedis")
	err = delayClient.Set(ctx, testStrKey, "wanglele", time.Duration(30*time.Second)).Err()
	if err != nil {
		t.Error("fail")
	} else {
		t.Log("pass")
	}

	v = client.Get(context.Background(), testStrKey).Val()
	if v == "wanglele" {
		t.Log("pass")
	} else {
		t.Error("fail")
	}
}
