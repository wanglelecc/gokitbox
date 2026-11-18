package event

import "sync"

type listenerEntry struct {
	identity *EventListener
	handler  EventHandler
}
type EventDispatcher struct {
	mu        sync.RWMutex
	listeners map[string][]listenerEntry
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{listeners: map[string][]listenerEntry{}}
}
func (d *EventDispatcher) AddEventListener(kind string, l *EventListener) {
	if l == nil || l.Handler == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.listeners == nil {
		d.listeners = map[string][]listenerEntry{}
	}
	d.listeners[kind] = append(d.listeners[kind], listenerEntry{l, l.Handler})
}
func (d *EventDispatcher) RemoveEventListener(kind string, l *EventListener) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	ls := d.listeners[kind]
	for i, v := range ls {
		if v.identity == l {
			d.listeners[kind] = append(ls[:i], ls[i+1:]...)
			if len(d.listeners[kind]) == 0 {
				delete(d.listeners, kind)
			}
			return true
		}
	}
	return false
}
func (d *EventDispatcher) HasEventListener(kind string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.listeners[kind]) > 0
}
func (d *EventDispatcher) DispatchEvent(e Event) bool {
	d.mu.RLock()
	callbacks := make([]EventHandler, 0, len(d.listeners[e.Type]))
	for _, l := range d.listeners[e.Type] {
		callbacks = append(callbacks, l.handler)
	}
	d.mu.RUnlock()
	e.Target = d
	for _, fn := range callbacks {
		fn(e)
	}
	return len(callbacks) > 0
}
