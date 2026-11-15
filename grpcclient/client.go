package grpcclient

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	conn      *grpc.ClientConn
	options   Options
	closeOnce sync.Once
	closeErr  error
}

// NewClientWithOptions creates a lazy channel; it does not prove connectivity.
func NewClientWithOptions(o Options, extensions ...grpc.DialOption) (*Client, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	copyMetadata := map[string]string{}
	for k, v := range o.Metadata {
		copyMetadata[k] = v
	}
	o.Metadata = copyMetadata
	c := &Client{options: o}
	var transport credentials.TransportCredentials
	if o.Insecure {
		transport = insecure.NewCredentials()
	} else {
		cfg, err := o.TLS.load()
		if err != nil {
			return nil, err
		}
		transport = credentials.NewTLS(cfg)
	}
	options := []grpc.DialOption{grpc.WithTransportCredentials(transport), grpc.WithChainUnaryInterceptor(c.unary), grpc.WithChainStreamInterceptor(c.stream), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(o.MaxRecvMsgSize), grpc.MaxCallSendMsgSize(o.MaxSendMsgSize))}
	if o.BearerToken != "" {
		options = append(options, grpc.WithPerRPCCredentials(tokenCredentials{token: o.BearerToken, secure: !o.Insecure}))
	}
	options = append(options, extensions...)
	conn, err := grpc.NewClient(o.Target, options...)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return c, nil
}
func (c *Client) Conn() *grpc.ClientConn { return c.conn }
func (c *Client) Close() error {
	c.closeOnce.Do(func() { c.closeErr = c.conn.Close() })
	return c.closeErr
}
func (c *Client) withMetadata(ctx context.Context) context.Context {
	existing, _ := metadata.FromOutgoingContext(ctx)
	result := metadata.MD{}
	for key, value := range c.options.Metadata {
		result.Set(key, value)
	}
	for key, values := range existing {
		result[key] = append([]string(nil), values...)
	}
	return metadata.NewOutgoingContext(ctx, result)
}
func (c *Client) unary(ctx context.Context, method string, req, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	callCtx, cancel := context.WithTimeout(c.withMetadata(ctx), c.options.CallTimeout)
	defer cancel()
	return invoke(callCtx, method, req, reply, conn, opts...)
}
func (c *Client) stream(ctx context.Context, desc *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(c.withMetadata(ctx), desc, conn, method, opts...)
}

type tokenCredentials struct {
	token  string
	secure bool
}

func (c tokenCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + c.token}, nil
}
func (c tokenCredentials) RequireTransportSecurity() bool { return c.secure }
