package app

import (
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"testing"
)

func TestBadSecretLeavesTransportAvailable(t *testing.T) {
	c := config.Default()
	c.Feishu.AppSecretEncrypted = "not-valid-dpapi"
	a := &Application{Config: c}
	if err := a.refreshTransport(); err == nil {
		t.Fatal("expected decryption error")
	}
	if a.Transport == nil {
		t.Fatal("transport must remain available for status and shutdown")
	}
}
