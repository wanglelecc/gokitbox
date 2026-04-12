# RpcxClient RPC 客户端

RpcxClient 基于 [rpcx](https://github.com/smallnest/rpcx) 封装，提供服务发现和负载均衡能力，支持多种注册中心。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/rpcxclient
```

## 特性

- 支持多种服务发现机制（ZooKeeper、Etcd、Redis、Consul）
- 自动负载均衡（加权轮询）
- 自动故障转移
- 内置熔断机制
- 超时控制
- 服务鉴权支持

## 构建标签

本包使用条件编译，需要通过 build tags 指定注册中心类型：

```shell
# ZooKeeper
go build -tags zookeeper ./...

# Etcd
go build -tags etcd ./...

# Redis
go build -tags redis ./...

# Consul
go build -tags consul ./...
```

## 配置

```ini
[Registry]
; 注册中心地址，多个以空格分隔
addrs = 10.90.70.205:2181 10.90.71.147:2181 10.90.71.159:2181
; 服务基础路径
basePath = /tal_zhongtai_weixin
; 服务分组
group = dev
; RPC 调用超时（默认 30s）
rpcCallTimeout = 30s
; RPC 连接超时（默认 1s）
rpcConnectTimeout = 5s

; 客户端鉴权配置
[RpcxAuth]
appId = 2010101
appKey = fe02fnelfn92rnknl
```

## 使用示例

```go
package main

import (
    "context"
    "log"
    
    "github.com/wanglelecc/gokitbox/rpcxclient"
)

func main() {
    // 检查配置是否初始化成功
    if !rpcxclient.IsConfigInitialized() {
        log.Fatal("rpcxclient config not initialized")
    }
    
    // 创建 XClient
    xclient, err := rpcxclient.NewXClient("MyService")
    if err != nil {
        log.Fatal(err)
    }
    defer xclient.Close()
    
    // 调用服务
    ctx := context.Background()
    args := &Args{A: 10, B: 20}
    reply := &Reply{}
    
    err = xclient.Call(ctx, "Mul", args, reply)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("result: %d", reply.C)
}
```

## API 说明

### 创建客户端

```go
// NewXClient 创建指定服务的 XClient
func NewXClient(serviceName string) (client.XClient, error)

// NewXClientWithPlugins 创建带插件的 XClient
func NewXClientWithPlugins(serviceName string, plugins ...client.Plugin) (client.XClient, error)
```

### 配置获取

```go
// 获取服务发现地址
addrs := rpcxclient.GetSdAddress()

// 获取服务基础路径
basePath := rpcxclient.GetServiceBasePath()

// 获取分组
group := rpcxclient.GetGroup()

// 获取超时时间
timeout := rpcxclient.GetCallTimeout()

// 获取客户端选项
option := rpcxclient.GetClientOption()
```

### 故障模式

```go
// GetFailMode 获取故障模式（默认 Failover）
failMode := rpcxclient.GetFailMode()
// Failover - 自动切换到其他节点
// Failfast - 快速失败
// Failtry - 重试当前节点
// Failbackup - 发送到多台服务器，使用最快返回的
```

### 选择模式

```go
// GetSelectMode 获取节点选择模式（默认 WeightedRoundRobin）
selectMode := rpcxclient.GetSelectMode()
// RandomSelect - 随机
// RoundRobin - 轮询
// WeightedRoundRobin - 加权轮询
// WeightedICMP - 根据网络质量选择
// ConsistentHash - 一致性哈希
// Closest - 选择最近的
```

## 更多文档

- [rpcx 官方文档](https://doc.rpcx.io/)
- [rpcx GitHub](https://github.com/smallnest/rpcx)
