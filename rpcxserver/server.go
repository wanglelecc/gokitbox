package rpcxserver

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wanglelecc/gokitbox/bootstrap"
	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/logger"

	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/server"
	"github.com/spf13/cast"
)

// isShutdownError 判断是否为关闭相关的错误
func isShutdownError(err error) bool {
	if err == nil {
		return false
	}
	if err == server.ErrServerClosed {
		return true
	}
	// 使用 strings.Contains 更健壮地检查错误信息
	errStr := err.Error()
	return strings.Contains(errStr, "server closed") || strings.Contains(errStr, "mux: server closed")
}

// Server struct
type Server struct {
	server       *server.Server
	Opts         Options
	beforeFuncs  []bootstrap.BeforeServerStartFunc
	afterFuncs   []bootstrap.AfterServerStopFunc
	sigChan      chan os.Signal // 统一信号通道
	shutdownOnce sync.Once      // 确保只关闭一次

	// 平滑关闭配置（不从配置文件读取，使用默认值或通过方法设置）
	preUnregisterWait  time.Duration // 服务注销前等待，默认2秒
	postUnregisterWait time.Duration // 服务注销后等待，默认3秒
	shutdownTimeout    time.Duration // 关闭超时，默认30秒
	forceShutdown      bool          // 是否强制退出，默认true

	// nonce 相关 - 每个 Server 实例独立，避免全局竞态和 goroutine 泄漏
	usedNonces       map[string]time.Time // 已使用的 nonce 集合
	nonceMutex       sync.RWMutex         // nonce 操作锁
	nonceCleanupStop chan struct{}        // 停止 nonce 清理协程
	nonceCleanupOnce sync.Once            // 确保只启动一次清理协程
	nonceStopOnce    sync.Once            // 确保只停止一次清理协程

	// 鉴权初始化标记
	authInitialized sync.Once // 确保只初始化一次鉴权

	// 鉴权数据 - 每实例独立，避免多实例互相覆盖
	validAuth         map[string]string
	accessFn          ValidAccess
	authMu            sync.RWMutex
	timestampDuration time.Duration
	durationOnce      sync.Once
}

// NewServer get server instance
func NewServer(options ...OptionFunc) *Server {
	opts := DefaultOptions()

	for _, o := range options {
		o(&opts)
	}

	srv := &Server{
		Opts: opts,
		// 设置平滑关闭默认值
		preUnregisterWait:  2 * time.Second,
		postUnregisterWait: 3 * time.Second,
		shutdownTimeout:    30 * time.Second,
		forceShutdown:      true,
		// 初始化 nonce 集合
		usedNonces: make(map[string]time.Time),
		// 初始化鉴权数据
		validAuth: make(map[string]string),
	}
	srv.sigChan = make(chan os.Signal, 10)
	return srv
}

// NewServerWithOptions with options
func NewServerWithOptions(opts Options) *Server {
	srv := &Server{
		Opts: opts,
		// 设置平滑关闭默认值
		preUnregisterWait:  2 * time.Second,
		postUnregisterWait: 3 * time.Second,
		shutdownTimeout:    30 * time.Second,
		forceShutdown:      true,
		// 初始化 nonce 集合
		usedNonces: make(map[string]time.Time),
		// 初始化鉴权数据
		validAuth: make(map[string]string),
	}
	srv.sigChan = make(chan os.Signal, 10)
	return srv
}

// ConfigureOptions 更新配置
func (srv *Server) ConfigureOptions(options ...OptionFunc) {
	for _, o := range options {
		o(&srv.Opts)
	}
}

// SetShutdownTimeout 设置关闭超时时间（默认30秒）
// 用于等待现有请求处理完成的最长时间，超时后强制关闭
func (srv *Server) SetShutdownTimeout(timeout time.Duration) *Server {
	srv.shutdownTimeout = timeout
	return srv
}

