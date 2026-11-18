package redisdao

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

type ReplayStore struct {
	Client *redis.Client
	Prefix string
}

func (s ReplayStore) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if s.Client == nil || s.Prefix == "" || ttl <= 0 {
		return false, errors.New("redis: invalid replay store")
	}
	operation, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.Client.SetNX(operation, s.Prefix+":"+key, "1", ttl).Result()
}
