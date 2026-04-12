package rpcxserver

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	"github.com/wanglelecc/gokitbox/logger"

	"github.com/smallnest/rpcx/log"
)

var (
	skip      = 3
	callerKey = "x_runtime_caller"
	tag       = "rpcxServer"
)

type rpcxLogger struct{}

func init() {
	log.SetLogger(&rpcxLogger{})
}

func (l *rpcxLogger) Debug(v ...interface{}) {
	logger.Dx(context.Background(), tag, fmt.Sprint(v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Debugf(format string, v ...interface{}) {
	logger.Dx(context.Background(), tag, fmt.Sprintf(format, v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Info(v ...interface{}) {
	logger.Ix(context.Background(), tag, fmt.Sprint(v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Infof(format string, v ...interface{}) {
	logger.Ix(context.Background(), tag, fmt.Sprintf(format, v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Warn(v ...interface{}) {
	logger.Wx(context.Background(), tag, fmt.Sprint(v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Warnf(format string, v ...interface{}) {
	logger.Wx(context.Background(), tag, fmt.Sprintf(format, v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Error(v ...interface{}) {
	logger.Ex(context.Background(), tag, fmt.Sprint(v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Errorf(format string, v ...interface{}) {
	logger.Ex(context.Background(), tag, fmt.Sprintf(format, v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Fatal(v ...interface{}) {
	logger.Fx(context.Background(), tag, fmt.Sprint(v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Fatalf(format string, v ...interface{}) {
	logger.Fx(context.Background(), tag, fmt.Sprintf(format, v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Panic(v ...interface{}) {
	logger.Px(context.Background(), tag, fmt.Sprint(v...), callerKey, runtimeCaller(skip))
}

func (l *rpcxLogger) Panicf(format string, v ...interface{}) {
	logger.Px(context.Background(), tag, fmt.Sprintf(format, v...), callerKey, runtimeCaller(skip))
}

func runtimeCaller(s int) (position string) {
	_, file, line, ok := runtime.Caller(s)
	if ok {
		position = file + ":" + strconv.Itoa(line)
	} else {
		position = "EMPTY"
	}

	return
}
