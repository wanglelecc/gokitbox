package rpcxclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/smallnest/rpcx/client"
)

type WrapClient struct {
	Discovery client.ServiceDiscovery
	xclient   client.XClient
	wrap      RpcxWrap
}

type InitXClientFunc func(c client.XClient) error

func NewWrapClient(basePath, servicePath string, failMode client.FailMode, selectMode client.SelectMode, option client.Option, fns ...InitXClientFunc) (*WrapClient, error) {
	wClient := new(WrapClient)

	// 初始化注册中心
	wClient.Discovery = getClientDiscovery(basePath, servicePath)
	if wClient.Discovery == nil {
		return nil, errors.New("failed to initialize service discovery: discovery is nil")
	}

	wClient.xclient = client.NewXClient(servicePath, failMode, selectMode, wClient.Discovery, option)
	if wClient.xclient == nil {
		wClient.Discovery.Close()
		return nil, errors.New("failed to create xclient: xclient is nil")
	}

	for _, fn := range fns {
		if err := fn(wClient.xclient); err != nil {
			wClient.xclient.Close()
			wClient.Discovery.Close()
			return nil, fmt.Errorf("failed to initialize xclient: %w", err)
		}
	}

	if appId != "" {
		wClient.xclient.Auth(appId)
	}

	wClient.wrap = NewDefaultWrap(servicePath)
	return wClient, nil
}

func (w *WrapClient) WrapCall(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	if w.xclient == nil {
		return errors.New("xclient is nil, client not properly initialized")
	}

	return w.wrap.WrapCall(w.xclient, ctx, serviceMethod, args, reply)
}

func (w *WrapClient) Close() error {
	var errs []error

	if w.xclient != nil {
		if err := w.xclient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// 关闭 Discovery 资源
	if w.Discovery != nil {
		w.Discovery.Close()
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close client: %v", errs)
	}
	return nil
}

// XClient 返回底层的 XClient，用于高级操作
func (w *WrapClient) XClient() client.XClient {
	return w.xclient
}

// IsValid 检查客户端是否有效
func (w *WrapClient) IsValid() bool {
	return w != nil && w.xclient != nil && w.Discovery != nil
}
