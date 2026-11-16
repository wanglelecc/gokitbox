package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wanglelecc/gokitbox/logger"
)

// ContextConsumeCallback is the cancellable form of the legacy callback.
// An error always means failure, even when the returned bool is true.
type ContextConsumeCallback func(context.Context, string, string, []byte) (bool, error)

type RuntimeOptions struct {
	HandlerTimeout   time.Duration `json:"handlerTimeout" yaml:"handlerTimeout"`
	RetryDelay       time.Duration `json:"retryDelay" yaml:"retryDelay"`
	DeliveryTimeout  time.Duration `json:"deliveryTimeout" yaml:"deliveryTimeout"`
	OperationTimeout time.Duration `json:"operationTimeout" yaml:"operationTimeout"`
	SessionTimeout   time.Duration `json:"sessionTimeout" yaml:"sessionTimeout"`
	MaxPollInterval  time.Duration `json:"maxPollInterval" yaml:"maxPollInterval"`
}

func (o RuntimeOptions) defaults() RuntimeOptions {
	if o.HandlerTimeout == 0 {
		o.HandlerTimeout = 30 * time.Second
	}
	if o.RetryDelay == 0 {
		o.RetryDelay = 500 * time.Millisecond
	}
	if o.DeliveryTimeout == 0 {
		o.DeliveryTimeout = 10 * time.Second
	}
	if o.OperationTimeout == 0 {
		o.OperationTimeout = 3 * time.Second
	}
	if o.SessionTimeout == 0 {
		o.SessionTimeout = 30 * time.Second
	}
	if o.MaxPollInterval == 0 {
		o.MaxPollInterval = 5 * time.Minute
	}
	return o
}
func (o RuntimeOptions) Validate() error {
	if o.HandlerTimeout < 0 || o.RetryDelay < 0 || o.DeliveryTimeout < 0 || o.OperationTimeout < 0 || o.MaxPollInterval < 0 || o.SessionTimeout < 0 {
		return errors.New("worker: negative runtime duration")
	}
	o = o.defaults()
	if o.HandlerTimeout > 24*time.Hour || o.RetryDelay > 24*time.Hour || o.DeliveryTimeout > 24*time.Hour || o.OperationTimeout > 24*time.Hour {
		return errors.New("worker: runtime duration exceeds 24 hours")
	}
	if o.OperationTimeout < time.Millisecond || o.DeliveryTimeout < time.Millisecond {
		return errors.New("worker: I/O and delivery timeout must be at least 1ms")
	}
	if o.SessionTimeout < 3*time.Second || o.SessionTimeout > time.Hour || o.MaxPollInterval < o.SessionTimeout {
		return errors.New("worker: sessionTimeout must be 3s..1h and fit maxPollInterval")
	}
	if o.MaxPollInterval > 24*time.Hour || o.MaxPollInterval < time.Second {
		return errors.New("worker: maxPollInterval outside Kafka millisecond range")
	}
	return nil
}
func (o RuntimeOptions) validateKafka(c *KafkaConfig) error {
	o = o.defaults()
	if o.OperationTimeout < time.Second || o.OperationTimeout > 5*time.Minute {
		return errors.New("worker: Kafka operationTimeout must be 1s..5m")
	}
	// Divide the available poll budget to avoid duration multiplication overflow.
	// Three failure publishes, three commits and their four retry waits are
	// reserved separately from business retries. Duration bounds above keep
	// the fixed sum safe; division avoids multiplying an untrusted FailCount.
	fixed := 3*o.DeliveryTimeout + 3*o.OperationTimeout + 4*o.RetryDelay + 5*time.Second
	remaining := o.MaxPollInterval - fixed
	attempt := o.HandlerTimeout + o.RetryDelay
	if remaining < o.HandlerTimeout || uint64(c.FailCount) > uint64((remaining-o.HandlerTimeout)/attempt) {
		return errors.New("worker: Kafka handler/retry/delivery budget must fit maxPollInterval")
	}
	return nil
}

type consumerRuntime struct {
	ctx      context.Context
	cancel   context.CancelFunc
	options  RuntimeOptions
	callback ContextConsumeCallback
	errorsMu sync.Mutex
	drainErr error
}

func newRuntime(o RuntimeOptions, callback ContextConsumeCallback) *consumerRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	return &consumerRuntime{ctx: ctx, cancel: cancel, options: o.defaults(), callback: callback}
}
func adaptCallback(callback ConsumeCallback) ContextConsumeCallback {
	if callback == nil {
		return nil
	}
	return func(_ context.Context, tpl, key string, value []byte) (bool, error) {
		return callback(tpl, key, value), nil
	}
}
func (r *consumerRuntime) call(ctx context.Context, legacy ConsumeCallback, tpl, key string, value []byte) (ok bool) {
	o := RuntimeOptions{}.defaults()
	if r != nil {
		o = r.options
	}
	callCtx, cancel := context.WithTimeout(ctx, o.HandlerTimeout)
	defer cancel()
	defer func() {
		if v := recover(); v != nil {
			ok = false
			logger.Ex(callCtx, "worker.callback", "handler panic", "type", fmt.Sprintf("%T", v), "key", key)
		}
	}()
	var err error
	// Each attempt borrows its own payload so handler mutation cannot corrupt
	// later retries or the original bytes sent to a failure queue.
	payload := bytes.Clone(value)
	if r != nil && r.callback != nil {
		ok, err = r.callback(callCtx, tpl, key, payload)
	} else if legacy != nil {
		ok = legacy(tpl, key, payload)
	} else {
		err = errors.New("worker: nil callback")
	}
	if err == nil {
		err = callCtx.Err()
	}
	if err != nil {
		logger.Wx(callCtx, "worker.callback", "handler failed", "key", key, "error", err)
		return false
	}
	return ok
}
func (r *consumerRuntime) context() context.Context {
	if r == nil {
		return context.Background()
	}
	return r.ctx
}
func (r *consumerRuntime) opts() RuntimeOptions {
	if r == nil {
		return RuntimeOptions{}.defaults()
	}
	return r.options
}

