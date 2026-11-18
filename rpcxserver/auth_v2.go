package rpcxserver

import (
	"context"
	"errors"
	"github.com/smallnest/rpcx/protocol"
	"github.com/wanglelecc/gokitbox/bootstrap"
	"github.com/wanglelecc/gokitbox/rpcxauth"
	"time"
)

type AuthOptions struct {
	Domain       string
	MaxClockSkew time.Duration
	ReplayStore  rpcxauth.ReplayStore
}

func (srv *Server) InitRpcxAuthV2(values map[string]string, o AuthOptions, fns ...ValidAccess) bootstrap.BeforeServerStartFunc {
	copied := map[string]string{}
	for k, v := range values {
		if k != "timestampValidate" {
			copied[k] = v
		}
	}
	return func() error {
		if o.MaxClockSkew == 0 {
			o.MaxClockSkew = 2 * time.Minute
		}
		if o.ReplayStore == nil || o.MaxClockSkew < time.Second || o.MaxClockSkew > 10*time.Minute {
			return errors.New("rpcxauth: shared replay store and 1s..10m clock skew required")
		}
		for k, v := range copied {
			if k == "" || v == "" || len(k) > 128 {
				return errors.New("rpcxauth: invalid credential configuration")
			}
		}
		srv.authInitialized.Do(func() {
			srv.authMu.Lock()
			defer srv.authMu.Unlock()
			srv.validAuth = copied
			srv.authV2 = o
			srv.accessFn = srv.validAuthAccess
			if len(fns) > 0 {
				srv.accessFn = fns[0]
			}
			srv.server.AuthFunc = func(ctx context.Context, req *protocol.Message, token string) error { return srv.auth(ctx, req, token) }
		})
		return nil
	}
}
