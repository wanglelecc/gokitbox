package producer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/producer/common"
	"github.com/wanglelecc/gokitbox/producer/kafka"
	"github.com/wanglelecc/gokitbox/producer/meta"

	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type InitP func(exit chan struct{}, fallBack chan<- []byte) common.MQ

var (
	pid      = 0
	initMap  = make(map[string]InitP)
	initLock sync.RWMutex
	std      *ProxyManager
	initOnce sync.Once
)

func AddMQ(mqType string, init InitP) {
	initLock.Lock()
	defer initLock.Unlock()
	initMap[mqType] = init
}

func init() {
	pid = os.Getpid()
}

func Init() {
	initOnce.Do(func() {
		initProducer()
	})
}

func initProducer() {
	// 初始化 Kafka
	kafkaEnable := cast.ToBool(config.GetConfDefault("KafkaProxy", "enable", "false"))
	if kafkaEnable {
		AddMQ(meta.Kafka, kafka.CInit)
	}

	std = NewProxyManager()
}

type ProxyManager struct {
	input chan []byte

	mqMap map[string]common.MQ
	quits []chan struct{}
	exit  chan string

	grayMode bool
}

func NewProxyManager() *ProxyManager {
	pm := new(ProxyManager)
	bufferLimit := cast.ToInt64(config.GetConfDefault("Producer", "bufferLimit", "2000"))
	pm.grayMode = cast.ToBool(config.GetConfDefault("MQProxy", "grayMode", "false"))

	pm.exit = make(chan string, 1)
	pm.input = make(chan []byte, bufferLimit+bufferLimit/2)
	pm.mqMap = make(map[string]common.MQ)

	pm.quits = make([]chan struct{}, 0)

	initLock.RLock()
	defer initLock.RUnlock()
	for mqType, init := range initMap {
		quit := make(chan struct{}, 1)
		mq := init(quit, pm.input)
		mq.SetFailMode()
		pm.mqMap[mqType] = mq
		pm.quits = append(pm.quits, quit)
	}

	go pm.Run()

	return pm
}

// Kafka 消息投递
func (m *ProxyManager) Kafka(topic string, msg []byte, key ...string) error {
	mq := m.mqMap[meta.Kafka]
	if mq == nil {
		return errors.New("Kafka proxy does not exist")
	}

	strs := []string{strconv.Itoa(pid)}
	strs = append(strs, strconv.FormatInt(time.Now().UnixNano()/1000000, 10))
	logid := strings.Join(strs, ".")
	if len(key) > 0 {
		logid = key[0]
	}

	if m.grayMode {
		topic = topic + "_gray"
	}

	//line := internal.BytePool.Get().([]byte)
	line := []byte("")
	line = append(line, []byte(cast.ToString(time.Now().Unix()*10))...)
	line = append(line, ' ')
	line = append(line, []byte(meta.Kafka)...)
	line = append(line, ' ')
	line = append(line, []byte(topic)...)
	line = append(line, ' ')
	line = append(line, []byte(logid)...)
	line = append(line, ' ')
	line = append(line, msg...)

	mq.Input() <- line

	return nil
}
func Kafka(ctx context.Context, topic string, msg []byte, key ...string) error {
	Init()

	if std == nil {
		return errors.New("producer std is nil")
	}

	msg = appendTraceId(ctx, msg)
	return std.Kafka(topic, msg, key...)
}

// Redis
func Redis(ctx context.Context, client *redis.Client, topic string, msg []byte) error {
	Init()

	msg = appendTraceId(ctx, msg)
	return client.RPush(ctx, topic, string(msg)).Err()
}

// Delay
func Delay(ctx context.Context, client *redis.Client, topic string, msg []byte, score float64) error {
	Init()

	msg = appendTraceId(ctx, msg)
	return client.ZAdd(ctx, topic, redis.Z{
		Score:  score,
		Member: string(msg),
	}).Err()
}

func (m *ProxyManager) Run() {
	for {
		select {
		case msg := <-m.input:
			items := bytes.Split(msg, []byte(" "))
			mqType := ""
			if len(items) > 2 {
				mqType = string(items[1])
			}
			mq := m.mqMap[mqType] // 判断消息类型，并将其发送到对应的mq
			if mq != nil {
				mq.Input() <- msg
			} else {
				logger.Ex(context.Background(), "ProxyManager", fmt.Sprintf("not support mq type%s", string(mqType)))
			}
		case <-m.exit:

			for _, mq := range m.mqMap {
				mq.Close()
			}

			for _, quit := range m.quits {
				<-quit
			}

			return
		}
	}
}

func (m *ProxyManager) Close() {
	m.exit <- "shutdown"
}

func Close() {
	if std != nil {
		std.Close()
	}
}

func appendTraceId(ctx context.Context, body []byte) []byte {
	bodyStr := string(body)
	if !(json.Valid(body) && strings.HasPrefix(bodyStr, "{") && strings.HasSuffix(bodyStr, "}")) {
		return body
	}

	traceId := cast.ToString(ctx.Value("trace_id"))
	if traceId == "" {
		return body
	}

	if gjson.GetBytes(body, "trace_id").String() != "" {
		return body
	}

	nBody, err := sjson.SetBytes(body, "trace_id", traceId)
	if err != nil {
		return body
	}

	return nBody
}
