package rpcxclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/share"
	"github.com/wanglelecc/gokitbox/configsource"
	"github.com/wanglelecc/gokitbox/logger"
)

var (
	sAddress           []string // 注册中心地址
	username           string   // 用户名
	password           string   // 密码
	bucket             string   // bucket
	defaultBasePath    string
	group              string
	appId              string
	appKey             string
	mu                 sync.RWMutex
	rpcxOptCallTimeout time.Duration // 调用超时
	rpcxConnectTimeout time.Duration // 连接超时
	configInitialized  bool          // 配置是否成功初始化
)

// Options are supplied after the application's configuration has been loaded.
type Options struct {
	Addrs                                                      []string
	Username, Password, Bucket, BasePath, Group, AppId, AppKey string
	CallTimeout, ConnectTimeout                                time.Duration
}

func InitWithOptions(o Options) error {
	if len(o.Addrs) == 0 || o.BasePath == "" {
		return errors.New("rpcxclient: registry addresses and basePath are required")
	}
	if o.CallTimeout == 0 {
		o.CallTimeout = 30 * time.Second
	}
	if o.ConnectTimeout == 0 {
		o.ConnectTimeout = time.Second
	}
	if o.CallTimeout < 0 || o.ConnectTimeout < 0 {
		return errors.New("rpcxclient: timeouts must be positive")
	}
	mu.Lock()
	defer mu.Unlock()
	if configInitialized {
		return errors.New("rpcxclient: already initialized; recreate clients to change options")
	}
	sAddress = append([]string(nil), o.Addrs...)
	username = o.Username
	password = o.Password
	bucket = o.Bucket
	defaultBasePath = o.BasePath
	group = o.Group
	appId = o.AppId
	appKey = o.AppKey
	rpcxOptCallTimeout = o.CallTimeout
	rpcxConnectTimeout = o.ConnectTimeout
	configInitialized = true
	return nil
}

// InitFromConfig is a legacy bridge; applications should use InitWithOptions.
func InitFromConfig() error {
	callTimeout, err := time.ParseDuration(configsource.GetConfDefault("Registry", "rpcCallTimeout", "30s"))
	if err != nil {
		return errors.New("rpcxclient: invalid rpcCallTimeout")
	}
	connectTimeout, err := time.ParseDuration(configsource.GetConfDefault("Registry", "rpcConnectTimeout", "1s"))
	if err != nil {
		return errors.New("rpcxclient: invalid rpcConnectTimeout")
	}
	return InitWithOptions(Options{Addrs: configsource.GetConfArr("Registry", "addrs"), Username: strings.TrimSpace(configsource.GetConf("Registry", "username")), Password: configsource.GetConf("Registry", "password"), Bucket: configsource.GetConf("Registry", "bucket"), BasePath: configsource.GetConf("Registry", "basePath"), Group: configsource.GetConf("Registry", "group"), AppId: configsource.GetConf("RpcxAuth", "appId"), AppKey: configsource.GetConf("RpcxAuth", "appKey"), CallTimeout: callTimeout, ConnectTimeout: connectTimeout})
}

var legacyConfigMu sync.Mutex

func ensureConfig() {
	legacyConfigMu.Lock()
	defer legacyConfigMu.Unlock()
	if !IsConfigInitialized() {
		if err := InitFromConfig(); err != nil {
			logger.Wx(context.Background(), "rpcxclient.config", "configuration not ready")
		}
	}
}

// safeGetConfArr 安全地获取配置数组
func safeGetConfArr(section, key string) []string {
	defer func() {
		if r := recover(); r != nil {
			logger.Wx(context.Background(), "rpcxclient.safeGetConfArr", "获取配置失败", "section", section, "key", key, "panic", fmt.Sprintf("%v", r))
		}
	}()
	return configsource.GetConfArr(section, key)
}

// safeGetConfStringMap 安全地获取配置 map
func safeGetConfStringMap(section string) map[string]string {
	defer func() {
		if r := recover(); r != nil {
			logger.Wx(context.Background(), "rpcxclient.safeGetConfStringMap", "获取配置失败", "section", section, "panic", fmt.Sprintf("%v", r))
		}
	}()
	return configsource.GetConfStringMap(section)
}

// IsConfigInitialized 检查配置是否成功初始化
func IsConfigInitialized() bool {
	mu.RLock()
	defer mu.RUnlock()
	return configInitialized
}

func GetSdAddress() []string {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	// 返回副本而非引用，防止外部修改
	result := make([]string, len(sAddress))
	copy(result, sAddress)
	return result
}

func GetUsername() string {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	return username
}

func GetPassword() string {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	return password
}

func GetBucket() string {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	return bucket
}

func GetServiceBasePath() string {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	return defaultBasePath
}

func GetFailMode() client.FailMode {
	return client.Failover
}

func GetSelectMode() client.SelectMode {
	return client.WeightedRoundRobin
}

func GetClientOption() client.Option {
	ensureConfig()
	mu.RLock()
	connectTimeout := rpcxConnectTimeout
	clientGroup := group
	mu.RUnlock()

	option := client.Option{
		Retries:           2,
		RPCPath:           share.DefaultRPCPath,
		ConnectTimeout:    connectTimeout,
		SerializeType:     protocol.MsgPack,
		CompressType:      protocol.None,
		BackupLatency:     10 * time.Millisecond,
		Heartbeat:         true,
		HeartbeatInterval: 1 * time.Second,
		Group:             clientGroup,
	}

	// if failed 5 times, return error immediately, and will try to connect after 10 seconds
	option.GenBreaker = func() client.Breaker {
		return client.NewConsecCircuitBreaker(5, 10*time.Second)
	}
	return option
}

// getClientDiscovery 获取服务发现实例
// 注意：当前实现每次都创建新的 discovery，不缓存
func getClientDiscovery(basePath, servicePath string) (discovery client.ServiceDiscovery) {
	ensureConfig()

	ctx := context.Background()
	tag := "rpcxclient.getClientDiscovery"

	if basePath == "" {
		mu.RLock()
		basePath = defaultBasePath
		mu.RUnlock()
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Ex(ctx, tag, "初始化服务发现对象失败", "error", fmt.Sprintf("%v", r))
			discovery = nil
		}
	}()

	var err error
	discovery, err = initClientDiscovery(basePath)
	if err != nil {
		logger.Ex(ctx, tag, "初始化服务发现对象失败", "error", err.Error(), "basePath", basePath)
		return nil
	}

	if discovery == nil {
		logger.Ex(ctx, tag, "初始化服务发现对象失败: discovery is nil", "basePath", basePath)
		return nil
	}

	return discovery
}

// GetAppId 获取 AppId
func GetAppId() string {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	return appId
}

// GetAppKey 获取 AppKey（谨慎使用，仅在需要生成签名时使用）
func GetAppKey() string {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	return appKey
}

// ValidateConfig 检查客户端配置是否完整
func ValidateConfig() error {
	if !IsConfigInitialized() {
		return errors.New("rpcxclient config not initialized")
	}

	mu.RLock()
	defer mu.RUnlock()

	if len(sAddress) == 0 {
		return errors.New("registry address not configured")
	}

	if defaultBasePath == "" {
		return errors.New("registry basePath not configured")
	}

	return nil
}

// GetCallTimeout 获取调用超时时间
func GetCallTimeout() time.Duration {
	ensureConfig()
	mu.RLock()
	defer mu.RUnlock()
	return rpcxOptCallTimeout
}
