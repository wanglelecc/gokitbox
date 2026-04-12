package consumer

import (
	"fmt"
	"sync"
	"time"
)

// ComponentType 标识产生告警的消费者组件
type ComponentType string

const (
	ComponentKafka    ComponentType = "kafka"
	ComponentRabbitmq ComponentType = "rabbitmq"
	ComponentRedis    ComponentType = "redis"
	ComponentDelay    ComponentType = "delay"
)

// AlertType 告警类型
type AlertType string

const (
	// AlertPanic 消费 goroutine 发生 panic
	AlertPanic AlertType = "panic"
	// AlertRestart 消费 goroutine 发生异常重启
	AlertRestart AlertType = "restart"
	// AlertConnLost 与 broker/server 的连接断开
	AlertConnLost AlertType = "conn_lost"
	// AlertConnRestored 与 broker/server 的连接恢复
	AlertConnRestored AlertType = "conn_restored"
	// AlertMsgDiscard 消息超出重试次数被丢弃（写入失败队列或直接丢弃）
	AlertMsgDiscard AlertType = "msg_discard"
	// AlertQueueBacklog 队列积压超过阈值
	AlertQueueBacklog AlertType = "queue_backlog"
)

// AlertEvent 告警事件，携带完整上下文，供应用层决策
type AlertEvent struct {
	Component ComponentType // 组件类型
	Name      string        // 连接标识（topic / queue / addr）
	Type      AlertType     // 告警类型
	Message   string        // 可读描述
	Err       error         // 关联错误（可为 nil）
	Time      time.Time     // 告警时间

	// SuppressedCount > 0 表示本次窗口内被收敛的同类告警数量
	// 第一条不被收敛，SuppressedCount 仅出现在窗口过期后的下一条事件上
	SuppressedCount int64
}

func (e AlertEvent) String() string {
	s := fmt.Sprintf("[%s] component=%s name=%s msg=%s", e.Type, e.Component, e.Name, e.Message)
	if e.Err != nil {
		s += fmt.Sprintf(" err=%v", e.Err)
	}
	if e.SuppressedCount > 0 {
		s += fmt.Sprintf(" (suppressed %d in prev window)", e.SuppressedCount)
	}
	return s
}

// AlertFunc 应用层注册的告警回调，在独立 goroutine 中调用，panic 会被 recover
type AlertFunc func(AlertEvent)

// alertKey 告警收敛的维度键
type alertKey struct {
	typ       AlertType
	component ComponentType
	name      string
}

type alertEntry struct {
	windowEnd       time.Time
	suppressedCount int64
}

// AlertManager 管理告警收敛与分发
//
// 收敛策略：同一 (type, component, name) 组合在一个窗口期内只透传第一条；
// 窗口到期后的下一条会携带前一窗口被抑制的数量（SuppressedCount），让应用层感知积压程度。
type AlertManager struct {
	fn      AlertFunc
	window  time.Duration
	mu      sync.Mutex
	entries map[alertKey]*alertEntry
}

// NewAlertManager 创建 AlertManager
//   - fn: 告警回调，为 nil 时 Emit 为空操作
//   - window: 同类告警收敛窗口（推荐 1~5 分钟），≤ 0 时默认 1 分钟
func NewAlertManager(fn AlertFunc, window time.Duration) *AlertManager {
	if window <= 0 {
		window = time.Minute
	}
	return &AlertManager{
		fn:      fn,
		window:  window,
		entries: make(map[alertKey]*alertEntry),
	}
}

// Emit 发送告警事件（异步，带收敛逻辑）
// 调用方不需要关心并发，可从任意 goroutine 调用
func (a *AlertManager) Emit(event AlertEvent) {
	if a == nil || a.fn == nil {
		return
	}

	event.Time = time.Now()

	key := alertKey{typ: event.Type, component: event.Component, name: event.Name}

	a.mu.Lock()
	entry, exists := a.entries[key]
	now := event.Time

	if exists && now.Before(entry.windowEnd) {
		// 仍在收敛窗口内，抑制
		entry.suppressedCount++
		a.mu.Unlock()
		return
	}

	// 窗口已过期或首次出现：将前一窗口的抑制数量附到本次事件
	if exists {
		event.SuppressedCount = entry.suppressedCount
		entry.windowEnd = now.Add(a.window)
		entry.suppressedCount = 0
	} else {
		a.entries[key] = &alertEntry{windowEnd: now.Add(a.window)}
	}
	a.mu.Unlock()

	go a.safeCall(event)
}

func (a *AlertManager) safeCall(event AlertEvent) {
	defer func() { recover() }() // 隔离回调 panic，防止影响消费者 goroutine
	a.fn(event)
}
