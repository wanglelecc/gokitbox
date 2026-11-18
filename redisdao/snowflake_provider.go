package redisdao

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/wanglelecc/gokitbox/tools/uSnowflake"
	"strconv"
	"strings"
	"time"
)

type RedisNodeProvider struct {
	client    *redis.Client
	keyPrefix string
}

func NewRedisNodeProvider() *RedisNodeProvider {
	return NewRedisNodeProviderWithClient(NewSimpleRedis("redis"))
}
func NewRedisNodeProviderWithClient(c *redis.Client) *RedisNodeProvider {
	return &RedisNodeProvider{client: c, keyPrefix: "gokitbox:snowflake:v2"}
}
func (p *RedisNodeProvider) key(project string) (string, error) {
	if p.client == nil || project == "" || strings.ContainsAny(project, "{}") {
		return "", errors.New("snowflake: client and valid project required")
	}
	return p.keyPrefix + ":{" + project + "}", nil
}

// InitializePool is an explicit first-deployment operation. Never reset an existing pool.
func (p *RedisNodeProvider) InitializePool(ctx context.Context, project string) error {
	k, err := p.key(project)
	if err != nil {
		return err
	}
	return p.client.HSetNX(ctx, k, "initialized", "1").Err()
}

const acquireNodeScript = `
if redis.call('HGET',KEYS[1],'initialized')~='1' then return redis.error_reply('ID_POOL_NOT_INITIALIZED') end
local t=redis.call('TIME');local now=t[1]*1000+math.floor(t[2]/1000)
for node=0,1023 do
 local untilAt=tonumber(redis.call('HGET',KEYS[1],'until:'..node) or '0')
 if untilAt<=now then
 redis.call('HSET',KEYS[1],'owner:'..node,ARGV[1],'until:'..node,now+10000)
 return node
 end
end
return redis.error_reply('ID_POOL_EXHAUSTED')`
const reserveNodeScript = `
if redis.call('HGET',KEYS[1],'initialized')~='1' then return redis.error_reply('ID_POOL_NOT_INITIALIZED') end
local t=redis.call('TIME');local now=t[1]*1000+math.floor(t[2]/1000)
if redis.call('HGET',KEYS[1],'owner:'..ARGV[1])~=ARGV[2] or tonumber(redis.call('HGET',KEYS[1],'until:'..ARGV[1]) or '0')<=now then return redis.error_reply('ID_LEASE_LOST') end
local start=math.max(now,tonumber(redis.call('HGET',KEYS[1],'high:'..ARGV[1]) or '0')+1)
if start>now+2000 then return redis.error_reply('ID_CLOCK_OR_RATE_LIMIT') end
local finish=start+999
redis.call('HSET',KEYS[1],'high:'..ARGV[1],finish,'until:'..ARGV[1],now+10000)
return {start,finish}`

func (p *RedisNodeProvider) Acquire(ctx context.Context, project, service string) (uSnowflake.Reservation, error) {
	k, err := p.key(project)
	if err != nil {
		return nil, err
	}
	var b [24]byte
	if _, err = rand.Read(b[:]); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(b[:])
	node, err := p.client.Eval(ctx, acquireNodeScript, []string{k}, token).Int64()
	if err != nil {
		return nil, fmt.Errorf("snowflake acquire: %w", err)
	}
	return &nodeReservation{client: p.client, key: k, token: token, node: node}, nil
}

type nodeReservation struct {
	client     *redis.Client
	key, token string
	node       int64
}

func (l *nodeReservation) Reserve(ctx context.Context) (uSnowflake.Window, error) {
	values, err := l.client.Eval(ctx, reserveNodeScript, []string{l.key}, l.node, l.token).Slice()
	if err != nil {
		return uSnowflake.Window{}, err
	}
	if len(values) != 2 {
		return uSnowflake.Window{}, errors.New("snowflake: invalid reservation response")
	}
	start, e := strconv.ParseInt(fmt.Sprint(values[0]), 10, 64)
	if e != nil {
		return uSnowflake.Window{}, e
	}
	end, e := strconv.ParseInt(fmt.Sprint(values[1]), 10, 64)
	return uSnowflake.Window{Node: l.node, Start: start, End: end, ValidFor: 10 * time.Second}, e
}
func (l *nodeReservation) Close(ctx context.Context) error {
	return l.client.Eval(ctx, `if redis.call('HGET',KEYS[1],'owner:'..ARGV[1])==ARGV[2] then redis.call('HDEL',KEYS[1],'owner:'..ARGV[1],'until:'..ARGV[1]);return 1 end return 0`, []string{l.key}, l.node, l.token).Err()
}

// Deprecated: legacy allocation is bounded and non-recycling; new code uses Acquire.
func (p *RedisNodeProvider) MakeNode(ctx context.Context, project, service string) (int64, error) {
	return 0, errors.New("snowflake: bare node allocation retired; use Acquire timestamp reservations")
}
