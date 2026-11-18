package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wanglelecc/gokitbox/logger"
)

type RedisConsumer struct {
	runtime   *consumerRuntime
	stopOnce  sync.Once
	closeDone chan struct{}
	closeErr  error

	quit         chan struct{}
	callback     ConsumeCallback
	wg           *sync.WaitGroup
	clients      []*redis.Client
	successCount int64
	errorCount   int64
	closed       int32 // 关闭标志，使用 atomic 操作，0表示未关闭，1表示已关闭

	// 健康指标
	activeGoroutines int32
	panicCount       int64
	restartCount     int64

	alertMgr *AlertManager
}

func NewRedisConsumer(configs []*RedisConfig, callback ConsumeCallback, alertMgr *AlertManager) (consumer *RedisConsumer, err error) {
	return NewRedisConsumerWithContext(configs, adaptCallback(callback), RuntimeOptions{}, alertMgr)
}
func NewRedisConsumerWithContext(configs []*RedisConfig, callback ContextConsumeCallback, options RuntimeOptions, alertMgr *AlertManager) (consumer *RedisConsumer, err error) {
	cfg := (&Configure{Enabled: &ConsumeEnabled{Redis: true}, Redis: configs, Runtime: options}).Snapshot()
	if callback == nil {
		return nil, errors.New("worker: callback is required")
	}
	if validationErr := cfg.Validate(); validationErr != nil {
		return nil, validationErr
	}
	configs = cfg.Redis

	if len(configs) == 0 {
		err = errors.New("redis config not found")
		return
	}
	consumer = new(RedisConsumer)
	consumer.runtime = newRuntime(options, callback)
	consumer.alertMgr = alertMgr
	consumer.quit = make(chan struct{})
	consumer.wg = new(sync.WaitGroup)
	consumer.clients = make([]*redis.Client, 0)

	consumer.wg.Add(1)
	go consumer.count()

	// 按 addr+username+password+db 复用 redis.Client，避免相同地址创建多个独立连接池
	clientCache := make(map[redisConnection]*redis.Client)
	for _, config := range configs {
		cacheKey := redisConnection{config.Addr, config.Username, config.Password, config.Db}
		client, ok := clientCache[cacheKey]
		if !ok {
			client = redis.NewClient(&redis.Options{
				Addr:                  config.Addr,
				Username:              config.Username,
				Password:              config.Password,
				DB:                    config.Db,
				ReadTimeout:           options.defaults().OperationTimeout,
				WriteTimeout:          options.defaults().OperationTimeout,
				ContextTimeoutEnabled: true,
				DialTimeout:           options.defaults().OperationTimeout,
				MaxRetries:            -1,
			})
			clientCache[cacheKey] = client
			consumer.clients = append(consumer.clients, client) // 只记录唯一 client，用于 Close
		}

		for i := 0; i < config.ConsumerCount; i++ {
			consumer.wg.Add(1)
			go consumer.consume(client, config)
		}
	}

	return
}

func (r *RedisConsumer) count() {
	defer r.wg.Done()

	ctx := context.Background()
	tag := "RedisCount"

	t := time.NewTicker(time.Second * 300)
	defer t.Stop()

	for {
		select {
		case <-r.quit:
			logger.Ix(ctx, tag, "count goroutine exit")
			return
		case <-t.C:
			succ := atomic.SwapInt64(&r.successCount, 0)
			fail := atomic.SwapInt64(&r.errorCount, 0)
			logger.Ix(ctx, tag, fmt.Sprintf("Stat succ:%d,fail:%d", succ, fail))
		}
	}
}

// consume 外层自愈循环：doConsume 异常退出后按指数退避重启
func (r *RedisConsumer) consume(client *redis.Client, config *RedisConfig) {
	defer r.wg.Done()
	atomic.AddInt32(&r.activeGoroutines, 1)
	defer atomic.AddInt32(&r.activeGoroutines, -1)

	ctx := r.runtime.context()
	bo := newBackoff(time.Second, 30*time.Second)

	for {
		select {
		case <-r.quit:
			return
		default:
		}

		if exitClean := r.doConsume(ctx, client, config, bo); exitClean {
			return
		}

		atomic.AddInt64(&r.restartCount, 1)
		wait := bo.Next()
		logger.Wx(ctx, "RedisConsume", fmt.Sprintf("goroutine restarting in %v", wait))
		r.alertMgr.Emit(AlertEvent{
			Component: ComponentRedis,
			Name:      config.Queue,
			Type:      AlertRestart,
			Message:   fmt.Sprintf("consumer goroutine restarting in %v", wait),
		})
		select {
		case <-r.quit:
			return
		case <-time.After(wait):
		}
	}
}

