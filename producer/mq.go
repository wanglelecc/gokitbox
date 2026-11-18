package producer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/wanglelecc/gokitbox/configsource"
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/producer/common"
	"github.com/wanglelecc/gokitbox/producer/kafka"
	"github.com/wanglelecc/gokitbox/producer/meta"
)

type InitP func(exit chan struct{}, fallBack chan<- []byte) common.MQ

var (
	pid           = 0
	initMap       = make(map[string]InitP)
	initLock      sync.RWMutex
	std           *ProxyManager
	stdMu         sync.Mutex
	defaultClosed bool
)

func AddMQ(mqType string, init InitP) {
	initLock.Lock()
	defer initLock.Unlock()
	initMap[mqType] = init
}

func init() {
	pid = os.Getpid()
}

// Start eagerly validates/acquires SDK resources. Failed construction can be retried.
func Start() error {
	stdMu.Lock()
	defer stdMu.Unlock()
	if defaultClosed {
		return ErrProducerClosed
	}
	if std != nil {
		return nil
	}
	optionsMu.Lock()
	started = true
	var o Options
	if configuredOptions == nil {
		o = legacyOptions()
	} else {
		o = *configuredOptions
		o.Kafka.Brokers = append([]string(nil), o.Kafka.Brokers...)
	}
	optionsMu.Unlock()
	m, err := New(o)
	if err != nil {
		optionsMu.Lock()
		started = false
		optionsMu.Unlock()
		return err
	}
	std = m
	return nil
}
func Init() {
	if err := Start(); err != nil {
		logger.Ex(context.Background(), "producer.Init", "initialization failed", "error", err)
	}
}

type ProxyManager struct {
	input chan []byte

	mqMap map[string]common.MQ
	quits []chan struct{}
	exit  chan string
	done  chan struct{} // 通知 Close 调用方 Run goroutine 已退出

	stopping  chan struct{}
	closeOnce sync.Once
	grayMode  bool
	closeCtx  context.Context
	closeErr  error
	admission sync.RWMutex
}

func NewProxyManager() *ProxyManager { return NewProxyManagerWithOptions(currentOptions()) }

// NewProxyManagerWithOptions retains the old panic-on-construction-error contract.
// Prefer New when initialization errors must be returned to application startup.
func NewProxyManagerWithOptions(o Options) *ProxyManager {
	m, err := New(o)
	if err != nil {
		panic(err)
	}
	return m
}
func New(o Options) (pm *ProxyManager, err error) {
	if err = o.Validate(); err != nil {
		return nil, err
	}
	pm = &ProxyManager{grayMode: o.GrayMode, exit: make(chan string, 1), done: make(chan struct{}), stopping: make(chan struct{}), input: make(chan []byte, o.BufferLimit), mqMap: map[string]common.MQ{}}
	// Legacy factories are copied before invoking user code; no global lock is held.
	initLock.RLock()
	factories := make(map[string]InitP, len(initMap))
	for name, f := range initMap {
		factories[name] = f
	}
	initLock.RUnlock()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("producer factory panic: %v", r)
		}
		if err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for _, mq := range pm.mqMap {
				if closer, ok := mq.(interface{ CloseContext(context.Context) error }); ok {
					err = errors.Join(err, closer.CloseContext(ctx))
				} else {
					// Legacy Close may be asynchronous or block; rollback has an explicit budget.
					closed := make(chan error, 1)
					go func(mq common.MQ) {
						var closeErr error
						defer func() {
							if r := recover(); r != nil {
								closeErr = fmt.Errorf("producer rollback close panic: %v", r)
							}
							closed <- closeErr
						}()
						mq.Close()
					}(mq)
					select {
					case e := <-closed:
						err = errors.Join(err, e)
					case <-ctx.Done():
						err = errors.Join(err, fmt.Errorf("producer rollback incomplete: %w", ctx.Err()))
					}
				}
			}
			for _, quit := range pm.quits {
				select {
				case <-quit:
				case <-ctx.Done():
					err = errors.Join(err, fmt.Errorf("producer rollback resource still running: %w", ctx.Err()))
				}
			}
			pm = nil
		}
	}()
	if o.KafkaEnabled {
		backend, e := kafka.New(o.Kafka)
		if e != nil {
			return pm, e
		}
		pm.mqMap[meta.Kafka] = backend
	}
	for name, factory := range factories {
		// Kafka is exclusively constructed from this instance's Options.
		if name == meta.Kafka {
			continue
		}
		quit := make(chan struct{})
		mq := factory(quit, pm.input)
		if mq == nil {
			return pm, fmt.Errorf("producer: nil backend %s", name)
		}
		pm.mqMap[name] = mq
		pm.quits = append(pm.quits, quit)
		mq.SetFailMode()
	}
	go pm.Run()
	return pm, nil
}

