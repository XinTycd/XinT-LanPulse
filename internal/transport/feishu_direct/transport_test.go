package feishudirect

import (
	"context"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"testing"
	"time"
)

func TestOnlyPrivateChatsAreDispatched(t *testing.T) {
	id, chat, content := "ou_test", "chat", `{"text":"/status"}`
	events := make(chan IncomingMessage, 1)
	tr := New("", "")
	tr.SetHandler(func(_ context.Context, m IncomingMessage) error { events <- m; return nil })
	event := &larkim.P2MessageReceiveV1{Event: &larkim.P2MessageReceiveV1Data{
		Sender:  &larkim.EventSender{SenderId: &larkim.UserId{OpenId: &id}},
		Message: &larkim.EventMessage{ChatId: &chat, Content: &content},
	}}
	for _, typ := range []string{"group", "topic_group", ""} {
		event.Event.Message.ChatType = &typ
		if err := tr.onMessage(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-events:
		t.Fatal("group message dispatched")
	default:
	}
	typ := "p2p"
	event.Event.Message.ChatType = &typ
	if err := tr.onMessage(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	select {
	case <-events:
	case <-time.After(time.Second):
		t.Fatal("private message not dispatched")
	}
}
