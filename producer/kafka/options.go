package kafka

import (
	"errors"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/wanglelecc/gokitbox/configsource"
)

type Options struct {
	Brokers                  []string
	Sasl                     bool
	Username, Password       string
	WaitAll                  bool
	Compression, Partitioner string
	Timeout                  time.Duration
	BufferLimit              int
	FailMode                 string
	// Zero uses the bounded defaults: 3 retries, 100ms backoff, 2m total budget.
	MaxRetries      int
	RetryBackoff    time.Duration
	RetryBudget     time.Duration
	MaxMessageBytes int
}

func DefaultOptions() Options {
	return Options{Compression: "none", Partitioner: "consistent_random", Timeout: 30 * time.Second, BufferLimit: 1000, FailMode: "discard"}
}
func (o Options) Validate() error {
	if len(o.Brokers) == 0 || o.Timeout < time.Millisecond || o.Timeout.Milliseconds() > 2147483647 || (o.BufferLimit < 0 || o.BufferLimit > 1000000) {
		return errors.New("kafka: invalid brokers/timeout/buffer options")
	}
	if o.MaxRetries < 0 || o.MaxRetries > 100 || o.RetryBackoff < 0 || (o.RetryBackoff > 0 && o.RetryBackoff < time.Millisecond) || o.RetryBackoff > time.Hour || o.RetryBudget < 0 || (o.RetryBudget > 0 && o.RetryBudget < time.Millisecond) || o.RetryBudget > 24*time.Hour || (o.MaxMessageBytes < 0 || (o.MaxMessageBytes > 0 && o.MaxMessageBytes < 1000)) || o.MaxMessageBytes > 1000000000 {
		return errors.New("kafka: invalid retry/size limits")
	}
	switch o.Compression {
	case "none", "gzip", "snappy", "lz4", "zstd":
	default:
		return errors.New("kafka: invalid compression")
	}
	for _, addr := range o.Brokers {
		if strings.TrimSpace(addr) == "" || strings.ContainsAny(addr, " \t\r\n") {
			return errors.New("kafka: empty broker address")
		}
	}
	switch o.Partitioner {
	case "random", "consistent", "consistent_random", "murmur2", "murmur2_random", "fnv1a", "fnv1a_random":
	default:
		return errors.New("kafka: invalid partitioner")
	}
	if o.Sasl && (o.Username == "" || o.Password == "") {
		return errors.New("kafka: SASL credentials are required")
	}
	if o.FailMode != "retry" && o.FailMode != "discard" {
		return errors.New("kafka: invalid fail mode")
	}
	return nil
}
func legacyOptions() Options {
	o := DefaultOptions()
	o.Brokers = configsource.GetConfArr("KafkaProxy", "brokers")
	o.Sasl = configsource.GetConf("KafkaProxy", "sasl") == "true"
	o.Username = configsource.GetConf("KafkaProxy", "user")
	o.Password = configsource.GetConf("KafkaProxy", "password")
	o.WaitAll = configsource.GetConf("KafkaProxy", "KafkaWaitAll") == "true"
	o.Compression = configsource.GetConfDefault("KafkaProxy", "KafkaCompression", "none")
	if o.Compression == "0" {
		o.Compression = "none"
	}
	o.Partitioner = configsource.GetConfDefault("KafkaProxy", "KafkaPartitioner", "consistent_random")
	if o.Partitioner == "0" {
		o.Partitioner = "consistent_random"
	}
	value := configsource.GetConfDefault("KafkaProxy", "KafkaProducerTimeout", "30s")
	if d, err := time.ParseDuration(value); err == nil {
		o.Timeout = d
	} else {
		o.Timeout = time.Duration(cast.ToInt(value)) * time.Millisecond
	}
	o.BufferLimit = cast.ToInt(configsource.GetConfDefault("KafkaProxy", "bufferLimit", "1000"))
	o.FailMode = configsource.GetConfDefault("KafkaProxy", "failMode", "discard")
	o.MaxRetries = cast.ToInt(configsource.GetConfDefault("KafkaProxy", "maxRetries", "0"))
	o.MaxMessageBytes = cast.ToInt(configsource.GetConfDefault("KafkaProxy", "maxMessageBytes", "0"))
	if d, err := time.ParseDuration(configsource.GetConfDefault("KafkaProxy", "retryBackoff", "100ms")); err == nil {
		o.RetryBackoff = d
	} else {
		o.RetryBackoff = -1
	}
	if d, err := time.ParseDuration(configsource.GetConfDefault("KafkaProxy", "retryBudget", "2m")); err == nil {
		o.RetryBudget = d
	} else {
		o.RetryBudget = -1
	}
	return o
}

// LegacyOptions is only for callers retaining section-based configuration.
func LegacyOptions() Options { return legacyOptions() }

func (o Options) normalized() Options {
	o.Brokers = append([]string(nil), o.Brokers...)
	if o.MaxRetries == 0 {
		o.MaxRetries = 3
	}
	if o.RetryBackoff == 0 {
		o.RetryBackoff = 100 * time.Millisecond
	}
	if o.RetryBudget == 0 {
		o.RetryBudget = 2 * time.Minute
	}
	if o.MaxMessageBytes == 0 {
		o.MaxMessageBytes = 1000000
	}
	return o
}
