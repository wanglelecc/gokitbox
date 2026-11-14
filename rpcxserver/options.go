package rpcxserver

import (
	"time"
)

const (
	// StatusOn 启用
	StatusOn = "on"
	// StatusOff 关闭
	StatusOff = "off"

	// 默认超时配置
	DefaultReadTimeout  = 30 * time.Second
	DefaultWriteTimeout = 30 * time.Second
)

// RegistryOptions 服务注册中心配置
type RegistryOptions struct {
	Status         string        `ini:"status" yaml:"status"`
	Addrs          []string      `ini:"addrs" yaml:"addrs"`
	BasePath       string        `ini:"basePath" yaml:"basePath"`
	UpdateInterval time.Duration `ini:"updateInterval" yaml:"updateInterval"`
	UserName       string        `ini:"username" yaml:"username"`
	Password       string        `ini:"password" yaml:"password"`
	Bucket         string        `ini:"bucket" yaml:"bucket"`
	Group          string        `ini:"group" yaml:"group"`
}

type OptionFunc func(*Options)

// Options server options
type Options struct {
	Network      string        `ini:"network" yaml:"network"`
	Addr         string        `ini:"addr" yaml:"addr"`
	Port         string        `ini:"port" yaml:"port"`
	WriteTimeout time.Duration `ini:"writeTimeout" yaml:"writeTimeout"`
	ReadTimeout  time.Duration `ini:"readTimeout" yaml:"readTimeout"`
	RegistryOpts RegistryOptions
}

// DefaultOptions default config
func DefaultOptions() Options {
	return Options{
		Network:      "tcp",
		Addr:         "127.0.0.1",
		Port:         "18900",
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
	}
}

func Network(n string) OptionFunc {
	return func(o *Options) {
		o.Network = n
	}
}

func Addr(a string) OptionFunc {
	return func(o *Options) {
		o.Addr = a
	}
}

func Port(p string) OptionFunc {
	return func(o *Options) {
		o.Port = p
	}
}

func ReadTimeout(t time.Duration) OptionFunc {
	return func(o *Options) {
		o.ReadTimeout = t
	}
}

func WriteTimeout(t time.Duration) OptionFunc {
	return func(o *Options) {
		o.WriteTimeout = t
	}
}

func WithRegistryOptions(registryOpts RegistryOptions) OptionFunc {
	return func(o *Options) {
		o.RegistryOpts = registryOpts
	}
}
