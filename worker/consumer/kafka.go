package consumer

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/spf13/cast"
	"github.com/wanglelecc/gokitbox/logger"
)

const (
	INT32_MAX = 2147483647 - 1000
)

type KafkaConsumer struct {
	runtime        *consumerRuntime
	sessionFactory func(*KafkaConfig) (kafkaSession, error)
	stopOnce       sync.Once
	closeDone      chan struct{}
	closeErr       error

	quit         chan struct{}
	callback     ConsumeCallback
	successCount int64
	errorCount   int64
	wg           *sync.WaitGroup
	closed       int32

	// 健康指标
	activeGoroutines int32
	panicCount       int64
	restartCount     int64

	// Failure producers are initialized per connection before launching consumers.
	producers map[kafkaConnection]kafkaFailureProducer

	alertMgr *AlertManager
}

func NewKafkaConsumer(configs []*KafkaConfig, callback ConsumeCallback, alertMgr *AlertManager) (consumer *KafkaConsumer, err error) {
	return NewKafkaConsumerWithContext(configs, adaptCallback(callback), RuntimeOptions{}, alertMgr)
}
func NewKafkaConsumerWithContext(configs []*KafkaConfig, callback ContextConsumeCallback, options RuntimeOptions, alertMgr *AlertManager) (consumer *KafkaConsumer, err error) {
	cfg := (&Configure{Enabled: &ConsumeEnabled{Kafka: true}, Kafka: configs, Runtime: options}).Snapshot()
	if callback == nil {
		return nil, errors.New("worker: callback is required")
	}
	if validationErr := cfg.Validate(); validationErr != nil {
		return nil, validationErr
	}
	configs = cfg.Kafka

	if len(configs) == 0 {
		err = errors.New("kafka config not found")
		return
	}
	consumer = new(KafkaConsumer)
	consumer.runtime = newRuntime(options, callback)
	consumer.alertMgr = alertMgr
	consumer.quit = make(chan struct{})
	consumer.wg = new(sync.WaitGroup)

	// The manager validates runtime Options. Keep direct constructor compatibility
	// (including nil SASL for plaintext), but reject nil entries before allocation.
	for _, cfg := range configs {
		if cfg == nil {
			return nil, errors.New("kafka: nil consumer configuration")
		}
	}
	consumer.producers, err = createFailureProducers(configs, func(cfg *KafkaConfig) (kafkaFailureProducer, error) {
		return newKafkaProducerWithOptions(cfg, options.defaults())
	})
	if err != nil {
		return nil, err
	}
	consumer.wg.Add(1)
	go consumer.count()

	for _, config := range configs {
		for i := 0; i < config.ConsumerCount; i++ {
			consumer.wg.Add(1)
			go consumer.consume(config)
		}
	}

	return
}

// CloseWithTimeout 带超时的优雅关闭
func (k *KafkaConsumer) CloseWithTimeout(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := k.CloseContext(ctx); err != nil {
		logger.Wx(context.Background(), "KafkaConsumer.Close", "shutdown incomplete", "error", err)
	}
}

