package consumer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wanglelecc/gokitbox/configsource"
)

var (
	configPath         = "../config/worker.json"
	configPathLock     sync.RWMutex
	configPathExplicit bool
)

// consume enabled
type ConsumeEnabled struct {
	Kafka    bool `json:"kafka" yaml:"kafka"`
	Rabbitmq bool `json:"rabbitmq" yaml:"rabbitmq"`
	Redis    bool `json:"redis" yaml:"redis"`
	Delay    bool `json:"delay" yaml:"delay"`
}

// consumer config
type Configure struct {
	Runtime  RuntimeOptions    `json:"runtime" yaml:"runtime"`
	Enabled  *ConsumeEnabled   `json:"enabled" yaml:"enabled"`
	Kafka    []*KafkaConfig    `json:"kafka" yaml:"kafka"`
	Rabbitmq []*RabbitmqConfig `json:"rabbitmq" yaml:"rabbitmq"`
	Redis    []*RedisConfig    `json:"redis" yaml:"redis"`
	Delay    []*DelayConfig    `json:"delay" yaml:"delay"`
}

// kafka config
type SASL struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
}

type KafkaConfig struct {
	Host          []string `json:"host" yaml:"host"`
	Topic         string   `json:"topic" yaml:"topic"`
	SASL          *SASL    `json:"sasl" yaml:"sasl"`
	FailTopic     string   `json:"failTopic" yaml:"failTopic"`
	FailCount     int      `json:"failCount" yaml:"failCount"`
	ConsumerGroup string   `json:"consumerGroup" yaml:"consumerGroup"`
	ConsumerCount int      `json:"consumerCount" yaml:"consumerCount"`
	TplMode       int      `json:"tplMode" yaml:"tplMode"`
	TplName       string   `json:"tplName" yaml:"tplName"`
}

// rabbitmq config
type RabbitmqConfig struct {
	URL           string `json:"url" yaml:"url"`
	FailCount     int    `json:"failCount" yaml:"failCount"`
	FailExchange  string `json:"failExchange" yaml:"failExchange"`
	IsReject      bool   `json:"isReject" yaml:"isReject"`
	PrefetchCount int    `json:"prefetchCount" yaml:"prefetchCount"`
	ConsumerQueue string `json:"consumerQueue" yaml:"consumerQueue"`
	ConsumerCount int    `json:"consumerCount" yaml:"consumerCount"`
	TplMode       int    `json:"tplMode" yaml:"tplMode"`
	TplName       string `json:"tplName" yaml:"tplName"`
}

// redis config
type RedisConfig struct {
	DeliveryMode  string `json:"deliveryMode" yaml:"deliveryMode"`
	ConsumerCount int    `json:"consumerCount" yaml:"consumerCount"`
	Addr          string `json:"addr" yaml:"addr"`
	Username      string `json:"username" yaml:"username"`
	Password      string `json:"password" yaml:"password"`
	Db            int    `json:"db" yaml:"db"`
	Queue         string `json:"queue" yaml:"queue"`
	FailQueue     string `json:"failQueue" yaml:"failQueue"`
	FailCount     int    `json:"failCount" yaml:"failCount"`
	TplMode       int    `json:"tplMode" yaml:"tplMode"`
	TplName       string `json:"tplName" yaml:"tplName"`
}

// delay config
type DelayConfig struct {
	DeliveryMode  string `json:"deliveryMode" yaml:"deliveryMode"`
	ConsumerCount int    `json:"consumerCount" yaml:"consumerCount"`
	Addr          string `json:"addr" yaml:"addr"`
	Username      string `json:"username" yaml:"username"`
	Password      string `json:"password" yaml:"password"`
	Db            int    `json:"db" yaml:"db"`
	Queue         string `json:"queue" yaml:"queue"`
	FailQueue     string `json:"failQueue" yaml:"failQueue"`
	FailCount     int    `json:"failCount" yaml:"failCount"`
	TplMode       int    `json:"tplMode" yaml:"tplMode"`
	TplName       string `json:"tplName" yaml:"tplName"`
}

// LoadConfigure reads an explicit JSON or YAML path without consulting global
// application config. New applications should inject Configure directly.
func LoadConfigure(path string) (*Configure, error) {
	cfg := new(Configure)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		doc, err := configsource.ReadYAML(path)
		if err != nil {
			return nil, err
		}
		if err := doc.Decode(cfg); err != nil {
			return nil, err
		}
	case ".json":
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(cfg); err != nil {
			return nil, fmt.Errorf("worker: invalid JSON configuration")
		}
		var extra interface{}
		if err := decoder.Decode(&extra); err != io.EOF {
			return nil, fmt.Errorf("worker: expected one JSON document")
		}
	default:
		return nil, fmt.Errorf("worker: expected .json, .yaml or .yml configuration")
	}
	if cfg.Enabled == nil {
		cfg.Enabled = &ConsumeEnabled{}
	}
	return cfg, nil
}

