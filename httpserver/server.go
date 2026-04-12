package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wanglelecc/gokitbox/bootstrap"
	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/httpserver/validator"
	"github.com/wanglelecc/gokitbox/logger"

	"github.com/gin-gonic/gin"
)

type Server struct {
	server          *http.Server
	engine          *gin.Engine // 直接持有 Engine，避免 Handler 字段类型断言
	beforeFuncs     []bootstrap.BeforeServerStartFunc
	afterFuncs      []bootstrap.AfterServerStopFunc
	opts            ServerOptions
	sigChan         chan os.Signal // 接收系统退出信号
	shutdownDone    chan struct{}  // 用于同步 shutdown 完成
	shutdownTimeout int64          // 优雅关闭超时时间，原子读写（单位：纳秒）
	served          int32          // atomic 防重入：0=未启动，1=已启动
}

// NewServer 从配置文件 [HttpServer] 段加载配置并创建 Server。
// 配置加载失败时使用默认配置。section 参数可指定其他配置段名。
func NewServer(section ...string) (*Server, error) {
	sec := "HttpServer"
	if len(section) > 0 && section[0] != "" {
		sec = section[0]
	}

	var opts ServerOptions
	if err := config.ConfMapToStruct(sec, &opts); err != nil {
		// 配置加载失败，使用默认配置
		opts = DefaultOptions()
	} else {
		// 用默认值填充未配置项（空值表示未配置）
		opts = fillDefaults(opts, DefaultOptions())
	}

	return NewServerWithOptions(opts)
}

// fillDefaults 用默认值填充 opts 中的空值字段
func fillDefaults(opts, defaults ServerOptions) ServerOptions {
	if opts.Mode == "" {
		opts.Mode = defaults.Mode
	}
	if opts.Addr == "" {
		opts.Addr = defaults.Addr
	}
	if opts.ReadHeaderTimeout == "" {
		opts.ReadHeaderTimeout = defaults.ReadHeaderTimeout
	}
	if opts.ReadTimeout == "" {
		opts.ReadTimeout = defaults.ReadTimeout
	}
	if opts.WriteTimeout == "" {
		opts.WriteTimeout = defaults.WriteTimeout
	}
	if opts.IdleTimeout == "" {
		opts.IdleTimeout = defaults.IdleTimeout
	}
	if opts.ShutdownTimeout == "" {
		opts.ShutdownTimeout = defaults.ShutdownTimeout
	}
	return opts
}

// NewServerWithOptions 使用自定义配置创建 Server
func NewServerWithOptions(opts ServerOptions) (*Server, error) {
	s := new(Server)
	s.opts = opts

	shutdownTimeout, err := opts.GetShutdownTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid ShutdownTimeout: %w", err)
	}
	atomic.StoreInt64(&s.shutdownTimeout, int64(shutdownTimeout))

	if err := s.InitGinValidatorChinese(); err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	// 使用配置的 Mode，必须在 gin.New() 之前调用
	if opts.Mode != "" {
		gin.SetMode(opts.Mode)
	}

	s.engine = gin.New()

	readHeaderTimeout, err := opts.GetReadHeaderTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid ReadHeaderTimeout: %w", err)
	}
	readTimeout, err := opts.GetReadTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid ReadTimeout: %w", err)
	}
	writeTimeout, err := opts.GetWriteTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid WriteTimeout: %w", err)
	}
	idleTimeout, err := opts.GetIdleTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid IdleTimeout: %w", err)
	}

	s.server = &http.Server{
		Addr:              opts.Addr,
		Handler:           s.engine,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	s.sigChan = make(chan os.Signal, 1)
	s.shutdownDone = make(chan struct{})

	return s, nil
}