// Kafka 消息投递
var ErrProducerClosed = kafka.ErrClosed

// Kafka preserves the legacy blocking API. New callers should pass a context.
func (m *ProxyManager) Kafka(topic string, msg []byte, key ...string) error {
	return m.KafkaContext(context.Background(), topic, msg, key...)
}

// KafkaContext reports local queue acceptance, not broker acknowledgement.
func (m *ProxyManager) KafkaContext(ctx context.Context, topic string, msg []byte, key ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-m.stopping:
		return ErrProducerClosed
	default:
	}

	mq := m.mqMap[meta.Kafka]
	if mq == nil {
		return errors.New("Kafka proxy does not exist")
	}

	strs := []string{strconv.Itoa(pid)}
	strs = append(strs, strconv.FormatInt(time.Now().UnixNano()/1000000, 10))
	logid := strings.Join(strs, ".")
	if len(key) > 0 {
		logid = key[0]
	}

	if m.grayMode {
		topic = topic + "_gray"
	}

	// Legacy Kafka(nil) sent an empty value, never a compacted-topic tombstone.
	if msg == nil {
		msg = []byte{}
	}
	message := common.Message{Topic: topic, Key: []byte(logid), Value: msg}
	if err := common.ValidateMessage(message); err != nil {
		return err
	}
	if typed, ok := mq.(interface {
		Submit(context.Context, common.Message) error
	}); ok {
		return typed.Submit(ctx, message)
	}
	m.admission.RLock()
	defer m.admission.RUnlock()
	select {
	case <-m.stopping:
		return ErrProducerClosed
	default:
	}
	line, err := common.Encode(message)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.stopping:
		return ErrProducerClosed
	case mq.Input() <- line:
		return nil
	}
}

func Kafka(ctx context.Context, topic string, msg []byte, key ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := Start(); err != nil {
		return err
	}
	stdMu.Lock()
	m := std
	stdMu.Unlock()
	msg = appendTraceId(ctx, msg)
	return m.KafkaContext(ctx, topic, msg, key...)
}

// Redis
func Redis(ctx context.Context, client *redis.Client, topic string, msg []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil {
		return errors.New("producer: Redis client is required")
	}
	msg = appendTraceId(ctx, msg)
	return client.RPush(ctx, topic, string(msg)).Err()
}

// Delay
func Delay(ctx context.Context, client *redis.Client, topic string, msg []byte, score float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil {
		return errors.New("producer: Redis client is required")
	}
	msg = appendTraceId(ctx, msg)
	return client.ZAdd(ctx, topic, redis.Z{
		Score:  score,
		Member: string(msg),
	}).Err()
}

func (m *ProxyManager) Run() {
	for {
		select {
		case msg := <-m.input:
			if err := m.forward(msg); err != nil {
				m.closeErr = errors.Join(m.closeErr, err)
			}
		case <-m.exit:
			m.shutdown()
			return
		}
	}
}
func (m *ProxyManager) forward(raw []byte) error {
	message, err := common.Decode(raw)
	if err != nil {
		return err
	}
	mq := m.mqMap[message.Backend]
	if mq == nil {
		return errors.New("producer: fallback backend unavailable")
	}
	// Historical fallback has no receipts. Never silently discard an owned envelope.
	select {
	case mq.Input() <- raw:
		return nil
	case <-m.stopping:
		return errors.New("producer: fallback not delivered during shutdown: outcome unknown")
	}
}
func (m *ProxyManager) shutdown() {
	m.admission.Lock()
	m.admission.Unlock()
	ctx := m.closeCtx
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case raw := <-m.input:
			m.closeErr = errors.Join(m.closeErr, m.forward(raw))
		default:
			goto drained
		}
	}
