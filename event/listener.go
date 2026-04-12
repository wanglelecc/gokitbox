package event

// 监听器函数
type EventHandler func(event Event)

// 监听器
type EventListener struct {
	Handler EventHandler
}

// 创建监听器
func NewEventListener(h EventHandler) *EventListener {
	l := new(EventListener)
	l.Handler = h
	return l
}
