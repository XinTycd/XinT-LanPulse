package main

import (
	"strings"
	"testing"
)

func TestPingRejectsInvalidAddresses(t *testing.T) {
	for _, address := range []string{"", "example.com", "::1", "999.1.1.1"} {
		if got := (&App{}).Ping(address); !strings.Contains(got, "有效") {
			t.Fatalf("%q: %q", address, got)
		}
	}
}
