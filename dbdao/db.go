package dbdao

import (
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cast"
	"github.com/wanglelecc/gokitbox/configsource"
	"xorm.io/xorm"
)

var initOnce sync.Once

type DBDao struct {
	Engine *xorm.EngineGroup
}

var (
	dbInstance   map[string]*DBDao
	showSql      bool
	slowDuration time.Duration
)

func initialize() error {
	initOnce.Do(func() {
		optionsMu.Lock()
		started = true
		opts := configuredOptions
		optionsMu.Unlock()
		if opts == nil {
			o := legacyOptions()
			opts = &o
		}
		initErr = initDbWithOptions(*opts)
	})
	return initErr
}

// InitWithOptions is the explicit initialization API. Configure may instead be
// called at bootstrap when legacy singleton access should remain lazy.
func InitWithOptions(o Options) error {
	if err := Configure(o); err != nil {
		return err
	}
	return initialize()
}

func legacyOptions() Options {
	values := configsource.GetConfStringMap("MysqlConfig")
	o := Options{Driver: "mysql", Clusters: configsource.GetConfArrayMap("MysqlCluster"), ShowSql: values["showSql"] == "true", SlowDuration: time.Duration(cast.ToInt(values["slowDuration"])) * time.Millisecond, MaxOpenConns: 100, MaxIdleConns: 30, ConnMaxLifetime: 3000 * time.Second}
	if n := cast.ToInt(values["maxConn"]); n > 0 {
		o.MaxOpenConns = n
	}
	if n := cast.ToInt(values["maxIdle"]); n > 0 {
		o.MaxIdleConns = n
	}
	if o.MaxIdleConns > o.MaxOpenConns {
		o.MaxIdleConns = o.MaxOpenConns
	}
	return o
}

func initDbWithOptions(o Options) error {
	if err := o.Validate(); err != nil {
		return err
	}
	showSql = o.ShowSql
	slowDuration = o.SlowDuration
	engines := make(map[string]*DBDao, len(o.Clusters))
	for name, dsns := range o.Clusters {
		engine, err := xorm.NewEngineGroup(o.Driver, dsns)
		if err != nil {
			for _, db := range engines {
				db.Close()
			}
			return fmt.Errorf("dbdao: cannot create cluster %q", name)
		}
		engine.SetMaxOpenConns(o.MaxOpenConns)
		engine.SetMaxIdleConns(o.MaxIdleConns)
		engine.SetConnMaxLifetime(o.ConnMaxLifetime)
		engine.ShowSQL(o.ShowSql)
		engine.SetLogger(dbLogger)
		engines[name] = &DBDao{Engine: engine}
	}
	dbInstance = engines
	return nil
}

// CloseAll closes all engines initialized by this module.
func CloseAll() error {
	var first error
	for _, db := range dbInstance {
		if err := db.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func GetDbInstance(db string) *DBDao {
	Init()

	if instances, ok := dbInstance[db]; ok {
		return instances
	} else {
		return nil
	}
}

// func (this *DBDao) GetSession() *xorm.Session {
//	return this.Engine.NewSession()
// }

func (this *DBDao) Close() error {
	return this.Engine.Close()
}

func Init() {
	if err := initialize(); err != nil {
		panic(err)
	}
}

// Initialize constructs engines using previously configured module Options.
func Initialize() error { return initialize() }
