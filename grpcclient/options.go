// Package grpcclient owns explicit per-target gRPC connections, not global configuration.
package grpcclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"
)

type TLSOptions struct {
	CAFile     string `yaml:"caFile"`
	CertFile   string `yaml:"certFile"`
	KeyFile    string `yaml:"keyFile"`
	ServerName string `yaml:"serverName"`
}
type Options struct {
	Target         string            `yaml:"target"`
	CallTimeout    time.Duration     `yaml:"callTimeout"`
	Insecure       bool              `yaml:"insecure"`
	TLS            TLSOptions        `yaml:"tls"`
	BearerToken    string            `yaml:"bearerToken"`
	Metadata       map[string]string `yaml:"metadata"`
	MaxRecvMsgSize int               `yaml:"maxRecvMsgSize"`
	MaxSendMsgSize int               `yaml:"maxSendMsgSize"`
}

func DefaultOptions() Options {
	return Options{CallTimeout: 3 * time.Second, MaxRecvMsgSize: 4 << 20, MaxSendMsgSize: 4 << 20}
}

var metadataKey = regexp.MustCompile(`^[0-9a-z_.-]+$`)

func (o Options) Validate() error {
	if strings.TrimSpace(o.Target) == "" || strings.ContainsAny(o.Target, " \r\n\t") {
		return errors.New("grpcclient: target is required and must not contain whitespace")
	}
	if o.CallTimeout <= 0 || o.MaxRecvMsgSize <= 0 || o.MaxSendMsgSize <= 0 {
		return errors.New("grpcclient: timeouts and limits must be positive")
	}
	t := o.TLS
	if (t.CertFile == "") != (t.KeyFile == "") {
		return errors.New("grpcclient: certFile and keyFile must be paired")
	}
	if o.Insecure && (t.CAFile != "" || t.CertFile != "" || t.KeyFile != "" || t.ServerName != "") {
		return errors.New("grpcclient: insecure and TLS options are mutually exclusive")
	}
	for key, value := range o.Metadata {
		if !metadataKey.MatchString(key) || strings.HasPrefix(key, "grpc-") || strings.HasSuffix(key, "-bin") || (key == "authorization" && o.BearerToken != "") {
			return errors.New("grpcclient: invalid or conflicting metadata key")
		}
		for _, ch := range value {
			if ch < 32 || ch > 126 {
				return errors.New("grpcclient: metadata values must be printable ASCII")
			}
		}
	}
	return nil
}
func (o TLSOptions) load() (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: o.ServerName}
	if o.CAFile != "" {
		data, err := os.ReadFile(o.CAFile)
		if err != nil {
			return nil, errors.New("grpcclient: cannot read CA")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(data) {
			return nil, errors.New("grpcclient: invalid CA")
		}
		cfg.RootCAs = pool
	}
	if o.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
		if err != nil {
			return nil, errors.New("grpcclient: cannot load client certificate/key")
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}