// doConsume 单次消费生命周期
// 返回 true 表示收到 quit 信号，正常退出；false 表示异常，外层需重启
func (r *RedisConsumer) doConsume(ctx context.Context, client *redis.Client, config *RedisConfig, bo *backoff) (exitClean bool) {
	if config.DeliveryMode != "legacy" {
		return r.runtime.consumeReliable(ctx, r.quit, client, config.Queue, "list", config.FailQueue, config.FailCount, config.TplName, r.callback, &r.successCount, &r.errorCount)
	}
	defer func() {
		if p := recover(); p != nil {
			atomic.AddInt64(&r.panicCount, 1)
			r.alertMgr.Emit(AlertEvent{Component: ComponentRedis, Name: config.Queue, Type: AlertPanic, Message: fmt.Sprintf("consumer panic (%T)", p)})
			logger.Ex(ctx, "RedisConsume", "consumer panic", "type", fmt.Sprintf("%T", p))
			exitClean = false
		}
	}()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		if stopRequested(ctx, r.quit) {
			return true
		}
		select {
		case <-ticker.C:
			checkCtx, stop := boundedContext(ctx, r.runtime.opts().OperationTimeout)
			length, err := client.LLen(checkCtx, config.Queue).Result()
			stop()
			if err != nil {
				logger.Wx(ctx, "RedisConsume", "backlog check failed", "queue", config.Queue, "error", err)
			} else if length > 2000 {
				r.alertMgr.Emit(AlertEvent{Component: ComponentRedis, Name: config.Queue, Type: AlertQueueBacklog, Message: fmt.Sprintf("queue backlog %d > 2000", length)})
			}
		default:
		}

		operation, cancel := boundedContext(ctx, r.runtime.opts().OperationTimeout)
		values, err := client.BLPop(operation, time.Second, config.Queue).Result()
		cancel()
		if err == redis.Nil {
			if !waitRetry(ctx, r.quit, 100*time.Millisecond) {
				return true
			}
			continue
		}
		if err != nil {
			logger.Wx(ctx, "RedisConsume", "pop failed", "error", err)
			return stopRequested(ctx, r.quit)
		}
		if len(values) != 2 {
			logger.Wx(ctx, "RedisConsume", "invalid BLPOP response")
			return false
		}
		value := values[1]
		var forward func(context.Context) error
		if config.FailQueue != "" {
			forward = func(operation context.Context) error { return client.RPush(operation, config.FailQueue, value).Err() }
		}
		restore := func(operation context.Context) error { return client.RPush(operation, config.Queue, value).Err() }
		resolved, err := r.runtime.processLegacy(ctx, r.quit, r.callback, config.TplName, config.Queue, []byte(value), config.FailCount, &r.successCount, &r.errorCount, forward, restore, func() {
			r.alertMgr.Emit(AlertEvent{Component: ComponentRedis, Name: config.Queue, Type: AlertMsgDiscard, Message: "no failure queue; retry limit reached"})
		})
		if err != nil {
			r.runtime.recordError(err)
			logger.Ex(ctx, "RedisConsume", "message restore failed", "queue", config.Queue, "error", err)
		}
		if !resolved {
			return stopRequested(ctx, r.quit)
		}
		bo.Reset()
	}
}

// CloseWithTimeout 带超时的优雅关闭
func (r *RedisConsumer) CloseWithTimeout(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := r.CloseContext(ctx); err != nil {
		logger.Wx(context.Background(), "RedisConsumer.Close", "shutdown incomplete", "error", err)
	}
}

// CloseContext reports the actual drain outcome. A retry waits for the same
// shutdown instead of returning success while the first caller is still draining.
func (r *RedisConsumer) CloseContext(ctx context.Context) error {
	r.stopOnce.Do(func() {
		r.closeDone = make(chan struct{})
		atomic.StoreInt32(&r.closed, 1)
		close(r.quit)
		go func() {
			defer close(r.closeDone)
			stopCancellation := r.runtime.cancelOnDeadline(ctx)
			defer stopCancellation()
			if r.runtime != nil {
				defer r.runtime.cancel()
			}
			r.wg.Wait()
			r.closeErr = errors.Join(r.closeErr, r.runtime.drainError())
			for _, client := range r.clients {
				if err := client.Close(); err != nil {
					r.closeErr = errors.Join(r.closeErr, err)
				}
			}
		}()
	})
	select {
	case <-r.closeDone:
		return r.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close 保持向后兼容
func (r *RedisConsumer) Close() {
	r.CloseWithTimeout(10 * time.Second)
}
