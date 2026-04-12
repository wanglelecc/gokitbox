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

	"github.com/wanglelecc/gokitbox/logger"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/spf13/cast"
)

const (
	INT32_MAX = 2147483647 - 1000
)

type KafkaConsumer struct {
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

	// produce 在 NewKafkaConsumer 启动时初始化（fail-fast），避免消费中途初始化失败导致死循环
	produce *kafka.Producer

	alertMgr *AlertManager
}

func NewKafkaConsumer(configs []*KafkaConfig, callback ConsumeCallback, alertMgr *AlertManager) (consumer *KafkaConsumer, err error) {
	if len(configs) == 0 {
		err = errors.New("kafka config not found")
		return
	}
	consumer = new(KafkaConsumer)
	consumer.callback = callback
	consumer.alertMgr = alertMgr
	consumer.quit = make(chan struct{})
	consumer.wg = new(sync.WaitGroup)

	consumer.wg.Add(1)
	go consumer.count()

	// 如有任何配置了 FailTopic，启动时初始化 Producer（fail-fast）
	// 所有 config 共用同一 Producer，假设同一 Kafka 集群
	for _, cfg := range configs {
		if cfg.FailTopic != "" {
			consumer.produce, err = newKafkaProducer(cfg)
			if err != nil {
				return nil, fmt.Errorf("init kafka producer: %w", err)
			}
			break
		}
	}

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
	if !atomic.CompareAndSwapInt32(&k.closed, 0, 1) {
		return
	}

	ctx := context.Background()
	tag := "KafkaClose"

	logger.Ix(ctx, tag, fmt.Sprintf("Closing Kafka consumer (timeout: %v)", timeout))
	close(k.quit)

	done := make(chan struct{})
	go func() {
		k.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Ix(ctx, tag, "Kafka consumer closed gracefully")
	case <-time.After(timeout):
		logger.Wx(ctx, tag, fmt.Sprintf(
			"Kafka consumer close timeout after %v, some goroutines may still be running", timeout))
	}

	// Flush 并关闭 Producer：确保 FailTopic 缓冲消息全部投递后再退出
	if k.produce != nil {
		remaining := k.produce.Flush(int(timeout.Milliseconds()))
		if remaining > 0 {
			logger.Wx(ctx, tag, fmt.Sprintf("producer flush timeout, %d messages unsent", remaining))
		}
		k.produce.Close()
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
		"api.version.request":   "true",
		"auto.offset.reset":     "earliest",
		"heartbeat.interval.ms": 3000,

		// 使用 Kafka 消费分组机制时，消费者超时时间。当 Broker 在该时间内没有收到消费者的心跳时，认为该消费者故障失败，Broker
		// 发起重新 Rebalance 过程。目前该值的配置必须在 Broker 配置group.min.session.timeout.ms=6000和group.max.session.timeout.ms=300000 之间
		"session.timeout.ms": 30000,

		"max.poll.interval.ms":      120000,
		"fetch.max.bytes":           1024000,
		"max.partition.fetch.bytes": 256000,

		// 禁用自动提交，改为手动提交以确保消息处理成功后才提交offset
		"enable.auto.commit": false,
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
		logger.Fx(ctx, "Consume", "NewConsumer error", "error", err)
		return nil, fmt.Errorf("new consumer: %w", err)
	}

	if err = consumer.SubscribeTopics([]string{cfg.Topic}, nil); err != nil {
		consumer.Close()
		logger.Fx(ctx, "Consume", "SubscribeTopics error", "error", err)
		return nil, fmt.Errorf("subscribe topics: %w", err)
	}

	return consumer, nil
}

// newKafkaProducer 创建 Kafka Producer，供 NewKafkaConsumer 在启动时调用（fail-fast）
func newKafkaProducer(cfg *KafkaConfig) (*kafka.Producer, error) {
	pCfg := &kafka.ConfigMap{
		"bootstrap.servers":             strings.Join(cfg.Host, ","),
		"api.version.request":           "true",
		"message.max.bytes":             1000000,
		"linger.ms":                     500,
		"sticky.partitioning.linger.ms": 1000,

		// 请求发生错误时重试次数，建议将该值设置为大于0，失败重试最大程度保证消息不丢失
		"retries": INT32_MAX,

		// 发送请求失败时到下一次重试请求之间的时间
		"retry.backoff.ms": 1000,

		// 设置客户端内部重试间隔
		"reconnect.backoff.max.ms": 3000,

		// 用户不显式配置时，默认值为1
		"acks": "1",
	}

	p, err := kafka.NewProducer(pCfg)
	if err != nil {
		return nil, fmt.Errorf("new kafka producer: %w", err)
	}
	return p, nil
}

