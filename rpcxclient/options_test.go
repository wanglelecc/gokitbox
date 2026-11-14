package rpcxclient

import (
	"sync"
	"testing"
	"time"
)

func TestExplicitOptions(t *testing.T) {
	if IsConfigInitialized() {
		t.Fatal("configuration was read at import time")
	}
	if err := InitWithOptions(Options{}); err == nil {
		t.Fatal("invalid options accepted")
	}
	addresses := []string{"localhost:2181"}
	if err := InitWithOptions(Options{Addrs: addresses, BasePath: "/test", AppId: "test-app", Group: "test-group"}); err != nil {
		t.Fatal(err)
	}
	addresses[0] = "changed"
	if GetSdAddress()[0] != "localhost:2181" || GetAppId() != "test-app" || GetCallTimeout() != 30*time.Second || GetClientOption().Group != "test-group" {
		t.Fatal("options lost or aliased")
	}
	if err := ValidateConfig(); err != nil {
		t.Fatal(err)
	}
	if err := InitWithOptions(Options{Addrs: []string{"other"}, BasePath: "/other"}); err == nil {
		t.Fatal("live reconfiguration was accepted")
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); GetClientOption(); GetAppId(); GetSdAddress(); IsConfigInitialized() }()
	}
	wg.Wait()
}
