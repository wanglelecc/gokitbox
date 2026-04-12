package event

import "fmt"

// 事件调度接口
type IEventDispatcher interface {
	// 事件监听
	AddEventListener(eventType string, listener *EventListener)
	// 移除事件监听
	RemoveEventListener(eventType string, listener *EventListener) bool
	// 是否包含事件
	HasEventListener(eventType string) bool
	// 事件派发
	DispatchEvent(event Event) bool
}

// 事件类型基类
type Event struct {
	// 事件触发实例
	Target IEventDispatcher
	// 事件类型
	Type string
	// 事件携带数据源
	Object interface{}
}

// 事件调度器中存放的单元
type EventSaver struct {
	Type      string
	Listeners []*EventListener
}

// 创建事件
func NewEvent(eventType string, object interface{}) Event {
	e := Event{Type: eventType, Object: object}
	return e
}

// 克隆事件
func (this *Event) Clone() *Event {
	e := new(Event)
	e.Type = this.Type
	e.Target = this.Target
	return e
}

func (this *Event) ToString() string {
	return fmt.Sprintf("Event Type %v", this.Type)
}
