// Package grpcserver provides an explicitly configured, signal-free gRPC server.
package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type TLSOptions struct {
	Enabled           bool   `yaml:"enabled"`
	CertFile          string `yaml:"certFile"`
	KeyFile           string `yaml:"keyFile"`
	ClientCAFile      string `yaml:"clientCAFile"`
	RequireClientCert bool   `yaml:"requireClientCert"`
}
type Options struct {
	Addr               string                                                   `yaml:"addr"`
	MaxRecvMsgSize     int                                                      `yaml:"maxRecvMsgSize"`
	MaxSendMsgSize     int                                                      `yaml:"maxSendMsgSize"`
	ShutdownTimeout    time.Duration                                            `yaml:"shutdownTimeout"`
	Health             bool                                                     `yaml:"health"`
	Reflection         bool                                                     `yaml:"reflection"`
	TLS                TLSOptions                                               `yaml:"tls"`
	BearerToken        string                                                   `yaml:"bearerToken"`
	AuthFunc           func(context.Context, string) error                      `yaml:"-"`
	ErrorMapper        func(error) error                                        `yaml:"-"`
	LogFunc            func(context.Context, string, codes.Code, time.Duration) `yaml:"-"`
	UnaryInterceptors  []grpc.UnaryServerInterceptor                            `yaml:"-"`
	StreamInterceptors []grpc.StreamServerInterceptor                           `yaml:"-"`
}

func DefaultOptions() Options {
	return Options{Addr: "127.0.0.1:19090", MaxRecvMsgSize: 4 << 20, MaxSendMsgSize: 4 << 20, ShutdownTimeout: 30 * time.Second, Health: true}
}
func (o Options) Validate() error {
	if _, _, err := net.SplitHostPort(o.Addr); err != nil {
		return errors.New("grpcserver: invalid addr")
	}
	if o.MaxRecvMsgSize <= 0 || o.MaxSendMsgSize <= 0 || o.ShutdownTimeout <= 0 {
		return errors.New("grpcserver: limits and shutdownTimeout must be positive")
	}
	t := o.TLS
	if !t.Enabled {
		if t.CertFile != "" || t.KeyFile != "" || t.ClientCAFile != "" || t.RequireClientCert {
			return errors.New("grpcserver: TLS fields require enabled TLS")
		}
		return nil
	}
	if t.CertFile == "" || t.KeyFile == "" {
		return errors.New("grpcserver: TLS certFile and keyFile are required")
	}
	if t.RequireClientCert != (t.ClientCAFile != "") {
		return errors.New("grpcserver: clientCAFile and requireClientCert must be configured together")
	}
	return nil
}
func (o TLSOptions) load() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
	if err != nil {
		return nil, errors.New("grpcserver: cannot load TLS certificate/key")
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}
	if o.RequireClientCert {
		data, err := os.ReadFile(o.ClientCAFile)
		if err != nil {
			return nil, errors.New("grpcserver: cannot read client CA")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(data) {
			return nil, errors.New("grpcserver: invalid client CA")
		}
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg, nil
}
