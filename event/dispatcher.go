package event

// 事件调度器基类
type EventDispatcher struct {
	savers []*EventSaver
}

// 创建事件派发器
func NewEventDispatcher() *EventDispatcher {
	return new(EventDispatcher)
}

// 事件调度器添加事件
func (this *EventDispatcher) AddEventListener(eventType string, listener *EventListener) {
	for _, saver := range this.savers {
		if saver.Type == eventType {
			saver.Listeners = append(saver.Listeners, listener)
			return
		}
	}

	saver := &EventSaver{Type: eventType, Listeners: []*EventListener{listener}}
	this.savers = append(this.savers, saver)
}

// 事件调度器移除某个监听
func (this *EventDispatcher) RemoveEventListener(eventType string, listener *EventListener) bool {
	for _, saver := range this.savers {
		if saver.Type == eventType {
			for i, l := range saver.Listeners {
				if listener == l {
					saver.Listeners = append(saver.Listeners[:i], saver.Listeners[i+1:]...)
					return true
				}
			}
		}
	}
	return false
}

// 事件调度器是否包含某个类型的监听
func (this *EventDispatcher) HasEventListener(eventType string) bool {
	for _, saver := range this.savers {
		if saver.Type == eventType {
			return true
		}
	}
	return false
}

// 事件调度器派发事件
func (this *EventDispatcher) DispatchEvent(event Event) bool {
	for _, saver := range this.savers {
		if saver.Type == event.Type {
			for _, listener := range saver.Listeners {
				event.Target = this
				listener.Handler(event)
			}
			return true
		}
	}
	return false
}
