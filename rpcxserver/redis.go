//go:build redis
// +build redis

package rpcxserver

import (
	"strings"

	"github.com/rpcxio/libkv/store"
	"github.com/rpcxio/rpcx-redis/serverplugin"
)

func AddRegistryPlugin(s *Server) error {
	plugin := &serverplugin.RedisRegisterPlugin{
		ServiceAddress: s.Opts.Network + "@" + s.Opts.Addr + ":" + s.Opts.Port,
		RedisServers:   s.Opts.RegistryOpts.Addrs,
		BasePath:       s.Opts.RegistryOpts.BasePath,
		UpdateInterval: s.Opts.RegistryOpts.UpdateInterval,
		Options:        &store.Config{},
	}

	if s.Opts.RegistryOpts.UserName != "" {
		plugin.Options.Username = strings.TrimSpace(s.Opts.RegistryOpts.UserName)
	}

	if s.Opts.RegistryOpts.Password != "" {
		plugin.Options.Password = strings.TrimSpace(s.Opts.RegistryOpts.Password)
	}

	if s.Opts.RegistryOpts.Bucket != "" {
		plugin.Options.Bucket = strings.TrimSpace(s.Opts.RegistryOpts.Bucket)
	}

	err := plugin.Start()

	if err != nil {
		return err
	}

	s.server.Plugins.Add(plugin)

	return nil
}
