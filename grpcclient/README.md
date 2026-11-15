# grpcclient

每个实例拥有自己的 grpc.ClientConn、target、凭证与 metadata，无全局配置。

```go
opts := grpcclient.DefaultOptions()
opts.Target = "dns:///control.example.com:443"
client, err := grpcclient.NewClientWithOptions(opts)
// 检查 err，然后交给生成的客户端：
api := pb.NewYourServiceClient(client.Conn())
defer client.Close()
```

基于 grpc.NewClient 延迟连接，构造成功不证明连通。默认 TLS 使用系统 CA，可配置私有 CA/ServerName 和成对的客户端证书/key；只有显式 Insecure 才走明文，不能与 TLS 选项混用。BearerToken 注入 authorization；Metadata 支持默认值与每次 outgoing context 覆盖，不支持二进制配置值。Close 幂等，并发安全。

默认 unary 超时 3 秒，尊重更短的调用方 deadline；长流完全由调用方 context 控制。默认收发上限 4 MiB。不额外配置业务重试，不持有应用 DTO，不读取配置文件。原生 DialOption 支持自定义 resolver/dialer/stats 等，由应用负责相容性、凭证与所有权。应用退出时必须 Close。
