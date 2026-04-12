//go:build etcd
// +build etcd

package rpcxserver

import (
	"github.com/rpcxio/rpcx-etcd/serverplugin"
)

func AddRegistryPlugin(s *Server) error {
	plugin := &serverplugin.EtcdV3RegisterPlugin{
		ServiceAddress: s.Opts.Network + "@" + s.Opts.Addr + ":" + s.Opts.Port,
		EtcdServers:    s.Opts.RegistryOpts.Addrs,
		BasePath:       s.Opts.RegistryOpts.BasePath,
		UpdateInterval: s.Opts.RegistryOpts.UpdateInterval,
	}

	err := plugin.Start()

	if err != nil {
		return err
	}

	s.server.Plugins.Add(plugin)

	return nil
}