drained:
	for _, mq := range m.mqMap {
		if closer, ok := mq.(interface{ CloseContext(context.Context) error }); ok {
			err := closer.CloseContext(ctx)
			// Wait reports final cleanup results rather than an earlier caller timeout.
			if waiter, ok := mq.(interface{ Wait(context.Context) error }); ok {
				m.closeErr = errors.Join(m.closeErr, waiter.Wait(context.Background()))
			} else {
				m.closeErr = errors.Join(m.closeErr, err)
			}
		} else {
			mq.Close()
		}
	}
	for _, quit := range m.quits {
		<-quit
	}
	close(m.done)
}
func (m *ProxyManager) Wait(ctx context.Context) error {
	select {
	case <-m.done:
		return m.closeErr
	default:
	}
	select {
	case <-m.done:
		return m.closeErr
	case <-ctx.Done():
		return fmt.Errorf("producer shutdown incomplete: %w", ctx.Err())
	}
}
func (m *ProxyManager) CloseContext(ctx context.Context) error {
	m.closeOnce.Do(func() {
		m.closeCtx = ctx
		if m.stopping != nil {
			close(m.stopping)
		}
		m.exit <- "shutdown"
	})
	return m.Wait(ctx)
}

// Snapshot exposes asynchronous delivery outcomes without changing publish semantics.
func (m *ProxyManager) Snapshot() kafka.Stats {
	if k, ok := m.mqMap[meta.Kafka].(*kafka.CKafka); ok {
		return k.Snapshot()
	}
	return kafka.Stats{}
}
func (m *ProxyManager) Close() {
	if err := m.CloseContext(context.Background()); err != nil {
		logger.Ex(context.Background(), "producer.Close", "close failed", "error", err)
	}
}
func CloseContext(ctx context.Context) error {
	stdMu.Lock()
	defaultClosed = true
	m := std
	stdMu.Unlock()
	if m == nil {
		return nil
	}
	return m.CloseContext(ctx)
}
func Close() {
	if err := CloseContext(context.Background()); err != nil {
		logger.Ex(context.Background(), "producer.Close", "close failed", "error", err)
	}
}

func appendTraceId(ctx context.Context, body []byte) []byte {
	bodyStr := string(body)
	if !(json.Valid(body) && strings.HasPrefix(bodyStr, "{") && strings.HasSuffix(bodyStr, "}")) {
		return body
	}

	traceId := cast.ToString(ctx.Value("trace_id"))
	if traceId == "" {
		return body
	}

	if gjson.GetBytes(body, "trace_id").String() != "" {
		return body
	}

	nBody, err := sjson.SetBytes(body, "trace_id", traceId)
	if err != nil {
		return body
	}

	return nBody
}

func legacyOptions() Options {
	return Options{KafkaEnabled: cast.ToBool(configsource.GetConfDefault("KafkaProxy", "enable", "false")), Kafka: kafka.LegacyOptions(), BufferLimit: cast.ToInt(configsource.GetConfDefault("Producer", "bufferLimit", "2000")), GrayMode: cast.ToBool(configsource.GetConfDefault("MQProxy", "grayMode", "false"))}
}

// Snapshot returns default-instance delivery counters; it does not initialize it.
func Snapshot() kafka.Stats {
	stdMu.Lock()
	m := std
	stdMu.Unlock()
	if m == nil {
		return kafka.Stats{}
	}
	return m.Snapshot()
}

// Wait waits for actual default-instance shutdown without initiating it.
func Wait(ctx context.Context) error {
	stdMu.Lock()
	m := std
	stdMu.Unlock()
	if m == nil {
		return nil
	}
	return m.Wait(ctx)
}
