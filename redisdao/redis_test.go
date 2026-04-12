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

	// reset init redis
	// initRedis()

	ctx := context.Background()
	client := NewSimpleRedis("redis")

	err := client.Set(ctx, testStrKey, "wanglele", time.Duration(30*time.Second)).Err()
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
