package kafka

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	native "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/wanglelecc/gokitbox/producer/common"
)

var ErrClosed = errors.New("producer: closed")

// Stats counts messages, not attempts. Failed is terminal; Dropped is its subset
// for forced abandonment (unknown outcome or legacy Input overflow). Pending includes local queues.
type Stats struct {
	Accepted, Delivered, Failed, Retried, Dropped, Pending int64
	LastError                                              string
}
type job struct {
	ready    chan struct{}
	message  common.Message
	created  time.Time
	next     time.Time
	attempts int
	inflight bool
}

type nativeProducer interface {
	Produce(*native.Message, chan native.Event) error
	Events() chan native.Event
	Flush(int) int
	Close()
}

type CKafka struct {
	input       chan []byte // compatibility only; Submit is the context-aware admission API
	queue       chan *job
	slots       chan struct{}
	producer    nativeProducer
	cfg         *native.ConfigMap
	options     Options
	exit        chan struct{}
	done        chan struct{}
	closeOnce   sync.Once
	admission   sync.RWMutex
	stopMu      sync.Mutex
	stopContext context.Context
	shutdownErr error
	statsMu     sync.Mutex
	stats       Stats
}

func buildProducerConfig() (*native.ConfigMap, error) { return buildConfig(legacyOptions()) }
func buildConfig(o Options) (*native.ConfigMap, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	o = o.normalized()
	// librdkafka requires message.timeout.ms > linger.ms, including short timeouts.
	timeout := min(o.Timeout, o.RetryBudget)
	linger := min(int64(500), timeout.Milliseconds()-1)
	cfg := &native.ConfigMap{
		"bootstrap.servers": strings.Join(o.Brokers, ","), "message.max.bytes": o.MaxMessageBytes,
		"queue.buffering.max.messages": max(1, o.BufferLimit+1),
		"linger.ms":                    int(linger), "sticky.partitioning.linger.ms": int(2 * linger),
		"message.timeout.ms": int(timeout.Milliseconds()), "retries": 2147482647,
		"retry.backoff.ms": 1000, "reconnect.backoff.max.ms": 3000,
		"acks": "1", "compression.type": o.Compression, "partitioner": o.Partitioner,
		"security.protocol": "plaintext",
	}
	if o.WaitAll {
		(*cfg)["acks"] = "all"
	}
	if o.Sasl {
		(*cfg)["security.protocol"] = "sasl_plaintext"
		(*cfg)["sasl.mechanism"] = "PLAIN"
		(*cfg)["sasl.username"] = o.Username
		(*cfg)["sasl.password"] = o.Password
	}
	return cfg, nil
}

// New constructs native resources eagerly and returns configuration errors.
// It verifies local SDK configuration, not broker connectivity/readiness.
func New(o Options) (*CKafka, error) {
	cfg, err := buildConfig(o)
	if err != nil {
		return nil, err
	}
	p, err := native.NewProducer(cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}
	k := &CKafka{producer: p, cfg: cfg, options: o.normalized(), input: make(chan []byte, o.BufferLimit), queue: make(chan *job, o.BufferLimit), slots: make(chan struct{}, max(1, o.BufferLimit+1)), exit: make(chan struct{}), done: make(chan struct{})}
	go k.run(nil)
	return k, nil
}
func CInit(quit chan struct{}, fallback chan<- []byte) common.MQ {
	return InitWithOptions(legacyOptions(), quit, fallback)
}

