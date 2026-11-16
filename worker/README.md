# Worker

消费者管理器，支持多种消息中间件：kafka, rabbitmq, redis...

## 安装
```shell script
go get github.com/wanglelecc/gokitbox/worker
```

> **RabbitMQ 依赖说明**：底层使用 `github.com/rabbitmq/amqp091-go`（RabbitMQ 官方维护），已替换原停更的 `github.com/streadway/amqp`，API 兼容无需修改业务代码。

## 配置
```json
{
  "enabled": {
    "kafka": false,
    "rabbitmq": false,
    "redis": true,
    "delay": true
  },
  "kafka": [
    {
      "consumerGroup": "demo_consumer",
      "consumerCount": 1,
      "host": [
        "10.11.11.1:9092",
        "10.11.11.2:9092",
        "10.11.11.3:9092"
      ],
      "sasl": {
        "enabled": false,
        "user": "",
        "password": ""
      },
      "topic": "gokit_message",
      "failTopic": "",
      "failCount": 3,
      "tplName": "",
      "tplMode": 1
    }
  ],
  "rabbitmq": [
    {
      "url": "amqp://guest:guest@127.0.0.1:5672/",
      "consumerQueue": "message_queue",
      "consumerCount": 2,
      "prefetchCount": 10,
      "failExchange": "",
      "isReject": false,
      "failCount": 3,
      "tplName": "",
      "tplMode": 1
    }
  ],
  "redis": [
    {
      "consumerCount": 1,
      "addr": "127.0.0.1:6379",
      "username": "",
      "password": "",
      "db": 0,
      "queue": "message_list",
      "failQueue": "",
      "failCount": 3,
      "tplName": "echo"
    }
  ],
  "delay": [
    {
      "consumerCount": 1,
      "addr": "127.0.0.1:6379",
      "username": "",
      "password": "",
      "db": 0,
      "queue": "message_delay",
      "failQueue": "",
      "failCount": 3,
      "tplName": "echo"
    }
  ]
}
```

### 公共字段
| 字段 | 说明 |
|------|------|
| `consumerCount` | 消费者协程数量 |
| `failCount` | 业务处理失败最大重试次数，超出后转入失败队列 |
| `tplName` | 消费逻辑注册名，为空时默认取 topic/queue 名称 |
| `tplMode` | 兼容保留字段；当前按 tplName 或 topic/routing key/queue 选择 handler，尚未实现 tag 拆分模式 |

### Kafka 专有字段
| 字段 | 说明 |
|------|------|
| `consumerGroup` | 消费组名 |
| `host` | Broker 地址列表 |
| `sasl` | SASL 认证配置，`enabled: true` 时启用 PLAIN 机制 |
| `topic` | 消费的 topic |
| `failTopic` | 超出重试次数后转入的失败 topic，为空则丢弃 |

### RabbitMQ 专有字段
| 字段 | 说明 |
|------|------|
| `url` | 连接地址，格式 `amqp://user:pass@host:port/vhost` |
| `consumerQueue` | 消费的队列名 |
| `prefetchCount` | 每个 Channel 的预取消息数（QoS） |
| `failExchange` | 超出重试次数后转发的 Exchange，为空则执行 Reject |
| `isReject` | Reject 时是否重新入队（requeue），`false` 表示丢弃 |

> RabbitMQ 每个消费协程独立持有一个连接和 Channel，网络断开后自动重连，无需手动干预。

### Redis / Delay 专有字段
| 字段 | 说明 |
|------|------|
| `addr` | Redis 地址 |
| `username` | Redis 用户名，Redis 6.0+ ACL 场景使用，留空则不传 |
| `password` | Redis 密码 |
| `db` | Redis DB 编号 |
| `queue` | 消费的 List key（Redis）或 ZSet key（Delay） |
| `failQueue` | 超出重试次数后转入的失败 key，为空则打日志丢弃 |

> `delay` 使用 Redis ZSet 实现延迟队列，score 为 Unix 时间戳，到期后自动消费。

## 初始化
```go
import (
    "github.com/wanglelecc/gokitbox/configsource"
    "github.com/wanglelecc/gokitbox/worker"
)

appCfg := configsource.GetConfStringMap("App")

s := worker.NewWorker()
s.AddBeforeServerStartFunc(
    bootstrap.InitLogger(appCfg["env"], appCfg["name"], "gokit", version.TAG),
    s.RegisterHandle(app.HandleMap()),
    s.InitConsumer(),
)
s.AddAfterServerStopFunc(s.CloseConsumer())

err := s.Serve()
if err != nil {
    log.Printf("worker stop err:%v", err)
} else {
    log.Printf("worker exit")
}
```

> 默认消费者处理器放在 `app/handle.go`

## 可选配置

```go
s := worker.NewWorker().
    SetShutdownTimeout(60 * time.Second). // 优雅关闭等待时长，默认 30s
    SetRetryDelay(1 * time.Second)        // 业务处理失败后的重试间隔，默认 500ms
```

| 方法 | 默认值 | 说明 |
|------|--------|------|
| `SetShutdownTimeout(d)` | 30s | 收到停止信号后等待消费者完成的最大时长，处理耗时较长的业务建议调大 |
| `SetRetryDelay(d)` | 500ms | 业务 Handle 返回 `false` 后到下次重试的等待时间 |

## 健康检查边界

当前尚无公开的 `Health()/HealthReport`，旧示例已移除。不要把消费 goroutine 活跃等同于 broker 可用或消息已成功确认；统一健康快照、lag 和告警资源上限留在后续阶段。

