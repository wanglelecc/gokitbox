package worker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/wanglelecc/gokitbox/bootstrap"
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/worker/consumer"

	"github.com/tidwall/gjson"
)

// contextKey 用于 context.WithValue 的专用类型，避免与第三方包的 key 碰撞
type contextKey string

type Worker struct {
	beforeFuncs []bootstrap.BeforeServerStartFunc
	afterFuncs  []bootstrap.AfterServerStopFunc
	exit        chan os.Signal

	consumers *consumer.ConsumeManager

	// 超时配置
	shutdownTimeout time.Duration // 优雅关闭超时时间，默认30秒
	retryDelay      time.Duration // 消息处理失败后重试延迟，默认500ms

	// 告警配置
	alertFunc   consumer.AlertFunc // 应用层注册的告警回调，为 nil 时不发告警
	alertWindow time.Duration      // 告警收敛窗口，默认 1 分钟
}

type Task struct {
	Cmd  string
	Ctx  context.Context
	Data []byte
}

func NewWorker() *Worker {
	w := new(Worker)
	w.exit = make(chan os.Signal, 1)

	// 设置合理的默认值
	w.shutdownTimeout = 30 * time.Second  // 默认30秒，满足大部分场景
	w.retryDelay = 500 * time.Millisecond // 默认500ms

	return w
}

func (w *Worker) Serve() error {
	var err error
	for _, fn := range w.beforeFuncs {
		err = fn()
		if err != nil {
			return err
		}
	}

	signal.Notify(w.exit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(w.exit)

	w.waitShutdown()

	for _, fn := range w.afterFuncs {
		fn()
	}

	return nil
}

func (w *Worker) waitShutdown() {
	<-w.exit

	ctx := context.Background()
	logger.Ix(ctx, "worker", fmt.Sprintf(
		"Received shutdown signal, starting graceful shutdown (timeout: %v)...",
		w.shutdownTimeout))

	startTime := time.Now()

	// 同步调用：CloseWithTimeout 内部处理超时和日志，此处无需额外 goroutine。
	// 避免异步关闭与 afterFuncs 里的 CloseConsumer() 并发执行的竞态问题。
	if w.consumers != nil {
		w.consumers.CloseWithTimeout(w.shutdownTimeout)
	}

	logger.Ix(ctx, "worker", fmt.Sprintf(
		"Graceful shutdown completed in %v", time.Since(startTime)))
}

func (w *Worker) parse(key string, value []byte) *Task {
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey("log_id"), logger.GenLoggerId())
	ctx = context.WithValue(ctx, contextKey("start"), time.Now())

	task := new(Task)
	task.Data = value
	task.Cmd = key

	// 处理链路ID，为空生成新链路ID埋入上下文
	traceId := gjson.Get(string(task.Data), "trace_id").String()
	if traceId == "" {
		traceId = logger.GenTraceId()
	}
	ctx = context.WithValue(ctx, contextKey("trace_id"), traceId)

	task.Ctx = ctx

	return task
}

func (w *Worker) deal(tplName string, key string, value []byte) (ret bool) {
	// 解析消息
	task := w.parse(key, value)

	defer func() {
		if r := recover(); r != nil {
			logger.Ex(task.Ctx, "recovery", "worker deal recovery error", "error", fmt.Sprintf("%v", r), "stacks", string(debug.Stack()), "key", key, "value", string(value))
			ret = false
		}
	}()

	cmd := task.Cmd
	if tplName != "" {
		cmd = tplName
	}

	if h, err := GetHandle(cmd); err == nil {
		ret, err = h(task.Ctx, task.Data)
	} else {
		ret = false
		logger.Ex(task.Ctx, "deal", "execute handle error", "error", err.Error(), "key", key, "value", string(value))
	}

	if !ret {
		// 使用配置的重试延迟（而非硬编码）
		time.Sleep(w.retryDelay)
	}

	logger.Dx(task.Ctx, "deal", fmt.Sprintf("ret:%t, key:%s, value:%s", ret, key, string(value)))

	return ret
}

func (w *Worker) AddBeforeServerStartFunc(fns ...bootstrap.BeforeServerStartFunc) {
	for _, fn := range fns {
		w.beforeFuncs = append(w.beforeFuncs, fn)
	}
}

func (w *Worker) AddAfterServerStopFunc(fns ...bootstrap.AfterServerStopFunc) {
	for _, fn := range fns {
		w.afterFuncs = append(w.afterFuncs, fn)
	}
}

// SetShutdownTimeout 设置优雅关闭超时时间
// 默认30秒。对于处理耗时较长的业务（如大文件处理），可调用此方法设置更长的超时时间
// 示例: worker.NewWorker().SetShutdownTimeout(60 * time.Second)
func (w *Worker) SetShutdownTimeout(timeout time.Duration) *Worker {
	if timeout > 0 {
		w.shutdownTimeout = timeout
	}
	return w
}

// SetRetryDelay 设置消息处理失败后的重试延迟
// 默认500ms。可根据业务特点和下游服务恢复时间调整
// 示例: worker.NewWorker().SetRetryDelay(1 * time.Second)
func (w *Worker) SetRetryDelay(delay time.Duration) *Worker {
	if delay > 0 {
		w.retryDelay = delay
	}
	return w
}

// SetAlertFunc 注册告警回调，当消费者发生故障时触发
//   - fn: 告警回调函数，在独立 goroutine 中调用，内部 panic 会被 recover 隔离
//   - window: 同类告警收敛窗口，同一 (type, component, name) 组合在窗口期内只触发一次
//     推荐 1~5 分钟；≤ 0 时默认 1 分钟
//
// 示例:
//
//	worker.NewWorker().SetAlertFunc(func(e consumer.AlertEvent) {
//	    log.Printf("[ALERT] %s", e)
//	    // 可接入钉钉/短信/PagerDuty...
//	}, 2*time.Minute)
func (w *Worker) SetAlertFunc(fn consumer.AlertFunc, window time.Duration) *Worker {
	w.alertFunc = fn
	w.alertWindow = window
	return w
}

// 注册处理器
func (w *Worker) RegisterHandle(handles map[string]Handle) bootstrap.BeforeServerStartFunc {
	return func() error {
		BatchRegisterHandle(handles)
		return nil
	}
}

// 初始化 Consumer
func (w *Worker) InitConsumer() bootstrap.BeforeServerStartFunc {
	return func() error {
		var alertMgr *consumer.AlertManager
		if w.alertFunc != nil {
			alertMgr = consumer.NewAlertManager(w.alertFunc, w.alertWindow)
		}
		w.consumers = consumer.NewConsumeManager(w.deal, alertMgr)
		return nil
	}
}

// 关闭 Consumer
func (w *Worker) CloseConsumer() bootstrap.AfterServerStopFunc {
	return func() {
		ctx := context.Background()
		logger.Wx(ctx, "Exec", "Process Stop...")
		w.consumers.Close()
		logger.Wx(ctx, "Exec", "Process Stop Complete")
	}
}

// 设置消费者配置文件路径
func SetConfigPath(path string) {
	consumer.SetConfigPath(path)
}
