//go:build zookeeper
// +build zookeeper

package rpcxclient

import (
	zClient "github.com/rpcxio/rpcx-zookeeper/client"
	"github.com/smallnest/rpcx/client"
)

func initClientDiscovery(basePath string) (client.ServiceDiscovery, error) {
	return zClient.NewZookeeperDiscoveryTemplate(basePath, GetSdAddress(), nil)
}
