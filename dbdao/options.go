package dbdao

import (
	"errors"
	"sync"
	"time"
)

// Options belongs to the database module; it contains no file/section names.
type Options struct {
	Driver                     string
	Clusters                   map[string][]string
	ShowSql                    bool
	SlowDuration               time.Duration
	MaxOpenConns, MaxIdleConns int
	ConnMaxLifetime            time.Duration
}

func (o Options) Validate() error {
	if o.Driver == "" || o.MaxOpenConns <= 0 || o.MaxIdleConns < 0 || o.MaxIdleConns > o.MaxOpenConns || o.ConnMaxLifetime < 0 || o.SlowDuration < 0 {
		return errors.New("dbdao: invalid driver/pool options")
	}
	for name, dsns := range o.Clusters {
		if name == "" || len(dsns) == 0 {
			return errors.New("dbdao: invalid cluster")
		}
		for _, dsn := range dsns {
			if dsn == "" {
				return errors.New("dbdao: empty DSN")
			}
		}
	}
	return nil
}

var optionsMu sync.Mutex
var configuredOptions *Options
var started bool
var initErr error

// Configure installs a defensive copy before the first client is constructed.
func Configure(o Options) error {
	if err := o.Validate(); err != nil {
		return err
	}
	copyOptions := o
	copyOptions.Clusters = make(map[string][]string, len(o.Clusters))
	for k, v := range o.Clusters {
		copyOptions.Clusters[k] = append([]string(nil), v...)
	}
	optionsMu.Lock()
	defer optionsMu.Unlock()
	if configuredOptions != nil || started {
		return errors.New("dbdao: options already configured or initialized")
	}
	configuredOptions = &copyOptions
	return nil
}