func LoadDefaultConfigure() (*Configure, error) {
	configPathLock.RLock()
	cp, explicit := configPath, configPathExplicit
	configPathLock.RUnlock()
	if !explicit {
		cp = configsource.GetConfDefault("App", "workerConfigPath", cp)
	}
	return LoadConfigure(cp)
}
func loadConfigure() *Configure {
	cfg, err := LoadDefaultConfigure()
	if err != nil {
		panic(fmt.Sprintf("failed to load consumer configuration: %v", err))
	}
	return cfg
}

func SetConfigPath(path string) {
	configPathLock.Lock()
	defer configPathLock.Unlock()
	configPath = path
	configPathExplicit = true
}

// Validate checks enabled backends without connecting to any broker.
func (c *Configure) Validate() error {
	if c == nil || c.Enabled == nil {
		return fmt.Errorf("worker: enabled configuration is required")
	}
	if err := c.Runtime.Validate(); err != nil {
		return err
	}
	if c.Enabled.Kafka {
		if len(c.Kafka) == 0 {
			return fmt.Errorf("worker: kafka consumers are required")
		}
		for _, v := range c.Kafka {
			if v == nil || len(v.Host) == 0 || v.Topic == "" || v.ConsumerGroup == "" || v.ConsumerCount <= 0 || v.FailCount < 0 {
				return fmt.Errorf("worker: invalid kafka consumer")
			}
			for _, host := range v.Host {
				if strings.TrimSpace(host) == "" {
					return fmt.Errorf("worker: empty Kafka broker")
				}
			}
			if err := c.Runtime.validateKafka(v); err != nil {
				return err
			}
			if v.SASL != nil && v.SASL.Enabled && (v.SASL.User == "" || v.SASL.Password == "") {
				return fmt.Errorf("worker: enabled Kafka SASL requires user and password")
			}
		}
	}
	if c.Enabled.Rabbitmq {
		if len(c.Rabbitmq) == 0 {
			return fmt.Errorf("worker: rabbitmq consumers are required")
		}
		for _, v := range c.Rabbitmq {
			if v == nil || v.URL == "" || v.ConsumerQueue == "" || v.ConsumerCount <= 0 || v.FailCount < 0 || v.PrefetchCount < 0 {
				return fmt.Errorf("worker: invalid rabbitmq consumer")
			}
		}
	}
	if c.Enabled.Redis {
		if len(c.Redis) == 0 {
			return fmt.Errorf("worker: redis consumers are required")
		}
		for _, v := range c.Redis {
			if v == nil || v.Addr == "" || v.Queue == "" || v.ConsumerCount <= 0 || v.Db < 0 || v.FailCount < 0 || (v.DeliveryMode != "" && v.DeliveryMode != "reliable" && v.DeliveryMode != "legacy") || (v.FailQueue != "" && v.FailQueue == v.Queue) {
				return fmt.Errorf("worker: invalid redis consumer")
			}
		}
	}
	if c.Enabled.Delay {
		if len(c.Delay) == 0 {
			return fmt.Errorf("worker: delay consumers are required")
		}
		for _, v := range c.Delay {
			if v == nil || v.Addr == "" || v.Queue == "" || v.ConsumerCount <= 0 || v.Db < 0 || v.FailCount < 0 || (v.DeliveryMode != "" && v.DeliveryMode != "reliable" && v.DeliveryMode != "legacy") || (v.FailQueue != "" && v.FailQueue == v.Queue) {
				return fmt.Errorf("worker: invalid delay consumer")
			}
		}
	}
	return nil
}

// Snapshot prevents caller mutation after goroutines have been started.
func (c *Configure) Snapshot() *Configure {
	if c == nil {
		return nil
	}
	copy := *c
	if c.Enabled != nil {
		enabled := *c.Enabled
		copy.Enabled = &enabled
	}
	copy.Kafka = make([]*KafkaConfig, len(c.Kafka))
	for i, v := range c.Kafka {
		if v != nil {
			item := *v
			item.Host = append([]string(nil), v.Host...)
			if v.SASL != nil {
				sasl := *v.SASL
				item.SASL = &sasl
			}
			copy.Kafka[i] = &item
		}
	}
	copy.Rabbitmq = make([]*RabbitmqConfig, len(c.Rabbitmq))
	for i, v := range c.Rabbitmq {
		if v != nil {
			item := *v
			copy.Rabbitmq[i] = &item
		}
	}
	copy.Redis = make([]*RedisConfig, len(c.Redis))
	for i, v := range c.Redis {
		if v != nil {
			item := *v
			copy.Redis[i] = &item
		}
	}
	copy.Delay = make([]*DelayConfig, len(c.Delay))
	for i, v := range c.Delay {
		if v != nil {
			item := *v
			copy.Delay[i] = &item
		}
	}
	return &copy
}
