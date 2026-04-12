//go:build consul
// +build consul

package rpcxserver

import (
	"strings"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/rpcxio/libkv/store"
	"github.com/rpcxio/rpcx-consul/serverplugin"
)

func AddRegistryPlugin(s *Server) error {
	plugin := &serverplugin.ConsulRegisterPlugin{
		ServiceAddress: s.Opts.Network + "@" + s.Opts.Addr + ":" + s.Opts.Port,
		ConsulServers:  s.Opts.RegistryOpts.Addrs,
		BasePath:       s.Opts.RegistryOpts.BasePath,
		Metrics:        metrics.NewRegistry(),
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
