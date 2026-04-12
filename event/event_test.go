package event

import (
	"sync/atomic"
	"testing"
	"time"
)

const helloWorld = "helloWorld"

// 实例化分发器
var dispatcher = NewEventDispatcher()

func TestEvent(t *testing.T) {
	// 注册监听器
	dispatcher.AddEventListener(helloWorld, NewEventListener(myEventListener))

	time.Sleep(time.Second * 1)

	// 事触发事件
	dispatcher.DispatchEvent(NewEvent(helloWorld, nil))
}

// 事件回调
func myEventListener(event Event) {
	// 使用标准输出
	println("event.name:", event.Type, "  event.object:", event.Object)
}

func TestNewEventDispatcher(t *testing.T) {
	d := NewEventDispatcher()
	if d == nil {
		t.Error("NewEventDispatcher() returned nil")
	}
}

func TestNewEvent(t *testing.T) {
	data := map[string]string{"key": "value"}
	e := NewEvent(helloWorld, data)

	if e.Type != helloWorld {
		t.Errorf("Event.Type = %s, want %s", e.Type, helloWorld)
	}
	if e.Object == nil {
		t.Error("Event.Object is nil")
	}
}

func TestNewEventListener(t *testing.T) {
	called := false
	callback := func(event Event) {
		called = true
	}

	listener := NewEventListener(callback)
	if listener == nil {
		t.Error("NewEventListener() returned nil")
	}

	// 测试回调
	listener.Handler(Event{Type: helloWorld})
	if !called {
		t.Error("Listener callback was not called")
	}
}

func TestAddAndDispatchEvent(t *testing.T) {
	d := NewEventDispatcher()
	var received Event

	listener := NewEventListener(func(event Event) {
		received = event
	})

	d.AddEventListener(helloWorld, listener)

	// 触发事件
	testData := "test data"
	d.DispatchEvent(NewEvent(helloWorld, testData))

	time.Sleep(100 * time.Millisecond) // 等待事件处理

	if received.Type != helloWorld {
		t.Errorf("Received event type = %s, want %s", received.Type, helloWorld)
	}
	// 不能直接比较 interface{}，需要类型断言
	if data, ok := received.Object.(string); !ok || data != testData {
		t.Errorf("Received event data = %v, want %v", received.Object, testData)
	}
}

func TestRemoveEventListener(t *testing.T) {
	d := NewEventDispatcher()
	callCount := int32(0)

	listener := NewEventListener(func(event Event) {
		atomic.AddInt32(&callCount, 1)
	})

	d.AddEventListener(helloWorld, listener)

	// 触发事件
	d.DispatchEvent(NewEvent(helloWorld, nil))
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("Call count before remove = %d, want 1", callCount)
	}

	// 移除监听器
	d.RemoveEventListener(helloWorld, listener)

	// 再次触发事件
	d.DispatchEvent(NewEvent(helloWorld, nil))
	time.Sleep(100 * time.Millisecond)

	// 计数应该仍然是1
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("Call count after remove = %d, want 1", callCount)
	}
}

func TestHasEventListener(t *testing.T) {
	d := NewEventDispatcher()
	listener := NewEventListener(func(event Event) {})

	if d.HasEventListener(helloWorld) {
		t.Error("HasEventListener() should return false for empty dispatcher")
	}

	d.AddEventListener(helloWorld, listener)

	if !d.HasEventListener(helloWorld) {
		t.Error("HasEventListener() should return true after adding listener")
	}
}

func TestMultipleListeners(t *testing.T) {
	d := NewEventDispatcher()
	callCount := int32(0)

	listener1 := NewEventListener(func(event Event) {
		atomic.AddInt32(&callCount, 1)
	})
	listener2 := NewEventListener(func(event Event) {
		atomic.AddInt32(&callCount, 1)
	})

	d.AddEventListener(helloWorld, listener1)
	d.AddEventListener(helloWorld, listener2)

	d.DispatchEvent(NewEvent(helloWorld, nil))
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("Multiple listeners call count = %d, want 2", callCount)
	}
}

func TestDispatchWithNoListeners(t *testing.T) {
	d := NewEventDispatcher()
	// 没有监听器时触发事件不应 panic
	result := d.DispatchEvent(NewEvent("nonexistent.event", nil))
	if result {
		t.Error("DispatchEvent() should return false when no listeners")
	}
}

func TestConcurrentDispatch(t *testing.T) {
	d := NewEventDispatcher()
	callCount := int32(0)

	d.AddEventListener(helloWorld, NewEventListener(func(event Event) {
		atomic.AddInt32(&callCount, 1)
	}))

	// 并发触发事件
	for i := 0; i < 100; i++ {
		go d.DispatchEvent(NewEvent(helloWorld, nil))
	}

	time.Sleep(500 * time.Millisecond)

	if atomic.LoadInt32(&callCount) != 100 {
		t.Errorf("Concurrent call count = %d, want 100", callCount)
	}
}
