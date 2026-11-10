package feishu

import (
	"context"
	"github.com/xint-tech/xint-lanpulse/internal/command"
	"github.com/xint-tech/xint-lanpulse/internal/security"
	"strings"
	"time"
)

type IncomingMessage struct{ EventID, UserID, ChatID, Text string }
type Reply func(context.Context, string, string) error
type Handler struct {
	Commands *command.Service
	Events   *EventStore
	Limiter  *security.RateLimiter
	Reply    Reply
	Log      func(string)
}

func (h *Handler) Handle(ctx context.Context, m IncomingMessage) error {
	seen, err := h.Events.Seen(m.EventID)
	if err != nil {
		return err
	}
	if seen {
		if h.Log != nil {
			h.Log("忽略重复飞书事件 " + m.EventID)
		}
		return nil
	}
	if !h.Limiter.Allow(m.UserID) {
		return h.Reply(ctx, m.ChatID, "⏳ 查询过于频繁，请稍后再试。")
	}
	if h.Log != nil {
		h.Log("收到飞书用户 " + m.UserID + " 的消息")
	}
	response := h.Commands.Execute(ctx, m.UserID, m.Text)
	for _, part := range splitReply(response, 3500) {
		if err := h.Reply(ctx, m.ChatID, part); err != nil {
			return err
		}
	}
	return nil
}
func NewHandler(commands *command.Service, events *EventStore, reply Reply, log func(string)) *Handler {
	return &Handler{Commands: commands, Events: events, Reply: reply, Log: log, Limiter: security.NewRateLimiter(2 * time.Second)}
}
func splitReply(text string, limit int) []string {
	if len([]rune(text)) <= limit {
		return []string{text}
	}
	lines := strings.Split(text, "\n")
	parts := []string{}
	current := ""
	for _, line := range lines {
		candidate := line
		if current != "" {
			candidate = current + "\n" + line
		}
		if len([]rune(candidate)) <= limit {
			current = candidate
			continue
		}
		if current != "" {
			parts = append(parts, current)
		}
		current = line
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
