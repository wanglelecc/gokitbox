package producer

import (
	"errors"
	"sync"

	"github.com/wanglelecc/gokitbox/producer/kafka"
)

type Options struct {
	KafkaEnabled bool
	Kafka        kafka.Options
	BufferLimit  int
	GrayMode     bool
}

func (o Options) Validate() error {
	if o.BufferLimit < 0 || o.BufferLimit > 1000000 {
		return errors.New("producer: buffer limit out of range")
	}
	if o.KafkaEnabled {
		return o.Kafka.Validate()
	}
	return nil
}

var optionsMu sync.Mutex
var configuredOptions *Options
var started bool

func Configure(o Options) error {
	if err := o.Validate(); err != nil {
		return err
	}
	o.Kafka.Brokers = append([]string(nil), o.Kafka.Brokers...)
	optionsMu.Lock()
	defer optionsMu.Unlock()
	if configuredOptions != nil || started {
		return errors.New("producer: already configured or initialized")
	}
	configuredOptions = &o
	return nil
}
func currentOptions() Options {
	optionsMu.Lock()
	defer optionsMu.Unlock()
	if configuredOptions != nil {
		o := *configuredOptions
		o.Kafka.Brokers = append([]string(nil), o.Kafka.Brokers...)
		return o
	}
	return legacyOptions()
}
