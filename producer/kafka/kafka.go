package kafka

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/producer/common"
	"github.com/wanglelecc/gokitbox/producer/failmode"
	"github.com/wanglelecc/gokitbox/producer/meta"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/spf13/cast"
)

const (
	INT32_MAX = 2147483647 - 1000
)

type CKafka struct {
	input    chan []byte
	fallback chan<- []byte

	successCount int64
	errorCount   int64

	cfg      *kafka.ConfigMap
	producer *kafka.Producer
	failMode failmode.FailMode

	exit chan struct{}
}

func CInit(quit chan struct{}, fallBack chan<- []byte) common.MQ {
	cfg := &kafka.ConfigMap{
		"api.version.request":           "true",
		"message.max.bytes":             1000000,
		"linger.ms":                     500,
		"sticky.partitioning.linger.ms": 1000,

		// 请求发生错误时重试次数，建议将该值设置为大于0，失败重试最大程度保证消息不丢失
		"retries": INT32_MAX,

		// 发送请求失败时到下一次重试请求之间的时间
		"retry.backoff.ms": 1000,

		// 设置客户端内部重试间隔。
		"reconnect.backoff.max.ms": 3000,

		// 用户不显示配置时，默认值为1。用户根据自己的业务情况进行设置
		"acks": "1"}

	// 设置接入点，请通过控制台获取对应Topic的接入点。
	cfg.SetKey("bootstrap.servers", strings.Join(config.GetConfArr("KafkaProxy", "brokers"), ","))

	if config.GetConfDefault("KafkaProxy", "sasl", "false") != "true" {
		cfg.SetKey("security.protocol", "plaintext")
	} else {
		cfg.SetKey("sasl.mechanism", "PLAIN")
		cfg.SetKey("security.protocol", "sasl_plaintext")
		cfg.SetKey("sasl.username", config.GetConf("KafkaProxy", "user"))
		cfg.SetKey("sasl.password", config.GetConf("KafkaProxy", "password"))
		// cfg.SetKey("sasl.mechanism", config.GetConf("KafkaProxy", "mechanism"))
	}

	k := new(CKafka)
	k.cfg = cfg
	bufferLimit := cast.ToInt(config.GetConfDefault("KafkaProxy", "bufferLimit", "1000"))
	k.input = make(chan []byte, bufferLimit)
	k.fallback = fallBack
	k.exit = make(chan struct{}, 1)
	k.successCount = int64(0)
	k.errorCount = int64(0)

	k.initProducer()
	go k.event()
	go k.run(quit)

	return k
}

func (k *CKafka) Input() chan<- []byte {
	return k.input
}

func (k *CKafka) Close() {
	close(k.exit)
}

func (k *CKafka) SetFailMode() {
	var err error
	k.failMode, err = failmode.GetFailMode(meta.Kafka, config.GetConfDefault("KafkaProxy", "failMode", "discard"))
	if err != nil {
		logger.Wx(context.Background(), "SetFailMode", "Kafka SetFailMode error", "error", err.Error())
		// 默认使用 discard 模式
		k.failMode, _ = failmode.GetFailMode(meta.Kafka, meta.DISCARD)
	}
}

func (k *CKafka) initProducer() {
	producer, err := kafka.NewProducer(k.cfg)
	if err != nil {
		logger.Wx(context.Background(), "initProducer", "ProducerInit", "error", err.Error())
		panic(fmt.Sprintf("Failed to create Kafka producer: %v", err))
	}

	k.producer = producer
}

func (k *CKafka) event() {
	defer meta.Recovery()

	ctx := context.Background()
	tag := "kafka.success"

	for e := range k.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				logger.Wx(ctx, tag, fmt.Sprintf("delivery failed: %v", ev.TopicPartition))

				atomic.AddInt64(&k.errorCount, int64(1))
				if ev.TopicPartition.Metadata == nil {
					logger.Wx(ctx, tag, "TopicPartition.Metadata is nil")
					continue
				}

				items := bytes.SplitN([]byte(*ev.TopicPartition.Metadata), []byte(" "), 5)
				if len(items) < 5 {
					logger.Wx(ctx, tag, fmt.Sprintf("invalid metadata format: %s", *ev.TopicPartition.Metadata))
					continue
				}

				timestamp := string(items[0])
				loggerId := ev.Key
				data := ev.Value
				logFlag := cast.ToInt64(timestamp) % 10
				diff := time.Now().Unix() - cast.ToInt64(timestamp)/10

				// 首次错误或距离上次日志超过59秒时记录日志
				if logFlag == 0 || diff >= 59 {
					timestamp = cast.ToString(time.Now().Unix()*10 + 11) // 多加1秒，防止同一秒可能出现打多次
					logger.Ex(ctx, tag+".ProducerOutPut", fmt.Sprintf("Kafka Send Message Error:'%v',Topic:'%s',Data:'%v'", ev.TopicPartition.Error, *ev.TopicPartition.Topic, string(data)))
				}

				if k.failMode != nil {
					k.failMode.Do(k.fallback, []byte(timestamp+" "+meta.Kafka+" "+*ev.TopicPartition.Topic+" "+string(loggerId)+" "+string(data)), data, []interface{}{meta.Kafka, ev.TopicPartition.Error})
				}

			} else {
				logger.Dx(ctx, tag+".ProducerOutPut", fmt.Sprintf("kafka send message success. to %v", ev.TopicPartition))
			}
		}
	}
}

func (k *CKafka) run(quite chan struct{}) {
	defer meta.Recovery()

	ctx := context.Background()
	tag := "kafka.run"

	t := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-t.C:
			succ := atomic.SwapInt64(&k.successCount, 0)
			fail := atomic.SwapInt64(&k.errorCount, 0)
			logger.Ix(ctx, tag+".KafkaProducerOutPut", fmt.Sprintf("Stat succ:%d,fail:%d", succ, fail))
		case msg := <-k.input:
			items := bytes.SplitN(msg, []byte(" "), 5)
			timestamp := string(items[0])
			topic := string(items[2])
			loggerId := items[3]
			data := items[4]
			metadata := string(msg)

			err := k.producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &topic,
					Partition: kafka.PartitionAny,
					Metadata:  &metadata,
				},
				Value: data,
				Key:   loggerId,
			}, nil)

			if err != nil {
				// 使用 select 避免阻塞
				select {
				case k.fallback <- []byte(timestamp + " " + meta.Kafka + " " + topic + " " + string(loggerId) + " " + string(data)):
				default:
					logger.Wx(ctx, tag, fmt.Sprintf("fallback channel is full, message dropped: topic=%s", topic))
				}
			}

		case <-k.exit:
			t.Stop()
			logger.Ix(ctx, tag+".ProducerOutPut", "KafkaQuit begin")
			k.producer.Flush(15 * 1000)
			k.producer.Close()
			logger.Ix(ctx, tag+".ProducerOutPut", "KafkaQuit end")
			close(quite)
			return
		}
	}
}
