package redisdao

import (
	"fmt"
	"github.com/redis/go-redis/v9"
)

func NewSimpleRedis(instance string) *redis.Client {
	Init()
	return getInstance(instance)
}

func Close() {
	initMu.Lock()
	defer initMu.Unlock()
	stopped = true
	closeInstance()
}

// Client returns explicit lookup/initialization errors rather than a nil client.
func Client(name string) (*redis.Client, error) {
	if err := Initialize(); err != nil {
		return nil, err
	}
	c := getInstance(name)
	if c == nil {
		return nil, fmt.Errorf("redisdao: unknown instance %q", name)
	}
	return c, nil
}
