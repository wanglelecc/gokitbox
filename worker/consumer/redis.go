package consumer

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wanglelecc/gokitbox/logger"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

type RedisConsumer struct {
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
	if len(configs) == 0 {
		err = errors.New("redis config not found")
		return
	}
	consumer = new(RedisConsumer)
	consumer.callback = callback
	consumer.alertMgr = alertMgr
	consumer.quit = make(chan struct{})
	consumer.wg = new(sync.WaitGroup)
	consumer.clients = make([]*redis.Client, 0)

	consumer.wg.Add(1)
	go consumer.count()

	// 按 addr+username+password+db 复用 redis.Client，避免相同地址创建多个独立连接池
	clientCache := make(map[string]*redis.Client)
	for _, config := range configs {
		cacheKey := fmt.Sprintf("%s|%s|%s|%d", config.Addr, config.Username, config.Password, config.Db)
		client, ok := clientCache[cacheKey]
		if !ok {
			client = redis.NewClient(&redis.Options{
				Addr:     config.Addr,
				Username: config.Username,
				Password: config.Password,
				DB:       config.Db,
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

	ctx := context.Background()
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
	defer func() {
		if rec := recover(); rec != nil {
			atomic.AddInt64(&r.panicCount, 1)
			var msg string
			if err, ok := rec.(error); ok {
				msg = err.Error()
				logger.Ex(ctx, "IRedisPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			} else {
				msg = fmt.Sprintf("%v", rec)
				logger.Ex(ctx, "IRedisPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			}
			r.alertMgr.Emit(AlertEvent{
				Component: ComponentRedis,
				Name:      config.Queue,
				Type:      AlertPanic,
				Message:   msg,
			})
			exitClean = false
		}
	}()

	bo.Reset()
	logger.Ix(ctx, "RedisConsume", fmt.Sprintf("Start consume from redis %v:%v", config.Addr, config.Queue))

	delay := 0
	t := time.NewTicker(time.Second * 300)
	defer t.Stop()

	for {
		select {
		case <-r.quit:
			logger.Wx(ctx, "RedisConsume", "IREDIS_RECV_QUIT")
			return true
		case <-t.C:
			queueLen := client.LLen(ctx, config.Queue).Val()
			if queueLen > 2000 {
				logger.Wx(ctx, "RedisConsumePileUp", "redis queue pile up", "queue", config.Queue, "len", queueLen)
				r.alertMgr.Emit(AlertEvent{
					Component: ComponentRedis,
					Name:      config.Queue,
					Type:      AlertQueueBacklog,
					Message:   fmt.Sprintf("queue backlog %d > 2000", queueLen),
				})
			}
		default:
			values, err := client.BLPop(ctx, time.Second, config.Queue).Result()
			if err == redis.Nil {
				delay++
			} else if err != nil {
				delay++
				logger.Ex(ctx, "RedisConsume", "redis cmd exec error", "error", err.Error())
			} else {
				delay = 0

				fLen := len(values) / 2
				for i := 0; i < fLen; i++ {
					value := values[2*i+1]

					count := 0
					for {
						ret := r.callback(config.TplName, config.Queue, []byte(value))
						if ret {
							atomic.AddInt64(&r.successCount, int64(1))
							logger.Dx(ctx, "RedisConsume", fmt.Sprintf("return_true:KEY:%s,VAL:%s", string(config.Queue), string(value)))
							break
						} else {
							if count >= config.FailCount {
								atomic.AddInt64(&r.errorCount, int64(1))
								if config.FailQueue != "" {
									if err := client.RPush(ctx, config.FailQueue, value).Err(); err != nil {
										logger.Ex(ctx, "RedisConsume", fmt.Sprintf("send to fail queue error: %v, KEY:%s,VAL:%s", err, config.Queue, value))
									} else {
										logger.Dx(ctx, "RedisConsume", fmt.Sprintf("Send to Fail queue :KEY:%s,VAL:%s", string(config.Queue), string(value)))
									}
								} else {
									logger.Ex(ctx, "RedisConsume", fmt.Sprintf("Fail queue is empty:KEY:%s,VAL:%s", string(config.Queue), string(value)))
									r.alertMgr.Emit(AlertEvent{
										Component: ComponentRedis,
										Name:      config.Queue,
										Type:      AlertMsgDiscard,
										Message:   fmt.Sprintf("no fail queue configured, discarding after %d retries", config.FailCount),
									})
								}
								break
							}
							count++
						}
					}
				}
			}
		}

		if delay > 0 {
			if delay > 5 {
				delay = 5
			}
			// H-1: 用 select 替代 time.Sleep，quit 信号能立即打断等待
			select {
			case <-r.quit:
				return true
			case <-time.After(time.Duration(delay*100) * time.Millisecond):
			}
		}
	}
}

// CloseWithTimeout 带超时的优雅关闭
func (r *RedisConsumer) CloseWithTimeout(timeout time.Duration) {
	// 使用 atomic CAS 确保只关闭一次
	if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		// 已经关闭过了，直接返回
		return
	}

	ctx := context.Background()
	tag := "RedisClose"

	logger.Ix(ctx, tag, fmt.Sprintf("Closing Redis consumer (timeout: %v)", timeout))
	close(r.quit)

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Ix(ctx, tag, "Redis consumer closed gracefully")
	case <-time.After(timeout):
		logger.Wx(ctx, tag, fmt.Sprintf(
			"Redis consumer close timeout after %v", timeout))
	}

	// 关闭所有 redis 客户端
	for _, client := range r.clients {
		if err := client.Close(); err != nil {
			logger.Ex(ctx, tag, "Close redis client error", "error", err.Error())
		}
	}
}

// Close 保持向后兼容
func (r *RedisConsumer) Close() {
	r.CloseWithTimeout(10 * time.Second)
}
