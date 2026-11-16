package rpcxclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/share"
	"github.com/spf13/cast"

	"github.com/wanglelecc/gokitbox/logger"
)

const (
	DefaultWrapType = iota

	// 超时时间, 值为time.duration 字符串，如：1s
	// 配置文件：rpcCallTimeout 为全局， 为单独请求设置
	WrapClientCtxKeyCallTimeout = "WRAPCLIENT_CTX_KEY_CALLTIMEOUT"
)

// 上下文 key 常量
type ctxKey string

const (
	CtxKeyRpcID         ctxKey = "rpc_id"
	CtxKeyTraceID       ctxKey = "trace_id"
	CtxKeyLogID         ctxKey = "log_id"
	CtxKeyHostname      ctxKey = "hostname"
	CtxKeyIsPlayback    ctxKey = "IS_PLAYBACK"
	CtxKeyIsBenchmark   ctxKey = "IS_BENCHMARK"
	CtxKeyRPCXAppId     ctxKey = "RPCX_APPID"
	CtxKeyRPCXTimestamp ctxKey = "RPCX_TIMESTAMP"
	CtxKeyRPCXSign      ctxKey = "RPCX_SIGN"
	CtxKeySkipLog       ctxKey = "RPCXSKIPLOG"
)

type RpcxWrap interface {
	WrapCall(client.XClient, context.Context, string, interface{}, interface{}) error
}

type DefaultWrap struct {
	serviceName string
	options     *WrapOptions
	useNewAuth  bool // 是否使用新的 HMAC-SHA256 鉴权
}

// WrapOptions configure one direct RPCX caller without installing registry globals.
type WrapOptions struct {
	AppId, AppKey string
	CallTimeout   time.Duration
}

func NewDefaultWrapWithOptions(serviceName string, o WrapOptions) (RpcxWrap, error) {
	if (o.AppId == "") != (o.AppKey == "") {
		return nil, errors.New("rpcxclient: appId/appKey must be paired")
	}
	if o.CallTimeout <= 0 {
		return nil, errors.New("rpcxclient: callTimeout must be positive")
	}
	return &DefaultWrap{serviceName: serviceName, useNewAuth: true, options: &o}, nil
}

func NewDefaultWrap(serviceName string) RpcxWrap {
	w := new(DefaultWrap)
	w.serviceName = serviceName
	// 默认使用新的鉴权方式
	w.useNewAuth = true
	return w
}

// NewDefaultWrapWithAuth 创建指定鉴权方式的 Wrap
func NewDefaultWrapWithAuth(serviceName string, useNewAuth bool) RpcxWrap {
	w := new(DefaultWrap)
	w.serviceName = serviceName
	w.useNewAuth = useNewAuth
	return w
}

func (d *DefaultWrap) WrapCall(c client.XClient, ctx context.Context, serviceMethod string, args interface{}, reply interface{}) (err error) {
	tag := d.serviceName + "." + serviceMethod
	if skip := ctx.Value(CtxKeySkipLog); skip == nil {
		defer func() {
			logger.Ix(ctx, tag, fmt.Sprintf("[destinationAddr:%s], WrapCall args:[%+v],reply:[%+v]", d.getServerAddr(ctx), args, reply))
		}()
	}

	var authAppId, authAppKey string
	if d.options != nil {
		authAppId = d.options.AppId
		authAppKey = d.options.AppKey
	} else {
		authAppId = appId
		authAppKey = appKey
	}
	metadata := map[string]string{
		"rpc_id":       cast.ToString(ctx.Value(CtxKeyRpcID)),
		"trace_id":     cast.ToString(ctx.Value(CtxKeyTraceID)),
		"hostname":     cast.ToString(ctx.Value(CtxKeyHostname)),
		"IS_PLAYBACK":  cast.ToString(ctx.Value(CtxKeyIsPlayback)),
		"IS_BENCHMARK": cast.ToString(ctx.Value(CtxKeyIsBenchmark)),
	}

	if metadata["trace_id"] == "" {
		metadata["trace_id"] = cast.ToString(ctx.Value(CtxKeyLogID))
	}

	// 鉴权信息生成
	if cast.ToString(ctx.Value(CtxKeyRPCXAppId)) != "" {
		// 使用上下文传入的自定义鉴权信息
		metadata[AuthHeaderAppId] = cast.ToString(ctx.Value(CtxKeyRPCXAppId))
		metadata[AuthHeaderTimestamp] = cast.ToString(ctx.Value(CtxKeyRPCXTimestamp))
		metadata[AuthHeaderSign] = cast.ToString(ctx.Value(CtxKeyRPCXSign))
	} else if authAppId != "" && authAppKey != "" {
		// 使用全局配置的鉴权
		if d.useNewAuth {
			// 新的 HMAC-SHA256 + nonce 鉴权
			timestamp, nonce, sign := genRpcAuthCtx(authAppId, authAppKey)
			metadata[AuthHeaderAppId] = authAppId
			metadata[AuthHeaderTimestamp] = timestamp
			metadata[AuthHeaderNonce] = nonce
			metadata[AuthHeaderSign] = sign
		} else {
			// 兼容旧的 MD5 鉴权（已废弃）
			timestamp, sign := genRpcAuthCtxLegacy(authAppId, authAppKey)
			metadata[AuthHeaderTimestamp] = timestamp
			metadata[AuthHeaderSign] = sign
		}
	}

	ctx = context.WithValue(ctx, share.ReqMetaDataKey, metadata)

	// 调用超时控制
	if d.options != nil || rpcxOptCallTimeout > 0 {
		err = d.callWithTimeout(c, ctx, serviceMethod, args, reply)
	} else {
		err = c.Call(ctx, serviceMethod, args, reply)
	}

	return
}

// 超时控制
func (d *DefaultWrap) callWithTimeout(c client.XClient, ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	var timeout time.Duration
	if d.options != nil {
		timeout = d.options.CallTimeout
	} else {
		timeout = rpcxOptCallTimeout
	}
	if t := cast.ToString(ctx.Value(WrapClientCtxKeyCallTimeout)); t != "" && d.options == nil {
		if td, err := time.ParseDuration(t); err == nil && td > 0 {
			timeout = td
		}
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Call(callCtx, serviceMethod, args, reply)
}

func (d *DefaultWrap) getServerAddr(ctx context.Context) (serverAddr string) {
	if metaData := ctx.Value(share.ReqMetaDataKey); metaData != nil {
		m, ok := metaData.(map[string]string)
		if !ok {
			return
		}
		if addr, ok := m["DESTINATION_ADDR"]; ok {
			return addr
		}
	}

	return
}