// consume 外层自愈循环：doConsume 异常退出后按指数退避重启
func (k *KafkaConsumer) consume(config *KafkaConfig) {
	defer k.wg.Done()
	atomic.AddInt32(&k.activeGoroutines, 1)
	defer atomic.AddInt32(&k.activeGoroutines, -1)

	ctx := context.Background()
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

	consumer, err := k.initKafka(config)
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
	defer consumer.Close()

	// 连接成功，重置退避
	bo.Reset()
	logger.Ix(ctx, "KafkaConsume", fmt.Sprintf("Start consume from broker %v", config.Host))
	k.alertMgr.Emit(AlertEvent{
		Component: ComponentKafka,
		Name:      config.Topic,
		Type:      AlertConnRestored,
		Message:   fmt.Sprintf("connected to broker %v", config.Host),
	})

	for {
		// 检查退出信号
		select {
		case <-k.quit:
			logger.Wx(ctx, "KafkaConsume", "IKAFKA_RECV_QUIT")
			return true
		default:
		}

		// 使用1秒超时，避免无限期阻塞，确保能及时响应quit信号
		event, err := consumer.ReadMessage(time.Second)
		if err != nil {
			if kafkaErr, ok := err.(kafka.Error); ok {
				// 超时是正常情况，继续下一轮循环（会检查quit信号）
				if kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				logger.Ex(ctx, "KafkaConsume", "ReadMessage error", "error", err.Error(), "code", kafkaErr.Code())
			} else {
				logger.Ex(ctx, "KafkaConsume", "ReadMessage error", "error", err.Error())
			}
			select {
			case <-k.quit:
				return true
			case <-time.After(time.Millisecond * 100):
			}
			continue
		}

		// 处理消息
		needCommit := false
		count := 0
		for {
			ret := k.callback(config.TplName, config.Topic, event.Value)
			if ret {
				atomic.AddInt64(&k.successCount, int64(1))
				logger.Dx(ctx, "KafkaConsume", fmt.Sprintf("return_true:KEY:%s,OFFSET:%d,PARTITION:%d", string(event.Key), event.TopicPartition.Offset, event.TopicPartition.Partition))
				needCommit = true
				break
			}

			logger.Ex(ctx, "KafkaConsume", fmt.Sprintf("return_false:KEY:%s,VAL:%s,OFFSET:%d,PARTITION:%d", string(event.Key), string(event.Value), event.TopicPartition.Offset, event.TopicPartition.Partition))

			if count >= config.FailCount {
				if config.FailTopic != "" {
					sent := false
					for i := 0; i < 3; i++ {
						if err := k.sendByHashPartition(config.FailTopic, event.Value, event.Key); err != nil {
							logger.Ex(ctx, "KafkaConsume", fmt.Sprintf("SendToFailTopic error:%s %s %v", config.FailTopic, string(event.Value), err))
							select {
							case <-k.quit:
								return true
							case <-time.After(time.Millisecond * 100):
							}
							continue
						}
						sent = true
						break
					}
					if sent {
						logger.Dx(ctx, "KafkaConsume", fmt.Sprintf("Sent to Fail Topic:KEY:%s,VAL:%s,OFFSET:%d,PARTITION:%d", string(event.Key), string(event.Value), event.TopicPartition.Offset, event.TopicPartition.Partition))
						needCommit = true
					} else {
						logger.Ex(ctx, "KafkaConsume", fmt.Sprintf("Failed to send to Fail Topic after 3 retries:KEY:%s,VAL:%s", string(event.Key), string(event.Value)))
						k.alertMgr.Emit(AlertEvent{
							Component: ComponentKafka,
							Name:      config.Topic,
							Type:      AlertMsgDiscard,
							Message:   fmt.Sprintf("failed to send to fail topic %s after 3 retries, offset=%d", config.FailTopic, event.TopicPartition.Offset),
						})
						// 发送失败队列失败，不提交offset，下次重试
						needCommit = false
					}
				} else {
					logger.Ex(ctx, "KafkaConsume", fmt.Sprintf("Fail Topic is empty:KEY:%s,VAL:%s,OFFSET:%d,PARTITION:%d", string(event.Key), string(event.Value), event.TopicPartition.Offset, event.TopicPartition.Partition))
					k.alertMgr.Emit(AlertEvent{
						Component: ComponentKafka,
						Name:      config.Topic,
						Type:      AlertMsgDiscard,
						Message:   fmt.Sprintf("no fail topic configured, discarding after %d retries, offset=%d", config.FailCount, event.TopicPartition.Offset),
					})
					// 没有配置失败队列，提交offset避免重复消费
					needCommit = true
				}
				break
			}
			count++
			select {
			case <-k.quit:
				return true
			case <-time.After(time.Millisecond * 100):
			}
		}

		// 手动提交 offset，最多重试 3 次，防止网络抖动导致 FailTopic 重复写入
		if needCommit {
			committed := false
			for i := 0; i < 3; i++ {
				if _, err := consumer.CommitMessage(event); err != nil {
					logger.Ex(ctx, "KafkaConsume", fmt.Sprintf("Commit offset error (attempt %d/3)", i+1), "error", err.Error(), "offset", event.TopicPartition.Offset, "partition", event.TopicPartition.Partition)
					select {
					case <-k.quit:
						return true
					case <-time.After(time.Millisecond * 100):
					}
					continue
				}
				committed = true
				break
			}
			if !committed {
				atomic.AddInt64(&k.errorCount, int64(1))
				logger.Ex(ctx, "KafkaConsume", "Commit offset failed after 3 retries, message may be re-consumed", "offset", event.TopicPartition.Offset, "partition", event.TopicPartition.Partition)
			}
		}
	}
}

// sendByHashPartition 同步发送消息到指定 topic，等待 broker 确认后才返回
func (k *KafkaConsumer) sendByHashPartition(topic string, data []byte, key []byte) error {
	if k.produce == nil {
		return fmt.Errorf("producer not initialized (FailTopic requires a valid producer)")
	}

	deliveryChan := make(chan kafka.Event, 1)
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
		Key:            key,
	}
	if err := k.produce.Produce(msg, deliveryChan); err != nil {
		return fmt.Errorf("produce: %w", err)
	}

	e := <-deliveryChan
	m, ok := e.(*kafka.Message)
	if !ok {
		return fmt.Errorf("unexpected delivery event type: %T", e)
	}
	if m.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
	}

	logger.Dx(context.Background(), "SendByHashPartition", fmt.Sprintf("[KAFKA_OUT]topic:%s,data:%s", topic, string(data)))
	return nil
}
