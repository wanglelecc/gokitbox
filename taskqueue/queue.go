// Package taskqueue implements at-least-once Redis tasks with persistent ownership.
// The coordinator must retain committed writes. Handlers must be idempotent.
package taskqueue

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

var ErrLeaseLost = errors.New("taskqueue: lease lost")
var ErrEmpty = redis.Nil
var ErrDuplicate = errors.New("taskqueue: task id already pending")

type Message struct {
	ID      string
	Payload []byte
	Attempt int
	token   string
}
type Queue struct {
	client     *redis.Client
	source     string
	kind       string
	keys       []string
	visibility time.Duration
}

func New(c *redis.Client, source, kind string, visibility time.Duration) (*Queue, error) {
	if c == nil || source == "" || (kind != "list" && kind != "delay") || visibility < 30*time.Millisecond || visibility > 24*time.Hour {
		return nil, errors.New("taskqueue: invalid client/source/kind/visibility")
	}
	digest := sha256.Sum256([]byte(kind + "\x00" + source))
	prefix := "gokitbox:tasks:v1:" + hex.EncodeToString(digest[:]) + ":"
	keys := []string{source}
	for _, suffix := range []string{"ready", "data", "active", "owner", "tokens", "attempts", "scheduled", "dead"} {
		keys = append(keys, prefix+suffix)
	}
	return &Queue{client: c, source: source, kind: kind, keys: keys, visibility: visibility}, nil
}
func token() (string, error) {
	var b [24]byte
	_, err := rand.Read(b[:])
	return hex.EncodeToString(b[:]), err
}

// Enqueue accepts stable caller IDs; zero due means ready immediately. Duplicate pending IDs are rejected.
func (q *Queue) Enqueue(ctx context.Context, m Message, due time.Time) error {
	if m.ID == "" || len(m.ID) > 256 || len(m.Payload) > 1<<20 {
		return errors.New("taskqueue: id required (<=256 bytes), payload <=1MiB")
	}
	n, err := q.client.Eval(ctx, enqueueScript, q.keys, m.ID, m.Payload, due.UnixMilli(), due.IsZero()).Int()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrDuplicate
	}
	return nil
}
func (q *Queue) Claim(ctx context.Context) (Message, error) {
	t, err := token()
	if err != nil {
		return Message{}, err
	}
	result, err := q.client.Eval(ctx, claimScript, q.keys, t, q.visibility.Milliseconds(), q.kind).Slice()
	if err != nil {
		return Message{}, err
	}
	if len(result) != 3 {
		return Message{}, errors.New("taskqueue: malformed claim")
	}
	attempt, err := strconv.Atoi(fmt.Sprint(result[2]))
	return Message{ID: fmt.Sprint(result[0]), Payload: []byte(fmt.Sprint(result[1])), Attempt: attempt, token: t}, err
}
func (q *Queue) act(ctx context.Context, m Message, action, target string) error {
	keys := append([]string(nil), q.keys...)
	keys = append(keys, target)
	if action == "fail" {
		destination, err := New(q.client, target, q.kind, q.visibility)
		if err != nil {
			return err
		}
		keys = append(keys, destination.keys[1:]...)
	}
	n, err := q.client.Eval(ctx, actionScript, keys, m.ID, m.token, action, q.visibility.Milliseconds(), q.kind).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrLeaseLost
	}
	return nil
}
func (q *Queue) Ack(ctx context.Context, m Message) error   { return q.act(ctx, m, "ack", q.source) }
func (q *Queue) Retry(ctx context.Context, m Message) error { return q.act(ctx, m, "retry", q.source) }
func (q *Queue) Renew(ctx context.Context, m Message) error { return q.act(ctx, m, "renew", q.source) }

// Fail atomically forwards a stable ID to another reliable queue, or retains an internal dead letter.
func (q *Queue) Fail(ctx context.Context, m Message, target string) error {
	if target == "" {
		return q.act(ctx, m, "dead", q.source)
	}
	if target == q.source {
		return errors.New("taskqueue: failure source must differ")
	}
	return q.act(ctx, m, "fail", target)
}
func (q *Queue) DeadLetters(ctx context.Context, start, stop int64) ([]Message, error) {
	ids, err := q.client.LRange(ctx, q.keys[8], start, stop).Result()
	if err != nil {
		return nil, err
	}
	messages := make([]Message, 0, len(ids))
	for _, id := range ids {
		payload, err := q.client.HGet(ctx, q.keys[2], id).Bytes()
		if err != nil {
			return nil, err
		}
		messages = append(messages, Message{ID: id, Payload: payload})
	}
	return messages, nil
}
func (q *Queue) RequeueDead(ctx context.Context, id string) error {
	n, err := q.client.Eval(ctx, `if redis.call('LREM',KEYS[9],1,ARGV[1])==0 then return 0 end redis.call('RPUSH',KEYS[2],ARGV[1]);return 1`, q.keys, id).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrEmpty
	}
	return nil
}
func (q *Queue) DeleteDead(ctx context.Context, id string) error {
	n, err := q.client.Eval(ctx, `if redis.call('LREM',KEYS[9],1,ARGV[1])==0 then return 0 end redis.call('HDEL',KEYS[3],ARGV[1]);redis.call('HDEL',KEYS[7],ARGV[1]);return 1`, q.keys, id).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrEmpty
	}
	return nil
}

type contextKey struct{}

func WithTaskID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}
func TaskID(ctx context.Context) string { id, _ := ctx.Value(contextKey{}).(string); return id }

// Stats is an observational snapshot; message transitions remain atomic in Lua.
type Stats struct{ Stored, Ready, Active, Scheduled, Dead, LegacySource int64 }

func (q *Queue) Stats(ctx context.Context) (Stats, error) {
	pipe := q.client.Pipeline()
	stored := pipe.HLen(ctx, q.keys[2])
	ready := pipe.LLen(ctx, q.keys[1])
	active := pipe.ZCard(ctx, q.keys[3])
	scheduled := pipe.ZCard(ctx, q.keys[7])
	dead := pipe.LLen(ctx, q.keys[8])
	var source *redis.IntCmd
	if q.kind == "list" {
		source = pipe.LLen(ctx, q.source)
	} else {
		source = pipe.ZCard(ctx, q.source)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return Stats{}, err
	}
	return Stats{stored.Val(), ready.Val(), active.Val(), scheduled.Val(), dead.Val(), source.Val()}, nil
}
