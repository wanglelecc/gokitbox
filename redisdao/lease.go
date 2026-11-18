package redisdao

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

var ErrLockHeld = errors.New("redis: lock held")
var ErrLeaseLost = errors.New("redis: lease lost")

type Lease struct {
	client     *redis.Client
	key, token string
	mu         sync.Mutex
	released   bool
}

func Acquire(ctx context.Context, c *redis.Client, key string, ttl time.Duration) (*Lease, error) {
	if c == nil || key == "" || ttl < time.Millisecond {
		return nil, errors.New("redis: client/key/positive millisecond ttl required")
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	l := &Lease{client: c, key: key, token: hex.EncodeToString(b[:])}
	ok, err := c.SetNX(ctx, key, l.token, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLockHeld
	}
	return l, nil
}
func (r *Redis) Acquire(ctx context.Context, key string, ttl time.Duration) (*Lease, error) {
	return Acquire(ctx, r.client, key, ttl)
}
func (l *Lease) Renew(ctx context.Context, ttl time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return ErrLeaseLost
	}
	if ttl < time.Millisecond {
		return errors.New("redis: ttl must be at least 1ms")
	}
	n, err := l.client.Eval(ctx, `if redis.call('GET',KEYS[1])==ARGV[1] then return redis.call('PEXPIRE',KEYS[1],ARGV[2]) end return 0`, []string{l.key}, l.token, ttl.Milliseconds()).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrLeaseLost
	}
	return nil
}
func (l *Lease) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return nil
	}
	n, err := l.client.Eval(ctx, `if redis.call('GET',KEYS[1])==ARGV[1] then return redis.call('DEL',KEYS[1]) end return 0`, []string{l.key}, l.token).Int()
	if err != nil {
		return err
	}
	l.released = true
	if n != 1 {
		return ErrLeaseLost
	}
	return nil
}
