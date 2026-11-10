package feishudirect

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/xint-tech/xint-lanpulse/internal/transport"
)

type IncomingMessage struct{ EventID, UserID, ChatID, Text string }
type Handler func(context.Context, IncomingMessage) error

// Transport owns all Feishu SDK calls. It creates one official SDK long
// connection and relies on the SDK's built-in reconnection support.
type Transport struct {
	mu            sync.RWMutex
	appID, secret string
	state         transport.Status
	handler       Handler
	onStatus      func(transport.Status)
	cancel        context.CancelFunc
	generation    uint64
	api           *lark.Client
}

func New(appID, secret string) *Transport {
	return &Transport{appID: appID, secret: secret, state: transport.Status{Detail: "飞书连接：未配置"}}
}
func (t *Transport) SetHandler(h Handler) { t.mu.Lock(); defer t.mu.Unlock(); t.handler = h }
func (t *Transport) SetStatusListener(listener func(transport.Status)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onStatus = listener
}
func (t *Transport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.cancel != nil {
		t.mu.Unlock()
		return nil
	}
	if t.appID == "" || t.secret == "" {
		t.state = transport.Status{Detail: "飞书连接：缺少 App ID 或 App Secret"}
		t.mu.Unlock()
		return fmt.Errorf("请先保存完整的飞书 App ID 和 App Secret")
	}
	child, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.generation++
	generation := t.generation
	t.state = transport.Status{Detail: "飞书连接：连接中"}
	t.api = lark.NewClient(t.appID, t.secret)
	dispatcher := larkevent.NewEventDispatcher("", "").OnP2MessageReceiveV1(t.onMessage)
	client := larkws.NewClient(t.appID, t.secret, larkws.WithEventHandler(dispatcher), larkws.WithOnReady(func() { t.setStatus(true, "飞书连接：已连接") }), larkws.WithOnReconnecting(func() { t.setStatus(false, "飞书连接：正在重连") }), larkws.WithOnReconnected(func() { t.setStatus(true, "飞书连接：已连接") }), larkws.WithOnDisconnected(func() { t.setStatus(false, "飞书连接：已断开") }), larkws.WithOnError(func(error) { t.setStatus(false, "飞书连接：连接异常，正在重连") }))
	t.mu.Unlock()
	go func() {
		err := client.Start(child)
		t.mu.Lock()
		if t.generation == generation {
			t.cancel = nil
			if child.Err() == nil && err != nil {
				t.state = transport.Status{Detail: "飞书连接：连接失败"}
			}
		}
		t.mu.Unlock()
	}()
	return nil
}
func (t *Transport) Stop(context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	t.generation++
	t.state = transport.Status{Detail: "飞书连接：已停止"}
	return nil
}
func (t *Transport) SendReply(ctx context.Context, m transport.ReplyMessage) error {
	t.mu.RLock()
	api := t.api
	t.mu.RUnlock()
	if api == nil {
		return fmt.Errorf("飞书客户端尚未启动")
	}
	content, err := json.Marshal(map[string]string{"text": m.Text})
	if err != nil {
		return err
	}
	resp, err := api.Im.V1.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().ReceiveIdType("chat_id").Body(larkim.NewCreateMessageReqBodyBuilder().ReceiveId(m.ChatID).MsgType("text").Content(string(content)).Build()).Build())
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("飞书回复失败：%d %s", resp.Code, resp.Msg)
	}
	return nil
}
func (t *Transport) Status() transport.Status { t.mu.RLock(); defer t.mu.RUnlock(); return t.state }
func (t *Transport) setStatus(connected bool, detail string) {
	t.mu.Lock()
	t.state = transport.Status{Connected: connected, Detail: detail}
	listener := t.onStatus
	state := t.state
	t.mu.Unlock()
	if listener != nil {
		listener(state)
	}
}
func (t *Transport) onMessage(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Sender == nil || event.Event.Sender.SenderId == nil || event.Event.Sender.SenderId.OpenId == nil || event.Event.Message == nil || event.Event.Message.ChatId == nil || event.Event.Message.Content == nil {
		return nil
	}
	// Device information and administrative commands are available in private
	// conversations only; never send a response into a group chat.
	if event.Event.Message.ChatType == nil || *event.Event.Message.ChatType != "p2p" {
		return nil
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(*event.Event.Message.Content), &body); err != nil {
		return nil
	}
	id := ""
	if event.EventV2Base != nil && event.EventV2Base.Header != nil {
		id = event.EventV2Base.Header.EventID
	}
	t.mu.RLock()
	h := t.handler
	t.mu.RUnlock()
	if h == nil {
		return nil
	}
	message := IncomingMessage{EventID: id, UserID: *event.Event.Sender.SenderId.OpenId, ChatID: *event.Event.Message.ChatId, Text: body.Text}
	// The platform expects the event callback to return within three seconds.
	// Deduplication is persisted by the handler before the command executes;
	// run network work asynchronously so a slow ICMP target cannot cause retry.
	go func() { _ = h(context.Background(), message) }()
	return nil
}