## 服务骨架示例
https://github.com/wanglelecc/gokitbox/worker-demo
> 新项目可以直接克隆，替换名称并直接使用。

### 应用层资源分组与 Kafka 失败投递

应用可以在主 YAML 中按资源组织多个 topic/queue，再解析为本库的显式 `consumer.Configure`。命名实例引用、订阅 defaults 与覆盖关系属于应用加载层；本库不查找应用的全局资源配置，旧 JSON/YAML Configure 继续兼容。

Kafka 失败消息 Producer 按 broker 地址集合和 SASL 认证信息分别创建；同连接多个 topic 复用 Producer，不同连接或凭据互相隔离。Producer 使用对应消费者资源的认证设置，初始化失败时回收已创建资源，关闭时统一 Flush/Close。此处保留现有 SASL/PLAIN 协议选项，没有新增 TLS 配置。

## 第一、二阶段：可靠转交与可取消生命周期

新应用继续在主 app.yaml 中配置资源组，由应用解析为 `consumer.Configure`；本库直接接收显式配置，不再要求额外文件。原 JSON/YAML 配置及 `Handle(ctx, []byte) (bool, error)` 保留；`err != nil` 一律按失败处理，不能用 `(true, err)` 确认消息。

```yaml
worker:
  runtime:
    handlerTimeout: 30s
    retryDelay: 500ms
    deliveryTimeout: 10s
    operationTimeout: 3s
    sessionTimeout: 30s
    maxPollInterval: 5m
  # enabled 与 kafka/rabbitmq/redis/delay 资源组保持原来的配置方式。
```

RuntimeOptions 的零值采用默认值，不表示禁用；可以通过 `consumer.DefaultRuntimeOptions()` 获取默认模型。JSON 新选项支持 `"30s"` 等时长字符串和原始纳秒数，未知字段拒绝。Kafka operationTimeout 为 1 秒至 5 分钟，maxPollInterval 不超过 24 小时，handler 重试、三次失败转交、三次提交及等待必须整体容纳于 maxPollInterval；sessionTimeout 还须符合实际 broker 的分组设置。长任务应同时调整 handlerTimeout/maxPollInterval，不只增大关闭等待时间。

新的 `NewConsumeManagerWithContext`、各 `New*ConsumerWithContext` 接受：

```go
func(ctx context.Context, tplName, key string, value []byte) (bool, error)
```

原 bool-only ConsumeCallback 构造器继续可用，但无法强制停止忽略取消的旧回调。每次尝试提供独立 payload，处理器修改入参不会污染后续重试或失败转交。Worker 自动使用新回调桥，业务 Handle 中应尊重传入 ctx。处理器注册属于 Worker 实例；旧全局注册仍作为兼容回退，配置与本地处理器 map 进行防御性复制。可静态确定的 handler 名称会在开始消费前校验；RabbitMQ 的动态 routing key 在收到消息后解析。

- Kafka 在一个会话中串行处理并提交；失败转交未获确认、提交失败/结果未知时结束会话，不读取后续消息跨过失败 offset。失败 Producer 开启幂等和 acks=all，投递本身仍可能因“目标已收到但源提交失败”而重复，业务须幂等。故障事件被持续排空，关闭统一等待；不再因未读错误事件让 Flush 假性超时。
- Delay 用单 key Lua 原子检查到期时间并取出；未来 member 不删除、不回写，不覆盖并发改期。这不等于新增了 Redis Cluster Client。
- RabbitMQ 失败转交使用 publisher confirm 与 mandatory/return，保留消息 ID、持久化模式及必要属性；无法确认/无法路由时保留源消息未 Ack 并关闭会话。目标持久化仍要求正确的 broker/queue 配置；不照搬可能属于其他发布用户的 UserId。未配置失败 Exchange 时，保留原 IsReject/requeue 策略。
- Redis/Delay 在正常取消或失败转交不可用时尝试恢复源消息；Delay 恢复使用 NX，避免覆盖新排期。恢复也失败时记录错误并使关闭结果暴露风险，不宣称可靠消费成功。

Start/Wait/Stop 为不注册信号的现代入口；Serve 保留信号兼容并调用相同状态机，包括启动期间收到信号。Stop 立即停止取新消息、允许在途工作在预算内完成，预算到期取消其 context。Stop 可超时，Wait 仍等待实际资源与 hooks 完成；重复 Stop 等待同一个关闭结果。AfterStop hooks 只在真实启动后、资源 drain 结束后执行一次。SetShutdownTimeout 对两个入口均生效，SetRetryDelay 设置统一 runner 的重试间隔，不再与适配器固定延迟叠加。setter/hook 注册应在启动前完成。

SDK 的同步 Kafka commit/close 和忽略 context 的 legacy 代码不能被 Go 安全强杀；最多保留该会话拥有的在途操作，结束前不销毁仍在使用的 native handle。Stop 按调用预算返回错误，Wait 不伪造结束，也不继续读取并提交后续消息。不得在 handler 内同步等待关闭自身 Worker。

### 第三阶段仍未包含

当前 Redis List/ZSet 模式仍是取出即删除：进程崩溃、命令已执行但回复丢失、源队列恢复失败都可能丢消息；ZSet 同 body member 还会合并。这一阶段没有新增 processing/lease/ack/reclaim 或改变 key 布局，不应对旧模式承诺至少一次或 exactly-once。需要这些保证时应单独启用后续可靠消费模式并设计业务幂等。
