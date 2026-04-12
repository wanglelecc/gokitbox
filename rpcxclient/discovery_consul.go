//go:build consul
// +build consul

package rpcxclient

import (
	"github.com/rpcxio/libkv/store"
	cClient "github.com/rpcxio/rpcx-consul/client"
	"github.com/smallnest/rpcx/client"
)

func initClientDiscovery(basePath string) (client.ServiceDiscovery, error) {

	o := &store.Config{}

	if GetUsername() != "" {
		o.Username = GetUsername()
	}

	if GetPassword() != "" {
		o.Password = GetPassword()
	}

	if GetBucket() != "" {
		o.Bucket = GetBucket()
	}

	return cClient.NewConsulDiscoveryTemplate(basePath, GetSdAddress(), o)
}
