package config

import (
	"os"
	"sync"
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

// TestLoadUnknownAnySource 验证 C-1 修复：未注册的 any source 返回 error 而非 panic
func TestLoadUnknownAnySource(t *testing.T) {
	_, err := Load("non_existent_source")
	if err == nil {
		t.Fatal("Load() with unknown any source should return error, got nil")
	}
}

// TestLoadYmlExtension 验证 config 重构：.yml 后缀文件被识别为 YAML
func TestLoadYmlExtension(t *testing.T) {
	dir, _ := os.Getwd()
	ymlPath := dir + "/config/app.yml"

	cfg, err := Load(ymlPath)
	if err != nil {
		t.Fatalf("Load() .yml file failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() .yml file returned nil config")
	}
	name := cfg.MustValue("goconfig", "name", "")
	if name != "goyml" {
		t.Errorf("Load() .yml name = %q, want %q", name, "goyml")
	}
}

// TestLoadYamlExtension 验证 .yaml 后缀文件识别（回归测试）
func TestLoadYamlExtension(t *testing.T) {
	dir, _ := os.Getwd()
	yamlPath := dir + "/config/app.yaml"

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load() .yaml file failed: %v", err)
	}
	name := cfg.MustValue("goconfig", "name", "")
	if name != "goyaml" {
		t.Errorf("Load() .yaml name = %q, want %q", name, "goyaml")
	}
}

// TestConcurrentInitConfig 验证并发 InitConfig 不发生 data race（需配合 -race）
func TestConcurrentInitConfig(t *testing.T) {
	dir, _ := os.Getwd()
	// 重置状态
	ClearConfigCache()
	SetConfigPath(dir + "/config/app.yaml")

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			InitConfig()
			_ = GetConf("goconfig", "name")
		}()
	}
	wg.Wait()

	// 恢复原始路径，不影响其他测试
	SetConfigPath(dir + "/config/app.yaml")
}

// TestSetConfigPathResetsGCfg 验证 SetConfigPath 正确重置全局配置
func TestSetConfigPathResetsGCfg(t *testing.T) {
	dir, _ := os.Getwd()

	// 先初始化
	SetConfigPath(dir + "/config/app.yaml")
	InitConfig()
	before := GetConf("goconfig", "name")
	if before == "" {
		t.Fatal("config not loaded before reset")
	}

	// 重置为不存在的路径，再次 GetConf 应返回空
	SetConfigPath("/tmp/nonexistent_config_test.yaml")
	ClearConfigCache()
	val := GetConf("goconfig", "name")
	if val != "" {
		t.Errorf("after SetConfigPath to nonexistent, GetConf should return empty, got %q", val)
	}

	// 恢复
	SetConfigPath(dir + "/config/app.yaml")
	ClearConfigCache()
}