// CloseContext reports the actual drain outcome. A retry waits for the same
// shutdown instead of returning success while the first caller is still draining.
func (k *KafkaConsumer) CloseContext(ctx context.Context) error {
	k.stopOnce.Do(func() {
		k.closeDone = make(chan struct{})
		atomic.StoreInt32(&k.closed, 1)
		close(k.quit)
		go func() {
			defer close(k.closeDone)
			stopCancellation := k.runtime.cancelOnDeadline(ctx)
			defer stopCancellation()
			if k.runtime != nil {
				defer k.runtime.cancel()
			}
			k.wg.Wait()
			k.closeErr = errors.Join(k.closeErr, k.runtime.drainError())
			deadline := time.Now().Add(10 * time.Second)
			if configured, ok := ctx.Deadline(); ok && configured.Before(deadline) {
				deadline = configured
			}
			for _, producer := range k.producers {
				if remaining := producer.Flush(int(max(0, time.Until(deadline)).Milliseconds())); remaining > 0 {
					k.closeErr = errors.Join(k.closeErr, fmt.Errorf("kafka: %d messages not flushed", remaining))
				}
				producer.Close()
			}

		}()
	})
	select {
	case <-k.closeDone:
		return k.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close 保持向后兼容
func (k *KafkaConsumer) Close() {
	k.CloseWithTimeout(10 * time.Second)
}

func (k *KafkaConsumer) count() {
	defer k.wg.Done()

	ctx := context.Background()
	tag := "kafkaCount"

	t := time.NewTicker(time.Second * 300)
	defer t.Stop()

	for {
		select {
		case <-k.quit:
			logger.Ix(ctx, tag, "count goroutine exit")
			return
		case <-t.C:
			succ := atomic.SwapInt64(&k.successCount, 0)
			fail := atomic.SwapInt64(&k.errorCount, 0)
			logger.Ix(ctx, tag, fmt.Sprintf("Stat succ:%d,fail:%d", succ, fail))
		}
	}
}

func (k *KafkaConsumer) initKafka(cfg *KafkaConfig) (*kafka.Consumer, error) {
	ctx := context.Background()

	kafkaCfg := &kafka.ConfigMap{

		"auto.offset.reset":     "earliest",
		"heartbeat.interval.ms": int(k.runtime.opts().SessionTimeout / (3 * time.Millisecond)),

		// 使用 Kafka 消费分组机制时，消费者超时时间。当 Broker 在该时间内没有收到消费者的心跳时，认为该消费者故障失败，Broker
		// 发起重新 Rebalance 过程。目前该值的配置必须在 Broker 配置group.min.session.timeout.ms=6000和group.max.session.timeout.ms=300000 之间
		"session.timeout.ms": int(k.runtime.opts().SessionTimeout / time.Millisecond),

		"max.poll.interval.ms":               int(k.runtime.opts().MaxPollInterval / time.Millisecond),
		"socket.timeout.ms":                  int(k.runtime.opts().OperationTimeout / time.Millisecond),
		"socket.connection.setup.timeout.ms": int(k.runtime.opts().OperationTimeout / time.Millisecond),
		"api.version.request.timeout.ms":     int(k.runtime.opts().OperationTimeout / time.Millisecond),
		"fetch.wait.max.ms":                  min(500, max(0, int(k.runtime.opts().OperationTimeout/time.Millisecond)-1000)),
		"fetch.max.bytes":                    1024000,
		"max.partition.fetch.bytes":          256000,

		// 禁用自动提交，改为手动提交以确保消息处理成功后才提交offset
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,
	}

	kafkaCfg.SetKey("bootstrap.servers", strings.Join(cfg.Host, ","))
	kafkaCfg.SetKey("group.id", cfg.ConsumerGroup)

	if cfg.SASL != nil && cfg.SASL.Enabled {
		kafkaCfg.SetKey("sasl.mechanism", "PLAIN")
		kafkaCfg.SetKey("security.protocol", "sasl_plaintext")
		kafkaCfg.SetKey("sasl.username", cfg.SASL.User)
		kafkaCfg.SetKey("sasl.password", cfg.SASL.Password)
	} else {
		kafkaCfg.SetKey("security.protocol", "plaintext")
	}

	consumer, err := kafka.NewConsumer(kafkaCfg)
	if err != nil {
		logger.Ex(ctx, "Consume", "NewConsumer error", "error", err)
		return nil, fmt.Errorf("new consumer: %w", err)
	}

	if err = consumer.SubscribeTopics([]string{cfg.Topic}, nil); err != nil {
		consumer.Close()
		logger.Ex(ctx, "Consume", "SubscribeTopics error", "error", err)
		return nil, fmt.Errorf("subscribe topics: %w", err)
	}

	return consumer, nil
}

// newKafkaProducer 创建 Kafka Producer，供 NewKafkaConsumer 在启动时调用（fail-fast）
func newKafkaProducer(cfg *KafkaConfig) (kafkaFailureProducer, error) {
	return newKafkaProducerWithOptions(cfg, RuntimeOptions{}.defaults())
}
func newKafkaProducerWithOptions(cfg *KafkaConfig, options RuntimeOptions) (kafkaFailureProducer, error) {
	pCfg := kafkaProducerOptions(cfg)
	timeoutMs := int(options.DeliveryTimeout / time.Millisecond)
	(*pCfg)["message.timeout.ms"] = timeoutMs
	(*pCfg)["socket.timeout.ms"] = int(options.OperationTimeout / time.Millisecond)
	(*pCfg)["socket.connection.setup.timeout.ms"] = int(options.OperationTimeout / time.Millisecond)
	(*pCfg)["api.version.request.timeout.ms"] = min(300000, int(options.OperationTimeout/time.Millisecond))
	linger := min(5, timeoutMs/4)
	(*pCfg)["linger.ms"] = linger
	(*pCfg)["sticky.partitioning.linger.ms"] = 2 * linger

	p, err := kafka.NewProducer(pCfg)
	if err != nil {
		return nil, fmt.Errorf("new kafka producer: %w", err)
	}
	return ownFailureProducer(p), nil
}

// consume 外层自愈循环：doConsume 异常退出后按指数退避重启
func (k *KafkaConsumer) consume(config *KafkaConfig) {
	defer k.wg.Done()
	atomic.AddInt32(&k.activeGoroutines, 1)
	defer atomic.AddInt32(&k.activeGoroutines, -1)

	ctx := k.runtime.context()
	bo := newBackoff(time.Second, 30*time.Second)

	for {
		select {
		case <-k.quit:
			return
		default:
		}

		if exitClean := k.doConsume(ctx, config, bo); exitClean {
			return
		}

		atomic.AddInt64(&k.restartCount, 1)
		wait := bo.Next()
		logger.Wx(ctx, "KafkaConsume", fmt.Sprintf("goroutine restarting in %v", wait))
		k.alertMgr.Emit(AlertEvent{
			Component: ComponentKafka,
			Name:      config.Topic,
			Type:      AlertRestart,
			Message:   fmt.Sprintf("consumer goroutine restarting in %v", wait),
		})
		select {
		case <-k.quit:
			return
		case <-time.After(wait):
		}
	}
}

// doConsume 单次消费生命周期：初始化连接 → 消费循环 → 退出
// 返回 true 表示收到 quit 信号，正常退出；false 表示异常，外层需重启
func (k *KafkaConsumer) doConsume(ctx context.Context, config *KafkaConfig, bo *backoff) (exitClean bool) {
	defer func() {
		if r := recover(); r != nil {
			atomic.AddInt64(&k.panicCount, 1)
			var msg string
			if err, ok := r.(error); ok {
				msg = err.Error()
				logger.Ex(ctx, "IKafkaPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			} else {
				msg = fmt.Sprintf("%v", r)
				logger.Ex(ctx, "IKafkaPanicRecover", fmt.Sprintf("panic recovered: %v\nstack:%v", msg, cast.ToString(debug.Stack())))
			}
			k.alertMgr.Emit(AlertEvent{
				Component: ComponentKafka,
				Name:      config.Topic,
				Type:      AlertPanic,
				Message:   msg,
			})
			exitClean = false
		}
	}()

	var consumer kafkaSession
	var err error
	if k.sessionFactory != nil {
		consumer, err = k.sessionFactory(config)
	} else {
		consumer, err = k.initKafka(config)
	}
	if err != nil {
		logger.Ex(ctx, "KafkaConsume", fmt.Sprintf("init kafka error: %v", err))
		k.alertMgr.Emit(AlertEvent{
			Component: ComponentKafka,
			Name:      config.Topic,
			Type:      AlertConnLost,
			Message:   "init kafka consumer failed",
			Err:       err,
		})
		return false
	}
	owner := &kafkaSessionOwner{session: consumer}
	defer func() {
		if err := owner.close(); err != nil {
			k.runtime.recordError(err)
			logger.Wx(ctx, "KafkaConsume", "close failed", "error", err)
		}
	}()

	// 连接成功，重置退避
	logger.Ix(ctx, "KafkaConsume", fmt.Sprintf("Start consume from broker %v", config.Host))
	k.alertMgr.Emit(AlertEvent{
		Component: ComponentKafka,
		Name:      config.Topic,
		Type:      AlertConnRestored,
		Message:   fmt.Sprintf("connected to broker %v", config.Host),
	})

	return k.consumeSession(ctx, config, bo, owner)
}

// Processing is serial within a session. Any unresolved delivery or commit ends
// the session before another ReadMessage: later offsets cannot cross a hole.
func (k *KafkaConsumer) consumeSession(ctx context.Context, config *KafkaConfig, bo *backoff, owner *kafkaSessionOwner) bool {
	for {
		if stopRequested(ctx, k.quit) {
			return true
		}
		event, err := owner.session.ReadMessage(200 * time.Millisecond)
		if err != nil {
			var kafkaErr kafka.Error
			if errors.As(err, &kafkaErr) && kafkaErr.Code() == kafka.ErrTimedOut {
				continue
			}
			logger.Wx(ctx, "KafkaConsume", "read failed", "error", err)
			return stopRequested(ctx, k.quit)
		}
		if err = k.processMessage(ctx, config, owner, event); err != nil {
			logger.Wx(ctx, "KafkaConsume", "message unresolved; closing session without advancing", "error", err, "offset", event.TopicPartition.Offset, "partition", event.TopicPartition.Partition)
			return stopRequested(ctx, k.quit)
		}
		bo.Reset()
	}
}
func (k *KafkaConsumer) processMessage(ctx context.Context, config *KafkaConfig, owner *kafkaSessionOwner, event *kafka.Message) error {
	options := k.runtime.opts()
	processed := false
	for attempt := 0; attempt <= config.FailCount; attempt++ {
		if stopRequested(ctx, k.quit) {
			return context.Canceled
		}
		if k.runtime.call(ctx, k.callback, config.TplName, config.Topic, event.Value) {
			processed = true
			atomic.AddInt64(&k.successCount, 1)
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt < config.FailCount && !waitRetry(ctx, k.quit, options.RetryDelay) {
			return context.Canceled
		}
	}
	if !processed {
		atomic.AddInt64(&k.errorCount, 1)
		if config.FailTopic != "" {
			var last error
			for attempt := 0; attempt < 3; attempt++ {
				if stopRequested(ctx, k.quit) {
					return context.Canceled
				}
				last = k.sendByHashPartitionContext(ctx, config, config.FailTopic, event.Value, event.Key)
				if last == nil {
					break
				}
				if attempt < 2 && !waitRetry(ctx, k.quit, options.RetryDelay) {
					return context.Canceled
				}
			}
			if last != nil {
				return fmt.Errorf("failure topic not acknowledged: %w", last)
			}
		} else {
			// Preserve the documented explicit-discard policy when no fail topic exists.
			k.alertMgr.Emit(AlertEvent{Component: ComponentKafka, Name: config.Topic, Type: AlertMsgDiscard, Message: "no failure topic; retry limit reached"})
		}
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		commitCtx, cancel := boundedContext(ctx, options.OperationTimeout)
		last = owner.commit(commitCtx, event)
		cancel()
		if last == nil {
			return nil
		}
		// A timed-out commit has an uncertain outcome. Never start another operation
		// against this session or read a later message while it remains outstanding.
		if errors.Is(last, context.DeadlineExceeded) || errors.Is(last, context.Canceled) {
			return last
		}
		if attempt < 2 && !waitRetry(ctx, k.quit, options.RetryDelay) {
			return context.Canceled
		}
	}
	return fmt.Errorf("commit failed; refusing to advance: %w", last)
}

type kafkaSession interface {
	ReadMessage(time.Duration) (*kafka.Message, error)
	CommitMessage(*kafka.Message) ([]kafka.TopicPartition, error)
	Close() error
}
type kafkaSessionOwner struct {
	session    kafkaSession
	operations sync.WaitGroup
}

func (o *kafkaSessionOwner) commit(ctx context.Context, event *kafka.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	result := make(chan error, 1)
	o.operations.Add(1)
	go func() { defer o.operations.Done(); _, err := o.session.CommitMessage(event); result <- err }()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (o *kafkaSessionOwner) close() error { o.operations.Wait(); return o.session.Close() }

// sendByHashPartition 同步发送消息到指定 topic，等待 broker 确认后才返回
func (k *KafkaConsumer) sendByHashPartition(cfg *KafkaConfig, topic string, data []byte, key []byte) error {
	return k.sendByHashPartitionContext(k.runtime.context(), cfg, topic, data, key)
}
func (k *KafkaConsumer) sendByHashPartitionContext(parent context.Context, cfg *KafkaConfig, topic string, data, key []byte) error {
	ctx, cancel := boundedContext(parent, k.runtime.opts().DeliveryTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	producer := k.producers[kafkaResourceKey(cfg)]
	if producer == nil {
		return fmt.Errorf("producer not initialized (FailTopic requires a valid producer)")
	}

	deliveryChan := make(chan kafka.Event, 1)
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
		Key:            key,
	}
	if err := producer.Produce(msg, deliveryChan); err != nil {
		return fmt.Errorf("produce: %w", err)
	}

	var e kafka.Event
	select {
	case e = <-deliveryChan:
	case <-ctx.Done():
		return ctx.Err()
	}
	m, ok := e.(*kafka.Message)
	if !ok {
		return fmt.Errorf("unexpected delivery event type: %T", e)
	}
	if m.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
	}

	logger.Dx(ctx, "SendByHashPartition", "failure message acknowledged", "topic", topic, "bytes", len(data))
	return nil
}
