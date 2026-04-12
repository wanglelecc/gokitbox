# RpcxServer

RpcxServer 基于 `rpcx` 封装，方便集成到 `gokit` 项目中，无代码侵入，保留原汁原味的 `github.com/smallnest/rpcx`, 享受原版的维护更新。支持 `zookeeper`, `etcd` 两种服务发现方式。

## 安装
```shell script
go get  github.com/wanglelecc/gokitbox/rpcxserver
```

## 配置
```ini
[Server]
network = tcp
port = 9999

[Registry]
; 是否启用注册中心 on:启用   off:不启用
status = on
; 注册中心地址，多个地址以空格分割
addrs = 10.90.70.205:2181 10.90.71.147:2181 10.90.71.159:2181
basePath = /tal_zhongtai_weixin
; 注册更新时间 1m表示一分钟
updateInterval = 1m
; 应用分组
group = dev

; 服务端鉴权配置
[ValidRpcxAuth]
2010101=fe02fnelfn92rnknl
1203434=jeofnen823rubkj2j

; 客户端鉴权配置
[RpcxAuth]
appId=2010101
appKey=fe02fnelfn92rnknl 

```

## 初始化
```go
import (
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/rpcxserver"
	"github.com/wanglelecc/gokitbox/rpcxserver/bootstrap"
	"github.com/wanglelecc/gokitbox/rpcxserver/middleware"
)

addr, err := logger.Extract("")
	if err != nil {
		panic(err)
	}

	// 注册中间件
	middleware.Use(middleware.ContextOption())

	// 实例化RPCX服务
	s := rpcxserver.NewServer(rpcxserver.Addr(addr))
	
    // 注册前置方法，初始化日志，加载配置，启动注册中心，注入RPC服务...
    s.AddBeforeServerStartFunc(bootstrap.InitLogger("dev", "RpcxServerDemo", "Cloud", "1.0.1"),
		s.InitConfig(),
		s.InitRegistry(),
		s.DisableHTTPGateway(),
		s.RegisterServiceWithPlugin("RpcxServerDemo", app.NewService(), ""))

    // 注册服务后置方法
	s.AddAfterServerStopFunc(bootstrap.CloseLogger())

    // 启动服务并监听
	err = s.Serve()
	if err != nil {
		panic(err)
	}
```

## 服务示例
```go
import (
	"context"
	"github.com/wanglelecc/gokitbox/rpcxserver/middleware"
	"rpcxserver-demo/app/service"
	"rpcxserver-demo/rpc/proto"
)

type Service struct {
	middleware.RpcxService
}

func NewService() *Service {
	s := new(Service)
	s.Init()

	return s
}

// 登录接口示例
func (s *Service) Login(ctx context.Context, req *proto.LoginRequest, resp *proto.LoginResponse) error {
	fn := func(w *middleware.WrapContext) error {
		return service.UserServiceStd.Login(w.GetCtx(), req, resp)
	}

	return s.WrapCall(ctx, "Login", fn)
}

```
> 默认服务注册放在 `app/service.go`

## 操作文档
https://doc.rpcx.io/

## 服务骨架示例
https://github.com/wanglelecc/gokitbox/rpcxserver-demo
> 新项目可以直接克隆，替换名称并直接使用。