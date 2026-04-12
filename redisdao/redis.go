package redisdao

import (
	"github.com/redis/go-redis/v9"
)

func NewSimpleRedis(instance string) *redis.Client {
	Init()
	return getInstance(instance)
}

func Close() {
	closeInstance()
}
