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

type DelayConsumer struct {
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
	if len(configs) == 0 {
		err = errors.New("delay config not found")
		return
	}
	consumer = new(DelayConsumer)
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

	ctx := context.Background()
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
	defer func() {
		if rec := recover(); rec != nil {
			atomic.AddInt64(&d.panicCount, 1)
			var msg string
			if err, ok := rec.(error); ok {
				msg = err.Error()
				logger.Ex(ctx, "IDelayPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			} else {
				msg = fmt.Sprintf("%v", rec)
				logger.Ex(ctx, "IDelayPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			}
			d.alertMgr.Emit(AlertEvent{
				Component: ComponentDelay,
				Name:      config.Queue,
				Type:      AlertPanic,
				Message:   msg,
			})
			exitClean = false
		}
	}()

	bo.Reset()
	logger.Ix(ctx, "DelayConsume", fmt.Sprintf("Start consume from delay %v:%v", config.Addr, config.Queue))

	delay := 0
	t := time.NewTicker(time.Second * 300)
	defer t.Stop()

	for {
		select {
		case <-d.quit:
			logger.Wx(ctx, "DelayConsume", "IDELAY_RECV_QUIT")
			return true
		case <-t.C:
			queueLen := client.ZCount(ctx, config.Queue, "1", cast.ToString(time.Now().Unix())).Val()
			if queueLen > 2000 {
				logger.Wx(ctx, "DelayConsumePileUp", "Delay queue pile up", "queue", config.Queue, "len", queueLen)
				d.alertMgr.Emit(AlertEvent{
					Component: ComponentDelay,
					Name:      config.Queue,
					Type:      AlertQueueBacklog,
					Message:   fmt.Sprintf("delay queue backlog %d > 2000", queueLen),
				})
			}
		default:
			member, err := d.pop(ctx, client, config.Queue, time.Now().Unix())
			if errors.Is(err, redis.Nil) {
				delay++
			} else if err != nil {
				delay++
				logger.Ex(ctx, "DelayConsume", "redis cmd exec error", "error", err.Error())
			} else {
				delay = 0
				count := 0
				for {
					ret := d.callback(config.TplName, config.Queue, []byte(member))
					if ret {
						atomic.AddInt64(&d.successCount, int64(1))
						logger.Dx(ctx, "DelayConsume", fmt.Sprintf("return_true:KEY:%s,VAL:%s", string(config.Queue), member))
						break
					} else {
						if count >= config.FailCount {
							atomic.AddInt64(&d.errorCount, int64(1))
							if config.FailQueue != "" {
								failZ := redis.Z{Score: float64(time.Now().Unix()), Member: member}
								if err := client.ZAdd(ctx, config.FailQueue, failZ).Err(); err != nil {
									logger.Ex(ctx, "DelayConsume", fmt.Sprintf("send to fail queue error: %v, KEY:%s,VAL:%s", err, config.Queue, member))
								} else {
									logger.Dx(ctx, "DelayConsume", fmt.Sprintf("Send to Fail queue :KEY:%s,VAL:%s", string(config.Queue), member))
								}
							} else {
								logger.Ex(ctx, "DelayConsume", fmt.Sprintf("Fail queue is empty:KEY:%s,VAL:%s", string(config.Queue), member))
								d.alertMgr.Emit(AlertEvent{
									Component: ComponentDelay,
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

		if delay > 0 {
			if delay > 5 {
				delay = 5
			}
			// H-1: 用 select 替代 time.Sleep，quit 信号能立即打断等待
			select {
			case <-d.quit:
				return true
			case <-time.After(time.Duration(delay*100) * time.Millisecond):
			}
		}
	}
}

// pop 从延迟 ZSet 中弹出一个已到期（score ≤ nowUnix）的成员。
//
// 实现说明：
//   - ZPopMin 是 Redis 原子命令，并发调用不会返回同一元素，无重复消费问题。
//   - 弹出后校验分值：未到期则 ZAdd 回写，对 Redis Cluster / 代理完全兼容。
//   - 去掉 ZCount 预检，减少一次 RTT；ZPopMin 返回空即无到期元素。
func (d *DelayConsumer) pop(ctx context.Context, client *redis.Client, queue string, nowUnix int64) (member string, err error) {
	vs, err := client.ZPopMin(ctx, queue, 1).Result()
	if err != nil {
		return "", err
	}
	if len(vs) == 0 {
		return "", redis.Nil
	}

	v := vs[0]
	if cast.ToInt64(v.Score) > nowUnix {
		// 最小分值元素尚未到期，回写后告知调用方无可消费元素
		if putErr := client.ZAdd(ctx, queue, v).Err(); putErr != nil {
			// 回写失败仍返回 Nil，该元素丢失风险由日志告警；不提前消费
			logger.Ex(ctx, "DelayPop", fmt.Sprintf("ZAdd back failed: %v, member may be lost: %v", putErr, v.Member))
		}
		return "", redis.Nil
	}

	return cast.ToString(v.Member), nil
}

// CloseWithTimeout 带超时的优雅关闭
func (d *DelayConsumer) CloseWithTimeout(timeout time.Duration) {
	// 使用 atomic CAS 确保只关闭一次
	if !atomic.CompareAndSwapInt32(&d.closed, 0, 1) {
		// 已经关闭过了，直接返回
		return
	}

	ctx := context.Background()
	tag := "DelayClose"

	logger.Ix(ctx, tag, fmt.Sprintf("Closing Delay consumer (timeout: %v)", timeout))
	close(d.quit)

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Ix(ctx, tag, "Delay consumer closed gracefully")
	case <-time.After(timeout):
		logger.Wx(ctx, tag, fmt.Sprintf(
			"Delay consumer close timeout after %v", timeout))
	}

	// 关闭所有 redis 客户端
	for _, client := range d.clients {
		if err := client.Close(); err != nil {
			logger.Ex(ctx, tag, "Close redis client error", "error", err.Error())
		}
	}
}

// Close 保持向后兼容
func (d *DelayConsumer) Close() {
	d.CloseWithTimeout(10 * time.Second)
}