// SetPreUnregisterWait 设置服务注销前等待时间（默认2秒）
// 用于防止滚动更新时出现短暂无可用节点
func (srv *Server) SetPreUnregisterWait(wait time.Duration) *Server {
	srv.preUnregisterWait = wait
	return srv
}

// SetPostUnregisterWait 设置服务注销后等待时间（默认3秒）
// 用于等待注册中心同步，让客户端感知节点下线
func (srv *Server) SetPostUnregisterWait(wait time.Duration) *Server {
	srv.postUnregisterWait = wait
	return srv
}

// SetForceShutdown 设置是否启用强制关闭（默认true）
// 当收到第二次信号时，是否立即强制退出
func (srv *Server) SetForceShutdown(force bool) *Server {
	srv.forceShutdown = force
	return srv
}

// SetShutdownWaits 批量设置等待时间（便捷方法）
func (srv *Server) SetShutdownWaits(preWait, postWait, timeout time.Duration) *Server {
	srv.preUnregisterWait = preWait
	srv.postUnregisterWait = postWait
	srv.shutdownTimeout = timeout
	return srv
}

// Start 初始化各种插件
func (srv *Server) Serve() error {
	tag := "rpcxServe"
	ctx := context.Background()

	// init rpc server
	srv.server = server.NewServer()

	// before func
	for _, fn := range srv.beforeFuncs {
		err := fn()
		if err != nil {
			return err
		}
	}

	// 统一信号处理
	signal.Notify(srv.sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go srv.handleSignals()

	logger.Ix(ctx, tag, fmt.Sprintf("server start listen on %s@%s:%s", srv.Opts.Network, srv.Opts.Addr, srv.Opts.Port))

	server.WithReadTimeout(srv.Opts.ReadTimeout)(srv.server)
	server.WithWriteTimeout(srv.Opts.WriteTimeout)(srv.server)
	err := srv.server.Serve(srv.Opts.Network, srv.Opts.Addr+":"+srv.Opts.Port)

	if err != nil && !isShutdownError(err) {
		logger.Ex(ctx, tag, "server error", "error", err.Error())
	} else {
		err = nil
	}

	// 等待关闭完成后执行清理函数
	for _, fn := range srv.afterFuncs {
		fn()
	}

	return err
}

// handleSignals 统一信号处理，支持第一次信号优雅关闭，第二次信号强制退出
func (srv *Server) handleSignals() {
	tag := "handleSignals"
	signalCount := 0

	for sig := range srv.sigChan {
		signalCount++
		logger.Ix(context.Background(), tag, "received signal", "signal", sig.String(), "count", signalCount)

		if signalCount == 1 {
			// 第一次信号：触发优雅关闭
			go srv.gracefulShutdown()
		} else {
			// 第二次及以后：强制退出
			logger.Wx(context.Background(), tag, "received second signal, forcing shutdown immediately", "signal", sig.String())
			if srv.forceShutdown {
				os.Exit(1)
			}
		}
	}
}

// gracefulShutdown 执行优雅关闭流程
func (srv *Server) gracefulShutdown() {
	tag := "gracefulShutdown"
	ctx := context.Background()

	// 确保只执行一次关闭流程
	srv.shutdownOnce.Do(func() {
		logger.Ix(ctx, tag, "starting graceful shutdown...")

		// 1. 延迟等待，防止新服务未启动导致无可用节点
		if srv.preUnregisterWait > 0 {
			logger.Ix(ctx, tag, "waiting before unregister", "duration", srv.preUnregisterWait)
			time.Sleep(srv.preUnregisterWait)
		}

		// 2. 从注册中心注销服务
		if srv.server != nil {
			logger.Ix(ctx, tag, "unregistering from service registry...")
			srv.server.UnregisterAll()
			logger.Ix(ctx, tag, "unregistered from service registry successfully")
		}

		// 3. 等待注册中心同步，让客户端感知节点下线
		if srv.postUnregisterWait > 0 {
			logger.Ix(ctx, tag, "waiting for registry sync", "duration", srv.postUnregisterWait)
			time.Sleep(srv.postUnregisterWait)
		}

		// 4. 关闭服务，等待现有连接处理完成
		if srv.server != nil {
			shutdownCtx := context.Background()
			var cancel context.CancelFunc

			// 设置关闭超时
			if srv.shutdownTimeout > 0 {
				shutdownCtx, cancel = context.WithTimeout(context.Background(), srv.shutdownTimeout)
				defer cancel()
				logger.Ix(ctx, tag, "shutting down server with timeout", "timeout", srv.shutdownTimeout)
			} else {
				logger.Ix(ctx, tag, "shutting down server without timeout")
			}

			// 执行关闭
			err := srv.server.Shutdown(shutdownCtx)
			if err != nil {
				if err == context.DeadlineExceeded {
					logger.Wx(ctx, tag, "server shutdown timeout, some connections may be forcefully closed")
				} else {
					logger.Ex(ctx, tag, "server shutdown error", "error", err.Error())
				}
			} else {
				logger.Ix(ctx, tag, "server shutdown successfully")
			}
		}

		// 5. 停止 nonce 清理协程
		srv.stopNonceCleanup()

		// 6. 停止信号监听（注意：gracefulShutdown 只执行一次，所以不会关闭两次）
		signal.Stop(srv.sigChan)
		close(srv.sigChan)

		logger.Ix(ctx, tag, "graceful shutdown completed")
	})
}

// AddBeforeServerStartFunc add before function
func (srv *Server) AddBeforeServerStartFunc(fns ...bootstrap.BeforeServerStartFunc) {
	for _, fn := range fns {
		srv.beforeFuncs = append(srv.beforeFuncs, fn)
	}
}

// AddAfterServerStopFunc add after function
func (srv *Server) AddAfterServerStopFunc(fns ...bootstrap.AfterServerStopFunc) {
	for _, fn := range fns {
		srv.afterFuncs = append(srv.afterFuncs, fn)
	}
}

// RegisterServiceWithName 用于注册自己的服务
func (srv *Server) RegisterServiceWithName(name string, recv interface{}, metadata string) bootstrap.BeforeServerStartFunc {
	return func() error {
		return srv.server.RegisterName(name, recv, metadata)
	}
}

// RegisterServiceWithPlugin 用于注册带插件功能的服务
func (srv *Server) RegisterServiceWithPlugin(name string, recv interface{}, metadata string) bootstrap.BeforeServerStartFunc {
	return func() error {
		if srv.server == nil {
			return errors.New("server not initialized, please call Serve() first or register in BeforeServerStartFunc")
		}

		if metadata == "" {
			metadata = "group=" + srv.Opts.RegistryOpts.Group
		}

		if srv.server.Plugins != nil {
			srv.server.Plugins.DoUnregister(name)
		}

		return srv.server.RegisterName(name, recv, metadata)
	}
}

// DisableHTTPGateway 禁用本地网关模式
func (srv *Server) DisableHTTPGateway() bootstrap.BeforeServerStartFunc {
	return func() error {
		srv.server.DisableHTTPGateway = true
		return nil
	}
}

// AddPlugins 添加rpcx plugin
func (srv *Server) AddPlugins(plugins ...server.Plugin) {
	for _, plugin := range plugins {
		srv.server.Plugins.Add(plugin)
	}
}

// InitConfig 初始化配置
func (srv *Server) InitConfig(ops ...string) bootstrap.BeforeServerStartFunc {
	return func() error {
		sec := "RpcServer"

		if len(ops) > 0 && ops[0] != "" {
			sec = ops[0]
		}

		err := config.ConfMapToStruct(sec, &srv.Opts)
		return err
	}
}

// InitRegistry 初始化注册中心
func (srv *Server) InitRegistry() bootstrap.BeforeServerStartFunc {
	return func() error {
		regOpts := RegistryOptions{}
		err := config.ConfMapToStruct("Registry", &regOpts)

		if err != nil {
			return err
		}
		// 如果不启用注册中心，则不初始化
		if regOpts.Status != StatusOn {
			return nil
		}

		if len(regOpts.Addrs) == 0 {
			return errors.New("can not found registry config")
		}

		srv.ConfigureOptions(WithRegistryOptions(regOpts))

		return AddRegistryPlugin(srv)
	}
}

// RegisterPlugin 预留的插件注册接口（暂未实现）
func (srv *Server) RegisterPlugin() bootstrap.BeforeServerStartFunc {
	return func() error {
		// TODO: 实现具体的插件注册逻辑
		return nil
	}
}

// Server 获取rpcx server
func (srv *Server) Server() *server.Server {
	return srv.server
}

// Shutdown 主动触发平滑关闭（用于程序主动关闭场景）
func (srv *Server) Shutdown() error {
	srv.gracefulShutdown()
	return nil
}

// Close 关闭资源（实现 io.Closer 接口）
func (srv *Server) Close() error {
	return srv.Shutdown()
}

// ========== 鉴权相关 ==========

type ValidAccess func(string) (string, bool)

func (srv *Server) validAuthAccess(appId string) (string, bool) {
	srv.authMu.RLock()
	appKey, ok := srv.validAuth[appId]
	srv.authMu.RUnlock()
	return appKey, ok
}

// InitRpcxAuth 初始化rpcx鉴权
func (srv *Server) InitRpcxAuth(fns ...ValidAccess) bootstrap.BeforeServerStartFunc {
	return func() error {
		var initErr error
		srv.authInitialized.Do(func() {
			srv.authMu.Lock()
			srv.validAuth = config.GetConfStringMap("ValidRpcxAuth")
			srv.server.AuthFunc = func(ctx context.Context, req *protocol.Message, token string) error {
				return srv.auth(ctx, req, token)
			}
			if len(fns) > 0 {
				srv.accessFn = fns[0]
			} else {
				srv.accessFn = srv.validAuthAccess
			}
			srv.authMu.Unlock()

			// 启动 nonce 清理协程
			srv.startNonceCleanup()
		})
		return initErr
	}
}

// auth 执行 RPC 鉴权
// 支持两种模式:
// 1. 新安全模式（推荐）: 需要 X-Auth-AppId, X-Auth-Timestamp, X-Auth-Nonce, X-Auth-Sign
// 2. 兼容模式: 使用 token 参数（已废弃，保留向后兼容）
func (srv *Server) auth(ctx context.Context, req *protocol.Message, token string) error {
	// 优先从 Metadata 获取 appId
	appId, ok := req.Metadata["X-Auth-AppId"]
	if !ok {
		// 向后兼容：使用 token 参数
		if token == "" {
			return errors.New("missing X-Auth-AppId")
		}
		appId = token
	}

	srv.authMu.RLock()
	fn := srv.accessFn
	srv.authMu.RUnlock()

	key, ok := fn(appId)
	if !ok {
		return errors.New("invalid appId")
	}

	timestamp, ok := req.Metadata["X-Auth-Timestamp"]
	if !ok {
		// 向后兼容：尝试旧的时间戳 key
		timestamp, ok = req.Metadata["X-Auth-TimeStamp"]
		if !ok {
			return errors.New("invalid timestamp")
		}
	}

	if !srv.checkTimestamp(cast.ToInt64(timestamp)) {
		return errors.New("expired timestamp")
	}

	// 获取 nonce（新安全模式必需）
	nonce, hasNonce := req.Metadata["X-Auth-Nonce"]

	sign, ok := req.Metadata["X-Auth-Sign"]
	if !ok {
		return errors.New("invalid sign")
	}

	// 根据是否有 nonce 选择校验方式
	if hasNonce {
		// 新安全模式: HMAC-SHA256 + nonce
		if !srv.checkNonce(appId, key, timestamp, nonce, sign) {
			return errors.New("check sign fail")
		}
	} else {
		// 向后兼容模式: MD5（不推荐）
		logger.Wx(ctx, "auth", "using deprecated MD5 auth", "appId", appId)
		if !checkLegacy(appId, key, timestamp, sign) {
			return errors.New("check sign fail")
		}
	}
	return nil
}

// checkTimestamp 检查时间戳是否在有效范围内
func (srv *Server) checkTimestamp(ts int64) bool {
	srv.durationOnce.Do(func() {
		srv.authMu.RLock()
		durationStr, ok := srv.validAuth["timestampValidate"]
		srv.authMu.RUnlock()

		if ok {
			duration, err := time.ParseDuration(durationStr)
			if err == nil {
				srv.timestampDuration = duration
				return
			}
		}
		// 默认 2 分钟，比原来的 5 分钟更严格
		srv.timestampDuration = 2 * time.Minute
	})

	if srv.timestampDuration.Seconds() == 0 {
		return true
	}

	now := time.Now().Unix()
	// 使用秒级差值计算，避免 time.Add 的溢出问题
	durationSec := int64(srv.timestampDuration.Seconds())
	start := now - durationSec
	end := now + durationSec

	if ts < start || ts > end {
		return false
	}
	return true
}

// checkLegacy 兼容旧的 MD5 鉴权（已废弃，仅用于向后兼容）
func checkLegacy(appId, key, timestamp, sign string) bool {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(appId + "&" + timestamp + key))
	cipherStr := md5Ctx.Sum(nil)
	signstr := hex.EncodeToString(cipherStr)
	return signstr == sign
}

