package config

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// 设置测试配置路径
	dir, _ := os.Getwd()
	SetConfigPath(dir + "/config/app.yaml")
	code := m.Run()
	os.Exit(code)
}

func TestGetConf(t *testing.T) {
	tests := []struct {
		name     string
		section  string
		key      string
		expected string
	}{
		{
			name:     "获取 hosts 配置",
			section:  "goconfig",
			key:      "hosts",
			expected: "127.0.0.1 127.0.0.2 127.0.0.3",
		},
		{
			name:     "获取 name 配置",
			section:  "goconfig",
			key:      "name",
			expected: "goyaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConf(tt.section, tt.key)
			if got != tt.expected {
				t.Errorf("GetConf(%s, %s) = %v, want %v", tt.section, tt.key, got, tt.expected)
			}
		})
	}
}

func TestGetConfDefault(t *testing.T) {
	tests := []struct {
		name         string
		section      string
		key          string
		defaultValue string
		wantDefault  bool
	}{
		{
			name:         "存在的配置",
			section:      "goconfig",
			key:          "name",
			defaultValue: "default",
			wantDefault:  false,
		},
		{
			name:         "不存在的配置返回默认值",
			section:      "notexist",
			key:          "notkey",
			defaultValue: "default_value",
			wantDefault:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConfDefault(tt.section, tt.key, tt.defaultValue)
			if tt.wantDefault {
				if got != tt.defaultValue {
					t.Errorf("GetConfDefault() = %v, want default %v", got, tt.defaultValue)
				}
			}
		})
	}
}

func TestGetConfArr(t *testing.T) {
	tests := []struct {
		name         string
		section      string
		key          string
		expectedLen  int
		expectedItem string
	}{
		{
			name:         "获取 hosts 数组",
			section:      "goconfig",
			key:          "hosts",
			expectedLen:  3,
			expectedItem: "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConfArr(tt.section, tt.key)
			if len(got) != tt.expectedLen {
				t.Errorf("GetConfArr() len = %v, want %v", len(got), tt.expectedLen)
			}
			if tt.expectedLen > 0 && got[0] != tt.expectedItem {
				t.Errorf("GetConfArr()[0] = %v, want %v", got[0], tt.expectedItem)
			}
		})
	}
}

func TestGetConfStringMap(t *testing.T) {
	tests := []struct {
		name         string
		section      string
		expectedKey  string
		expectedVal  string
		shouldExist  bool
	}{
		{
			name:         "获取 StringMap",
			section:      "goyamlStringMap",
			expectedKey:  "name",
			expectedVal:  "goyaml",
			shouldExist:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConfStringMap(tt.section)
			if tt.shouldExist {
				if val, ok := got[tt.expectedKey]; !ok || val != tt.expectedVal {
					t.Errorf("GetConfStringMap()[%s] = %v, want %v", tt.expectedKey, val, tt.expectedVal)
				}
			}
		})
	}
}

func TestGetConfArrayMap(t *testing.T) {
	tests := []struct {
		name        string
		section     string
		expectedKey string
		expectedLen int
	}{
		{
			name:        "获取 ArrayMap",
			section:     "goyamlArrayMap",
			expectedKey: "name",
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConfArrayMap(tt.section)
			if val, ok := got[tt.expectedKey]; !ok || len(val) != tt.expectedLen {
				t.Errorf("GetConfArrayMap()[%s] len = %v, want %v", tt.expectedKey, len(val), tt.expectedLen)
			}
		})
	}
}

func TestConfMapToStruct(t *testing.T) {
	// 简化测试，只验证不报错
	tests := []struct {
		name      string
		section   string
		wantErr   bool
	}{
		{
			name:    "映射到结构体",
			section: "goconfig",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestConfig struct {
				Name string `yaml:"name"`
			}
			var cfg TestConfig
			err := ConfMapToStruct(tt.section, &cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfMapToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// 验证字段映射
			if cfg.Name != "goyaml" {
				t.Errorf("ConfMapToStruct() Name = %v, want goyaml", cfg.Name)
			}
		})
	}
}

func TestSetConfigPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "设置配置路径",
			path: "/tmp/test.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：这会改变全局配置路径，测试后需要恢复
			// configFilename 是小写的内部变量，这里只测试不报错
			SetConfigPath(tt.path)
			// 验证可以通过再次调用不报错
			SetConfigPath(tt.path)
		})
	}
}
