package socketserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Session struct {
	draining                atomic.Bool
	server                  *Server
	wire                    wireConn
	id, remoteAddr, network string
	created                 time.Time
	ctx                     context.Context
	cancel                  context.CancelFunc
	done                    chan struct{}
	incoming                chan Message
	writeSlots              chan struct{}
	writeGate               chan struct{}
	stateMu                 sync.Mutex
	closed                  bool
	reason                  CloseReason
	closeOnce               sync.Once
	writes                  sync.WaitGroup
	lastSeen                atomic.Int64
	authenticated           atomic.Bool
	queuedBytes             atomic.Int64
	maintenanceIndex        int
	maintenanceDue          time.Time
}

func newSession(server *Server, wire wireConn, id string) *Session {
	ctx, cancel := context.WithCancel(server.runtimeCtx)
	now := time.Now()
	s := &Session{server: server, wire: wire, id: id, remoteAddr: wire.RemoteAddr().String(), network: server.options.Network, created: now, ctx: ctx, cancel: cancel, done: make(chan struct{}), incoming: make(chan Message, server.options.ReadQueueSize), writeSlots: make(chan struct{}, server.options.WriteQueueSize), writeGate: make(chan struct{}, 1), maintenanceIndex: -1}
	s.lastSeen.Store(now.UnixNano())
	s.writeGate <- struct{}{}
	return s
}
func (s *Session) ID() string               { return s.id }
func (s *Session) Network() string          { return s.network }
func (s *Session) RemoteAddr() string       { return s.remoteAddr }
func (s *Session) CreatedAt() time.Time     { return s.created }
func (s *Session) Context() context.Context { return s.ctx }
func (s *Session) Closed() bool             { s.stateMu.Lock(); defer s.stateMu.Unlock(); return s.closed }
func (s *Session) Authenticated() bool      { return s.authenticated.Load() }
func (s *Session) MarkAuthenticated() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.closed {
		return ErrClosed
	}
	s.authenticated.Store(true)
	return nil
}
func (s *Session) Reason() CloseReason { s.stateMu.Lock(); defer s.stateMu.Unlock(); return s.reason }
func (s *Session) Wait(ctx context.Context) error {
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Session) Close(reason CloseReason) { s.close(reason, true) }
func (s *Session) close(reason CloseReason, cancel bool) {
	s.closeOnce.Do(func() {
		s.stateMu.Lock()
		s.closed = true
		s.reason = reason
		s.stateMu.Unlock()
		s.server.unscheduleMaintenance(s)
		s.server.hub.remove(s)
		s.wire.Close()
	})
	if cancel {
		s.cancel()
	}
}
func (s *Session) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if message.Type != TextMessage && message.Type != BinaryMessage {
		return errors.New("socketserver: unsupported message type")
	}
	limit := s.server.options.MaxMessageBytes
	if limit > 0 && int64(len(message.Data)) > limit {
		return ErrMessageTooLarge
	}
	if !reserveBytes(&s.server.writeBytes, s.server.options.MaxWriteBytes, int64(len(message.Data))) {
		return ErrQueueFull
	}
	defer s.server.writeBytes.Add(-int64(len(message.Data)))
	select {
	case s.writeSlots <- struct{}{}:
		defer func() { <-s.writeSlots }()
	default:
		return ErrQueueFull
	}
	s.stateMu.Lock()
	if s.closed {
		s.stateMu.Unlock()
		return ErrClosed
	}
	s.writes.Add(1)
	s.stateMu.Unlock()
	defer s.writes.Done()
	callCtx, cancel := context.WithTimeout(ctx, s.server.options.WriteTimeout)
	defer cancel()
	select {
	case <-s.ctx.Done():
		return ErrClosed
	case <-callCtx.Done():
		return callCtx.Err()
	case <-s.writeGate:
	}
	defer func() { s.writeGate <- struct{}{} }()
	if s.Closed() {
		return ErrClosed
	}
	if err := callCtx.Err(); err != nil {
		return err
	}
	cancelled := make(chan struct{})
	stop := context.AfterFunc(callCtx, func() { s.Close(CloseReason{Code: "write_timeout", Err: callCtx.Err()}); close(cancelled) })
	deadline, _ := callCtx.Deadline()
	err := invoke(func() error { return s.wire.WriteMessage(message, deadline) })
	if !stop() {
		<-cancelled
	}
	if contextErr := callCtx.Err(); contextErr != nil {
		s.Close(CloseReason{Code: "write_timeout", Err: contextErr})
		return contextErr
	}
	if err != nil {
		// The socket deadline can fire before the context timer is scheduled.
		// Keep the errors.As target off the successful-write allocation path.
		var timeout net.Error
		if errors.As(err, &timeout) && timeout.Timeout() && !time.Now().Before(deadline) {
			err = context.DeadlineExceeded
		}
		s.Close(CloseReason{Code: "write_error", Err: err})
	}
	return err
}
func (s *Session) readLoop() (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("socketserver: read panic (%T)", p)
		}
	}()
	for {
		message, readErr := s.wire.ReadMessage()
		if message.Data != nil {
			size := int64(len(message.Data))
			limit := s.server.options.MaxMessageBytes
			if limit > 0 && size > limit {
				return ErrMessageTooLarge
			}
			if budget := s.server.options.ReadQueueBytes; budget > 0 && s.queuedBytes.Add(size) > budget {
				s.queuedBytes.Add(-size)
				return ErrMessageTooLarge
			} else if budget == 0 {
				s.queuedBytes.Add(size)
			}
			if !reserveBytes(&s.server.readBytes, s.server.options.MaxReadBytes, size) {
				s.queuedBytes.Add(-size)
				return ErrReadBudgetExceeded
			}
			// A custom codec may borrow its buffer. The queue always owns its bytes.
			if owner, ok := s.wire.(interface{ ownsReadBuffers() bool }); !ok || !owner.ownsReadBuffers() {
				message.Data = append([]byte{}, message.Data...)
			}
			select {
			case s.incoming <- message:
				s.lastSeen.Store(time.Now().UnixNano())
			case <-s.ctx.Done():
				s.releaseMessage(message)
				return s.ctx.Err()
			}
		}
		if readErr != nil {
			return readErr
		}
		if message.Data == nil {
			return io.ErrNoProgress
		}
	}
}
func (s *Session) consumeLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case message, ok := <-s.incoming:
			if !ok {
				return
			}
			if s.server.settings.hooks.OnMessage != nil {
				err := s.hook(func(ctx context.Context) error { return s.server.settings.hooks.OnMessage(ctx, s, message) })
				s.releaseMessage(message)
				if err != nil {
					s.server.report(s, "message", err)
					var panicErr *hookPanic
					if s.server.options.CloseOnHandlerError || errors.As(err, &panicErr) {
						s.Close(CloseReason{Code: "handler_error", Err: err})
						return
					}
				}
			} else {
				s.releaseMessage(message)
			}
		}
	}
}
func (s *Session) hook(fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.server.options.HookTimeout)
	defer cancel()
	err := invoke(func() error { return fn(ctx) })
	if err == nil {
		err = ctx.Err()
	}
	return err
}
func (s *Session) run() {
	defer func() {
		if p := recover(); p != nil {
			s.server.report(s, "session", fmt.Errorf("socketserver: session panic (%T)", p))
			s.Close(CloseReason{Code: "internal_error"})
		}
		s.Close(s.Reason())
		s.writes.Wait()
		for message := range s.incoming {
			s.releaseMessage(message)
		}
		if hook := s.server.settings.hooks.OnDisconnect; hook != nil {
			ctx, cancel := s.server.closeHookContext(s)
			err := invoke(func() error { return hook(ctx, s, s.Reason()) })
			cancel()
			if err != nil {
				s.server.report(s, "disconnect", err)
				s.server.recordCleanup(err)
			}
		}
		close(s.done)
		s.server.finishSession(s)
	}()
	if err := s.connect(); err != nil {
		s.server.report(s, "connect", err)
		s.Close(CloseReason{Code: "connect_error", Err: err})
		close(s.incoming)
		return
	}
	s.server.scheduleMaintenance(s, true)

	if s.Closed() {
		close(s.incoming)
		return
	}
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		err := s.readLoop()
		s.endRead(err)
		close(s.incoming)
	}()
	// The session owner consumes messages itself; only reads need another goroutine.
	s.consumeLoop()
	<-readDone

}

func (s *Session) connect() error {
	defer s.server.finishAdmission()
	if hook := s.server.settings.hooks.OnConnect; hook != nil {
		return s.hook(func(ctx context.Context) error { return hook(ctx, s) })
	}
	return nil
}
func (s *Session) releaseMessage(m Message) {
	size := int64(len(m.Data))
	s.queuedBytes.Add(-size)
	s.server.readBytes.Add(-size)
}

func (s *Session) endRead(err error) {
	if !s.Closed() {
		code := "read_error"
		if errors.Is(err, io.EOF) {
			code = "remote_closed"
		}
		// Remove from lookup immediately, but let already accepted frames finish.
		// TCP half-close may still need responses written by the message hook.
		s.draining.Store(true)
		s.server.unscheduleMaintenance(s)
		s.server.hub.remove(s)
		s.stateMu.Lock()
		if !s.closed {
			s.reason = CloseReason{Code: code, Err: err}
		}
		s.stateMu.Unlock()
	}
}
