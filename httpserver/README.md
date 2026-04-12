# HttpServer

HttpServer 基于 `gin` 封装，方便集成到 `gokit` 项目中，无代码侵入，保留原汁原味的 `github.com/gin-gonic/gin`, 可同步更新新版本。

## 安装
```shell script
go get github.com/wanglelecc/gokitbox/httpserver
```

## 配置
```ini
[HttpServer]
;TCP监听端口
addr = :8088
;运行模式. 可选debug/release/test
mode = release
;读取超时时间（支持 "10s", "500ms", "1m" 等格式，或纯数字表示秒数）
readTimeout = 10s
;写入超时时间
writeTimeout = 30s
;空闲超时时间（keep-alive 连接）
idleTimeout = 60s
```

**默认值说明：**
- `mode`: `release` - 默认运行模式
- `addr`: `:8088` - 默认监听 8088 端口
- `readTimeout`: `10s` - 读取请求超时 10 秒（包括请求体）
- `writeTimeout`: `30s` - 写入响应超时 30 秒
- `idleTimeout`: `60s` - Keep-Alive 空闲连接超时 60 秒

**配置格式：**
- 支持 Go Duration 格式：`"10s"`, `"500ms"`, `"1m"`, `"1h30m"` 等
- 支持纯数字（秒）：`10` 等同于 `"10s"`
- 配置为 0 或留空将使用默认值

**注意：** `mode` 配置项只有在使用 `NewServerWithOptions()` 时才能从配置文件中读取生效，因为 `gin.SetMode()` 必须在 `gin.New()` 之前调用。如果使用 `NewServer()` + `InitConfig()` 的方式，`mode` 配置会被忽略。

## 初始化

### 方式一：使用默认配置 + InitConfig（推荐用于已有项目）

```go
    
    import (
            "github.com/wanglelecc/gokitbox/httpserver"
            "github.com/wanglelecc/gokitbox/bootstrap"
            "github.com/wanglelecc/gokitbox/httpserver/middleware"
        )

	// 实例化 httpserver（使用默认配置）
	s := httpserver.NewServer()
	
    // 注册中间件 Logger，Recovery, Context, RequestHeader ...
    s.UseMiddleware(middleware.Logger(middleware.CheckCode0), middleware.Recovery(), middleware.Context(), middleware.RequestHeader())
	
    // 注册前置方法 初始化日志和配置
    s.AddBeforeServerStartFunc(bootstrap.InitLogger("dev", "httpServerDemo", "gomods", "0.0.1"), s.InitConfig())
	
    // 注册后置方法
    s.AddAfterServerStopFunc(bootstrap.CloseLogger())

	// 注册路由
	app.RegisterRouter(s.GinEngine())
    
    // 启动服务并监听
	err := s.Serve()
	if err != nil {
		log.Printf("Server stop err:%v", err)
	} else {
		log.Printf("Server exit")
	}

```

### 方式二：使用自定义配置（推荐用于新项目或需要通过配置文件设置 Mode）

```go
    import (
            "github.com/wanglelecc/gokitbox/config"
            "github.com/wanglelecc/gokitbox/httpserver"
            "github.com/wanglelecc/gokitbox/bootstrap"
            "github.com/wanglelecc/gokitbox/httpserver/middleware"
        )

    // 先初始化配置
    config.Init()
    
    // 从配置文件加载 ServerOptions
    var opts httpserver.ServerOptions
    err := config.ConfMapToStruct("HttpServer", &opts)
    if err != nil {
        // 使用默认配置
        opts = httpserver.DefaultOptions()
    }
    
    // 使用自定义配置创建 Server（此时 Mode 配置会生效）
    s := httpserver.NewServerWithOptions(opts)
    
    // 注册中间件
    s.UseMiddleware(middleware.Logger(middleware.CheckCode0), middleware.Recovery(), middleware.Context())
    
    // 注册路由
    app.RegisterRouter(s.GinEngine())
    
    // 启动服务
    err = s.Serve()
    if err != nil {
        log.Printf("Server stop err:%v", err)
    }
```

> **说明：** 
> - 方式一兼容老代码，但 `mode` 配置不会生效（因为 `gin.SetMode()` 必须在 `gin.New()` 之前调用）
> - 方式二可以完整支持所有配置项，包括 `mode` 配置

### 自定义优雅关闭超时时间（可选）

对于有长时间请求的应用（如文件上传、长轮询等），可以自定义优雅关闭的超时时间：

```go
    s := httpserver.NewServer()
    
    // 设置优雅关闭超时为 2 分钟（默认为 30 秒）
    s.SetShutdownTimeout(2 * time.Minute)
    
    // ... 其他配置
    
    err := s.Serve()
```

> **说明：**
> - 默认优雅关闭超时为 30 秒，适合大部分应用
> - 对于有 60s+ 长请求的场景，建议设置更长的超时时间
> - 超时后会自动强制关闭连接，避免进程无法退出
> 默认路由注册放在 `app/service.go`

## 操作文档
https://gin-gonic.com/zh-cn/docs/

## 服务骨架示例
https://github.com/wanglelecc/gokitbox/httpserver-demo
> 新项目可以直接克隆，替换名称并直接使用。