func (s *Server) Serve() error {
	if !atomic.CompareAndSwapInt32(&s.served, 0, 1) {
		return fmt.Errorf("httpserver: Serve already called, Server is not reusable")
	}

	for _, fn := range s.beforeFuncs {
		if err := fn(); err != nil {
			return err
		}
	}

	quit := make(chan struct{})
	signal.Notify(s.sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(s.sigChan)

	go s.waitShutdown(quit)
	logger.Ix(context.Background(), "httpserver", fmt.Sprintf("httpserver start and serve:%s", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		// 启动失败（如端口被占用），通知 waitShutdown 退出
		close(quit)
		// 等待 waitShutdown goroutine 完全退出，确保 Serve() 返回前无悬挂 goroutine
		<-s.shutdownDone
		for _, fn := range s.afterFuncs {
			fn()
		}
		return err
	}

	// 等待优雅关闭完成
	<-s.shutdownDone

	for _, fn := range s.afterFuncs {
		fn()
	}

	return nil
}

func (s *Server) waitShutdown(quit <-chan struct{}) {
	select {
	case <-s.sigChan:
		// 收到系统信号，执行优雅关闭
	case <-quit:
		// 服务启动失败，直接退出
		close(s.shutdownDone)
		return
	}

	timeout := time.Duration(atomic.LoadInt64(&s.shutdownTimeout))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logger.Ix(ctx, "httpserver", "shutdown http server ...", "timeout", timeout.String())

	err := s.server.Shutdown(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Ex(ctx, "httpserver", "shutdown timeout, forcing close", "timeout", timeout.String())
			if closeErr := s.server.Close(); closeErr != nil {
				logger.Ex(ctx, "httpserver", "force close error", "error", closeErr.Error())
			}
		} else {
			logger.Ex(ctx, "httpserver", "shutdown http server error", "error", err.Error())
		}
	} else {
		logger.Ix(ctx, "httpserver", "shutdown http server successfully")
	}

	close(s.shutdownDone)
}

func (s *Server) HTTPServer() *http.Server {
	return s.server
}

func (s *Server) GinEngine() *gin.Engine {
	return s.engine
}

// InitGinValidatorChinese 初始化 gin 中文校验器
func (s *Server) InitGinValidatorChinese() error {
	return validator.Init()
}

func (s *Server) UseMiddleware(middleware ...gin.HandlerFunc) {
	s.engine.Use(middleware...)
}

func (s *Server) AddBeforeServerStartFunc(fns ...bootstrap.BeforeServerStartFunc) {
	for _, fn := range fns {
		s.beforeFuncs = append(s.beforeFuncs, fn)
	}
}

func (s *Server) AddAfterServerStopFunc(fns ...bootstrap.AfterServerStopFunc) {
	for _, fn := range fns {
		s.afterFuncs = append(s.afterFuncs, fn)
	}
}

// SetShutdownTimeout 设置优雅关闭的超时时间。
// 对于有长时间请求的应用（如文件上传、长轮询等），可适当调大，默认 30 秒。
// 并发安全，可在 Serve() 启动前后调用。
func (s *Server) SetShutdownTimeout(timeout time.Duration) {
	atomic.StoreInt64(&s.shutdownTimeout, int64(timeout))
}

// InitConfig 从配置文件加载配置并更新 Server 配置。
// 注意：此方法已废弃，配置现在应在 NewServer() 时完成。
// 为了向后兼容，此方法保留但不再作为 BeforeServerStartFunc 使用。
// 如果需要在运行时重新加载配置，建议创建新的 Server 实例。
func (s *Server) InitConfig(ops ...string) error {
	sec := "HttpServer"
	if len(ops) > 0 && ops[0] != "" {
		sec = ops[0]
	}

	var opts ServerOptions
	if err := config.ConfMapToStruct(sec, &opts); err != nil {
		return err
	}

	// 用默认值填充空值
	opts = fillDefaults(opts, s.opts)

	// 验证配置有效性
	if _, err := opts.GetReadHeaderTimeout(); err != nil {
		return fmt.Errorf("invalid ReadHeaderTimeout: %w", err)
	}
	if _, err := opts.GetReadTimeout(); err != nil {
		return fmt.Errorf("invalid ReadTimeout: %w", err)
	}
	if _, err := opts.GetWriteTimeout(); err != nil {
		return fmt.Errorf("invalid WriteTimeout: %w", err)
	}
	if _, err := opts.GetIdleTimeout(); err != nil {
		return fmt.Errorf("invalid IdleTimeout: %w", err)
	}
	shutdownTimeout, err := opts.GetShutdownTimeout()
	if err != nil {
		return fmt.Errorf("invalid ShutdownTimeout: %w", err)
	}

	// 应用配置（注意：mode 无法在运行时更改，因为 gin.New() 已执行）
	if opts.Mode != s.opts.Mode {
		fmt.Fprintf(os.Stderr, "[httpserver] WARNING: mode changed from %q to %q but cannot take effect after Server created\n", s.opts.Mode, opts.Mode)
	}

	s.opts = opts
	s.server.Addr = opts.Addr
	s.server.ReadHeaderTimeout, _ = opts.GetReadHeaderTimeout()
	s.server.ReadTimeout, _ = opts.GetReadTimeout()
	s.server.WriteTimeout, _ = opts.GetWriteTimeout()
	s.server.IdleTimeout, _ = opts.GetIdleTimeout()
	atomic.StoreInt64(&s.shutdownTimeout, int64(shutdownTimeout))

	return nil
}
