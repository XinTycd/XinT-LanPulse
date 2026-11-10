package tests

import (
	"context"
	"github.com/xint-tech/xint-lanpulse/internal/command"
	"github.com/xint-tech/xint-lanpulse/internal/feishu"
	"testing"
)

func TestEventDeduplication(t *testing.T) {
	events, err := feishu.NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	h := feishu.NewHandler(serviceForEvent(t), events, func(context.Context, string, string) error { calls++; return nil }, nil)
	m := feishu.IncomingMessage{EventID: "evt_1", UserID: "ou_user", ChatID: "oc_1", Text: "15"}
	if err = h.Handle(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	if err = h.Handle(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("want 1 reply got %d", calls)
	}
}
func serviceForEvent(t *testing.T) *command.Service { svc, _, _ := service(true); return svc }
