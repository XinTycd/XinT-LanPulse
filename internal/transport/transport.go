package transport

import "context"

type ReplyMessage struct {
	ChatID string
	Text   string
}
type Status struct {
	Connected bool
	Detail    string
}
type Transport interface {
	Start(context.Context) error
	Stop(context.Context) error
	SendReply(context.Context, ReplyMessage) error
	Status() Status
}
