package main

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"net/netip"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/xint-tech/xint-lanpulse/internal/checker"
	"github.com/xuri/excelize/v2"
)

//go:embed all:frontend
var assets embed.FS

type App struct{}

func (a *App) Ping(address string, timeoutMS, attempts int) string {
	address = strings.TrimSpace(address)
	ip, err := netip.ParseAddr(address)
	if err != nil || !ip.Is4() {
		return "请输入有效的 IPv4 地址"
	}
	if timeoutMS < 100 || timeoutMS > 60000 || attempts < 1 || attempts > 10 {
		return "超时须为 100～60000 毫秒，次数须为 1～10"
	}
	result := checker.New().Ping(context.Background(), ip.String(), timeoutMS, attempts)
	if result.Online {
		return fmt.Sprintf("%s 在线（%d ms）", ip, result.LatencyMS)
	}
	return fmt.Sprintf("%s 暂未响应", ip)
}

// ExportMigration produces an XLSX accepted by the later device importer.
func (a *App) ExportMigration(addresses []string) (string, error) {
	if len(addresses) == 0 || len(addresses) > 254 {
		return "", fmt.Errorf("设备数量须为 1～254")
	}
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	for i, header := range []string{"设备IP", "设备名称", "设备区域"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellStr(sheet, cell, header); err != nil {
			return "", err
		}
	}
	seen := make(map[string]bool)
	for i, address := range addresses {
		ip, err := netip.ParseAddr(address)
		if err != nil || !ip.Is4() {
			return "", fmt.Errorf("设备地址无效：%s", address)
		}
		octets := ip.As4()
		if octets[0] != 192 || octets[1] != 168 || octets[2] != 20 || octets[3] == 0 || octets[3] == 255 || seen[ip.String()] {
			return "", fmt.Errorf("迁移表仅支持不重复的 192.168.20.1～254 地址：%s", address)
		}
		seen[ip.String()] = true
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetCellStr(sheet, cell, ip.String()); err != nil {
			return "", err
		}
	}
	data, err := f.WriteToBuffer()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data.Bytes()), nil
}

func main() {
	if err := wails.Run(&options.App{Title: "XinT-LanPulse v0.2.5", Width: 640, Height: 720,
		AssetServer: &assetserver.Options{Assets: assets}, Bind: []interface{}{&App{}}}); err != nil {
		panic(err)
	}
}
