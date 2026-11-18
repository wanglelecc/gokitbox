package rpcxclient

import (
	"errors"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"github.com/wanglelecc/gokitbox/rpcxauth"
	"sync"
)

type authPlugin struct{ app, key, domain string }

func (p *authPlugin) ClientBeforeEncode(r *protocol.Message) error {
	return rpcxauth.SignRequest(r, p.app, p.key, p.domain)
}

var authInstallMu sync.Mutex

// InstallAuth must run before the client is first used. A client owns immutable credentials.
func InstallAuth(c client.XClient, app, key, domain string) error {
	if app == "" && key == "" {
		return nil
	}
	if app == "" || key == "" || domain == "" {
		return errors.New("rpcxclient: paired credentials and domain required")
	}
	authInstallMu.Lock()
	defer authInstallMu.Unlock()
	plugins := c.GetPlugins()
	if plugins == nil {
		plugins = client.NewPluginContainer()
		c.SetPlugins(plugins)
	}
	for _, plugin := range plugins.All() {
		if p, ok := plugin.(*authPlugin); ok {
			if p.app != app || p.key != key || p.domain != domain {
				return errors.New("rpcxclient: client credentials cannot change")
			}
			return nil
		}
	}
	plugins.Add(&authPlugin{app, key, domain})
	return nil
}
