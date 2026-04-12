# Config 配置管理

Config 是一个简单易用的配置管理工具，支持 INI 和 YAML 格式，提供统一的配置读取接口。

## 安装

```shell
go get github.com/wanglelecc/gokitbox/config
```

## 特性

- 支持 INI、YAML 格式配置文件
- 多种指定配置文件位置的方法
- 一次加载，内存缓存，读取速度快
- 支持配置项与结构体自动映射
- 支持多配置文件加载

## 默认配置路径

```
./conf/app.ini
```

## 使用示例

```go
package main

import (
    "log"
    "github.com/wanglelecc/gokitbox/config"
)

func main() {
    // 修改配置路径（可选，默认 ./conf/app.ini）
    config.SetConfigPath("/home/dev/app.ini")

    // 获取单个配置项
    name := config.GetConf("goconfig", "name")

    // 获取配置项，不存在返回默认值
    port := config.GetConfDefault("server", "port", "8080")

    // 获取字符串数组
    hosts := config.GetConfs("goconfig", "hosts")
    // hosts = ["127.0.0.1", "127.0.0.2", "127.0.0.3"]

    // 获取整个 section 的 string map
    cfgMap := config.GetConfStringMap("goconfigStringMap")
    // cfgMap = {"name": "goconfig", "host": "127.0.0.1"}

    // 获取整个 section 的数组 map
    arrMap := config.GetConfArrayMap("goconfigArrayMap")
    // arrMap = {"name": ["goconfig1", "goconfig2"]}

    // 映射到结构体
    type Config struct {
        Max    int
        Port   int
        Rate   float64
        Hosts  []string
        Timeout string
    }
    var cfg Config
    err := config.ConfMapToStruct("goconfigObject", &cfg)
    if err != nil {
        log.Fatal(err)
    }
}
```

## 配置示例

### INI 格式

```ini
[goconfig]
name = goconfig
hosts = 127.0.0.1 127.0.0.2 127.0.0.3

[goconfigStringMap]
name = goconfig
host = 127.0.0.1

[goconfigArrayMap]
name = goconfig1 goconfig2

[goconfigObject]
max = 101
port = 9099
rate = 1.01
hosts = 127.0.0.1 127.0.0.2
timeout = 5s
```

### YAML 格式

```yaml
goconfig:
  name: goconfig
  hosts:
    - 127.0.0.1
    - 127.0.0.2

goconfigObject:
  max: 101
  port: 9099
  rate: 1.01
  timeout: 5s
```

## API 说明

```go
// 设置配置文件路径
func SetConfigPath(path string)

// 初始化配置（通常在启动时调用）
func Init()

// 获取指定 section 下指定 key 的值
func GetConf(sec, key string) string

// 获取指定 section 下指定 key 的值，不存在返回默认值
func GetConfDefault(sec, key, def string) string

// 获取指定 section 下指定 key 的字符串切片
func GetConfs(sec, key string) []string

// 获取指定 section 下所有配置，返回 map[string]string
func GetConfStringMap(sec string) map[string]string

// 获取指定 section 下所有配置，返回 map[string][]string
func GetConfArrayMap(sec string) map[string][]string

// 将指定 section 下配置映射到结构体
func ConfMapToStruct(sec string, v interface{}) error
```
