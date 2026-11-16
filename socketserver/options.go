// Package socketserver owns transport-neutral connection sessions. Business
// routing, authentication decisions and message delivery belong to application hooks.
package socketserver

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

type Options struct {
	Network               string        `yaml:"network"`
	Addr                  string        `yaml:"addr"`
	IdleTimeout           time.Duration `yaml:"idleTimeout"`
	LoginTimeout          time.Duration `yaml:"loginTimeout"`
	WriteTimeout          time.Duration `yaml:"writeTimeout"`
	HookTimeout           time.Duration `yaml:"hookTimeout"`
	ShutdownTimeout       time.Duration `yaml:"shutdownTimeout"`
	MaxConnections        int           `yaml:"maxConnections"`
	MaxPendingConnections int           `yaml:"maxPendingConnections"`
	MaxReadBytes          int64         `yaml:"maxReadBytes"`
	MaxWriteBytes         int64         `yaml:"maxWriteBytes"`
	MaxMessageBytes       int64         `yaml:"maxMessageBytes"`
	ReadBufferSize        int           `yaml:"readBufferSize"`
	ReadQueueSize         int           `yaml:"readQueueSize"`
	ReadQueueBytes        int64         `yaml:"readQueueBytes"`
	WriteQueueSize        int           `yaml:"writeQueueSize"`
	CheckInterval         time.Duration `yaml:"checkInterval"`
	MaintenanceInterval   time.Duration `yaml:"maintenanceInterval"`
	MaintenanceWorkers    int           `yaml:"maintenanceWorkers"`
	CloseOnHandlerError   bool          `yaml:"closeOnHandlerError"`
	TCPFraming            string        `yaml:"tcpFraming"`
	LineDelimiter         string        `yaml:"lineDelimiter"`
	WebSocketPath         string        `yaml:"websocketPath"`
	AllowedOrigins        []string      `yaml:"allowedOrigins"`
	AllowAllOrigins       bool          `yaml:"allowAllOrigins"`
}

func DefaultOptions() Options {
	return Options{Network: "tcp", Addr: "127.0.0.1:8888", IdleTimeout: 2 * time.Minute, LoginTimeout: 5 * time.Minute, WriteTimeout: 5 * time.Second, HookTimeout: 5 * time.Second, ShutdownTimeout: 30 * time.Second, MaxConnections: 10000, MaxPendingConnections: 256, MaxReadBytes: 128 << 20, MaxWriteBytes: 64 << 20, MaxMessageBytes: 1 << 20, ReadBufferSize: 1024, ReadQueueSize: 64, ReadQueueBytes: 1 << 20, WriteQueueSize: 64, CheckInterval: time.Second, MaintenanceInterval: time.Minute, MaintenanceWorkers: 8, TCPFraming: "raw", LineDelimiter: "\n", WebSocketPath: "/websocket"}
}
func (o Options) Validate() error {
	if o.Network != "tcp" && o.Network != "websocket" {
		return errors.New("socketserver: network must be tcp/websocket")
	}
	if _, _, err := net.SplitHostPort(o.Addr); err != nil {
		return errors.New("socketserver: addr requires host:port")
	}
	if o.IdleTimeout < 0 || o.LoginTimeout < 0 || o.MaintenanceInterval < 0 || o.WriteTimeout <= 0 || o.HookTimeout <= 0 || o.ShutdownTimeout <= 0 || o.CheckInterval <= 0 {
		return errors.New("socketserver: invalid timeout/interval")
	}
	if o.MaxPendingConnections < 0 || o.MaxReadBytes < 0 || o.MaxWriteBytes < 0 || o.MaxConnections < 0 || o.MaxMessageBytes < 0 || o.ReadQueueBytes < 0 || o.ReadBufferSize < 1 || o.ReadQueueSize < 1 || o.WriteQueueSize < 1 || o.MaintenanceWorkers < 1 {
		return errors.New("socketserver: invalid connection/message/queue limits")
	}
	switch o.TCPFraming {
	case "raw", "line", "length-prefix", "custom":
	default:
		return errors.New("socketserver: unknown TCP framing")
	}
	if o.Network == "tcp" && (o.TCPFraming == "line" || o.TCPFraming == "length-prefix") && o.MaxMessageBytes == 0 {
		return errors.New("socketserver: framed TCP requires a positive message size limit")
	}
	if o.TCPFraming == "line" && len(o.LineDelimiter) != 1 {
		return errors.New("socketserver: line delimiter must be one byte")
	}
	if !strings.HasPrefix(o.WebSocketPath, "/") || strings.ContainsAny(o.WebSocketPath, " ?#") {
		return errors.New("socketserver: invalid websocket path")
	}
	if o.AllowAllOrigins && len(o.AllowedOrigins) > 0 {
		return errors.New("socketserver: allowAllOrigins and allowedOrigins are mutually exclusive")
	}
	for _, origin := range o.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("socketserver: allowedOrigins require exact http(s) origins")
		}
	}
	return nil
}
