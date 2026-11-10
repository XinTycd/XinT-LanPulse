package command

import (
	"fmt"
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"strconv"
	"strings"
)

type Kind int

const (
	Invalid Kind = iota
	Ping
	PingSegment
	VideoInspect
	Net
	SetNet
	Status
	Help
)

type Parsed struct {
	Kind   Kind
	Value  string
	Prefix string
}

func Parse(text string) Parsed {
	v := strings.TrimSpace(text)
	lower := strings.ToLower(v)
	if _, err := config.ValidateSuffix(v); err == nil {
		return Parsed{Kind: Ping, Value: v}
	}
	fields := strings.Fields(lower)
	if len(fields) == 1 && fields[0] == "/ping" {
		return Parsed{Kind: Invalid}
	}
	if len(fields) == 2 && fields[0] == "/ping" {
		return Parsed{Kind: Ping, Value: fields[1]}
	}
	if len(fields) == 2 && fields[0] == "/vi" {
		return Parsed{Kind: VideoInspect, Value: fields[1]}
	}
	if len(fields) == 2 && strings.HasPrefix(fields[0], "/") {
		segment, err := strconv.Atoi(strings.TrimPrefix(fields[0], "/"))
		if err == nil && segment >= 0 && segment <= 255 {
			return Parsed{Kind: PingSegment, Value: fields[1], Prefix: fmt.Sprintf("192.168.%d", segment)}
		}
	}
	if lower == "/net" {
		return Parsed{Kind: Net}
	}
	if len(fields) == 2 && fields[0] == "/net" {
		return Parsed{Kind: SetNet, Value: fields[1]}
	}
	if lower == "/status" {
		return Parsed{Kind: Status}
	}
	if lower == "/help" {
		return Parsed{Kind: Help}
	}
	return Parsed{Kind: Invalid}
}
