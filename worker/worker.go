package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tidwall/gjson"
	"github.com/wanglelecc/gokitbox/bootstrap"
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/worker/consumer"
)

type Worker struct {
	mu                          sync.Mutex
	state                       string
	started                     bool
	beforeFuncs                 []bootstrap.BeforeServerStartFunc
	afterFuncs                  []bootstrap.AfterServerStopFunc
	handlers                    map[string]Handle
	exit                        chan os.Signal
	startDone, runDone          chan struct{}
	startOnce, stopOnce         sync.Once
	startCtx                    context.Context
	startCancel                 context.CancelFunc
	stopErr                     error
	consumers                   *consumer.ConsumeManager
	shutdownTimeout, retryDelay time.Duration
	retryConfigured             bool
	alertFunc                   consumer.AlertFunc
	alertWindow                 time.Duration
}
type Task struct {
	Cmd  string
	Ctx  context.Context
	Data []byte
}

func NewWorker() *Worker {
	return &Worker{state: "new", handlers: map[string]Handle{}, exit: make(chan os.Signal, 1), startDone: make(chan struct{}), runDone: make(chan struct{}), shutdownTimeout: 30 * time.Second, retryDelay: 500 * time.Millisecond}
}

func (w *Worker) Serve() error {
	signal.Notify(w.exit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(w.exit)
	// Observe termination while startup is still running. A buffered result
	// avoids stranding the startup goroutine if a legacy hook ignores cancel.
	started := make(chan error, 1)
	go func() { started <- w.Start(context.Background()) }()
	select {
	case <-w.exit:
		return w.Stop(context.Background())
	case err := <-started:
		if err != nil {
			return errors.Join(err, w.Stop(context.Background()))
		}
	}
	select {
	case <-w.exit:
		return w.Stop(context.Background())
	case <-w.runDone:
		return w.Wait()
	}
}

func (w *Worker) parse(key string, value []byte) *Task {
	return w.parseContext(context.Background(), key, value)
}
func (w *Worker) parseContext(ctx context.Context, key string, value []byte) *Task {
	ctx = context.WithValue(ctx, "log_id", logger.GenLoggerId())
	ctx = context.WithValue(ctx, "start", time.Now())
	trace := gjson.GetBytes(value, "trace_id").String()
	if trace == "" {
		trace = logger.GenTraceId()
	}
	ctx = context.WithValue(ctx, "trace_id", trace)
	return &Task{Cmd: key, Ctx: ctx, Data: value}
}
func (w *Worker) lookup(name string) (Handle, error) {
	w.mu.Lock()
	h, ok := w.handlers[name]
	w.mu.Unlock()
	if ok && h != nil {
		return h, nil
	}
	return GetHandle(name)
}
func (w *Worker) dealContext(ctx context.Context, tpl, key string, value []byte) (ok bool, err error) {
	task := w.parseContext(ctx, key, value)
	defer func() {
		if p := recover(); p != nil {
			ok = false
			err = fmt.Errorf("worker: handler panic (%T)", p)
			logger.Ex(task.Ctx, "worker.handler", "handler panic", "key", key, "error", err)
		}
	}()
	cmd := key
	if tpl != "" {
		cmd = tpl
	}
	h, err := w.lookup(cmd)
	if err != nil {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	ok, err = h(task.Ctx, task.Data)
	if err != nil {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	return ok, nil
}

// deal is the legacy bool-only bridge; retries now belong to the consumer runner.
func (w *Worker) deal(tpl, key string, value []byte) bool {
	ok, err := w.dealContext(context.Background(), tpl, key, value)
	if err != nil {
		logger.Wx(context.Background(), "worker.handler", "handler failed", "key", key, "error", err)
	}
	return ok && err == nil
}
func (w *Worker) AddBeforeServerStartFunc(fns ...bootstrap.BeforeServerStartFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.beforeFuncs = append(w.beforeFuncs, fns...)
}
func (w *Worker) AddAfterServerStopFunc(fns ...bootstrap.AfterServerStopFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.afterFuncs = append(w.afterFuncs, fns...)
}
func (w *Worker) SetShutdownTimeout(timeout time.Duration) *Worker {
	w.mu.Lock()
	defer w.mu.Unlock()
	if timeout > 0 && w.state == "new" {
		w.shutdownTimeout = timeout
	}
	return w
}
func (w *Worker) SetRetryDelay(delay time.Duration) *Worker {
	w.mu.Lock()
	defer w.mu.Unlock()
	if delay > 0 && w.state == "new" {
		w.retryDelay = delay
		w.retryConfigured = true
	}
	return w
}
func (w *Worker) SetAlertFunc(fn consumer.AlertFunc, window time.Duration) *Worker {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.state == "new" {
		w.alertFunc = fn
		w.alertWindow = window
	}
	return w
}
func (w *Worker) RegisterHandle(handles map[string]Handle) bootstrap.BeforeServerStartFunc {
	copy := make(map[string]Handle, len(handles))
	for name, h := range handles {
		copy[name] = h
	}
	return func() error {
		for name, h := range copy {
			if name == "" || h == nil {
				return errors.New("worker: non-empty handler name and function are required")
			}
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		for name, h := range copy {
			w.handlers[name] = h
		}
		return nil
	}
}
func (w *Worker) InitConsumer() bootstrap.BeforeServerStartFunc { return w.InitConsumerWithConfig(nil) }
func (w *Worker) InitConsumerWithConfig(cfg *consumer.Configure) bootstrap.BeforeServerStartFunc {
	cfg = cfg.Snapshot()
	return func() error {
		current := cfg.Snapshot()
		if current == nil {
			var err error
			current, err = consumer.LoadDefaultConfigure()
			if err != nil {
				return err
			}
		}
		w.mu.Lock()
		ctx := w.startCtx
		fn, window := w.alertFunc, w.alertWindow
		retry, override := w.retryDelay, w.retryConfigured
		existing := w.consumers
		w.mu.Unlock()
		if existing != nil {
			return errors.New("worker: consumers already initialized")
		}
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if override {
			current.Runtime.RetryDelay = retry
		}
		if err := current.Validate(); err != nil {
			return err
		}
		if err := w.validateHandlers(current); err != nil {
			return err
		}

		var alerts *consumer.AlertManager
		if fn != nil {
			alerts = consumer.NewAlertManager(fn, window)
		}
		manager, err := consumer.NewConsumeManagerWithContext(current, w.dealContext, alerts)
		if err != nil {
			return err
		}
		w.mu.Lock()
		w.consumers = manager
		w.mu.Unlock()
		return nil
	}
}
func (w *Worker) CloseConsumer() bootstrap.AfterServerStopFunc {
	return func() {
		w.mu.Lock()
		c := w.consumers
		w.mu.Unlock()
		if c != nil {
			c.Close()
		}
	}
}
func SetConfigPath(path string) { consumer.SetConfigPath(path) }

func (w *Worker) Start(ctx context.Context) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	if w.state != "new" {
		w.mu.Unlock()
		return errors.New("worker: already started or stopped")
	}
	w.state = "starting"
	w.started = true
	w.startCtx, w.startCancel = context.WithCancel(ctx)
	startCtx := w.startCtx
	hooks := append([]bootstrap.BeforeServerStartFunc(nil), w.beforeFuncs...)
	w.mu.Unlock()
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("worker: start hook panic (%T)", p)
		}
		w.mu.Lock()
		if err == nil && startCtx.Err() != nil {
			err = startCtx.Err()
		}
		if err == nil && w.state == "starting" {
			w.state = "running"
		}
		w.mu.Unlock()
		w.startOnce.Do(func() { close(w.startDone) })
		if err != nil {
			w.beginStop(context.Background())
		}
	}()
	for _, hook := range hooks {
		if err = startCtx.Err(); err != nil {
			return err
		}
		if hook == nil {
			return errors.New("worker: nil start hook")
		}
		if err = hook(); err != nil {
			return err
		}
	}
	return nil
}
func (w *Worker) Wait() error {
	w.mu.Lock()
	state := w.state
	w.mu.Unlock()
	if state == "new" {
		return errors.New("worker: not started")
	}
	<-w.runDone
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stopErr
}
func (w *Worker) beginStop(parent context.Context) {
	w.stopOnce.Do(func() {
		w.mu.Lock()
		wasNew := w.state == "new"
		w.state = "stopping"
		cancel := w.startCancel
		timeout := w.shutdownTimeout
		w.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if wasNew {
			w.startOnce.Do(func() { close(w.startDone) })
		}
		ctx, stop := context.WithTimeout(parent, timeout)
		go func() { defer stop(); w.shutdown(ctx) }()
	})
}
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	timeout := w.shutdownTimeout
	w.mu.Unlock()
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	w.beginStop(ctx)
	select {
	case <-w.runDone:
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.stopErr
	case <-waitCtx.Done():
		return waitCtx.Err()
	}
}
func (w *Worker) shutdown(ctx context.Context) {
	// An uncooperative legacy start hook cannot be killed. Stop can time out,
	// but Wait must remain open until it stops creating resources.
	<-w.startDone
	w.mu.Lock()
	manager := w.consumers
	var hooks []bootstrap.AfterServerStopFunc
	if w.started {
		hooks = append(hooks, w.afterFuncs...)
	}
	w.mu.Unlock()
	var result error
	if manager != nil {
		err := manager.CloseContext(ctx)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			err = manager.CloseContext(context.WithoutCancel(ctx))
		}
		result = errors.Join(result, err)
	}
	for _, hook := range hooks {
		err := func() (err error) {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("worker: stop hook panic (%T)", p)
				}
			}()
			if hook == nil {
				return errors.New("worker: nil stop hook")
			}
			hook()
			return nil
		}()
		result = errors.Join(result, err)
	}
	w.mu.Lock()
	w.stopErr = result
	w.state = "stopped"
	w.mu.Unlock()
	close(w.runDone)
}

func (w *Worker) validateHandlers(cfg *consumer.Configure) error {
	check := func(name string) error {
		_, err := w.lookup(name)
		if err != nil {
			return fmt.Errorf("worker: configured handler %q unavailable: %w", name, err)
		}
		return nil
	}
	if cfg.Enabled.Kafka {
		for _, v := range cfg.Kafka {
			name := v.TplName
			if name == "" {
				name = v.Topic
			}
			if err := check(name); err != nil {
				return err
			}
		}
	}
	if cfg.Enabled.Redis {
		for _, v := range cfg.Redis {
			name := v.TplName
			if name == "" {
				name = v.Queue
			}
			if err := check(name); err != nil {
				return err
			}
		}
	}
	if cfg.Enabled.Delay {
		for _, v := range cfg.Delay {
			name := v.TplName
			if name == "" {
				name = v.Queue
			}
			if err := check(name); err != nil {
				return err
			}
		}
	}
	// RoutingKey is not known until an AMQP delivery arrives. Validate explicit
	// template names here and resolve dynamic legacy routing keys per delivery.
	if cfg.Enabled.Rabbitmq {
		for _, v := range cfg.Rabbitmq {
			if v.TplName != "" {
				if err := check(v.TplName); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
