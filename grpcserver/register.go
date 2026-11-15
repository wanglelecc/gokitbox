package grpcserver

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/grpc"
)

type registration struct {
	desc           grpc.ServiceDesc
	implementation any
}
type collector struct {
	names   map[string]bool
	entries []registration
	err     error
}

func (c *collector) RegisterService(desc *grpc.ServiceDesc, impl any) {
	if c.err != nil {
		return
	}
	if desc == nil || desc.ServiceName == "" || strings.Contains(desc.ServiceName, "/") {
		c.err = errors.New("grpcserver: invalid service descriptor")
		return
	}
	if c.names[desc.ServiceName] {
		c.err = errors.New("grpcserver: duplicate or reserved service name")
		return
	}
	handler := reflect.TypeOf(desc.HandlerType)
	value := reflect.ValueOf(impl)
	if handler == nil || handler.Kind() != reflect.Ptr || handler.Elem().Kind() != reflect.Interface || !value.IsValid() || (value.Kind() == reflect.Ptr && value.IsNil()) || !value.Type().Implements(handler.Elem()) {
		c.err = errors.New("grpcserver: service does not implement its generated interface")
		return
	}
	methods := map[string]bool{}
	for _, method := range desc.Methods {
		if method.MethodName == "" || methods[method.MethodName] || method.Handler == nil {
			c.err = errors.New("grpcserver: invalid or duplicate method")
			return
		}
		methods[method.MethodName] = true
	}
	for _, method := range desc.Streams {
		if method.StreamName == "" || methods[method.StreamName] || method.Handler == nil {
			c.err = errors.New("grpcserver: invalid or duplicate stream")
			return
		}
		methods[method.StreamName] = true
	}
	copyDesc := *desc
	copyDesc.Methods = append([]grpc.MethodDesc(nil), desc.Methods...)
	copyDesc.Streams = append([]grpc.StreamDesc(nil), desc.Streams...)
	c.names[desc.ServiceName] = true
	c.entries = append(c.entries, registration{copyDesc, impl})
}

// Register executes the generated registration synchronously against a collector.
// It constructs no listener or native server. Do not retain the registrar.
func (s *Server) Register(fn func(grpc.ServiceRegistrar)) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != "new" {
		return errors.New("grpcserver: register before Start")
	}
	if fn == nil {
		return errors.New("grpcserver: registration callback is required")
	}
	names := map[string]bool{}
	for _, entry := range s.registrations {
		names[entry.desc.ServiceName] = true
	}
	if s.options.Health {
		names["grpc.health.v1.Health"] = true
	}
	if s.options.Reflection {
		names["grpc.reflection.v1.ServerReflection"] = true
		names["grpc.reflection.v1alpha.ServerReflection"] = true
	}
	c := &collector{names: names}
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("grpcserver: registration panic (%T)", p)
		}
	}()
	fn(c)
	if c.err != nil {
		return c.err
	}
	s.registrations = append(s.registrations, c.entries...)
	return nil
}
