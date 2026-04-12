package consumer

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cast"
	"github.com/wanglelecc/gokitbox/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitmqConsumer struct {
	quit     chan struct{}
	callback ConsumeCallback
	wg       *sync.WaitGroup
	configs  []*RabbitmqConfig
	closed   int32

	// 健康指标
	activeGoroutines int32
	panicCount       int64
	restartCount     int64

	alertMgr *AlertManager
}

func NewRabbitmqConsumer(configs []*RabbitmqConfig, callback ConsumeCallback, alertMgr *AlertManager) (*RabbitmqConsumer, error) {
	if len(configs) == 0 {
		return nil, errors.New("rabbitmq config not found")
	}

	consumer := &RabbitmqConsumer{
		callback: callback,
		quit:     make(chan struct{}),
		wg:       new(sync.WaitGroup),
		configs:  configs,
		alertMgr: alertMgr,
	}

	for _, config := range configs {
		for i := 0; i < config.ConsumerCount; i++ {
			consumer.wg.Add(1)
			go consumer.consume(config, i)
		}
	}

	return consumer, nil
}

// CloseWithTimeout 带超时的优雅关闭
func (r *RabbitmqConsumer) CloseWithTimeout(timeout time.Duration) {
	if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		return
	}

	ctx := context.Background()
	tag := "RabbitmqClose"

	logger.Ix(ctx, tag, fmt.Sprintf("Closing RabbitMQ consumer (timeout: %v)", timeout))
	close(r.quit)

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Ix(ctx, tag, "RabbitMQ consumer closed gracefully")
	case <-time.After(timeout):
		logger.Wx(ctx, tag, fmt.Sprintf("RabbitMQ consumer close timeout after %v", timeout))
	}
}

// Close 保持向后兼容
func (r *RabbitmqConsumer) Close() {
	r.CloseWithTimeout(10 * time.Second)
}

// dial 建立独立连接、channel 并注册消费者，每个 goroutine 独享，不跨 goroutine 共享
func (r *RabbitmqConsumer) dial(config *RabbitmqConfig) (*amqp.Connection, *amqp.Channel, <-chan amqp.Delivery, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("open channel: %w", err)
	}

	if err = channel.Qos(config.PrefetchCount, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return nil, nil, nil, fmt.Errorf("set qos: %w", err)
	}

	msgs, err := channel.Consume(
		config.ConsumerQueue,
		"",
		false, false, false, false, nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, nil, nil, fmt.Errorf("consume: %w", err)
	}

	return conn, channel, msgs, nil
}

