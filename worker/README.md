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
| `tplMode` | 消费模式：1 消费全量消息；2 按消息体首个空格前的 tag 路由 |

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
    "github.com/wanglelecc/gokitbox/config"
    "github.com/wanglelecc/gokitbox/worker"
)

appCfg := config.GetConfStringMap("App")

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

## 健康检查

```go
report := s.Health()

if report.Redis != nil {
    fmt.Printf("Redis: active=%d panic=%d restart=%d\n",
        report.Redis.ActiveGoroutines,
        report.Redis.PanicCount,
        report.Redis.RestartCount,
    )
}
```

`HealthReport` 包含已启用的各消费者快照，未启用的字段为 `nil`。

| 字段 | 说明 |
|------|------|
| `ActiveGoroutines` | 当前活跃消费协程数 |
| `PanicCount` | 累计 panic 恢复次数，持续升高说明业务代码有问题 |
| `RestartCount` | 累计自愈重启次数，持续升高说明下游连接不稳定 |

> 消费者内部已实现指数退避自愈（1s → 2s → 4s ... 上限 30s + ±25% jitter），连接断开或 panic 后自动重启，无需外部干预。

## 服务骨架示例
https://github.com/wanglelecc/gokitbox/worker-demo
> 新项目可以直接克隆，替换名称并直接使用。
