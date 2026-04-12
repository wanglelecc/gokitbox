//go:build !zookeeper && !etcd && !redis && !consul
// +build !zookeeper,!etcd,!redis,!consul

package rpcxserver

import "errors"

// AddRegistryPlugin 默认实现，当没有指定任何服务注册中心时返回错误
// 要使用服务注册功能，请在构建时添加 tags:
//   - zookeeper: 使用 ZooKeeper 服务注册
//   - etcd: 使用 Etcd 服务注册
//   - redis: 使用 Redis 服务注册
//   - consul: 使用 Consul 服务注册
//
// 示例: go build -tags zookeeper ./...
func AddRegistryPlugin(s *Server) error {
	return errors.New("no service registry mechanism selected. Please build with one of the following tags: zookeeper, etcd, redis, consul")
}
