package consumer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Each failure producer belongs to a connection identity, never to the first
// subscription in the process. Credentials are only map keys, never error text.
type kafkaConnection struct {
	brokers        string
	sasl           bool
	user, password string
}

func kafkaResourceKey(cfg *KafkaConfig) kafkaConnection {
	brokers := append([]string(nil), cfg.Host...)
	sort.Strings(brokers)
	key := kafkaConnection{brokers: strings.Join(brokers, "\x00")}
	if cfg.SASL != nil && cfg.SASL.Enabled {
		key.sasl = true
		key.user = cfg.SASL.User
		key.password = cfg.SASL.Password
	}
	return key
}

type kafkaFailureProducer interface {
	Produce(*kafka.Message, chan kafka.Event) error
	Flush(int) int
	Close()
}

func createFailureProducers(configs []*KafkaConfig, create func(*KafkaConfig) (kafkaFailureProducer, error)) (map[kafkaConnection]kafkaFailureProducer, error) {
	producers := map[kafkaConnection]kafkaFailureProducer{}
	for _, cfg := range configs {
		if cfg.FailTopic == "" {
			continue
		}
		key := kafkaResourceKey(cfg)
		if _, ok := producers[key]; ok {
			continue
		}
		p, err := create(cfg)
		if err != nil {
			for _, previous := range producers {
				previous.Close()
			}
			return nil, fmt.Errorf("initialize Kafka failure producer: %w", err)
		}
		producers[key] = p
	}
	return producers, nil
}
func kafkaProducerOptions(cfg *KafkaConfig) *kafka.ConfigMap {
	options := &kafka.ConfigMap{
		"bootstrap.servers": strings.Join(cfg.Host, ","), "message.max.bytes": 1000000,
		"linger.ms": 500, "sticky.partitioning.linger.ms": 1000, "retries": INT32_MAX, "retry.backoff.ms": 1000, "reconnect.backoff.max.ms": 3000, "acks": "all", "enable.idempotence": true,
		"security.protocol": "plaintext",
	}
	if cfg.SASL != nil && cfg.SASL.Enabled {
		(*options)["security.protocol"] = "sasl_plaintext"
		(*options)["sasl.mechanism"] = "PLAIN"
		(*options)["sasl.username"] = cfg.SASL.User
		(*options)["sasl.password"] = cfg.SASL.Password
	}
	return options
}

// librdkafka counts queued application events in Producer.Len/Flush, including
// connection errors. Drain them continuously so a recovered outage does not
// leave Flush reporting undelivered messages forever. Per-message delivery
// reports still go to the private channel supplied to Produce.
type nativeFailureProducer struct {
	producer   *kafka.Producer
	eventsDone chan struct{}
}

func ownFailureProducer(producer *kafka.Producer) kafkaFailureProducer {
	owner := &nativeFailureProducer{producer: producer, eventsDone: make(chan struct{})}
	go func() {
		defer close(owner.eventsDone)
		for range producer.Events() {
		}
	}()
	return owner
}
func (p *nativeFailureProducer) Produce(message *kafka.Message, delivery chan kafka.Event) error {
	return p.producer.Produce(message, delivery)
}
func (p *nativeFailureProducer) Flush(timeout int) int { return p.producer.Flush(timeout) }
func (p *nativeFailureProducer) Close()                { p.producer.Close(); <-p.eventsDone }
