package redisdao

import (
	"errors"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Options defines named standalone Redis clients. Cluster/Sentinel require a
// separate module/API; multiple addresses cannot silently replace one another.
type Options struct{ Instances map[string]redis.Options }

func (o Options) Validate() error {
	for name, v := range o.Instances {
		if name == "" || v.Addr == "" || v.DB < 0 || v.PoolSize < 0 || v.MinIdleConns < 0 || v.ReadTimeout < -2 || v.WriteTimeout < -2 || (v.PoolSize > 0 && v.MinIdleConns > v.PoolSize) {
			return errors.New("redisdao: invalid instance/pool options")
		}
	}
	return nil
}

var optionsMu sync.Mutex
var configuredOptions *Options
var started bool

func Configure(o Options) error {
	if err := o.Validate(); err != nil {
		return err
	}
	copied := Options{Instances: make(map[string]redis.Options, len(o.Instances))}
	for k, v := range o.Instances {
		if v.TLSConfig != nil {
			v.TLSConfig = v.TLSConfig.Clone()
		}
		copied.Instances[k] = v
	}
	optionsMu.Lock()
	defer optionsMu.Unlock()
	if configuredOptions != nil || started {
		return errors.New("redisdao: options already configured or initialized")
	}
	configuredOptions = &copied
	return nil
}
