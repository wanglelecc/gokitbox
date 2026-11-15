package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	mu                             sync.Mutex
	options                        Options
	extensions                     []grpc.ServerOption
	registrations                  []registration
	state                          string
	native                         *grpc.Server
	health                         *health.Server
	listener                       net.Listener
	startCancel                    context.CancelFunc
	startDone, serveDone, stopDone chan struct{}
	serveOnce, startOnce, stopOnce sync.Once
	serveErr, stopErr              error
	started                        bool
}

func NewServerWithOptions(o Options, extensions ...grpc.ServerOption) (*Server, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	o.UnaryInterceptors = append([]grpc.UnaryServerInterceptor(nil), o.UnaryInterceptors...)
	o.StreamInterceptors = append([]grpc.StreamServerInterceptor(nil), o.StreamInterceptors...)
	return &Server{options: o, extensions: append([]grpc.ServerOption(nil), extensions...), state: "new", startDone: make(chan struct{}), serveDone: make(chan struct{}), stopDone: make(chan struct{})}, nil
}

// Start binds synchronously and owns no process signals. Its context only bounds startup.
func (s *Server) Start(parent context.Context) (err error) {
	s.mu.Lock()
	if s.state != "new" {
		s.mu.Unlock()
		return errors.New("grpcserver: already started or stopped")
	}
	ctx, cancel := context.WithCancel(parent)
	s.startCancel = cancel
	s.state = "starting"
	s.started = true
	s.mu.Unlock()
	defer cancel()
	defer s.startOnce.Do(func() { close(s.startDone) })
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("grpcserver: startup panic (%T)", p)
		}
		if err != nil {
			s.mu.Lock()
			native, listener := s.native, s.listener
			s.state = "stopped"
			s.serveErr = err
			s.mu.Unlock()
			if native != nil {
				native.Stop()
			}
			if listener != nil {
				listener.Close()
			}
			s.serveOnce.Do(func() { close(s.serveDone) })
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	options := []grpc.ServerOption{grpc.MaxRecvMsgSize(s.options.MaxRecvMsgSize), grpc.MaxSendMsgSize(s.options.MaxSendMsgSize), grpc.ChainUnaryInterceptor(append([]grpc.UnaryServerInterceptor{s.unary}, s.options.UnaryInterceptors...)...), grpc.ChainStreamInterceptor(append([]grpc.StreamServerInterceptor{s.stream}, s.options.StreamInterceptors...)...)}
	if s.options.TLS.Enabled {
		tlsConfig, err := s.options.TLS.load()
		if err != nil {
			return err
		}
		options = append(options, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	options = append(options, s.extensions...)
	native := grpc.NewServer(options...)
	s.mu.Lock()
	s.native = native
	s.mu.Unlock()
	for _, entry := range s.registrations {
		native.RegisterService(&entry.desc, entry.implementation)
	}
	if s.options.Health {
		h := health.NewServer()
		h.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		healthpb.RegisterHealthServer(native, h)
		s.mu.Lock()
		s.health = h
		s.mu.Unlock()
	}
	if s.options.Reflection {
		reflection.Register(native)
	}
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", s.options.Addr)
	if err != nil {
		return fmt.Errorf("grpcserver: bind: %w", err)
	}
	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()
	ready := make(chan struct{})
	wrapped := &readyListener{Listener: listener, ready: ready}
	go func() {
		err := native.Serve(wrapped)
		if errors.Is(err, grpc.ErrServerStopped) {
			err = nil
		}
		s.mu.Lock()
		if s.serveErr == nil {
			s.serveErr = err
		}
		s.mu.Unlock()
		s.serveOnce.Do(func() { close(s.serveDone) })
	}()
	select {
	case <-ready:
		if err := ctx.Err(); err != nil {
			return err
		}
		s.mu.Lock()
		if s.state != "starting" {
			s.mu.Unlock()
			return context.Canceled
		}
		s.state = "running"
		if s.health != nil {
			s.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		}
		s.mu.Unlock()
		return nil
	case <-s.serveDone:
		s.mu.Lock()
		err := s.serveErr
		s.mu.Unlock()
		if err == nil {
			err = errors.New("grpcserver: stopped during startup")
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Server) Wait() error {
	s.mu.Lock()
	started := s.started
	s.mu.Unlock()
	if !started {
		return errors.New("grpcserver: not started")
	}
	<-s.serveDone
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serveErr
}
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// SetServing updates transport health. The application owns dependency health policy.
func (s *Server) SetServing(serving bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.health != nil {
		state := healthpb.HealthCheckResponse_NOT_SERVING
		if serving && s.state == "running" {
			state = healthpb.HealthCheckResponse_SERVING
		}
		s.health.SetServingStatus("", state)
	}
}

// Stop is idempotent and bounded even if a handler ignores cancellation.
func (s *Server) Stop(ctx context.Context) error {
	waitCtx, waitCancel := context.WithTimeout(ctx, s.options.ShutdownTimeout)
	defer waitCancel()
	s.stopOnce.Do(func() {
		s.mu.Lock()
		wasNew := s.state == "new"
		s.state = "stopping"
		cancel := s.startCancel
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if wasNew {
			s.startOnce.Do(func() { close(s.startDone) })
			s.serveOnce.Do(func() { close(s.serveDone) })
		}
		go func() {
			defer close(s.stopDone)
			stopCtx, cancel := context.WithTimeout(ctx, s.options.ShutdownTimeout)
			defer cancel()
			select {
			case <-s.startDone:
			case <-stopCtx.Done():
				s.mu.Lock()
				s.stopErr = stopCtx.Err()
				s.mu.Unlock()
				<-s.startDone
			}
			s.mu.Lock()
			native, h := s.native, s.health
			s.mu.Unlock()
			var err error
			if native != nil {
				if h != nil {
					h.Shutdown()
				}
				done := make(chan struct{})
				go func() { native.GracefulStop(); close(done) }()
				select {
				case <-done:
				case <-stopCtx.Done():
					err = stopCtx.Err()
					native.Stop()
				}
			}
			s.mu.Lock()
			s.stopErr = errors.Join(s.stopErr, err)
			s.state = "stopped"
			s.mu.Unlock()
		}()
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

type readyListener struct {
	net.Listener
	ready chan struct{}
	once  sync.Once
}

func (l *readyListener) Accept() (net.Conn, error) {
	l.once.Do(func() { close(l.ready) })
	return l.Listener.Accept()
}
