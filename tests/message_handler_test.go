package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/xint-tech/xint-lanpulse/internal/feishu"
)

func TestHandlerSplitsLargeReply(t *testing.T) {
	events, err := feishu.NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	parts := []string{}
	h := feishu.NewHandler(serviceForEvent(t), events, func(_ context.Context, _ string, text string) error { parts = append(parts, text); return nil }, nil)
	// /help is deliberately short; this checks the public handler still replies.
	if err := h.Handle(context.Background(), feishu.IncomingMessage{EventID: "split-test", UserID: "ou_user", ChatID: "oc_1", Text: "/help"}); err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 || !strings.Contains(parts[0], "/vi all") {
		t.Fatalf("unexpected reply: %#v", parts)
	}
}
