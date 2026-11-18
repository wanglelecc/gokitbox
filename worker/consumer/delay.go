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

type DelayConsumer struct {
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

func NewDelayConsumer(configs []*DelayConfig, callback ConsumeCallback, alertMgr *AlertManager) (consumer *DelayConsumer, err error) {
	return NewDelayConsumerWithContext(configs, adaptCallback(callback), RuntimeOptions{}, alertMgr)
}
func NewDelayConsumerWithContext(configs []*DelayConfig, callback ContextConsumeCallback, options RuntimeOptions, alertMgr *AlertManager) (consumer *DelayConsumer, err error) {
	cfg := (&Configure{Enabled: &ConsumeEnabled{Delay: true}, Delay: configs, Runtime: options}).Snapshot()
	if callback == nil {
		return nil, errors.New("worker: callback is required")
	}
	if validationErr := cfg.Validate(); validationErr != nil {
		return nil, validationErr
	}
	configs = cfg.Delay

	if len(configs) == 0 {
		err = errors.New("delay config not found")
		return
	}
	consumer = new(DelayConsumer)
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

func (d *DelayConsumer) count() {
	defer d.wg.Done()

	ctx := context.Background()
	tag := "DelayCount"

	t := time.NewTicker(time.Second * 300)
	defer t.Stop()

	for {
		select {
		case <-d.quit:
			logger.Ix(ctx, tag, "count goroutine exit")
			return
		case <-t.C:
			succ := atomic.SwapInt64(&d.successCount, 0)
			fail := atomic.SwapInt64(&d.errorCount, 0)
			logger.Ix(ctx, tag, fmt.Sprintf("Stat succ:%d,fail:%d", succ, fail))
		}
	}
}

// consume 外层自愈循环：doConsume 异常退出后按指数退避重启
func (d *DelayConsumer) consume(client *redis.Client, config *DelayConfig) {
	defer d.wg.Done()
	atomic.AddInt32(&d.activeGoroutines, 1)
	defer atomic.AddInt32(&d.activeGoroutines, -1)

	ctx := d.runtime.context()
	bo := newBackoff(time.Second, 30*time.Second)

	for {
		select {
		case <-d.quit:
			return
		default:
		}

		if exitClean := d.doConsume(ctx, client, config, bo); exitClean {
			return
		}

		atomic.AddInt64(&d.restartCount, 1)
		wait := bo.Next()
		logger.Wx(ctx, "DelayConsume", fmt.Sprintf("goroutine restarting in %v", wait))
		d.alertMgr.Emit(AlertEvent{
			Component: ComponentDelay,
			Name:      config.Queue,
			Type:      AlertRestart,
			Message:   fmt.Sprintf("consumer goroutine restarting in %v", wait),
		})
		select {
		case <-d.quit:
			return
		case <-time.After(wait):
		}
	}
}

// doConsume 单次消费生命周期
// 返回 true 表示收到 quit 信号，正常退出；false 表示异常，外层需重启
func (d *DelayConsumer) doConsume(ctx context.Context, client *redis.Client, config *DelayConfig, bo *backoff) (exitClean bool) {
	if config.DeliveryMode != "legacy" {
		return d.runtime.consumeReliable(ctx, d.quit, client, config.Queue, "delay", config.FailQueue, config.FailCount, config.TplName, d.callback, &d.successCount, &d.errorCount)
	}
	defer func() {
		if p := recover(); p != nil {
			atomic.AddInt64(&d.panicCount, 1)
			d.alertMgr.Emit(AlertEvent{Component: ComponentDelay, Name: config.Queue, Type: AlertPanic, Message: fmt.Sprintf("consumer panic (%T)", p)})
			logger.Ex(ctx, "DelayConsume", "consumer panic", "type", fmt.Sprintf("%T", p))
			exitClean = false
		}
	}()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		if stopRequested(ctx, d.quit) {
			return true
		}
		select {
		case <-ticker.C:
			checkCtx, stop := boundedContext(ctx, d.runtime.opts().OperationTimeout)
			length, err := client.ZCount(checkCtx, config.Queue, "-inf", fmt.Sprint(time.Now().Unix())).Result()
			stop()
			if err != nil {
				logger.Wx(ctx, "DelayConsume", "backlog check failed", "queue", config.Queue, "error", err)
			} else if length > 2000 {
				d.alertMgr.Emit(AlertEvent{Component: ComponentDelay, Name: config.Queue, Type: AlertQueueBacklog, Message: fmt.Sprintf("queue backlog %d > 2000", length)})
			}
		default:
		}

		operation, cancel := boundedContext(ctx, d.runtime.opts().OperationTimeout)
		value, err := d.pop(operation, client, config.Queue, time.Now().Unix())
		cancel()
		if errors.Is(err, redis.Nil) {
			if !waitRetry(ctx, d.quit, 100*time.Millisecond) {
				return true
			}
			continue
		}
		if err != nil {
			logger.Wx(ctx, "DelayConsume", "pop failed", "error", err)
			return stopRequested(ctx, d.quit)
		}
		var forward func(context.Context) error
		if config.FailQueue != "" {
			forward = func(operation context.Context) error {
				return client.ZAdd(operation, config.FailQueue, redis.Z{Score: float64(time.Now().Unix()), Member: value}).Err()
			}
		}
		restore := func(operation context.Context) error {
			return client.ZAddNX(operation, config.Queue, redis.Z{Score: float64(time.Now().Unix()), Member: value}).Err()
		}
		resolved, err := d.runtime.processLegacy(ctx, d.quit, d.callback, config.TplName, config.Queue, []byte(value), config.FailCount, &d.successCount, &d.errorCount, forward, restore, func() {
			d.alertMgr.Emit(AlertEvent{Component: ComponentDelay, Name: config.Queue, Type: AlertMsgDiscard, Message: "no failure queue; retry limit reached"})
		})
		if err != nil {
			d.runtime.recordError(err)
			logger.Ex(ctx, "DelayConsume", "message restore failed", "queue", config.Queue, "error", err)
		}
		if !resolved {
			return stopRequested(ctx, d.quit)
		}
		bo.Reset()
	}
}

// duePopScript leaves future members untouched. The due check and delete are
// one single-key operation, including on Redis Cluster.
const duePopScript = `
local item=redis.call('ZRANGE',KEYS[1],0,0,'WITHSCORES')
if #item==0 or tonumber(item[2])>tonumber(ARGV[1]) then return false end
redis.call('ZREM',KEYS[1],item[1])
return item[1]`

func (d *DelayConsumer) pop(ctx context.Context, client *redis.Client, queue string, nowUnix int64) (string, error) {
	return client.Eval(ctx, duePopScript, []string{queue}, nowUnix).Text()
}

// CloseWithTimeout 带超时的优雅关闭
func (d *DelayConsumer) CloseWithTimeout(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := d.CloseContext(ctx); err != nil {
		logger.Wx(context.Background(), "DelayConsumer.Close", "shutdown incomplete", "error", err)
	}
}

// CloseContext reports the actual drain outcome. A retry waits for the same
// shutdown instead of returning success while the first caller is still draining.
func (d *DelayConsumer) CloseContext(ctx context.Context) error {
	d.stopOnce.Do(func() {
		d.closeDone = make(chan struct{})
		atomic.StoreInt32(&d.closed, 1)
		close(d.quit)
		go func() {
			defer close(d.closeDone)
			stopCancellation := d.runtime.cancelOnDeadline(ctx)
			defer stopCancellation()
			if d.runtime != nil {
				defer d.runtime.cancel()
			}
			d.wg.Wait()
			d.closeErr = errors.Join(d.closeErr, d.runtime.drainError())
			for _, client := range d.clients {
				if err := client.Close(); err != nil {
					d.closeErr = errors.Join(d.closeErr, err)
				}
			}
		}()
	})
	select {
	case <-d.closeDone:
		return d.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close 保持向后兼容
func (d *DelayConsumer) Close() {
	d.CloseWithTimeout(10 * time.Second)
}
