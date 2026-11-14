package httpserver

import (
	"testing"
	"time"

	"github.com/wanglelecc/gokitbox/config"
)

func TestYAMLTimeoutOptions(t *testing.T) {
	cfg, err := config.NewSnapshot(map[string]interface{}{"HTTP": map[string]interface{}{"addr": ":1234", "readTimeout": "7s", "writeTimeout": "9s"}})
	if err != nil {
		t.Fatal(err)
	}
	var opts ServerOptions
	if err := cfg.GetSectionObject("HTTP", &opts); err != nil {
		t.Fatal(err)
	}
	got, err := opts.GetReadTimeout()
	if err != nil || got != 7*time.Second {
		t.Fatal("readTimeout silently ignored")
	}
	got, err = opts.GetWriteTimeout()
	if err != nil || got != 9*time.Second {
		t.Fatal("writeTimeout silently ignored")
	}
}
