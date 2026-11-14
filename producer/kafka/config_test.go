package kafka

import (
	"testing"

	"github.com/wanglelecc/gokitbox/config"
)

func TestEffectiveProducerConfig(t *testing.T) {
	cfg, err := config.NewSnapshot(map[string]interface{}{"KafkaProxy": map[string]interface{}{"brokers": []string{"localhost:9092"}, "KafkaWaitAll": true, "KafkaCompression": "gzip", "KafkaPartitioner": "consistent_random", "KafkaProducerTimeout": "7s"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := config.Install(cfg); err != nil {
		t.Fatal(err)
	}
	effective, err := buildProducerConfig()
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]interface{}{"acks": "all", "compression.type": "gzip", "partitioner": "consistent_random", "message.timeout.ms": 7000, "bootstrap.servers": "localhost:9092"} {
		value, err := effective.Get(key, nil)
		if err != nil || value != want {
			t.Fatalf("%s: got %v want %v err=%v", key, value, want, err)
		}
	}
}
