# producer 消息生产代理

消息生产者代理，支持 `kafka`, `redis`, `delay` 等类型。

## 安装

```shell script
go get github.com/wanglelecc/gokitbox/producer
```

## 配置

```ini
[MQProxy]
; 灰度模式
grayMode = false

[KafkaProxy]
enable = true
; 调整 confluent-kafka-go 作为基础包，sarama 保留，手动配置 type=sarama 开启, 默认: confluent
;type=sarama
KafkaWaitAll = true
KafkaCompression = true
KafkaPartitioner = round
KafkaProducerTimeout = 10
brokers = localhost:9092
sasl = false
user =
password =
valid = tal_exercise_submit tal_exam_submit
;消息失败处理模式，支持retry(重试)/save(保存到redis)/discard(直接丢弃)，默认为retry
failMode = discard
```

### 开始

```go
// 导入库
import github.com/wanglelecc/gokitbox/producer

// kafka 投递示例
producer.Kafka("kafka_topic", []byte("msg context..."))

// 投递 redis 队列
; rdsClient = redis 实例
producer.Redis(rdsClient, "gokit_rds_list", []byte("msg context..."))

// 延迟消息投递示例
producer.Delay(rdsClient, "gokit_rds_delay", []byte("msg context..."), 1684627485)
```