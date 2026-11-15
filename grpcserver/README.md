# grpcserver

显式 Options 的 gRPC 服务端，不读取配置、不注册信号、不退出进程。与 rpcxserver 并存，由应用选择协议。

```go
opts := grpcserver.DefaultOptions()
opts.Addr = "127.0.0.1:19090"
s, err := grpcserver.NewServerWithOptions(opts)
// 检查 err；注册生成的服务：
err = s.Register(func(r grpc.ServiceRegistrar) { pb.RegisterYourServiceServer(r, impl) })
// 检查 err；由应用生命周期依次调用：
err = s.Start(ctx)
err = s.Wait()
err = s.Stop(shutdownCtx)
```

Register 仅在 Start 前调用，执行同步注册回调并检查重复服务/接口匹配；不可保留 registrar 异步注册。Start 同步绑定，单实例不可重启，失败也要 Stop。Stop 幂等，受调用 context 与 ShutdownTimeout 共同约束；超时强制断开连接并返回错误，不能强杀忽略取消的业务 goroutine。

默认 4 MiB 收发、30 秒关闭预算，health 开启、reflection 关闭。默认绑定 loopback，不启用 TLS；生产环境按应用访问策略配置 TLS/mTLS 和 BearerToken/AuthFunc。所有 unary/stream（包括探针与反射）经过鉴权/recovery/错误映射/耗时日志。未知错误和 panic 对外返回 Internal，不泄漏错误原文；panic 只记录类型和堆栈。显式 status 和 context 错误保留。ErrorMapper 可注入业务映射，不能依赖应用包。

Health 是传输状态，应用用 SetServing 控制依赖健康策略。Options 提供 unary/stream interceptor 和 LogFunc；原生 ServerOption 用于 stats 等扩展，须由调用者检查与内置选项是否冲突。扩展回调同步执行，不得阻塞启动。证书在 Start 时加载，路径由应用解析。没有证书热更新、注册中心桥接或自动重试。
