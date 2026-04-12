package logger

import (
	"context"
)

type Builder interface {
	LoadConfig(config *Config)

	SetVersion(version string)

	SetDepartment(department string)

	SetName(name string)

	SetEnv(env string)

	LoggerX(ctx context.Context, lvl string, tag string, message string, fields ...string)

	LoggerF(ctx context.Context, lvl string, tag string, message string, fields ...string)

	Build(ctx context.Context) (expand []interface{})

	Sync() error
}

func SetJoinMod(mod bool) {
	joinMod = mod
}

var stdBuilder = new(zapBuilder)

func SetEnv(env string) {
	stdBuilder.SetEnv(env)
}

func SetName(name string) {
	stdBuilder.SetName(name)
}

func SetVersion(version string) {
	stdBuilder.SetVersion(version)
}

func SetDepartment(department string) {
	stdBuilder.SetDepartment(department)
}

func Sync() error {
	return stdBuilder.Sync()
}

// 分隔模式
func Dx(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "DEBUG", tag, message, fields...)
}

func Ix(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "INFO", tag, message, fields...)
}

func Wx(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "WARNING", tag, message, fields...)
}

func Ex(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "ERROR", tag, message, fields...)
}

func Fx(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "FATAL", tag, message, fields...)
}

func Px(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "FATAL", tag, message, fields...)
}

func DebugX(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "DEBUG", tag, message, fields...)
}

func InfoX(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "INFO", tag, message, fields...)
}

func WarningX(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "WARNING", tag, message, fields...)
}

func ErrorX(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "ERROR", tag, message, fields...)
}

func FatalX(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "FATAL", tag, message, fields...)
}

func PanicX(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerX(ctx, "PANIC", tag, message, fields...)
}

// 粘连模式
func Df(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "DEBUG", tag, message, fields...)
}

func If(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "INFO", tag, message, fields...)
}

func Wf(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "WARNING", tag, message, fields...)
}

func Ef(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "ERROR", tag, message, fields...)
}

func Ff(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "FATAL", tag, message, fields...)
}

func Pf(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "PANIC", tag, message, fields...)
}

func DebugF(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "DEBUG", tag, message, fields...)
}

func InfoF(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "INFO", tag, message, fields...)
}

func WarningF(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "WARNING", tag, message, fields...)
}

func ErrorF(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "ERROR", tag, message, fields...)
}

func FatalF(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "FATAL", tag, message, fields...)
}

func PanicF(ctx context.Context, tag string, message string, fields ...interface{}) {
	stdBuilder.LoggerF(ctx, "PANIC", tag, message, fields...)
}
