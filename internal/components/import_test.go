package components

import (
	"context"
	"net/netip"
	"testing"
)

func TestParseManifest(t *testing.T) {
	item, err := parse([]byte(`{"id":"camera-health","name":"摄像头画面检查","description":"检查画面状态"}`), "file:test.json")
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != "camera-health" || !item.Enabled {
		t.Fatalf("unexpected component: %#v", item)
	}
}

func TestPublicAddressAndDialRejectPrivateNetworks(t *testing.T) {
	for _, value := range []string{"127.0.0.1", "10.0.0.1", "169.254.1.1", "100.64.0.1", "::1", "::ffff:192.168.1.1"} {
		if publicAddress(netip.MustParseAddr(value)) {
			t.Fatalf("accepted %s", value)
		}
	}
	if !publicAddress(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("rejected public address")
	}
	if conn, err := dialPublic(context.Background(), "tcp", "127.0.0.1:443"); err == nil {
		conn.Close()
		t.Fatal("connected to loopback")
	}
	if _, err := FromURL("https://user:pass@example.com/component.json"); err == nil {
		t.Fatal("accepted URL credentials")
	}
}

func TestParseManifestRejectsReservedID(t *testing.T) {
	if _, err := parse([]byte(`{"id":"device-status","name":"x","description":"x"}`), "file:test.json"); err == nil {
		t.Fatal("expected reserved ID rejection")
	}
}
