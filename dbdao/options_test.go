package dbdao

import (
	"testing"
	"time"
)

func TestOptionsAreValidatedAndCopied(t *testing.T) {
	bad := Options{Driver: "mysql", MaxOpenConns: 1, MaxIdleConns: 2}
	if bad.Validate() == nil {
		t.Fatal("invalid pool accepted")
	}
	input := Options{Driver: "mysql", MaxOpenConns: 10, MaxIdleConns: 0, ConnMaxLifetime: time.Minute, Clusters: map[string][]string{"main": {"test-dsn"}}}
	if err := Configure(input); err != nil {
		t.Fatal(err)
	}
	input.Clusters["main"][0] = "changed"
	delete(input.Clusters, "main")
	optionsMu.Lock()
	got := configuredOptions.Clusters["main"][0]
	optionsMu.Unlock()
	if got != "test-dsn" {
		t.Fatal("caller mutated installed options")
	}
	if err := Configure(input); err == nil {
		t.Fatal("reconfiguration accepted")
	}
}
