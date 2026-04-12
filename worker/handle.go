package worker

import (
	"context"
	"fmt"
	"sync"
)

// 处理器
type Handle func(ctx context.Context, v []byte) (ret bool, err error)

var (
	handleLock sync.RWMutex
	handleMap  = make(map[string]Handle)
)

// 注册处理器
func RegisterHandle(name string, handle Handle) {
	handleLock.Lock()
	defer handleLock.Unlock()

	handleMap[name] = handle
}

// 批量注册处理器
func BatchRegisterHandle(handles map[string]Handle) {
	handleLock.Lock()
	defer handleLock.Unlock()
	for name, handle := range handles {
		handleMap[name] = handle
	}
}

// 获取处理器
func GetHandle(name string) (Handle, error) {
	handleLock.RLock()
	defer handleLock.RUnlock()

	if h, ok := handleMap[name]; ok {
		return h, nil
	}

	return nil, fmt.Errorf("handle is nil. name:%s", name)
}
