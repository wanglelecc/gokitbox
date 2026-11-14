package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStrictDocument(t *testing.T) {
	type Options struct {
		Name    string        `yaml:"name"`
		Enabled bool          `yaml:"enabled"`
		Timeout time.Duration `yaml:"timeout"`
		Hosts   []string      `yaml:"hosts"`
	}
	cases := []struct {
		name, text string
		valid      bool
	}{
		{"valid", "name: off\nenabled: false\ntimeout: 2s\nhosts: [a, b]\n", true},
		{"unknown", "name: x\npasswrod: secret-value\n", false},
		{"duplicate", "name: first\nname: secret-value\n", false},
		{"type", "enabled: secret-value\n", false},
		{"coercion", "name: true\n", false},
		{"duration", "timeout: 50\n", false},
		{"null", "name: null\n", false},
		{"multiple", "name: x\n---\nname: secret-value\n", false},
		{"syntax", "name: [secret-value\n", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var out Options
			err := DecodeYAML([]byte(tt.text), &out)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, err)
			}
			if err != nil && strings.Contains(err.Error(), "secret-value") {
				t.Fatal("error exposed source value")
			}
			if tt.valid && (out.Name != "off" || out.Timeout != 2*time.Second || len(out.Hosts) != 2) {
				t.Fatalf("unexpected result: %+v", out)
			}
		})
	}
}

func TestExplicitPathAndFreshRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	if err := os.WriteFile(path, []byte("App: {name: first}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := ReadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("App: {name: second}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	var old map[string]map[string]string
	if err := first.Decode(&old); err != nil {
		t.Fatal(err)
	}
	if old["App"]["name"] != "first" || second.MustValue("App", "name") != "second" {
		t.Fatal("read snapshot was cached or mutated")
	}
	t.Chdir(dir)
	if _, err := Load("app.yaml"); err != nil {
		t.Fatal(err)
	}
}

func TestYamlLegacyArraysAndSet(t *testing.T) {
	cfg, err := parseYaml([]byte("App:\n  status: off\n  hosts: [a, b]\n  sibling: keep\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MustValue("App", "status") != "off" {
		t.Fatal("YAML 1.1 coercion")
	}
	if !reflect.DeepEqual(cfg.MustValueArray("App", "hosts", " "), []string{"a", "b"}) {
		t.Fatal("native sequence lost")
	}
	if !reflect.DeepEqual(cfg.GetSectionList(), []string{"App"}) {
		t.Fatal("missing sections")
	}
	cfg.Set("App", "name", "new")
	if cfg.MustValue("App", "sibling") != "keep" {
		t.Fatal("Set removed sibling")
	}
	if err := cfg.GetSectionObject("absent", &struct{}{}); err == nil {
		t.Fatal("missing section accepted")
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				cfg.Set("App", "name", "new")
				cfg.MustValue("App", "name")
				cfg.GetSectionList()
			}
		}()
	}
	wg.Wait()
}

func TestDefaultForUnavailableSource(t *testing.T) {
	gCfgMu.RLock()
	oldPath := configFilename
	gCfgMu.RUnlock()
	t.Cleanup(func() { SetConfigPath(oldPath) })
	SetConfigPath(filepath.Join(t.TempDir(), "missing.yaml"))
	if got := GetConfDefault("App", "name", "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
}
