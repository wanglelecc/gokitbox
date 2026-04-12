package bootstrap

import (
	"github.com/wanglelecc/gokitbox/logger"
	"github.com/wanglelecc/gokitbox/producer"
)

type AfterServerStopFunc func()

func CloseLogger() AfterServerStopFunc {
	return func() {
		logger.Sync()
	}
}

func CloseProducer() AfterServerStopFunc {
	return func() {
		producer.Close()
		return
	}
}
