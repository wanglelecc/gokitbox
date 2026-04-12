package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/logger"
)

var (
	configPath     = "../config/consumer.json"
	configPathLock sync.RWMutex
)

// consume enabled
type ConsumeEnabled struct {
	Kafka    bool `json:"kafka"`
	Rabbitmq bool `json:"rabbitmq"`
	Redis    bool `json:"redis"`
	Delay    bool `json:"delay"`
}

// consumer config
type Configure struct {
	Enabled  *ConsumeEnabled   `json:"enabled"`
	Kafka    []*KafkaConfig    `json:"kafka"`
	Rabbitmq []*RabbitmqConfig `json:"rabbitmq"`
	Redis    []*RedisConfig    `json:"redis"`
	Delay    []*DelayConfig    `json:"delay"`
}

// kafka config
type SASL struct {
	Enabled  bool   `json:"enabled"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type KafkaConfig struct {
	Host          []string `json:"host"`
	Topic         string   `json:"topic"`
	SASL          *SASL    `json:"sasl"`
	FailTopic     string   `json:"failTopic"`
	FailCount     int      `json:"failCount"`
	ConsumerGroup string   `json:"consumerGroup"`
	ConsumerCount int      `json:"consumerCount"`
	TplMode       int      `json:"tplMode"`
	TplName       string   `json:"tplName"`
}

// rabbitmq config
type RabbitmqConfig struct {
	URL           string `json:"url"`
	FailCount     int    `json:"failCount"`
	FailExchange  string `json:"failExchange"`
	IsReject      bool   `json:"isReject"`
	PrefetchCount int    `json:"prefetchCount"`
	ConsumerQueue string `json:"consumerQueue"`
	ConsumerCount int    `json:"consumerCount"`
	TplMode       int    `json:"tplMode"`
	TplName       string `json:"tplName"`
}

// redis config
type RedisConfig struct {
	ConsumerCount int    `json:"consumerCount"`
	Addr          string `json:"addr"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Db            int    `json:"db"`
	Queue         string `json:"queue"`
	FailQueue     string `json:"failQueue"`
	FailCount     int    `json:"failCount"`
	TplMode       int    `json:"tplMode"`
	TplName       string `json:"tplName"`
}

// delay config
type DelayConfig struct {
	ConsumerCount int    `json:"consumerCount"`
	Addr          string `json:"addr"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Db            int    `json:"db"`
	Queue         string `json:"queue"`
	FailQueue     string `json:"failQueue"`
	FailCount     int    `json:"failCount"`
	TplMode       int    `json:"tplMode"`
	TplName       string `json:"tplName"`
}

func loadConfigure() *Configure {
	ctx := context.Background()

	configPathLock.Lock()
	configPath = config.GetConfDefault("App", "consumeConfigPath", configPath)
	if !strings.HasPrefix(configPath, "/") {
		configPath = config.Binhome() + "/" + configPath
	}
	cp := configPath
	configPathLock.Unlock()

	b, err := os.ReadFile(cp)
	if err != nil {
		logger.Fx(ctx, "consumer", "read config file error", "error", err.Error(), "config_path", cp)
		panic(fmt.Sprintf("failed to read consumer config file: %v", err))
	}

	cfg := new(Configure)

	err = json.Unmarshal(b, cfg)
	if err != nil {
		logger.Fx(ctx, "consumer", "json Unmarshal error", "error", err.Error())
		panic(fmt.Sprintf("failed to unmarshal consumer config: %v", err))
	}

	// Enabled 为指针，JSON 中缺失该字段时为 nil，补充零值防止调用方 nil deref
	if cfg.Enabled == nil {
		cfg.Enabled = &ConsumeEnabled{}
	}

	return cfg
}

func SetConfigPath(path string) {
	configPathLock.Lock()
	defer configPathLock.Unlock()
	configPath = path
}
