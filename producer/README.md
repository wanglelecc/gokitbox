# Producer 消息生产代理

支持 Kafka 异步投递，以及使用调用方 Redis client 的 List / Sorted Set 写入。

## 应用启动与默认实例

```go
opts := kafka.DefaultOptions()
opts.Brokers = []string{"localhost:9092"}
opts.WaitAll = true
opts.FailMode = "retry"
if err := producer.Configure(producer.Options{KafkaEnabled: true, Kafka: opts}); err != nil {
    return err
}
if err := producer.Start(); err != nil { // 校验真实 SDK 配置并创建资源
    return err
}
if err := producer.Kafka(ctx, "events", []byte(`{"id":1}`), "business-key"); err != nil {
    return err
}
// 先停止业务，再传入整个关闭流程剩余的预算。
if err := producer.CloseContext(shutdownCtx); err != nil {
    return err
}
```

以上标识分别来自 `github.com/wanglelecc/gokitbox/producer` 和 `producer/kafka`。
`Start` 验证本地 SDK 参数，不代表 broker 已连接或健康检查通过。
首次 `Kafka` 仍会兼容性启动，但应用应显式 `Start`，使配置错误在启动时返回。
构造失败不会发布半初始化单例，也不会消耗不可恢复的 `sync.Once`。
成功配置后不能热重配；默认实例关闭后不能重新打开。

**Kafka 返回 nil 只表示本地接受，不表示 broker 确认。** 调用 context 取消只影响等待入队，
不会撤回已经接受的消息。同步拒绝（非法 topic、单条过大、取消、关闭）直接返回 error。
异步最终失败通过 `Snapshot()`、日志和最终 `CloseContext` / `Wait` 错误可见。

## 独立实例

```go
manager, err := producer.New(producer.Options{KafkaEnabled: true, Kafka: opts})
if err != nil { return err }
if err := manager.KafkaContext(ctx, "events", body, "key with spaces"); err != nil { return err }
stats := manager.Snapshot()
_ = stats
return manager.CloseContext(shutdownCtx)
```

每个实例独立拥有 broker、凭据、灰度配置、队列、重试和 SDK handle。
`KafkaEnabled: false` 不创建 Kafka，即使曾注册过全局 Kafka factory。
`AddMQ` 保留为非 Kafka 历史插件入口；Kafka factory 覆盖已停止使用，须改传实例 Options。
`NewProxyManagerWithOptions`、底层 `InitWithOptions` 保留旧签名及构造失败 panic 行为；
新代码使用返回 error 的 `producer.New` / `kafka.New`。

内部使用结构化消息，通过 SDK `Message.Opaque` 关联投递结果。key 中的空格、二进制及 payload
不会串位；接受前复制数据，调用返回后可以安全复用原 buffer。旧 Kafka(nil) 保持空消息语义，
不会意外生成 compacted-topic tombstone；显式空 key 保留为空 key。

## 有界失败策略

`kafka.Options` 默认值与边界：

| 字段 | 默认 | 说明 |
|---|---|---|
| Timeout | 30s | SDK 单次投递超时；实际值不超过 RetryBudget |
| BufferLimit | 1000 | Go 输入队列容量；安全入口在途与重试共享最多容量+1 个额度 |
| FailMode | discard | 支持 discard / retry；不支持 save |
| MaxRetries | 3 | 0 使用默认3；最多100。需要完全禁用重试使用 discard |
| RetryBackoff | 100ms | 0 使用默认；指数退避最多放大64倍，非零要求至少1ms、最多1h |
| RetryBudget | 2m | 0 使用默认；非零要求至少1ms、最多24h，包含本地排队与在途时间 |
| MaxMessageBytes | 1000000 | 0 使用默认；非零至少1000、最多1000000000 |

同步 Produce 错误和异步最终失败走同一策略。QueueFull、传输/投递超时等临时错误才有资格重试；
非法参数、size、fatal 等不无限回退。librdkafka 本身仍在单次 Timeout 内重试；以上次数是再次
调用 Produce 的应用层次数。重试调度会预留一次 SDK 超时；总预算按10ms调度粒度检查，
在途超预算视为结果未知，SDK 可能已经写入。应用层重试可能产生重复，消费端仍需幂等。
SDK 也限制排队消息数，避免过期任务释放本地额度后无限堆积在 native queue。

入口进行 topic 与单条大小检查（含 key/topic 和协议开销预留）；SDK / broker 仍可能进一步拒绝。
`GrayMode` 会在 topic 后追加 `_gray`，追加后同样验证 topic。
`acks` 默认1，`WaitAll: true` 使用 all；没有自动开启 SDK 幂等，也没有持久化 outbox。

## 生命周期与统计

关闭先拒绝新的调用，排空本地已接受任务、重试与投递事件，检查 Flush 剩余量，再释放 native
handle。事件循环由实例单一协程拥有；`Wait` 的完成意味着该协程与 SDK 资源实际退出。
`Close` / `CloseContext` 幂等，第一次关闭传入的 context 决定排空预算；后续 Wait context
只限制调用者等待时间。调用者超时可先返回；后台继续完成清理，随后可 `Wait` 查询最终结果。

预算耗尽不能伪报成功：剩余任务终结为 unknown，并在返回错误中报告。即使提前某条消息被
`discard`，最终关闭也返回累计终结失败。`Snapshot()` 不触发初始化，可持续观测：

- Accepted：本地接受数。
- Delivered：收到成功 delivery report 的消息数。
- Failed：终结失败数，包含 Dropped。
- Retried：实际再次调用 Produce 的次数。
- Dropped：强制放弃（结果未知或历史裸 Input 容量不足）的消息数。
- Pending：Accepted - Delivered - Failed，含本地队列、重试与在途。
- LastError：最近终结错误或最终关闭错误，不包含 payload。

旧 `common.MQ.Input()` 仅作兼容：不能返回拒绝错误，禁止调用方关闭 channel 或在停止后继续发送。
新代码使用 `Submit` / `KafkaContext`；旧格式输入仍能解析，畸形输入会被明确计入 Failed。
非 Kafka 自定义 factory 必须自行保证初始化失败时释放尚未返回的资源，并履行 Close/quit 合约。
历史 `failmode.Retry.Do` 满队列会记录失败；新适配调用可用 `TryDo` 获取 `ErrRetryFull`。
Kafka 内置重试不再经过该无回执 fallback 通道。

## Redis 与 Delay

```go
err := producer.Redis(ctx, client, "events", body)
err = producer.Delay(ctx, client, "scheduled", body, float64(unixTimestamp))
```

这两个 helper 借用传入 client，不创建/关闭 Kafka 或 Redis，不受默认 Kafka 实例关闭状态控制。
它们返回 Redis 命令结果；不自动提供任务身份、恢复或可靠延迟队列协议。