// Stop accepting new messages immediately. In-flight work drains until the
// first shutdown caller's deadline, then its cooperative context is cancelled.
func (r *consumerRuntime) cancelOnDeadline(ctx context.Context) func() bool {
	if r == nil {
		return func() bool { return true }
	}
	return context.AfterFunc(ctx, r.cancel)
}
func waitRetry(ctx context.Context, quit <-chan struct{}, delay time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case <-quit:
		return false
	default:
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-quit:
		return false
	case <-timer.C:
		return true
	}
}
func stopRequested(ctx context.Context, quit <-chan struct{}) bool {
	if ctx.Err() != nil {
		return true
	}
	select {
	case <-quit:
		return true
	default:
		return false
	}
}
func boundedContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

type redisConnection struct {
	addr, username, password string
	db                       int
}

func (r *consumerRuntime) recordError(err error) {
	if r == nil || err == nil {
		return
	}
	r.errorsMu.Lock()
	r.drainErr = errors.Join(r.drainErr, err)
	r.errorsMu.Unlock()
}
func (r *consumerRuntime) drainError() error {
	if r == nil {
		return nil
	}
	r.errorsMu.Lock()
	defer r.errorsMu.Unlock()
	return r.drainErr
}

// Legacy Redis modes cannot recover a process crash. On an orderly failure or
// cancellation, try to put the owned payload back before leaving this session.
func (r *consumerRuntime) processLegacy(ctx context.Context, quit <-chan struct{}, legacy ConsumeCallback, tpl, key string, value []byte, retries int, successes, failures *int64, forward, restore func(context.Context) error, discard func()) (bool, error) {
	restoreOwned := func() (bool, error) {
		cleanup, cancel := boundedContext(context.WithoutCancel(ctx), r.opts().OperationTimeout)
		defer cancel()
		if err := restore(cleanup); err != nil {
			return false, fmt.Errorf("worker: unable to restore legacy in-flight message: %w", err)
		}
		return false, nil
	}
	for attempt := 0; attempt <= retries; attempt++ {
		if stopRequested(ctx, quit) {
			return restoreOwned()
		}
		if r.call(ctx, legacy, tpl, key, value) {
			atomic.AddInt64(successes, 1)
			return true, nil
		}
		if attempt < retries && !waitRetry(ctx, quit, r.opts().RetryDelay) {
			return restoreOwned()
		}
	}
	atomic.AddInt64(failures, 1)
	if stopRequested(ctx, quit) {
		return restoreOwned()
	}
	if forward == nil {
		discard()
		return true, nil
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if stopRequested(ctx, quit) {
			return restoreOwned()
		}
		operation, cancel := boundedContext(ctx, r.opts().OperationTimeout)
		last = forward(operation)
		cancel()
		if last == nil {
			return true, nil
		}
		if attempt < 2 && !waitRetry(ctx, quit, r.opts().RetryDelay) {
			return restoreOwned()
		}
	}
	logger.Wx(ctx, "worker.legacy", "failure queue unavailable; restoring source message", "key", key, "error", last)
	return restoreOwned()
}

func DefaultRuntimeOptions() RuntimeOptions { return (RuntimeOptions{}).defaults() }

// JSON accepts human-readable durations just like YAML, while retaining the
// original numeric time.Duration representation for Go-generated JSON files.
func (o *RuntimeOptions) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return errors.New("worker: invalid runtime options")
	}
	var next RuntimeOptions
	targets := map[string]*time.Duration{"handlerTimeout": &next.HandlerTimeout, "retryDelay": &next.RetryDelay, "deliveryTimeout": &next.DeliveryTimeout, "operationTimeout": &next.OperationTimeout, "sessionTimeout": &next.SessionTimeout, "maxPollInterval": &next.MaxPollInterval}
	for key, raw := range fields {
		target, ok := targets[key]
		if !ok {
			return fmt.Errorf("worker: unknown runtime option %q", key)
		}
		var err error
		if len(raw) > 0 && raw[0] == '"' {
			var text string
			err = json.Unmarshal(raw, &text)
			if err == nil {
				*target, err = time.ParseDuration(text)
			}
		} else {
			err = json.Unmarshal(raw, target)
		}
		if err != nil {
			return fmt.Errorf("worker: invalid duration for %s", key)
		}
	}
	*o = next
	return nil
}
