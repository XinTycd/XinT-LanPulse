package main

import (
	"context"
	"embed"
	"fmt"
	"net/netip"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/xint-tech/xint-lanpulse/internal/checker"
)

//go:embed all:frontend
var assets embed.FS

type App struct{}

func (a *App) Ping(address string) string {
	address = strings.TrimSpace(address)
	ip, err := netip.ParseAddr(address)
	if err != nil || !ip.Is4() {
		return "请输入有效的 IPv4 地址"
	}
	result := checker.New().Ping(context.Background(), ip.String(), 1000, 1)
	if result.Online {
		return fmt.Sprintf("%s 在线（%d ms）", ip, result.LatencyMS)
	}
	return fmt.Sprintf("%s 暂未响应", ip)
}

func main() {
	if err := wails.Run(&options.App{Title: "XinT-LanPulse v0.1.0", Width: 640, Height: 400,
		AssetServer: &assetserver.Options{Assets: assets}, Bind: []interface{}{&App{}}}); err != nil {
		panic(err)
	}
}
