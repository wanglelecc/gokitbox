package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wanglelecc/gokitbox/logger"
)

type ConsumeManager struct {
	stopOnce  sync.Once
	closeDone chan struct{}
	closeErr  error

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
	manager, err := NewConsumeManagerWithConfig(loadConfigure(), callback, alertMgr)
	if err != nil {
		panic(err)
	}
	return manager
}

// NewConsumeManagerWithConfig accepts application-owned validated configuration.
func NewConsumeManagerWithConfig(cfg *Configure, callback ConsumeCallback, alertMgr *AlertManager) (*ConsumeManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	manager := &ConsumeManager{cfg: cfg}
	var err error
	fail := func(kind string) (*ConsumeManager, error) {
		manager.Close()
		return nil, fmt.Errorf("worker: initialize %s consumer failed", kind)
	}
	if cfg.Enabled.Kafka {
		manager.kafka, err = NewKafkaConsumer(cfg.Kafka, callback, alertMgr)
		if err != nil {
			return fail("kafka")
		}
	}

	if cfg.Enabled.Rabbitmq {
		manager.rabbitmq, err = NewRabbitmqConsumer(cfg.Rabbitmq, callback, alertMgr)
		if err != nil {
			return fail("rabbitmq")
		}
	}

	if cfg.Enabled.Redis {
		manager.redis, err = NewRedisConsumer(cfg.Redis, callback, alertMgr)
		if err != nil {
			return fail("redis")
		}
	}

	if cfg.Enabled.Delay {
		manager.delay, err = NewDelayConsumer(cfg.Delay, callback, alertMgr)
		if err != nil {
			return fail("delay")
		}
	}

	return manager, nil
}

// CloseWithTimeout preserves the old API while reporting incomplete drains.
func (c *ConsumeManager) CloseWithTimeout(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := c.CloseContext(ctx); err != nil {
		logger.Wx(context.Background(), "ConsumeManager", "shutdown incomplete", "error", err)
	}
}
func (c *ConsumeManager) Close() { c.CloseWithTimeout(30 * time.Second) }
func (c *ConsumeManager) CloseContext(ctx context.Context) error {
	c.stopOnce.Do(func() {
		c.closeDone = make(chan struct{})
		atomic.StoreInt32(&c.closed, 1)
		go func() {
			defer close(c.closeDone)
			var wg sync.WaitGroup
			var mu sync.Mutex
			stop := func(close func(context.Context) error) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := close(context.WithoutCancel(ctx))
					mu.Lock()
					c.closeErr = errors.Join(c.closeErr, err)
					mu.Unlock()
				}()
			}
			if c.kafka != nil {
				stop(c.kafka.CloseContext)
			}
			if c.rabbitmq != nil {
				stop(c.rabbitmq.CloseContext)
			}
			if c.redis != nil {
				stop(c.redis.CloseContext)
			}
			if c.delay != nil {
				stop(c.delay.CloseContext)
			}
			wg.Wait()
		}()
	})
	select {
	case <-c.closeDone:
		return c.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
