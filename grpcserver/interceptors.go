package grpcserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/wanglelecc/gokitbox/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *Server) authenticate(ctx context.Context, method string) error {
	if s.options.BearerToken != "" {
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("authorization")
		if len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte("Bearer "+s.options.BearerToken)) != 1 {
			return status.Error(codes.Unauthenticated, "invalid credentials")
		}
	}
	if s.options.AuthFunc != nil {
		return s.options.AuthFunc(ctx, method)
	}
	return nil
}
func (s *Server) mapError(err error) (mapped error) {
	if err == nil {
		return nil
	}
	mapped = status.Error(codes.Internal, "internal server error")
	defer func() {
		if recover() != nil {
			mapped = status.Error(codes.Internal, "internal server error")
		}
	}()
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Err()
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	if s.options.ErrorMapper != nil {
		if value := s.options.ErrorMapper(err); value != nil {
			if _, ok := status.FromError(value); ok {
				return value
			}
		}
	}
	return mapped
}
func (s *Server) finish(ctx context.Context, method string, start time.Time, err error) {
	// A logging extension must not turn an otherwise handled RPC into a panic.
	defer func() { recover() }()
	code := status.Code(err)
	if s.options.LogFunc != nil {
		s.options.LogFunc(ctx, method, code, time.Since(start))
		return
	}
	logger.Ix(ctx, "grpcserver", "rpc completed", "method", method, "code", code.String(), "duration", time.Since(start))
}
func (s *Server) unary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
	start := time.Now()
	defer func() {
		if p := recover(); p != nil {
			s.logPanic(ctx, p)
			err = status.Error(codes.Internal, "internal server error")
		}
		err = s.mapError(err)
		s.finish(ctx, info.FullMethod, start, err)
	}()
	if err = s.authenticate(ctx, info.FullMethod); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}
func (s *Server) stream(service any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	start := time.Now()
	defer func() {
		if p := recover(); p != nil {
			s.logPanic(stream.Context(), p)
			err = status.Error(codes.Internal, "internal server error")
		}
		err = s.mapError(err)
		s.finish(stream.Context(), info.FullMethod, start, err)
	}()
	if err = s.authenticate(stream.Context(), info.FullMethod); err != nil {
		return err
	}
	return handler(service, stream)
}

// Panic values may contain credentials; log the type and stack only.
func (s *Server) logPanic(ctx context.Context, p any) {
	defer func() { recover() }()
	logger.Ex(ctx, "grpcserver", "handler panic", "type", fmt.Sprintf("%T", p), "stack", string(debug.Stack()))
}