// consume 外层自愈循环：连接断开后按指数退避重连，panic 后同样重启
func (r *RabbitmqConsumer) consume(config *RabbitmqConfig, idx int) {
	defer r.wg.Done()
	atomic.AddInt32(&r.activeGoroutines, 1)
	defer atomic.AddInt32(&r.activeGoroutines, -1)

	ctx := context.Background()
	tag := fmt.Sprintf("RabbitmqConsumer-%d", idx)
	bo := newBackoff(time.Second, 30*time.Second)

	for {
		select {
		case <-r.quit:
			return
		default:
		}

		if exitClean := r.doConsume(ctx, tag, config, bo); exitClean {
			return
		}

		atomic.AddInt64(&r.restartCount, 1)
		wait := bo.Next()
		logger.Wx(ctx, tag, fmt.Sprintf("reconnecting in %v", wait))
		r.alertMgr.Emit(AlertEvent{
			Component: ComponentRabbitmq,
			Name:      config.ConsumerQueue,
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

// doConsume 单次连接生命周期：dial → consumeLoop → 返回
// 返回 true 表示收到 quit 信号，正常退出；false 表示连接断开或 panic，外层需重连
func (r *RabbitmqConsumer) doConsume(ctx context.Context, tag string, config *RabbitmqConfig, bo *backoff) (exitClean bool) {
	defer func() {
		if rec := recover(); rec != nil {
			atomic.AddInt64(&r.panicCount, 1)
			var msg string
			if err, ok := rec.(error); ok {
				msg = err.Error()
				logger.Ex(ctx, "RabbitmqConsumerPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			} else {
				msg = fmt.Sprintf("%v", rec)
				logger.Ex(ctx, "RabbitmqConsumerPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			}
			r.alertMgr.Emit(AlertEvent{
				Component: ComponentRabbitmq,
				Name:      config.ConsumerQueue,
				Type:      AlertPanic,
				Message:   msg,
			})
			exitClean = false
		}
	}()

	conn, channel, msgs, err := r.dial(config)
	if err != nil {
		logger.Ex(ctx, tag, fmt.Sprintf("dial error: %v", err))
		r.alertMgr.Emit(AlertEvent{
			Component: ComponentRabbitmq,
			Name:      config.ConsumerQueue,
			Type:      AlertConnLost,
			Message:   "dial rabbitmq failed",
			Err:       err,
		})
		return false
	}

	// 连接成功，重置退避
	bo.Reset()
	logger.Ix(ctx, tag, fmt.Sprintf("connected, queue: %s", config.ConsumerQueue))
	r.alertMgr.Emit(AlertEvent{
		Component: ComponentRabbitmq,
		Name:      config.ConsumerQueue,
		Type:      AlertConnRestored,
		Message:   "connected to rabbitmq",
	})

	return r.consumeLoop(ctx, tag, config, conn, channel, msgs)
}

// consumeLoop 消息消费主循环，conn 断开或收到 quit 时返回，defer 保证连接清理
func (r *RabbitmqConsumer) consumeLoop(
	ctx context.Context,
	tag string,
	config *RabbitmqConfig,
	conn *amqp.Connection,
	channel *amqp.Channel,
	msgs <-chan amqp.Delivery,
) (exitClean bool) {
	defer conn.Close()
	defer channel.Close()

	for {
		select {
		case <-r.quit:
			logger.Wx(ctx, tag, "quit signal received")
			return true
		case event, ok := <-msgs:
			if !ok {
				logger.Wx(ctx, tag, "message channel closed")
				r.alertMgr.Emit(AlertEvent{
					Component: ComponentRabbitmq,
					Name:      config.ConsumerQueue,
					Type:      AlertConnLost,
					Message:   "rabbitmq message channel closed unexpectedly",
				})
				return false
			}
			r.handleMessage(ctx, tag, config, channel, event)
		}
	}
}

// handleMessage 处理单条消息，含重试和失败队列逻辑
func (r *RabbitmqConsumer) handleMessage(
	ctx context.Context,
	tag string,
	config *RabbitmqConfig,
	channel *amqp.Channel,
	event amqp.Delivery,
) {
	for retryCount := 0; ; retryCount++ {
		// H-4: 每次重试前检查关闭信号，避免业务重试阻塞优雅关闭
		// 未 Ack 的消息在 channel 关闭后由 AMQP broker 自动重新入队
		select {
		case <-r.quit:
			return
		default:
		}

		if r.callback(config.TplName, event.RoutingKey, event.Body) {
			logger.Dx(ctx, tag, fmt.Sprintf("processed: KEY:%s", event.RoutingKey))
			if err := r.ackWithRetry(ctx, &event, 3); err != nil {
				logger.Ex(ctx, tag, fmt.Sprintf("ACK failed after retries: KEY:%s", event.RoutingKey))
			}
			return
		}

		logger.Ex(ctx, tag, fmt.Sprintf("processing failed (attempt %d): KEY:%s", retryCount+1, event.RoutingKey))

		if retryCount < config.FailCount {
			// M-2: 避免 CPU 空转，加入退避等待；同时响应关闭信号（未 Ack 的消息由 broker 重新入队）
			select {
			case <-r.quit:
				return
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}

		// 超出重试次数，进入失败处理
		if config.FailExchange != "" {
			published := false
			for i := 0; i < 3; i++ {
				err := channel.Publish(
					config.FailExchange, event.RoutingKey, false, false,
					amqp.Publishing{ContentType: "text/plain", Body: event.Body},
				)
				if err != nil {
					logger.Ex(ctx, tag, fmt.Sprintf("publish to fail exchange (attempt %d): %v", i+1, err))
					select {
					case <-r.quit:
						return
					case <-time.After(100 * time.Millisecond):
					}
					continue
				}
				published = true
				break
			}

			if published {
				logger.Ix(ctx, tag, fmt.Sprintf("sent to fail exchange: %s", config.FailExchange))
				if err := r.ackWithRetry(ctx, &event, 3); err != nil {
					logger.Ex(ctx, tag, "ACK failed after sending to fail exchange")
				}
			} else {
				logger.Ex(ctx, tag, "publish to fail exchange failed after 3 attempts, rejecting")
				r.alertMgr.Emit(AlertEvent{
					Component: ComponentRabbitmq,
					Name:      config.ConsumerQueue,
					Type:      AlertMsgDiscard,
					Message:   fmt.Sprintf("failed to publish to fail exchange %s after 3 retries, key=%s", config.FailExchange, event.RoutingKey),
				})
				event.Reject(config.IsReject)
			}
		} else {
			logger.Wx(ctx, tag, fmt.Sprintf("no fail exchange configured, rejecting: KEY:%s", event.RoutingKey))
			r.alertMgr.Emit(AlertEvent{
				Component: ComponentRabbitmq,
				Name:      config.ConsumerQueue,
				Type:      AlertMsgDiscard,
				Message:   fmt.Sprintf("no fail exchange configured, discarding after %d retries, key=%s", config.FailCount, event.RoutingKey),
			})
			event.Reject(config.IsReject)
		}

		return
	}
}

func (r *RabbitmqConsumer) ackWithRetry(ctx context.Context, event *amqp.Delivery, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := event.Ack(false); err == nil {
			return nil
		} else {
			logger.Ex(ctx, "ackWithRetry", fmt.Sprintf("ACK failed (%d/%d): %v", i+1, maxRetries, err))
		}
		if i < maxRetries-1 {
			select {
			case <-r.quit:
				return fmt.Errorf("ack interrupted by shutdown")
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	return fmt.Errorf("ack failed after %d retries", maxRetries)
}