// InitWithOptions retains the historical panic-on-construction-error contract.
// New code should use New. Retries are instance-owned, never sent to fallback.
func InitWithOptions(o Options, quit chan struct{}, fallback chan<- []byte) common.MQ {
	k, err := New(o)
	if err != nil {
		panic(err)
	}
	if quit != nil {
		go func() { <-k.done; close(quit) }()
	}
	return k
}
func (k *CKafka) Input() chan<- []byte { return k.input }
func (k *CKafka) SetFailMode()         {} // policy is immutable and installed before startup
func (k *CKafka) Submit(ctx context.Context, m common.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := common.ValidateMessage(m); err != nil {
		return err
	}
	// Reserve protocol overhead as well as the key; native max.bytes applies to the whole record.
	if len(m.Value)+len(m.Key)+len(m.Topic)+64 > k.options.MaxMessageBytes {
		return errors.New("kafka: message exceeds max bytes")
	}
	k.admission.RLock()
	defer k.admission.RUnlock()
	select {
	case <-k.exit:
		return ErrClosed
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-k.exit:
		return ErrClosed
	case k.slots <- struct{}{}:
	}
	m.Key = bytes.Clone(m.Key)
	m.Value = bytes.Clone(m.Value)
	j := &job{message: m, created: time.Now(), ready: make(chan struct{})}
	select {
	case <-ctx.Done():
		<-k.slots
		return ctx.Err()
	case <-k.exit:
		<-k.slots
		return ErrClosed
	case k.queue <- j:
		k.statsMu.Lock()
		k.stats.Accepted++
		k.statsMu.Unlock()
		close(j.ready)
		return nil
	}
}
func (k *CKafka) Snapshot() Stats {
	k.statsMu.Lock()
	defer k.statsMu.Unlock()
	s := k.stats
	s.Pending = s.Accepted - s.Delivered - s.Failed
	return s
}
func (k *CKafka) Close() { k.beginClose(context.Background()) }
func (k *CKafka) beginClose(ctx context.Context) {
	k.closeOnce.Do(func() { k.stopMu.Lock(); k.stopContext = ctx; k.stopMu.Unlock(); close(k.exit) })
}
func (k *CKafka) CloseContext(ctx context.Context) error { k.beginClose(ctx); return k.Wait(ctx) }
func (k *CKafka) Wait(ctx context.Context) error {
	select {
	case <-k.done:
		return k.shutdownErr
	default:
	}
	select {
	case <-k.done:
		return k.shutdownErr
	case <-ctx.Done():
		return fmt.Errorf("kafka shutdown incomplete: %w", ctx.Err())
	}
}

