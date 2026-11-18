package failmode

import (
	"context"
	"errors"
	"sync"

	"github.com/wanglelecc/gokitbox/logger"
)

type InitFailMode func() (FailMode, error)

var (
	registryMu      sync.RWMutex
	initFailModeMap = make(map[string]InitFailMode, 0)
)

func AddFailMode(failMode string, init InitFailMode) {
	registryMu.Lock()
	defer registryMu.Unlock()
	initFailModeMap[failMode] = init
}

type FailMode interface {
	Do(chan<- []byte, []byte, []byte, []interface{})
}

func GetFailMode(mqType string, failMode string) (FailMode, error) {
	// fmt.Printf("-------mqtype is %v ---------fail mode is %v--------\n", mqType, failMode)
	ctx := context.Background()
	tag := "GetFailMode"

	registryMu.RLock()
	if len(initFailModeMap) == 0 {
		logger.Wx(ctx, tag, " fail mode is empty ")
	}

	init := initFailModeMap[failMode]
	registryMu.RUnlock()
	if init != nil {
		f, err := init()
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	return nil, errors.New("unsupported fail mode")
}
