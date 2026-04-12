//go:build !zookeeper && !etcd && !redis && !consul
// +build !zookeeper,!etcd,!redis,!consul

package rpcxclient

import (
	"errors"

	"github.com/smallnest/rpcx/client"
)

// initClientDiscovery 默认实现，当没有指定任何服务发现机制时返回错误
// 要使用服务发现功能，请在构建时添加 tags:
//   - zookeeper: 使用 ZooKeeper 服务发现
//   - etcd: 使用 Etcd 服务发现
//   - redis: 使用 Redis 服务发现
//   - consul: 使用 Consul 服务发现
//
// 示例: go build -tags zookeeper ./...
func initClientDiscovery(basePath string) (client.ServiceDiscovery, error) {
	return nil, errors.New("no service discovery mechanism selected. Please build with one of the following tags: zookeeper, etcd, redis, consul")
}
