//go:build windows

package tests

import (
	"context"
	"testing"

	"github.com/xint-tech/xint-lanpulse/internal/checker"
	"github.com/xint-tech/xint-lanpulse/internal/security"
)

func TestDPAPIRoundTrip(t *testing.T) {
	cipher, err := security.Encrypt("test-only-secret")
	if err != nil {
		t.Fatal(err)
	}
	if cipher == "test-only-secret" {
		t.Fatal("DPAPI output must not be plaintext")
	}
	plain, err := security.Decrypt(cipher)
	if err != nil || plain != "test-only-secret" {
		t.Fatalf("decrypt=%q err=%v", plain, err)
	}
}

func TestWindowsICMPPingLoopback(t *testing.T) {
	result := checker.New().Ping(context.Background(), "127.0.0.1", 1000, 1)
	if !result.Online {
		t.Fatalf("expected loopback ICMP response: %+v", result)
	}
}
