package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/wanglelecc/gokitbox/config"
	"github.com/wanglelecc/gokitbox/dbdao"
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/producer"
	"github.com/wanglelecc/gokitbox/redisdao"
	"github.com/wanglelecc/gokitbox/tools/uSnowflake"
)

type BeforeServerStartFunc func() error

// 初始化日志
func InitLogger(env, appName, department, version string) BeforeServerStartFunc {
	return func() error {
		section := "Log"
		logMap := config.GetConfStringMap(section) // 通过配置文件转为map[string]string

		var fileName string
		fileName, _ = logMap["fileName"]

		// 兼容未来云日志收集目录
		if strings.Index(fileName, "${APP_NAME}") > -1 {
			// get  app name
			cloudAppName := os.Getenv("CLOUD_DEPLOYMENT_NAME")
			if cloudAppName == "" {
				cloudAppName = appName
			}

			logMap["fileName"] = strings.Replace(fileName, "${APP_NAME}", cloudAppName, -1)
		}

		logger.SetEnv(env)
		logger.SetName(appName)
		logger.SetDepartment(department)
		logger.SetVersion(version)
		logConfig := logger.NewConfig().SetConfigMap(logMap)

		logger.InitWithConfig(logConfig)

		return nil
	}
}

func InitPprof() BeforeServerStartFunc {
	return func() error {
		enable := config.GetConf("Pprof", "enable")
		if enable == "true" {
			go pprofStart()
		}
		return nil
	}
}

func pprofStart() {
	ctx := context.Background()
	port := config.GetConf("Pprof", "port")

	if len(port) <= 0 {
		logger.Ex(ctx, "Pprof", fmt.Sprintf("pprof port:%s format wrong", port))
		return
	}

	logger.Ix(ctx, "Pprof", fmt.Sprintf("open pprof on port:%s", port))

	_ = http.ListenAndServe(":"+port, nil)
}

// 初始化消息生产者
func InitProducer() BeforeServerStartFunc {
	return func() error {
		producer.Init()
		return nil
	}
}

// 初始化 redis 连接池
func InitRedis() BeforeServerStartFunc {
	return func() error {
		redisdao.Init()
		return nil
	}
}

// 初始化 db 连接池
func InitDb() BeforeServerStartFunc {
	return func() error {
		dbdao.Init()
		return nil
	}
}

// 初始化 雪花算法 因子
func InitSnowflake(project string, service string) BeforeServerStartFunc {
	return func() error {
		provider := redisdao.NewRedisNodeProvider()
		uSnowflake.InitSnowflake(context.Background(), provider, project, service)
		return nil
	}
}
