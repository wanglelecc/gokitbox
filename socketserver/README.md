# socketserver

实例拥有的 TCP / WebSocket 长连接运行时。共用 Session、Hub、读队列、串行写入、登录/空闲截止与 Start/Wait/Stop；不依赖应用 Redis/Kafka、业务 handler 或配置全局源，不注册进程信号。

```go
opts := socketserver.DefaultOptions()
opts.Network = "websocket" // 或 tcp
opts.Addr = ":8888"
srv, err := socketserver.NewServerWithOptions(opts,
    socketserver.WithHooks(socketserver.Hooks{
        OnMessage: func(ctx context.Context, s *socketserver.Session, m socketserver.Message) error {
            return s.Send(ctx, m) // 保留 text/binary 类型
        },
    }),
)
if err != nil { return err }
// 把 srv.Start / srv.Wait / srv.Stop 接入应用 lifecycle。
```

- 构造不获取 listener，不启动 goroutine。Start 同步绑定，启动 context 只约束启动；Stop 幂等且受调用方 context 和 ShutdownTimeout 共同约束。实例停止后不可重启。
- `srv.Connections()` 提供 Get、Count、HasOnline、WriteMessage、Login、Disconnect。WriteMessage 发送原始字节，WS 默认 TextMessage；需要 binary 时用 Session.Send。Login 仅标记应用已认证，认证决策由应用作出。
- `Session.ID/Network/RemoteAddr/CreatedAt/Context/Closed/Wait` 是会话接口。ID 默认随机十六进制，可通过 WithIDGenerator 注入应用生成器。重复 ID 拒绝接管，避免覆盖现有连接。
- OnConnect 失败会关闭并回滚该会话，仍调用 OnDisconnect 以释放部分初始化资源。应用应跟踪自己实际完成的资源步骤；未接管的连接不会进入业务 hook。
- OnMessage 在同一会话按接收顺序执行；不同会话并行。OnMaintain 受固定 worker 数限制，可与消息/关闭 hook 并发，应用资源操作需支持并发或做原子所有权检查。OnError 可并发调用。
- hook panic 变成不泄漏 panic 值的错误。OnMessage 普通错误经 OnError 报告，默认继续；CloseOnHandlerError 可改成断开，panic 总是断开。基础库不重放业务消息。
- 关闭先从 Hub 移除，网络 Close 解除阻塞读取/写入。正常读结束会交付已接收的队列数据；TCP 半关闭允许队列中的处理函数完成响应。主动关闭/超时会取消处理，队列中未处理的数据不承诺投递。
- OnDisconnect 在读取消/消费/写入收敛后执行一次。应用 hooks 必须遵守 context；Go 无法强制终止忽略取消的函数，超时 Stop 返回错误，不伪称清理完成。

## 数据边界与限流

- TCP `raw` 只提供字节流分块，默认每块 1024 字节，不代表完整业务消息；reader 在整个连接中复用，不丢弃预读缓冲。
- `line` 使用一个分隔字节，读时移除、写时追加。`length-prefix` 使用 4 字节大端无符号 payload 长度；两者要求正的 MaxMessageBytes。实际设备协议必须匹配，不能直接替代既有私有协议。
- `custom` 由 WithCodecFactory 注入每会话独立的 FrameCodec。队列会复制 codec 返回的 payload，避免复用缓冲被下一次读取覆盖。
- WS 保留完整消息及 text/binary 类型，WebSocketPath 默认 /websocket。默认同源校验（没有 Origin 的设备客户端允许）；可显式配置 AllowedOrigins 或 AllowAllOrigins，二者互斥。
- MaxConnections 包含正在接管、活动和正在排空的会话；0 不限制。MaxMessageBytes=0、ReadQueueBytes=0 分别禁用消息/排队字节上限，仅用于兼容已有协议，生产值应显式配置。
- 接收队列按条数等待并支持取消；超过字节预算拒绝并断开，不静默丢弃。WriteQueueSize 限制同一会话活动与等待写入数，满时返回 ErrQueueFull。所有实际写入串行，受 WriteTimeout 和更短的请求 deadline 约束；写超时/取消会关闭该会话，不重用可能已经写入半帧的连接。
- IdleTimeout 以收到的数据帧/分块更新，不自动生成设备业务心跳；LoginTimeout 管理应用认证截止，0 可禁用。WS 原生控制帧与业务心跳分别处理，本轮不添加主动 Ping 协议。

WithHTTPRoutes 可在 WS HTTP mux 上挂载应用路径；回调在绑定端口前执行，panic 返回启动错误。Socket 监听与应用 RPC TLS 是独立端口；本组件本轮沿用普通 TCP/WS 部署，不新增 TLS/WSS 配置。

本地测试覆盖 TCP 预读/半关闭、分帧、WS 类型与 Origin/大小限制、队列取消、慢写、并发状态、接管失败、容量、多个实例、失败回滚与有界关闭。测试可被仓库策略忽略，应用仍应保留并执行。loopback benchmark 仅代表本机基线，不代表生产容量。

## 容量与调度

- 每连接保留一个会话消费者和一个 reader，共约 2 个 goroutine；消费者仍按顺序执行，正常 EOF 排空与 TCP 半关闭响应语义保持。
- 路由等维护回调由索引最小堆按到期时间排期，固定 MaintenanceWorkers 持续消费到期任务；忙时等待，不因小调度队列已满丢掉本轮任务。首次在半个至一个 interval 之间错峰，后续在 90%～100% interval 之间续期。回调耗时和排队仍可能造成实际延迟，需根据外部资源 TTL 留预算、观察 Stats，不承诺过载时准点完成。
- DefaultOptions 使用 64 条读队列、1 MiB 每连接读取 payload 预算、128 MiB Server 读取预算、64 MiB Server 写入预算、256 个并发接管；0 表示禁用相应字节/接管上限。单会话 ReadQueueBytes 与 Server MaxReadBytes 均涵盖已预留、排队与正在处理的 payload，在 hook 返回/取消排空后释放。达到读预算则断开该会话，达到写预算返回 ErrQueueFull，不做隐式重放。
- MaxWriteBytes 统计已接受 Send 调用的 payload 长度（含等待写入），返回时释放；不含调用者在进入 Send 前自行持有的内存。读预算在 transport 完成读取后预留，不能限制此前 WS 解码/自定义 codec 的临时分配。上述预算不是 RSS 硬上限；应同时配置 MaxMessageBytes、MaxConnections，并为内核、解码和 producer 缓冲预留内存。
- MaxPendingConnections 覆盖 ID 生成和 OnConnect 执行阶段，避免下游慢时无界接管；WS upgrade 之前的 HTTP 握手不在此计数范围，应在入口单独限制。MaxConnections 覆盖接管、活动与排空阶段。
- 内置 TCP codec 和 WS 每次读取返回独立 payload，核心直接转移所有权；自定义 codec 仍安全复制，复用 scratch buffer 不会覆盖已入队消息。没有引入无界 buffer pool。
- `srv.Stats()` 提供 Connections、PendingConnections、ReadBytes、WriteBytes、MaintenancePending、MaintenanceCalls、MaintenanceErrors、MaintenanceMaxDelay，可供应用指标系统采集。维护计数/最大调度延迟自实例启动累计，Pending 只统计堆中等待项，不含正在派发/执行项；快照各字段并非跨资源事务。

容量优化不会自动提高 MaxConnections，也不把配置上限当作生产容量。目标主机需验证真实 Redis/Kafka、日志、消息频率、故障恢复与长期路由有效性。
