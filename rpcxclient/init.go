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

	"github.com/wanglelecc/gokitbox/config"
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

func init() {
	var initErr error

	// 安全地读取配置
	defer func() {
		if r := recover(); r != nil {
			logger.Ex(context.Background(), "rpcxclient.init", "配置初始化失败", "panic", fmt.Sprintf("%v", r))
			configInitialized = false
		}
	}()

	sAddress = safeGetConfArr("Registry", "addrs")

	// 读取超时配置
	if callTimeout := config.GetConf("Registry", "rpcCallTimeout"); callTimeout != "" {
		if d, err := time.ParseDuration(callTimeout); err == nil {
			rpcxOptCallTimeout = d
		} else {
			logger.Wx(context.Background(), "rpcxclient.init", "rpcCallTimeout 解析失败，使用默认值", "error", err.Error())
			rpcxOptCallTimeout = 30 * time.Second // 默认 30 秒
		}
	} else {
		rpcxOptCallTimeout = 30 * time.Second // 默认 30 秒
	}

	rpcxConnectTimeout = time.Second // default 1s
	if connectTimeout := config.GetConf("Registry", "rpcConnectTimeout"); connectTimeout != "" {
		if d, err := time.ParseDuration(connectTimeout); err == nil {
			rpcxConnectTimeout = d
		} else {
			logger.Wx(context.Background(), "rpcxclient.init", "rpcConnectTimeout 解析失败，使用默认值", "error", err.Error())
		}
	}

	defaultBasePath = config.GetConf("Registry", "basePath")
	group = config.GetConf("Registry", "group")

	username = strings.TrimSpace(config.GetConf("Registry", "username"))
	password = strings.TrimSpace(config.GetConf("Registry", "password"))
	bucket = strings.TrimSpace(config.GetConf("Registry", "bucket"))

	// 安全读取鉴权配置
	rpcAuth := safeGetConfStringMap("RpcxAuth")
	if rpcAuth != nil {
		appId = rpcAuth["appId"]
		appKey = rpcAuth["appKey"]
	}

	configInitialized = true

	if initErr != nil {
		logger.Ex(context.Background(), "rpcxclient.init", "配置初始化出错", "error", initErr.Error())
	}
}

// safeGetConfArr 安全地获取配置数组
func safeGetConfArr(section, key string) []string {
	defer func() {
		if r := recover(); r != nil {
			logger.Wx(context.Background(), "rpcxclient.safeGetConfArr", "获取配置失败", "section", section, "key", key, "panic", fmt.Sprintf("%v", r))
		}
	}()
	return config.GetConfArr(section, key)
}

// safeGetConfStringMap 安全地获取配置 map
func safeGetConfStringMap(section string) map[string]string {
	defer func() {
		if r := recover(); r != nil {
			logger.Wx(context.Background(), "rpcxclient.safeGetConfStringMap", "获取配置失败", "section", section, "panic", fmt.Sprintf("%v", r))
		}
	}()
	return config.GetConfStringMap(section)
}

// IsConfigInitialized 检查配置是否成功初始化
func IsConfigInitialized() bool {
	return configInitialized
}

func GetSdAddress() []string {
	mu.RLock()
	defer mu.RUnlock()
	// 返回副本而非引用，防止外部修改
	result := make([]string, len(sAddress))
	copy(result, sAddress)
	return result
}

func GetUsername() string {
	mu.RLock()
	defer mu.RUnlock()
	return username
}

func GetPassword() string {
	mu.RLock()
	defer mu.RUnlock()
	return password
}

func GetBucket() string {
	mu.RLock()
	defer mu.RUnlock()
	return bucket
}

func GetServiceBasePath() string {
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
	mu.RLock()
	connectTimeout := rpcxConnectTimeout
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
		Group:             group,
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
	mu.Lock()
	defer mu.Unlock()

	ctx := context.Background()
	tag := "rpcxclient.getClientDiscovery"

	if basePath == "" {
		basePath = defaultBasePath
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
	mu.RLock()
	defer mu.RUnlock()
	return appId
}

// GetAppKey 获取 AppKey（谨慎使用，仅在需要生成签名时使用）
func GetAppKey() string {
	mu.RLock()
	defer mu.RUnlock()
	return appKey
}

// ValidateConfig 检查客户端配置是否完整
func ValidateConfig() error {
	if !configInitialized {
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
	mu.RLock()
	defer mu.RUnlock()
	return rpcxOptCallTimeout
}
