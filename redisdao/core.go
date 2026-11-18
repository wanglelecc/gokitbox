package redisdao

import (
	"crypto/tls"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wanglelecc/gokitbox/configsource"
)

var initMu sync.Mutex
var initialized, stopped bool

var instance = struct {
	sync.RWMutex
	redisInstances map[string]*redis.Client
}{redisInstances: make(map[string]*redis.Client, 0)}

func Init() {
	if err := Initialize(); err != nil {
		panic(err)
	}
}
func Initialize() error {
	initMu.Lock()
	defer initMu.Unlock()
	if stopped {
		return fmt.Errorf("redisdao: closed")
	}
	if initialized {
		return nil
	}
	optionsMu.Lock()
	started = true
	opts := configuredOptions
	optionsMu.Unlock()
	if opts == nil {
		o := legacyOptions()
		opts = &o
	}
	if err := opts.Validate(); err != nil {
		return err
	}
	clients := map[string]*redis.Client{}
	for name, option := range opts.Instances {
		copyOptions := option
		clients[name] = redis.NewClient(&copyOptions)
	}
	instance.Lock()
	instance.redisInstances = clients
	instance.Unlock()
	initialized = true
	return nil
}

func legacyOptions() Options {
	result := Options{Instances: map[string]redis.Options{}}

	confMapList := configsource.GetConfArrayMap("Redis")
	for k, v := range confMapList {
		options := redis.Options{
			Username:     "",
			Password:     "",
			DB:           0,
			PoolSize:     100,
			MinIdleConns: 50,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			MaxRetries:   0,
		}

		if p := configsource.GetConf("RedisConfig", k+".db"); p != "" {
			if rt, err := strconv.Atoi(p); err == nil {
				options.DB = rt
			} else {
				log.Printf("db strconv.Atoi(%+v) error:%v", p, err)
			}
		}

		// 连接池最大连接数量，注意：这里不包括 pub/sub，pub/sub 将使用独立的网络连接
		// 默认为 10 * runtime.GOMAXPROCS
		if p := configsource.GetConf("RedisConfig", k+".poolsize"); p != "" {
			if rt, err := strconv.Atoi(p); err == nil {
				options.PoolSize = rt
			} else {
				log.Printf("poolsize strconv.Atoi(%+v) error:%v", p, err)
			}
		}

		// 连接池保持的最小空闲连接数，它受到PoolSize的限制
		// 默认为0，不保持
		if p := configsource.GetConf("RedisConfig", k+".minidleconns"); p != "" {
			if rt, err := strconv.Atoi(p); err == nil {
				options.MinIdleConns = rt
			} else {
				log.Printf("minidleconns strconv.Atoi(%+v) error:%v", p, err)
			}
		}

		// 从网络连接中读取数据超时时间，可能的值：
		//  0 - 默认值，3秒
		// -1 - 无超时，无限期的阻塞
		// -2 - 不进行超时设置，不调用 SetReadDeadline 方法
		if p := configsource.GetConf("RedisConfig", k+".readtimeout"); p != "" {
			if rt, err := strconv.Atoi(p); err == nil {
				options.ReadTimeout = time.Second * time.Duration(rt)
			} else {
				log.Printf("readtimeout strconv.Atoi(%+v) error:%v", p, err)
			}
		}

		// 把数据写入网络连接的超时时间，可能的值：
		//  0 - 默认值，3秒
		// -1 - 无超时，无限期的阻塞
		// -2 - 不进行超时设置，不调用 SetWriteDeadline 方法
		if p := configsource.GetConf("RedisConfig", k+".writetimeout"); p != "" {
			if rt, err := strconv.Atoi(p); err == nil {
				options.WriteTimeout = time.Second * time.Duration(rt)
			} else {
				log.Printf("writetimeout strconv.Atoi(%+v) error:%v", p, err)
			}
		}

		// 当redis服务器版本在6.0以上时，作为ACL认证信息配合密码一起使用，
		// ACL是redis 6.0以上版本提供的认证功能，6.0以下版本仅支持密码认证。
		// 默认为空，不进行认证。
		if p := configsource.GetConf("RedisConfig", k+".username"); p != "" {
			options.Username = p
		}

		// 当redis服务器版本在6.0以上时，作为ACL认证信息配合密码一起使用，
		// 当redis服务器版本在6.0以下时，仅作为密码认证。
		// ACL是redis 6.0以上版本提供的认证功能，6.0以下版本仅支持密码认证。
		// 默认为空，不进行认证。
		if p := configsource.GetConf("RedisConfig", k+".password"); p != "" {
			options.Password = p
		}

		if p := configsource.GetConf("RedisConfig", k+".tlsinsecureskip"); p == "true" {
			options.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		}

		if p := configsource.GetConf("RedisConfig", k+".maxretries"); p != "" {
			if rt, err := strconv.Atoi(p); err == nil {
				options.MaxRetries = rt
			} else {
				log.Printf("maxretries strconv.Atoi(%+v) error:%v", p, err)
			}
		}

		for _, s := range v {
			options.Addr = s
			result.Instances[k] = options
		}
	}
	return result
}

func getInstance(server string) *redis.Client {
	instance.RLock()
	ins, ok := instance.redisInstances[server]
	instance.RUnlock()
	if ok && ins != nil {
		return ins
	}

	return nil
}

func setInstance(server string, client *redis.Client) {
	instance.Lock()
	instance.redisInstances[server] = client
	instance.Unlock()
}

func closeInstance() {
	instance.Lock()
	defer instance.Unlock()

	for k, c := range instance.redisInstances {
		c.Close()
		delete(instance.redisInstances, k)
	}
}
