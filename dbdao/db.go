package dbdao

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/logger"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cast"
	"xorm.io/xorm"
)

var initOnce sync.Once

type DBDao struct {
	Engine *xorm.EngineGroup
}

var (
	dbInstance   map[string]*DBDao
	curDbPoses   map[string]*uint64 // 当前选择的数据库
	showSql      bool
	slowDuration time.Duration
	maxConn      int = 100
	maxIdle      int = 30
)

func newDBDaoWithParams(hosts []string, driver string) (Db *DBDao) {
	Db = new(DBDao)
	// engine, err := xorm.NewEngine(driver, host)
	engine, err := xorm.NewEngineGroup(driver, hosts)

	// 必须先检查错误，再赋值给 Db.Engine
	if err != nil {
		logger.Pf(context.Background(), "dbdao", "创建数据库引擎失败 (hosts=%v): %v", hosts, err)
		panic(fmt.Sprintf("创建数据库引擎失败 (hosts=%v): %v", hosts, err))
	}

	Db.Engine = engine
	/*
	  Db.Engine.Logger.SetLevel(core.LOG_DEBUG)
	  Db.Engine.ShowSQL = true
	  Db.Engine.ShowInfo = true
	  Db.Engine.ShowDebug = true
	  Db.Engine.ShowErr = true
	  Db.Engine.ShowWarn = true
	*/
	Db.Engine.SetMaxOpenConns(maxConn)
	Db.Engine.SetMaxIdleConns(maxIdle)
	Db.Engine.SetConnMaxLifetime(time.Second * 3000)
	Db.Engine.ShowSQL(showSql)
	Db.Engine.SetLogger(dbLogger)
	return
}

func Init() {
	initOnce.Do(func() {
		initDb()
	})
}

func initDb() {
	dbInstance = make(map[string]*DBDao, 0)
	curDbPoses = make(map[string]*uint64)
	idc := ""
	showLog := config.GetConfStringMap("MysqlConfig")
	showSql = showLog["showSql"] == "true"
	slowDuration = time.Duration(cast.ToInt(showLog["slowDuration"])) * time.Millisecond
	maxConnConfig := cast.ToInt(showLog["maxConn"])
	if maxConnConfig > 0 {
		maxConn = maxConnConfig
	}

	maxIdleConfig := cast.ToInt(showLog["maxIdle"])
	if maxIdleConfig > 0 {
		maxIdle = maxIdleConfig
	}

	if maxIdle > maxConn {
		maxIdle = maxConn
	}

	for cluster, hosts := range config.GetConfArrayMap("MysqlCluster") {
		// 过滤IDC
		if cluster == idc {
			continue
		}

		instance := cluster
		dbInstance[instance] = newDBDaoWithParams(hosts, "mysql")
		curDbPoses[instance] = new(uint64)
	}
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
