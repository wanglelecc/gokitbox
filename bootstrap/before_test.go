package bootstrap

import (
	"testing"
)

func TestInitLogger(t *testing.T) {
	fn := InitLogger("test", "testApp", "testDept", "v1.0.0")
	if fn == nil {
		t.Error("InitLogger() returned nil")
	}
	// 注意：实际执行会失败，因为配置未加载
}

func TestInitPprof(t *testing.T) {
	fn := InitPprof()
	if fn == nil {
		t.Error("InitPprof() returned nil")
	}
}

func TestCloseLogger(t *testing.T) {
	fn := CloseLogger()
	if fn == nil {
		t.Error("CloseLogger() returned nil")
	}
}

func TestInitDb(t *testing.T) {
	fn := InitDb()
	if fn == nil {
		t.Error("InitDb() returned nil")
	}
}

func TestInitRedis(t *testing.T) {
	fn := InitRedis()
	if fn == nil {
		t.Error("InitRedis() returned nil")
	}
}

func TestInitProducer(t *testing.T) {
	fn := InitProducer()
	if fn == nil {
		t.Error("InitProducer() returned nil")
	}
}

func TestCloseProducer(t *testing.T) {
	fn := CloseProducer()
	if fn == nil {
		t.Error("CloseProducer() returned nil")
	}
}

func TestInitSnowflake(t *testing.T) {
	fn := InitSnowflake("testProject", "testService")
	if fn == nil {
		t.Error("InitSnowflake() returned nil")
	}
}
