//go:build etcd
// +build etcd

package rpcxclient

import (
	eClient "github.com/rpcxio/rpcx-etcd/client"
	"github.com/smallnest/rpcx/client"
)

func initClientDiscovery(basePath string) (client.ServiceDiscovery, error) {
	return eClient.NewEtcdV3DiscoveryTemplate(basePath, GetSdAddress(), true, nil)
}
