package producer

import (
	"testing"

	"github.com/wanglelecc/gokitbox/producer/kafka"
)

func TestOptionsAreValidatedAndCopied(t *testing.T) {
	o := Options{KafkaEnabled: true, Kafka: kafka.DefaultOptions(), BufferLimit: 0}
	o.Kafka.Brokers = []string{"localhost:9092"}
	if err := Configure(o); err != nil {
		t.Fatal(err)
	}
	o.Kafka.Brokers[0] = "changed"
	if currentOptions().Kafka.Brokers[0] != "localhost:9092" {
		t.Fatal("broker list was not copied")
	}
	if err := Configure(o); err == nil {
		t.Fatal("reconfiguration accepted")
	}
}
