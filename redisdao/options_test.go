package redisdao

import (
	"crypto/tls"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestOptionsAreValidatedAndCopied(t *testing.T) {
	input := Options{Instances: map[string]redis.Options{"main": {Addr: "localhost:6379", PoolSize: 5, MinIdleConns: 0, TLSConfig: &tls.Config{ServerName: "original"}}}}
	if err := Configure(input); err != nil {
		t.Fatal(err)
	}
	original := input.Instances["main"]
	original.TLSConfig.ServerName = "changed"
	delete(input.Instances, "main")
	optionsMu.Lock()
	got := configuredOptions.Instances["main"]
	optionsMu.Unlock()
	if got.TLSConfig.ServerName != "original" || got.MinIdleConns != 0 {
		t.Fatal("options aliased or explicit zero lost")
	}
	if err := Configure(input); err == nil {
		t.Fatal("reconfiguration accepted")
	}
	if err := (Options{Instances: map[string]redis.Options{"bad": {Addr: "local", PoolSize: 1, MinIdleConns: 2}}}).Validate(); err == nil {
		t.Fatal("invalid pool accepted")
	}
}
