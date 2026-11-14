package consumer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigureExplicitFormats(t *testing.T) {
	for _, tt := range []struct{ ext, body string }{{"yaml", "enabled: {redis: false}\nredis: []\n"}, {"json", `{"enabled":{"redis":false},"redis":[]}`}} {
		path := filepath.Join(t.TempDir(), "worker."+tt.ext)
		if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadConfigure(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := cfg.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestConfigureRejectsUnknownAndMissing(t *testing.T) {
	for _, body := range []string{`{"enabld":{}}`, `{"enabled":{}} {"enabled":{}}`} {
		path := filepath.Join(t.TempDir(), "worker.json")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfigure(path); err == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
	if err := (&Configure{Enabled: &ConsumeEnabled{Redis: true}}).Validate(); err == nil {
		t.Fatal("missing active consumers accepted")
	}
	if err := (&Configure{Enabled: &ConsumeEnabled{Delay: true}, Delay: []*DelayConfig{nil}}).Validate(); err == nil {
		t.Fatal("nil consumer accepted")
	}
}
