package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wanglelecc/gokitbox/logger"
)

type RabbitmqConsumer struct {
	runtime        *consumerRuntime
	sessionFactory func(context.Context, *RabbitmqConfig) (rabbitSession, error)
	stopOnce       sync.Once
	closeDone      chan struct{}
	closeErr       error

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
	return NewRabbitmqConsumerWithContext(configs, adaptCallback(callback), RuntimeOptions{}, alertMgr)
}
func NewRabbitmqConsumerWithContext(configs []*RabbitmqConfig, callback ContextConsumeCallback, options RuntimeOptions, alertMgr *AlertManager) (*RabbitmqConsumer, error) {
	cfg := (&Configure{Enabled: &ConsumeEnabled{Rabbitmq: true}, Rabbitmq: configs, Runtime: options}).Snapshot()
	if callback == nil {
		return nil, errors.New("worker: callback is required")
	}
	if validationErr := cfg.Validate(); validationErr != nil {
		return nil, validationErr
	}
	configs = cfg.Rabbitmq

	if len(configs) == 0 {
		return nil, errors.New("rabbitmq config not found")
	}

	consumer := &RabbitmqConsumer{
		runtime:  newRuntime(options, callback),
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := r.CloseContext(ctx); err != nil {
		logger.Wx(context.Background(), "RabbitmqConsumer.Close", "shutdown incomplete", "error", err)
	}
}

// CloseContext reports the actual drain outcome. A retry waits for the same
// shutdown instead of returning success while the first caller is still draining.
func (r *RabbitmqConsumer) CloseContext(ctx context.Context) error {
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
func (r *RabbitmqConsumer) Close() {
	r.CloseWithTimeout(10 * time.Second)
}

// Each goroutine owns a complete AMQP session. A failed or uncertain publish
// closes that session with the source delivery unacknowledged for redelivery.
type rabbitSession interface {
	deliveries() <-chan amqp.Delivery
	transferFailure(context.Context, string, amqp.Delivery) error
	settle(context.Context, amqp.Delivery, bool, bool) error
	Close() error
}

func (r *RabbitmqConsumer) consume(config *RabbitmqConfig, idx int) {
	defer r.wg.Done()
	atomic.AddInt32(&r.activeGoroutines, 1)
	defer atomic.AddInt32(&r.activeGoroutines, -1)
	ctx := r.runtime.context()
	bo := newBackoff(time.Second, 30*time.Second)
	for !stopRequested(ctx, r.quit) {
		if r.doConsume(ctx, fmt.Sprintf("rabbitmq-%d", idx), config, bo) {
			return
		}
		atomic.AddInt64(&r.restartCount, 1)
		delay := bo.Next()
		r.alertMgr.Emit(AlertEvent{Component: ComponentRabbitmq, Name: config.ConsumerQueue, Type: AlertRestart, Message: fmt.Sprintf("restarting in %v", delay)})
		if !waitRetry(ctx, r.quit, delay) {
			return
		}
	}
}
func (r *RabbitmqConsumer) doConsume(ctx context.Context, tag string, config *RabbitmqConfig, bo *backoff) (clean bool) {
	defer func() {
		if p := recover(); p != nil {
			atomic.AddInt64(&r.panicCount, 1)
			r.alertMgr.Emit(AlertEvent{Component: ComponentRabbitmq, Name: config.ConsumerQueue, Type: AlertPanic, Message: fmt.Sprintf("consumer panic (%T)", p)})
			logger.Ex(ctx, tag, "consumer panic", "type", fmt.Sprintf("%T", p))
			clean = false
		}
	}()
	var session rabbitSession
	var err error
	if r.sessionFactory != nil {
		session, err = r.sessionFactory(ctx, config)
	} else {
		session, err = openRabbitSession(ctx, config, r.runtime.opts())
	}
	if err != nil {
		logger.Wx(ctx, tag, "connect failed", "error", err)
		r.alertMgr.Emit(AlertEvent{Component: ComponentRabbitmq, Name: config.ConsumerQueue, Type: AlertConnLost, Message: "connect failed", Err: err})
		return stopRequested(ctx, r.quit)
	}
	r.alertMgr.Emit(AlertEvent{Component: ComponentRabbitmq, Name: config.ConsumerQueue, Type: AlertConnRestored, Message: "AMQP connection and subscription established"})
	defer func() {
		if err := session.Close(); err != nil {
			r.runtime.recordError(err)
			logger.Wx(ctx, tag, "close failed", "error", err)
		}
	}()
	for {
		if stopRequested(ctx, r.quit) {
			return true
		}
		select {
		case <-r.quit:
			return true
		case <-ctx.Done():
			return true
		case event, ok := <-session.deliveries():
			if !ok {
				return stopRequested(ctx, r.quit)
			}
			if err := r.handleDelivery(ctx, config, session, event); err != nil {
				logger.Wx(ctx, tag, "delivery unresolved; closing session", "error", err)
				return stopRequested(ctx, r.quit)
			}
			bo.Reset()
		}
	}
}
func (r *RabbitmqConsumer) handleDelivery(ctx context.Context, config *RabbitmqConfig, session rabbitSession, event amqp.Delivery) error {
	options := r.runtime.opts()
	processed := false
	for attempt := 0; attempt <= config.FailCount; attempt++ {
		if stopRequested(ctx, r.quit) {
			return context.Canceled
		}
		if r.runtime.call(ctx, r.callback, config.TplName, event.RoutingKey, event.Body) {
			processed = true
			break
		}
		if attempt < config.FailCount && !waitRetry(ctx, r.quit, options.RetryDelay) {
			return context.Canceled
		}
	}
	if !processed {
		if stopRequested(ctx, r.quit) {
			return context.Canceled
		}
		if config.FailExchange != "" {
			deliveryCtx, cancel := boundedContext(ctx, options.DeliveryTimeout)
			err := session.transferFailure(deliveryCtx, config.FailExchange, event)
			cancel()
			if err != nil {
				return fmt.Errorf("failure exchange not confirmed/routable: %w", err)
			}
		} else {
			r.alertMgr.Emit(AlertEvent{Component: ComponentRabbitmq, Name: config.ConsumerQueue, Type: AlertMsgDiscard, Message: "no failure exchange; applying configured reject/requeue policy"})
			if config.IsReject && !waitRetry(ctx, r.quit, options.RetryDelay) {
				return context.Canceled
			}
			operation, cancel := boundedContext(ctx, options.OperationTimeout)
			defer cancel()
			return session.settle(operation, event, true, config.IsReject)
		}
	}
	operation, cancel := boundedContext(ctx, options.OperationTimeout)
	defer cancel()
	// Acknowledgement writes are not broker confirms. An uncertain Ack is handled
	// by closing the session, never by blindly retrying the same delivery tag.
	return session.settle(operation, event, false, false)
}