// checkNonce 使用 HMAC-SHA256 进行签名验证，包含 nonce 防重放
func (srv *Server) checkNonce(appId, key, timestamp, nonce, sign string) bool {
	// 使用写锁保护，确保 check 和 mark 是原子操作，防止并发重放
	srv.nonceMutex.Lock()
	defer srv.nonceMutex.Unlock()

	// 1. 校验 nonce 是否已使用（防重放）
	_, exists := srv.usedNonces[nonce]
	if exists {
		return false
	}

	// 2. 先验证签名，签名正确才标记 nonce
	// 签名格式: HMAC-SHA256(appId + "|" + timestamp + "|" + nonce, key)
	message := appId + "|" + timestamp + "|" + nonce
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	expectedSign := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sign), []byte(expectedSign)) {
		return false
	}

	// 3. 签名验证通过，标记 nonce 为已使用
	nonceTTL := 10 * time.Minute
	srv.usedNonces[nonce] = time.Now().Add(nonceTTL)
	return true
}

// startNonceCleanup 启动 nonce 清理协程（每个 Server 实例只启动一次）
func (srv *Server) startNonceCleanup() {
	srv.nonceCleanupOnce.Do(func() {
		if srv.nonceCleanupStop != nil {
			return // 已经启动过
		}
		srv.nonceCleanupStop = make(chan struct{})
		go srv.cleanupNonces()
	})
}

// cleanupNonces 定期清理过期的 nonce
func (srv *Server) cleanupNonces() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			srv.nonceMutex.Lock()
			now := time.Now()
			for nonce, expireTime := range srv.usedNonces {
				if now.After(expireTime) {
					delete(srv.usedNonces, nonce)
				}
			}
			srv.nonceMutex.Unlock()
		case <-srv.nonceCleanupStop:
			return
		}
	}
}

// stopNonceCleanup 停止 nonce 清理协程
func (srv *Server) stopNonceCleanup() {
	srv.nonceStopOnce.Do(func() {
		if srv.nonceCleanupStop != nil {
			close(srv.nonceCleanupStop)
			srv.nonceCleanupStop = nil
		}
	})
}