func retryable(err error) bool {
	var e native.Error
	if !errors.As(err, &e) {
		return false
	}
	if e.IsFatal() {
		return false
	}
	switch e.Code() {
	case native.ErrQueueFull, native.ErrMsgTimedOut, native.ErrRequestTimedOut, native.ErrTransport, native.ErrAllBrokersDown:
		return true
	}
	return e.IsRetriable()
}
func (k *CKafka) run(quit chan struct{}) {
	active := make(map[*job]struct{})
	var fatalErr error
	// Cleanup always releases the native handle and signals actual completion.
	defer func() {
		if r := recover(); r != nil {
			fatalErr = fmt.Errorf("kafka producer panic: %v", r)
		}
		k.beginClose(context.Background())
		k.admission.Lock()
		k.admission.Unlock()
		for {
			select {
			case j := <-k.queue:
				active[j] = struct{}{}
			default:
				goto drained
			}
		}
	drained:
		for {
			select {
			case <-k.input:
				k.statsMu.Lock()
				k.stats.Accepted++
				k.stats.Failed++
				k.stats.Dropped++
				k.stats.LastError = "kafka: legacy input not processed before shutdown"
				k.statsMu.Unlock()
			default:
				goto rawDrained
			}
		}
	rawDrained:
		for j := range active {
			k.finish(j, errors.New("kafka: delivery outcome unknown during shutdown"), true)
			delete(active, j)
		}
		// Native Flush must be inspected. No pending work should remain on a normal close.
		// On forced close the budget is exhausted: Flush(0) only inspects, then Close purges.
		remaining := k.producer.Flush(0)
		if remaining > 0 {
			fatalErr = errors.Join(fatalErr, fmt.Errorf("kafka: %d native events/records remain at close", remaining))
		}
		k.producer.Close()
		s := k.Snapshot()
		if s.Failed > 0 {
			fatalErr = errors.Join(fatalErr, fmt.Errorf("kafka: %d terminal failures (%d unknown), last: %s", s.Failed, s.Dropped, s.LastError))
		}
		if fatalErr != nil {
			k.statsMu.Lock()
			k.stats.LastError = fatalErr.Error()
			k.statsMu.Unlock()
		}
		k.shutdownErr = fatalErr
		close(k.done)
		if quit != nil {
			close(quit)
		}
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	closing := false
	stop := k.exit
	legacyInput := k.input
	var deadline <-chan struct{}
	for {
		if closing && len(active) == 0 && len(k.queue) == 0 && len(k.input) == 0 {
			return
		}
		select {
		case <-stop:
			stop = nil
			closing = true
			k.admission.Lock()
			k.admission.Unlock()
			k.stopMu.Lock()
			ctx := k.stopContext
			k.stopMu.Unlock()
			deadline = ctx.Done()
		case <-deadline:
			fatalErr = errors.New("kafka: shutdown deadline exhausted")
			return
		case j := <-k.queue:
			if j.ready != nil {
				<-j.ready
			}
			active[j] = struct{}{}
			k.send(j, active)
		case b, ok := <-legacyInput:
			if !ok {
				legacyInput = nil
				continue
			}
			m, err := common.Decode(b)
			if err != nil {
				k.statsMu.Lock()
				k.stats.Accepted++
				k.stats.Failed++
				k.stats.LastError = err.Error()
				k.statsMu.Unlock()
				continue
			}
			// Legacy Input has no error return. Overflow is explicitly terminal and counted.
			k.statsMu.Lock()
			k.stats.Accepted++
			k.statsMu.Unlock()
			select {
			case k.slots <- struct{}{}:
				j := &job{message: m, created: time.Now()}
				active[j] = struct{}{}
				k.send(j, active)
			default:
				k.statsMu.Lock()
				k.stats.Failed++
				k.stats.Dropped++
				k.stats.LastError = "kafka: legacy input capacity exceeded"
				k.statsMu.Unlock()
			}
		case event, ok := <-k.producer.Events():
			if !ok {
				fatalErr = errors.New("kafka: native event stream closed unexpectedly")
				return
			}
			switch e := event.(type) {
			case *native.Message:
				j, ok := e.Opaque.(*job)
				if !ok {
					continue
				}
				if _, ok = active[j]; !ok {
					continue
				}
				j.inflight = false
				if e.TopicPartition.Error == nil {
					k.finish(j, nil, false)
					delete(active, j)
				} else {
					k.failure(j, e.TopicPartition.Error, active)
				}
			case native.Error:
				if e.IsFatal() {
					fatalErr = e
					return
				}
			}
		case now := <-ticker.C:
			for j := range active {
				if now.Sub(j.created) >= k.options.RetryBudget {
					k.finish(j, errors.New("kafka: total delivery budget exhausted"), j.inflight)
					delete(active, j)
					continue
				}
				if !j.inflight && !now.Before(j.next) {
					k.send(j, active)
				}
			}
		}
	}
}
func (k *CKafka) send(j *job, active map[*job]struct{}) {
	if time.Since(j.created) >= k.options.RetryBudget {
		k.finish(j, errors.New("kafka: retry budget exhausted"), false)
		delete(active, j)
		return
	}
	if j.attempts > 0 {
		k.statsMu.Lock()
		k.stats.Retried++
		k.statsMu.Unlock()
	}
	j.attempts++
	err := k.producer.Produce(&native.Message{TopicPartition: native.TopicPartition{Topic: &j.message.Topic, Partition: native.PartitionAny}, Key: j.message.Key, Value: j.message.Value, Opaque: j}, nil)
	if err != nil {
		k.failure(j, err, active)
	} else {
		j.inflight = true
	}
}
func (k *CKafka) failure(j *job, err error, active map[*job]struct{}) {
	if k.options.FailMode == "retry" && retryable(err) && j.attempts <= k.options.MaxRetries {
		delay := k.options.RetryBackoff * time.Duration(1<<min(j.attempts-1, 6))
		if time.Since(j.created)+delay+min(k.options.Timeout, k.options.RetryBudget) <= k.options.RetryBudget {
			j.next = time.Now().Add(delay)
			return
		}
	}
	k.finish(j, err, false)
	delete(active, j)
}
func (k *CKafka) finish(j *job, err error, unknown bool) {
	<-k.slots
	k.statsMu.Lock()
	defer k.statsMu.Unlock()
	if err == nil {
		k.stats.Delivered++
	} else {
		k.stats.Failed++
		k.stats.LastError = err.Error()
		log.Printf("producer delivery failed: topic=%s attempts=%d unknown=%t error=%v", j.message.Topic, j.attempts, unknown, err)
		if unknown {
			k.stats.Dropped++
		}
	}
}
