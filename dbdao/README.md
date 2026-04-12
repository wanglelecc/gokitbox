# DBDao 数据库访问

DBDao 基于 `go-xorm/xorm` 封装，提供数据库连接池管理和主从分离支持。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/dbdao
```

## 特性

- 支持一主多从架构
- 自动读写分离
- 连接池管理
- 慢查询日志
- SQL 执行日志
- 自动负载均衡（轮询选择从库）

## 配置

```ini
[MysqlConfig]
; 是否打开 SQL 执行记录
showSql = false
; 是否记录 SQL 执行时间（需打开 showSql）
showExecTime = false
; 慢查询阈值（毫秒），0 表示记录全部
slowDuration = 500
; 最大连接数（默认 100）
maxConn = 50
; 最大空闲连接数（默认 30）
maxIdle = 30

[MysqlCluster]
; 格式：实例名 = 主库连接串 从库1连接串 从库2连接串 ...
gokit = gokit_rw:password@tcp(localhost:3306)/gokit_test gokit_ro:password@tcp(localhost:3306)/gokit_test
blog = blog_rw:password@tcp(localhost:3306)/blog_test blog_ro:password@tcp(localhost:3306)/blog_test
```

> 支持一主多从，配置至少一主一从。

## 使用示例

```go
package main

import (
    "log"
    "time"
    _ "github.com/go-sql-driver/mysql"
    "github.com/wanglelecc/gokitbox/dbdao"
)

type User struct {
    Id      int64     `xorm:"pk autoincr"`
    Name    string    `xorm:"varchar(100)"`
    Email   string    `xorm:"varchar(200)"`
    Age     int
    Created time.Time `xorm:"created"`
    Updated time.Time `xorm:"updated"`
}

func main() {
    // 初始化数据库（通常在 bootstrap 中完成）
    dbdao.Init()
    
    // 获取指定实例
    db := dbdao.GetDbInstance("gokit")
    if db == nil {
        log.Fatal("数据库实例不存在")
    }
    
    // 获取主库引擎（用于写操作）
    master := db.Engine.Master()
    
    // 获取从库引擎（用于读操作，自动负载均衡）
    slave := db.Engine.Slave()
    
    // 插入数据
    user := User{Name: "张三", Email: "zhangsan@example.com", Age: 25}
    affected, err := master.Insert(&user)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("插入 %d 条记录，ID: %d", affected, user.Id)
    
    // 查询数据
    var users []User
    err = slave.Where("age > ?", 18).Limit(10, 0).Find(&users)
    if err != nil {
        log.Fatal(err)
    }
    
    // 单条查询
    var u User
    has, err := slave.Where("name = ?", "张三").Get(&u)
    if err != nil {
        log.Fatal(err)
    }
    if has {
        log.Printf("找到用户: %+v", u)
    }
    
    // 更新数据
    affected, err = master.Where("id = ?", u.Id).Update(&User{Age: 26})
    if err != nil {
        log.Fatal(err)
    }
    
    // 删除数据
    affected, err = master.Where("id = ?", u.Id).Delete(&User{})
    if err != nil {
        log.Fatal(err)
    }
    
    // 关闭连接
    db.Close()
}
```

## API 说明

```go
// 初始化数据库连接池
func Init()

// 获取指定名称的数据库实例
func GetDbInstance(name string) *DBDao

// 关闭数据库连接
func (d *DBDao) Close() error
```

## XORM 文档

- [XORM 使用手册](https://gobook.io/read/gitea.com/xorm/manual-zh-CN/)
- [XORM GitHub](https://github.com/go-xorm/xorm)
