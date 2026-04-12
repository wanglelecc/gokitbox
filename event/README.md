# Event 事件分发器

Event 是一个简单的事件分发器，用于解耦业务逻辑，将主流程和附属逻辑分离。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/event
```

## 特性

- 同步/异步事件监听
- 支持多监听器
- 动态添加/移除监听器
- 线程安全

## 使用示例

```go
package main

import (
    "fmt"
    "time"
    
    e "github.com/wanglelecc/gokitbox/event"
)

const (
    OrderCreated = "order.created"
    UserLoggedIn = "user.logged_in"
)

func main() {
    // 创建事件分发器
    dispatcher := e.NewEventDispatcher()
    
    // 创建监听器
    orderListener := e.NewEventListener(func(event e.Event) {
        fmt.Printf("[Order] 类型: %s, 数据: %+v\n", event.Type, event.Object)
    })
    
    // 添加监听器
    dispatcher.AddEventListener(OrderCreated, orderListener)
    
    // 也可以直接定义回调函数
    dispatcher.AddEventListener(UserLoggedIn, e.NewEventListener(func(event e.Event) {
        user := event.Object.(User)
        fmt.Printf("[Login] 用户 %s 登录成功\n", user.Name)
    }))
    
    // 触发事件
    dispatcher.DispatchEvent(e.NewEvent(OrderCreated, Order{ID: "ORD001", Amount: 100}))
    dispatcher.DispatchEvent(e.NewEvent(UserLoggedIn, User{ID: 1, Name: "张三"}))
    
    // 异步触发（非阻塞）
    go dispatcher.DispatchEvent(e.NewEvent(OrderCreated, Order{ID: "ORD002", Amount: 200}))
    
    time.Sleep(time.Second)
    
    // 移除监听器
    dispatcher.RemoveEventListener(OrderCreated, orderListener)
    
    // 再次触发（orderListener 不会再收到）
    dispatcher.DispatchEvent(e.NewEvent(OrderCreated, Order{ID: "ORD003", Amount: 300}))
}

type Order struct {
    ID     string
    Amount float64
}

type User struct {
    ID   int
    Name string
}
```

## API 说明

```go
// 创建事件分发器
func NewEventDispatcher() *EventDispatcher

// 创建事件
func NewEvent(eventType string, object interface{}) Event

// 创建事件监听器
func NewEventListener(callback func(Event)) *EventListener

// 添加事件监听器
func (d *EventDispatcher) AddEventListener(eventType string, listener *EventListener)

// 移除事件监听器
func (d *EventDispatcher) RemoveEventListener(eventType string, listener *EventListener)

// 分发事件
func (d *EventDispatcher) DispatchEvent(event Event)
```

## 使用场景

- 订单创建后发送通知
- 用户注册后初始化数据
- 操作日志记录
- 缓存刷新
- 业务解耦
