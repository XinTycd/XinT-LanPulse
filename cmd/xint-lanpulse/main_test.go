package main

import (
	"strings"
	"testing"
)

func TestPingRejectsInvalidAddresses(t *testing.T) {
	for _, address := range []string{"", "example.com", "::1", "999.1.1.1"} {
		if got := (&App{}).Ping(address, 1000, 1); !strings.Contains(got, "有效") {
			t.Fatalf("%q: %q", address, got)
		}
	}
}

func TestPingRejectsInvalidOptions(t *testing.T) {
	for _, options := range [][2]int{{99, 1}, {60001, 1}, {1000, 0}, {1000, 11}} {
		if got := (&App{}).Ping("127.0.0.1", options[0], options[1]); !strings.Contains(got, "超时") {
			t.Fatalf("%v: %q", options, got)
		}
	}
}
