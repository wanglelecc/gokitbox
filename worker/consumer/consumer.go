package consumer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wanglelecc/gokitbox/logger"
)

type ConsumeManager struct {
	cfg      *Configure
	wg       *sync.WaitGroup
	kafka    *KafkaConsumer
	rabbitmq *RabbitmqConsumer
	redis    *RedisConsumer
	delay    *DelayConsumer
	closed   int32
}

type ConsumeCallback func(TplName string, Key string, Value []byte) bool

func NewConsumeManager(callback ConsumeCallback, alertMgr *AlertManager) *ConsumeManager {
	ctx := context.Background()

	var err error
	manager := new(ConsumeManager)
	cfg := loadConfigure()
	manager.cfg = cfg

	if cfg.Enabled.Kafka {
		manager.kafka, err = NewKafkaConsumer(cfg.Kafka, callback, alertMgr)
		if err != nil {
			logger.Fx(ctx, "NewConsumerManager", "NewKafkaConsumer error", "error", err.Error())
			panic(fmt.Sprintf("failed to initialize kafka consumer: %v", err))
		}
	}

	if cfg.Enabled.Rabbitmq {
		manager.rabbitmq, err = NewRabbitmqConsumer(cfg.Rabbitmq, callback, alertMgr)
		if err != nil {
			logger.Fx(ctx, "NewConsumerManager", "NewRabbitmqConsumer error", "error", err.Error())
			panic(fmt.Sprintf("failed to initialize rabbitmq consumer: %v", err))
		}
	}

	if cfg.Enabled.Redis {
		manager.redis, err = NewRedisConsumer(cfg.Redis, callback, alertMgr)
		if err != nil {
			logger.Fx(ctx, "NewConsumerManager", "NewRedisConsumer error", "error", err.Error())
			panic(fmt.Sprintf("failed to initialize redis consumer: %v", err))
		}
	}

	if cfg.Enabled.Delay {
		manager.delay, err = NewDelayConsumer(cfg.Delay, callback, alertMgr)
		if err != nil {
			logger.Fx(ctx, "NewConsumerManager", "NewDelayConsumer error", "error", err.Error())
			panic(fmt.Sprintf("failed to initialize delay consumer: %v", err))
		}
	}

	return manager
}

// CloseWithTimeout 带超时的优雅关闭
func (c *ConsumeManager) CloseWithTimeout(timeout time.Duration) {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}

	ctx := context.Background()
	logger.Ix(ctx, "ConsumeManager", fmt.Sprintf(
		"Closing all consumers (timeout: %v)", timeout))

	startTime := time.Now()

	// 计算每个组件的超时时间（预留10%的 buffer）
	perConsumerTimeout := timeout * 9 / 10

	// 并发关闭所有消费者
	var wg sync.WaitGroup
	closeConsumer := func(name string, closer func(time.Duration)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			closer(perConsumerTimeout)
			logger.Dx(ctx, "ConsumeManager", fmt.Sprintf(
				"%s closed in %v", name, time.Since(start)))
		}()
	}

	if c.cfg.Enabled.Kafka && c.kafka != nil {
		closeConsumer("Kafka", c.kafka.CloseWithTimeout)
	}
	if c.cfg.Enabled.Rabbitmq && c.rabbitmq != nil {
		closeConsumer("RabbitMQ", c.rabbitmq.CloseWithTimeout)
	}
	if c.cfg.Enabled.Redis && c.redis != nil {
		closeConsumer("Redis", c.redis.CloseWithTimeout)
	}
	if c.cfg.Enabled.Delay && c.delay != nil {
		closeConsumer("Delay", c.delay.CloseWithTimeout)
	}

	// 等待所有关闭完成（带超时保护）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Ix(ctx, "ConsumeManager", fmt.Sprintf(
			"All consumers closed in %v", time.Since(startTime)))
	case <-time.After(timeout):
		logger.Wx(ctx, "ConsumeManager", fmt.Sprintf(
			"Consumer close timeout after %v", timeout))
	}
}

// Close 保持向后兼容（使用合理的默认超时）
func (c *ConsumeManager) Close() {
	c.CloseWithTimeout(30 * time.Second)
}
