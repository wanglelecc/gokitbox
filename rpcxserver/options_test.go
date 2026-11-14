package rpcxserver

import (
	"github.com/smallnest/rpcx/server"
	"testing"
)

func TestAuthOptionsAreCopiedBeforeHook(t *testing.T) {
	s := NewServer()
	s.server = server.NewServer()
	// This unit checks configuration ownership, not background nonce cleanup.
	s.nonceCleanupOnce.Do(func() {})
	values := map[string]string{"client-a": "test-key"}
	hook := s.InitRpcxAuthWithOptions(values)
	values["client-a"] = "changed"
	if err := hook(); err != nil {
		t.Fatal(err)
	}
	key, ok := s.validAuthAccess("client-a")
	if !ok || key != "test-key" {
		t.Fatal("authorization options were aliased")
	}
	if s.server.AuthFunc == nil {
		t.Fatal("auth hook did not apply options")
	}
}
