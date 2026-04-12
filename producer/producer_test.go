package producer

import (
	"context"
	"os"
	"os/user"
	"testing"
	"time"

	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/logger"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

func TestKafka(t *testing.T) {
	currUser, _ := user.Current()

	cfgMap := map[string]string{
		"fileName": currUser.HomeDir + "/logs/gokit/logger.log",
		"console":  "true",
		"level":    "DEBUG",
		"maxSize":  "200",
	}

	// 初始化日志环境
	logger.SetEnv("dev")
	logger.SetName("testLogger")
	logger.SetDepartment("gokit")
	logger.SetVersion("logger-v1.0.0")
	logConfig := logger.NewConfig().SetConfigMap(cfgMap)

	logger.InitWithConfig(logConfig)

	defer logger.Sync()

	dir, _ := os.Getwd()
	config.SetConfigPath(dir + "/conf/conf.ini")

	ctx := context.Background()
	err := Kafka(ctx, "gokit_go_producer", []byte("AABBCC."+time.Now().Format("2006-01-02 15:04:05")))
	defer Close()

	if err != nil {
		t.Errorf("kafka producer fail. err:%v", err)
	} else {
		t.Log("pass")
	}

	// new redis client
	client := redis.NewClient(&redis.Options{
		Addr:     config.GetConfDefault("Redis", "redis", "127.0.0.1:6379"),
		Password: config.GetConfDefault("RedisConfig", "redis.password", ""),
		DB:       cast.ToInt(config.GetConfDefault("RedisConfig", "redis.db", "0")),
	})
	err = Delay(ctx, client, "gokit_go_producer_delay", []byte("112233445566"), 1)
	if err != nil {
		t.Errorf("delay producer fail. err:%v", err)
	} else {
		t.Log("pass")
	}

	time.Sleep(2 * time.Second)
}
