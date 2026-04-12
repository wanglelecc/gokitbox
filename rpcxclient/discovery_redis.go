//go:build redis
// +build redis

package rpcxclient

import (
	"github.com/rpcxio/libkv/store"
	rClient "github.com/rpcxio/rpcx-redis/client"
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

	return rClient.NewRedisDiscoveryTemplate(basePath, GetSdAddress(), o)
}
