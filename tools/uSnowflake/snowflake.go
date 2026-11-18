package uSnowflake

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

// Window is an exclusive timestamp reservation in a durable coordination domain.
type Window struct {
	Node, Start, End int64
	ValidFor         time.Duration
}
type Reservation interface {
	Reserve(context.Context) (Window, error)
	Close(context.Context) error
}
type WindowProvider interface {
	Acquire(context.Context, string, string) (Reservation, error)
}

var ErrUnavailable = errors.New("snowflake: generator unavailable or lease lost")

type Generator struct {
	stop, done     chan struct{}
	stopOnce       sync.Once
	mu             sync.Mutex
	reservation    Reservation
	window         Window
	until          time.Time
	last, sequence int64
	closed         bool
}

const epoch = int64(1288834974657)

func New(ctx context.Context, p WindowProvider, project, service string) (*Generator, error) {
	if p == nil {
		return nil, ErrUnavailable
	}
	lease, err := p.Acquire(ctx, project, service)
	if err != nil {
		return nil, err
	}
	g := &Generator{reservation: lease, last: -1, stop: make(chan struct{}), done: make(chan struct{})}
	if err = g.reserve(ctx); err != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		return nil, errors.Join(err, lease.Close(cleanup))
	}
	go g.keepalive()
	return g, nil
}
func (g *Generator) reserve(ctx context.Context) error {
	start := time.Now()
	w, err := g.reservation.Reserve(ctx)
	if err != nil {
		return err
	}
	if w.Node < 0 || w.Node > 1023 || w.Start < epoch || w.End < w.Start || w.End-epoch >= (1<<41) || w.ValidFor <= 0 {
		return ErrUnavailable
	}
	if g.window.End != 0 && (w.Node != g.window.Node || w.Start <= g.window.End) {
		return errors.New("snowflake: reservation overlaps prior high-water mark")
	}
	g.window = w
	g.until = start.Add(w.ValidFor)
	return nil
}
func (g *Generator) Next(ctx context.Context) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if g.closed || !time.Now().Before(g.until) {
		return 0, ErrUnavailable
	}
	now := time.Now().UnixMilli()
	if now < g.last {
		g.closed = true
		return 0, errors.New("snowflake: wall clock moved backwards")
	}
	stamp := max(now, g.window.Start)
	sequence := int64(0)
	if stamp == g.last {
		sequence = g.sequence + 1
		if sequence > 4095 {
			stamp++
			sequence = 0
		}
	}
	if stamp > g.window.End {
		if err := g.reserve(ctx); err != nil {
			g.closed = true
			return 0, err
		}
		if stamp < g.window.Start {
			stamp = g.window.Start
			sequence = 0
		}
	}
	if stamp > g.window.End || !time.Now().Before(g.until) {
		return 0, ErrUnavailable
	}
	if wait := stamp - time.Now().UnixMilli(); wait > 0 {
		timer := time.NewTimer(time.Duration(wait) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if !time.Now().Before(g.until) {
		return 0, ErrUnavailable
	}
	// Commit sequence state only once an ID is returned. Cancellation at rollover
	// must never reset the previous millisecond's sequence and reissue old IDs.
	g.last, g.sequence = stamp, sequence
	return (stamp-epoch)<<22 | g.window.Node<<12 | sequence, nil
}
func (g *Generator) keepalive() {
	defer close(g.done)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-g.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			g.mu.Lock()
			if !g.closed {
				if err := g.reserve(ctx); err != nil {
					g.closed = true
				}
			}
			closed := g.closed
			g.mu.Unlock()
			cancel()
			if closed {
				return
			}
		}
	}
}
func (g *Generator) Close(ctx context.Context) error {
	g.stopOnce.Do(func() { close(g.stop) })
	select {
	case <-g.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	return g.reservation.Close(ctx)
}

var defaultMu sync.RWMutex
var defaultGenerator *Generator

func Initialize(ctx context.Context, p WindowProvider, project, service string) error {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultGenerator != nil {
		return errors.New("snowflake: already initialized")
	}
	g, err := New(ctx, p, project, service)
	if err == nil {
		defaultGenerator = g
	}
	return err
}

// InitSnowflake is a fail-closed compatibility bridge; random fallback is removed.
func InitSnowflake(ctx context.Context, p NodeProvider, project, service string) {
	wp, ok := p.(WindowProvider)
	if !ok {
		panic("snowflake: provider must support exclusive timestamp reservations")
	}
	if err := Initialize(ctx, wp, project, service); err != nil {
		panic(err)
	}
}
func NextInt64(ctx context.Context) (int64, error) {
	defaultMu.RLock()
	g := defaultGenerator
	defaultMu.RUnlock()
	if g == nil {
		return 0, ErrUnavailable
	}
	return g.Next(ctx)
}
func NextString(ctx context.Context) (string, error) {
	v, err := NextInt64(ctx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(v, 10), nil
}
func NewIdInt64() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	v, err := NextInt64(ctx)
	if err != nil {
		panic(err)
	}
	return v
}
func NewIdString() string { return strconv.FormatInt(NewIdInt64(), 10) }
func Close(ctx context.Context) error {
	defaultMu.RLock()
	g := defaultGenerator
	defaultMu.RUnlock()
	if g == nil {
		return nil
	}
	return g.Close(ctx)
}

// Wait surfaces renewal failure to an application lifecycle supervisor.
func (g *Generator) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-g.done:
		select {
		case <-g.stop:
			return nil
		default:
			return ErrUnavailable
		}
	}
}
func Wait() error {
	defaultMu.RLock()
	g := defaultGenerator
	defaultMu.RUnlock()
	if g == nil {
		return ErrUnavailable
	}
	return g.Wait(context.Background())
}
