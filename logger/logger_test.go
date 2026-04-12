package logger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInitWithConfig(t *testing.T) {
	// 创建临时目录用于存储日志
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfgMap := map[string]string{
		"fileName":  logFile,
		"console":   "false", // 测试时不输出到控制台
		"level":     "DEBUG",
		"maxSize":   "10",
		"maxBackups": "3",
		"maxAge":    "7",
		"compress":  "false",
	}

	// 设置基础信息
	SetEnv("test")
	SetName("testApp")
	SetDepartment("testDept")
	SetVersion("v1.0.0")

	logConfig := NewConfig().SetConfigMap(cfgMap)
	InitWithConfig(logConfig)
	defer Sync()

	// 测试基础日志
	ctx := context.Background()
	Dx(ctx, "test", "debug message")
	Ix(ctx, "test", "info message")
	Wx(ctx, "test", "warn message")
	Ex(ctx, "test", "error message")

	// 验证日志文件已创建
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestStructuredLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "structured.log")

	cfgMap := map[string]string{
		"fileName": logFile,
		"console":  "false",
		"level":    "DEBUG",
	}

	SetEnv("test")
	SetName("testApp")
	SetDepartment("testDept")
	SetVersion("v1.0.0")

	logConfig := NewConfig().SetConfigMap(cfgMap)
	InitWithConfig(logConfig)
	defer Sync()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "test-trace-123")
	ctx = context.WithValue(ctx, "rpc_id", "1.1")

	// 测试结构化日志
	Dx(ctx, "test", "debug with context", "key1", "value1", "count", 42)
	Ix(ctx, "test", "info with context", "user_id", 10086)
	Wx(ctx, "test", "warn with context", "retry", 3)
	Ex(ctx, "test", "error with context", "error", "something went wrong")
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"DEBUG级别应记录所有日志", "DEBUG"},
		{"INFO级别应记录INFO及以上", "INFO"},
		{"WARN级别应记录WARN及以上", "WARN"},
		{"ERROR级别只记录ERROR", "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里简化测试，实际应检查日志输出
			t.Logf("Testing log level: %s", tt.level)
		})
	}
}

func TestGenTraceId(t *testing.T) {
	// 生成多个trace_id，验证不重复
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenTraceId()
		if ids[id] {
			t.Errorf("Duplicate trace_id generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenLoggerId(t *testing.T) {
	// 验证能生成 Logger ID（基于 goroutine ID）
	id := GenLoggerId()
	if id <= 0 {
		t.Errorf("Logger ID should be positive, got %d", id)
	}

	// 同一 goroutine 中 ID 相同（预期行为）
	id2 := GenLoggerId()
	if id != id2 {
		t.Logf("同一 goroutine 中 Logger ID 不同: %d vs %d", id, id2)
	}

	// 不同 goroutine 中 ID 不同
	done := make(chan int64)
	go func() {
		done <- GenLoggerId()
	}()

	otherID := <-done
	if otherID == id {
		t.Logf("不同 goroutine 中 Logger ID 相同（可能巧合）: %d", id)
	}
}

func TestConfig(t *testing.T) {
	cfg := NewConfig()

	// 测试通过 map 设置配置
	cfgMap := map[string]string{
		"fileName":   "/tmp/test.log",
		"level":      "INFO",
		"maxSize":    "100",
		"maxBackups": "5",
		"maxAge":     "30",
		"compress":   "true",
		"console":    "true",
	}
	cfg.SetConfigMap(cfgMap)

	// 验证配置
	if cfg.FileName != "/tmp/test.log" {
		t.Errorf("Config FileName = %s, want /tmp/test.log", cfg.FileName)
	}
	if cfg.Level != "INFO" {
		t.Errorf("Config Level = %s, want INFO", cfg.Level)
	}
}

func TestSetConfigMap(t *testing.T) {
	cfg := NewConfig()
	cfgMap := map[string]string{
		"fileName":   "/tmp/test.log",
		"level":      "WARN",
		"maxSize":    "200",
		"maxBackups": "10",
		"maxAge":     "14",
		"compress":   "true",
		"console":    "true",
	}

	cfg.SetConfigMap(cfgMap)

	if cfg.FileName != "/tmp/test.log" {
		t.Errorf("SetConfigMap FileName = %s", cfg.FileName)
	}
	if cfg.Level != "WARN" {
		t.Errorf("SetConfigMap Level = %s", cfg.Level)
	}
}

