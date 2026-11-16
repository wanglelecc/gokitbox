package socketserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wanglelecc/gokitbox/logger"
)

type Server struct {
	mu                                  sync.Mutex
	options                             Options
	settings                            settings
	hub                                 *Hub
	state                               string
	started                             bool
	listener                            net.Listener
	httpServer                          *http.Server
	runtimeCtx                          context.Context
	runtimeCancel                       context.CancelFunc
	startCancel                         context.CancelFunc
	startDone, waitDone, stopDone       chan struct{}
	startOnce, waitOnce, stopOnce       sync.Once
	sessions                            map[*Session]struct{}
	pending                             map[wireConn]struct{}
	sessionsWG, workersWG               sync.WaitGroup
	maintenance                         chan *Session
	maintenanceMu                       sync.Mutex
	maintenanceHeap                     maintenanceHeap
	maintenanceWake                     chan struct{}
	admitting                           int
	readBytes, writeBytes               atomic.Int64
	maintenanceCalls, maintenanceErrors atomic.Uint64
	maintenanceMaxDelay                 atomic.Int64
	waitErr, stopErr                    error
	shutdownCtx                         context.Context
}

func NewServerWithOptions(o Options, extensions ...Option) (*Server, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	settings := settings{id: randomID}
	for _, apply := range extensions {
		if apply == nil {
			return nil, errors.New("socketserver: option is nil")
		}
		apply(&settings)
	}
	if settings.id == nil {
		return nil, errors.New("socketserver: ID generator is required")
	}
	if o.TCPFraming == "custom" && settings.codec == nil {
		return nil, errors.New("socketserver: custom framing requires a codec factory")
	}
	o.AllowedOrigins = append([]string(nil), o.AllowedOrigins...)
	return &Server{options: o, settings: settings, hub: newHub(), state: "new", startDone: make(chan struct{}), waitDone: make(chan struct{}), stopDone: make(chan struct{}), sessions: map[*Session]struct{}{}, pending: map[wireConn]struct{}{}, maintenanceWake: make(chan struct{}, 1)}, nil
}
func (s *Server) Connections() *Hub { return s.hub }
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}
func (s *Server) Start(parent context.Context) (err error) {
	s.mu.Lock()
	if s.state != "new" {
		s.mu.Unlock()
		return errors.New("socketserver: already started or stopped")
	}
	ctx, cancel := context.WithCancel(parent)
	s.startCancel = cancel
	s.started = true
	s.state = "starting"
	s.mu.Unlock()
	defer cancel()
	defer s.startOnce.Do(func() { close(s.startDone) })
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("socketserver: startup panic (%T)", p)
		}
		if err != nil {
			s.mu.Lock()
			listener := s.listener
			s.mu.Unlock()
			if listener != nil {
				listener.Close()
			}
			s.finishServe(err)
		}
	}()
	if err = ctx.Err(); err != nil {
		return err
	}
	var httpServer *http.Server
	if s.options.Network == "websocket" {
		mux := http.NewServeMux()
		mux.HandleFunc(s.options.WebSocketPath, s.websocketHandler)
		if s.settings.routes != nil {
			if err = invoke(func() error { s.settings.routes(mux); return nil }); err != nil {
				return err
			}
		}
		httpServer = &http.Server{Handler: mux, ReadHeaderTimeout: s.options.HookTimeout}
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.options.Addr)
	if err != nil {
		return fmt.Errorf("socketserver: bind: %w", err)
	}
	s.mu.Lock()
	s.listener = listener
	if s.state != "starting" || ctx.Err() != nil {
		s.mu.Unlock()
		return context.Canceled
	}
	s.runtimeCtx, s.runtimeCancel = context.WithCancel(context.WithoutCancel(parent))
	s.httpServer = httpServer
	s.maintenance = make(chan *Session)
	s.state = "running"
	s.workersWG.Add(2 + s.options.MaintenanceWorkers)
	for i := 0; i < s.options.MaintenanceWorkers; i++ {
		go s.maintenanceWorker()
	}
	go s.supervise()
	go s.dispatchMaintenance()
	s.mu.Unlock()
	go func() {
		err := invoke(func() error {
			if s.httpServer != nil {
				return s.httpServer.Serve(listener)
			}
			return s.serveTCP(listener)
		})
		s.finishServe(err)
	}()
	return nil
}
func (s *Server) finishServe(err error) {
	s.mu.Lock()
	expected := s.state == "stopping" || s.state == "stopped"
	if expected && (errors.Is(err, net.ErrClosed) || errors.Is(err, http.ErrServerClosed)) {
		err = nil
	}
	s.waitErr = err
	s.mu.Unlock()
	s.waitOnce.Do(func() { close(s.waitDone) })
}
func (s *Server) Wait() error {
	s.mu.Lock()
	started := s.started
	s.mu.Unlock()
	if !started {
		return errors.New("socketserver: not started")
	}
	<-s.waitDone
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waitErr
}
func (s *Server) admit(wire wireConn) error {
	s.mu.Lock()
	if s.state != "running" {
		s.mu.Unlock()
		return ErrClosed
	}
	if limit := s.options.MaxConnections; limit > 0 && len(s.sessions)+len(s.pending) >= limit {
		s.mu.Unlock()
		return ErrTooManyConnections
	}
	if limit := s.options.MaxPendingConnections; limit > 0 && s.admitting >= limit {
		s.mu.Unlock()
		return ErrTooManyConnections
	}
	s.admitting++
	s.pending[wire] = struct{}{}
	s.sessionsWG.Add(1)
	s.mu.Unlock()
	go s.takeOwnership(wire)
	return nil
}
func (s *Server) takeOwnership(wire wireConn) {
	var id string
	err := invoke(func() error { var err error; id, err = s.settings.id(); return err })
	s.mu.Lock()
	delete(s.pending, wire)
	if err != nil || id == "" || s.state != "running" {
		s.admitting--
		s.mu.Unlock()
		wire.Close()
		s.sessionsWG.Done()
		if err == nil {
			err = ErrClosed
		}
		s.report(nil, "identify", err)
		return
	}
	session := newSession(s, wire, id)
	if err = s.hub.add(session); err != nil {
		s.admitting--
		s.mu.Unlock()
		wire.Close()
		session.cancel()
		s.sessionsWG.Done()
		s.report(session, "identify", err)
		return
	}
	s.sessions[session] = struct{}{}
	s.mu.Unlock()
	session.run()
}
func (s *Server) finishSession(session *Session) {
	s.mu.Lock()
	delete(s.sessions, session)
	s.mu.Unlock()
	s.sessionsWG.Done()
}
func (s *Server) report(session *Session, operation string, err error) {
	if err == nil {
		return
	}
	defer func() { recover() }()
	ctx := context.Background()
	if session != nil {
		ctx = session.ctx
	}
	event := ErrorEvent{Session: session, Operation: operation, Err: err}
	if s.settings.hooks.OnError != nil {
		s.settings.hooks.OnError(ctx, event)
	} else {
		logger.Wx(ctx, "socketserver", operation, "error", err)
	}
}
func (s *Server) recordCleanup(err error) {
	s.mu.Lock()
	if s.state == "stopping" {
		s.stopErr = errors.Join(s.stopErr, err)
	}
	s.mu.Unlock()
}
func (s *Server) closeHookContext(session *Session) (context.Context, context.CancelFunc) {
	s.mu.Lock()
	ctx := s.shutdownCtx
	s.mu.Unlock()
	if ctx == nil {
		ctx = context.WithoutCancel(session.ctx)
	}
	return context.WithTimeout(ctx, s.options.HookTimeout)
}
func (s *Server) Stop(ctx context.Context) error {
	waitCtx, cancel := context.WithTimeout(ctx, s.options.ShutdownTimeout)
	defer cancel()
	s.stopOnce.Do(func() {
		s.mu.Lock()
		wasNew := s.state == "new"
		s.state = "stopping"
		startCancel := s.startCancel
		s.mu.Unlock()
		if startCancel != nil {
			startCancel()
		}
		if wasNew {
			s.startOnce.Do(func() { close(s.startDone) })
			s.waitOnce.Do(func() { close(s.waitDone) })
		}
		go s.shutdown(ctx)
	})
	select {
	case <-s.stopDone:
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.stopErr
	case <-waitCtx.Done():
		return waitCtx.Err()
	}
}
func (s *Server) shutdown(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, s.options.ShutdownTimeout)
	defer cancel()
	select {
	case <-s.startDone:
	default:
		select {
		case <-s.startDone:
		case <-ctx.Done():
			s.mu.Lock()
			s.stopErr = errors.Join(s.stopErr, ctx.Err())
			s.state = "stopped"
			s.mu.Unlock()
			close(s.stopDone)
			return
		}
	}
	s.mu.Lock()
	s.shutdownCtx = ctx
	listener, httpServer, runtimeCancel := s.listener, s.httpServer, s.runtimeCancel
	sessions := make([]*Session, 0, len(s.sessions))
	for session := range s.sessions {
		sessions = append(sessions, session)
	}
	pending := make([]wireConn, 0, len(s.pending))
	for conn := range s.pending {
		pending = append(pending, conn)
	}
	s.mu.Unlock()
	if listener != nil {
		listener.Close()
	}
	if httpServer != nil {
		httpServer.Close()
	}
	if runtimeCancel != nil {
		runtimeCancel()
	}
	for _, wire := range pending {
		wire.Close()
	}
	for _, session := range sessions {
		session.Close(CloseReason{Code: "server_shutdown"})
	}
	done := make(chan struct{})
	go func() { s.sessionsWG.Wait(); s.workersWG.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		s.mu.Lock()
		s.stopErr = errors.Join(s.stopErr, ctx.Err())
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.state = "stopped"
	s.mu.Unlock()
	close(s.stopDone)
}
func (s *Server) supervise() {
	defer s.workersWG.Done()
	ticker := time.NewTicker(s.options.CheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.runtimeCtx.Done():
			return
		case now := <-ticker.C:
			s.mu.Lock()
			sessions := make([]*Session, 0, len(s.sessions))
			for session := range s.sessions {
				sessions = append(sessions, session)
			}
			s.mu.Unlock()
			for _, session := range sessions {
				if session.Closed() || session.draining.Load() {
					continue
				}
				if idle := s.options.IdleTimeout; idle > 0 && now.Sub(time.Unix(0, session.lastSeen.Load())) >= idle {
					session.Close(CloseReason{Code: "idle_timeout"})
					continue
				}
				if login := s.options.LoginTimeout; login > 0 && !session.Authenticated() && now.Sub(session.created) >= login {
					session.Close(CloseReason{Code: "login_timeout"})
					continue
				}

			}
		}
	}
}
func (s *Server) finishAdmission() {
	s.mu.Lock()
	s.admitting--
	s.mu.Unlock()
}
