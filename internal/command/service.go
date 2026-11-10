package command

import (
	"context"
	"fmt"
	"github.com/xint-tech/xint-lanpulse/internal/checker"
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"github.com/xint-tech/xint-lanpulse/internal/security"
	"strings"
	"sync"
	"time"
)

type Store interface{ Save(config.Config) error }
type StatusProvider func() string
type Service struct {
	Config       *config.Config
	Store        Store
	Checker      checker.Checker
	Started      time.Time
	ComputerName string
	Status       StatusProvider
	OnLog        func(string)
	OnPingResult func(checker.Result)
}

func (s *Service) Execute(ctx context.Context, userID, message string) string {
	if !security.Allowed(*s.Config, userID) {
		return "⛔ 你没有使用 XinT-LanPulse 的权限。"
	}
	p := Parse(message)
	switch p.Kind {
	case Ping:
		return s.ping(ctx, userID, p.Value)
	case PingSegment:
		return s.pingPrefix(ctx, userID, p.Prefix, p.Value)
	case VideoInspect:
		return s.videoInspect(ctx, userID, p.Value)
	case Net:
		return fmt.Sprintf("当前 IP 网段：%s.x\n\n示例：\n发送 15，将检测 %s.15", s.Config.Network.Prefix, s.Config.Network.Prefix)
	case SetNet:
		return s.setNet(userID, p.Value)
	case Status:
		return s.status()
	case Help:
		return HelpText()
	default:
		return "输入无效，请发送 1～254 之间的数字。\n\n例如：\n15\n/ping 15"
	}
}
func (s *Service) ping(ctx context.Context, user, suffix string) string {
	return s.pingPrefix(ctx, user, s.Config.Network.Prefix, suffix)
}
func (s *Service) pingPrefix(ctx context.Context, user, prefix, suffix string) string {
	n, err := config.ValidateSuffix(suffix)
	if err != nil {
		return err.Error()
	}
	target := fmt.Sprintf("%s.%d", prefix, n)
	s.log(fmt.Sprintf("用户 %s 查询 %s", user, target))
	r := s.Checker.Ping(ctx, target, s.Config.Network.TimeoutMS, s.Config.Network.Attempts)
	if s.OnPingResult != nil {
		s.OnPingResult(r)
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if r.Online {
		s.log(fmt.Sprintf("%s 在线，延迟 %d ms", target, r.LatencyMS))
		return fmt.Sprintf("✅ IP 在线\n\n目标地址：%s\n响应时间：%d ms\n检测次数：%d\n检测时间：%s", target, r.LatencyMS, r.Attempts, now)
	}
	s.log(fmt.Sprintf("%s 暂未响应", target))
	return fmt.Sprintf("❌ IP 暂未响应\n\n目标地址：%s\n检测次数：%d\n单次超时：%d ms\n检测时间：%s", target, r.Attempts, r.TimeoutMS, now)
}
func (s *Service) videoInspect(ctx context.Context, user, suffix string) string {
	if suffix == "all" {
		return s.inspectAll(ctx, user, false)
	}
	if suffix == "no" {
		return s.inspectAll(ctx, user, true)
	}
	n, err := config.ValidateSuffix(suffix)
	if err != nil {
		return err.Error()
	}
	asset := config.Asset{Suffix: n, Name: "未登记", Area: "未登记"}
	for _, item := range s.Config.Assets {
		if item.Suffix == n {
			asset = item
			break
		}
	}
	target := fmt.Sprintf("192.168.20.%d", n)
	r := s.Checker.Ping(ctx, target, s.Config.Network.TimeoutMS, s.Config.Network.Attempts)
	if r.Address == "" {
		r.Address = target
	}
	if s.OnPingResult != nil {
		s.OnPingResult(r)
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	response := fmt.Sprintf("设备名称：%s\n设备区域：%s\n设备IP：%s", asset.Name, asset.Area, r.Address)
	if r.Online {
		return "✅ 监控设备在线\n\n" + response + fmt.Sprintf("\n响应时间：%d ms\n检测次数：%d\n检测时间：%s", r.LatencyMS, r.Attempts, now)
	}
	return "❌ 监控设备暂未响应\n\n" + response + fmt.Sprintf("\n响应时间：暂未响应\n检测次数：%d\n检测时间：%s", r.Attempts, now)
}

type deviceResult struct {
	asset  config.Asset
	result checker.Result
}

func (s *Service) inspectAll(ctx context.Context, user string, offlineOnly bool) string {
	assets := append([]config.Asset(nil), s.Config.Assets...)
	if len(assets) == 0 {
		return "设备列表为空，请先在客户端导入或添加设备。"
	}
	s.log(fmt.Sprintf("用户 %s 发起 %d 台设备的批量巡检", user, len(assets)))
	results := make([]deviceResult, len(assets))
	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := 8
	if len(assets) < workers {
		workers = len(assets)
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				asset := assets[index]
				target := fmt.Sprintf("192.168.20.%d", asset.Suffix)
				r := s.Checker.Ping(ctx, target, s.Config.Network.TimeoutMS, s.Config.Network.Attempts)
				if r.Address == "" {
					r.Address = target
				}
				if s.OnPingResult != nil {
					s.OnPingResult(r)
				}
				results[index] = deviceResult{asset: asset, result: r}
			}
		}()
	}
	for index := range assets {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	rows := []string{"设备IP | 设备名称 | 设备区域 | 状态 | 响应时间"}
	shown := 0
	for _, item := range results {
		if offlineOnly && item.result.Online {
			continue
		}
		name, area := item.asset.Name, item.asset.Area
		if name == "" {
			name = "未命名"
		}
		if area == "" {
			area = "未登记"
		}
		state, response := "❌ 无法连接", "暂未响应"
		if item.result.Online {
			state = "✅ 在线"
			response = fmt.Sprintf("%d ms", item.result.LatencyMS)
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s | %s | %s", item.result.Address, name, area, state, response))
		shown++
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if offlineOnly && shown == 0 {
		return fmt.Sprintf("✅ 所有 %d 台监控设备均可连接\n\n检测次数：%d\n检测时间：%s", len(assets), s.Config.Network.Attempts, now)
	}
	title := "监控设备巡检结果"
	if offlineOnly {
		title = "无法连接的监控设备"
	}
	s.log(fmt.Sprintf("批量巡检完成：共 %d 台，返回 %d 台", len(assets), shown))
	return fmt.Sprintf("%s\n共检测：%d 台；本次显示：%d 台\n检测次数：%d\n检测时间：%s\n\n%s", title, len(assets), shown, s.Config.Network.Attempts, now, strings.Join(rows, "\n"))
}
func (s *Service) setNet(user, input string) string {
	if !security.Admin(*s.Config, user) {
		return "⛔ 你没有修改 IP 网段的权限。"
	}
	prefix, err := config.NormalizePrefix(input)
	if err != nil {
		return "网段格式无效，请使用例如 192.168.30。"
	}
	old := s.Config.Network.Prefix
	next := *s.Config
	next.Network.Prefix = prefix
	if err = s.Store.Save(next); err != nil {
		return "❌ IP 网段保存失败，当前设置未修改。"
	}
	*s.Config = next
	s.log(fmt.Sprintf("管理员将网段修改为 %s", prefix))
	return fmt.Sprintf("✅ IP 网段已修改\n\n原网段：%s.x\n新网段：%s.x\n\n现在发送 15，将检测 %s.15", old, prefix, prefix)
}
func (s *Service) status() string {
	extra := "飞书连接：未连接"
	if s.Status != nil {
		extra = s.Status()
	}
	return fmt.Sprintf("✅ XinT-LanPulse 运行正常\n\n%s\n当前网段：%s.x\n运行电脑：%s\n程序版本：v0.3.0\n运行时间：%s", extra, s.Config.Network.Prefix, s.ComputerName, formatDuration(time.Since(s.Started)))
}
func (s *Service) log(msg string) {
	if s.OnLog != nil {
		s.OnLog(msg)
	}
}
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	return strings.TrimPrefix(d.String(), "0s")
}
func HelpText() string {
	return "XinT-LanPulse 命令帮助\n\n/vi 15\n检测单台 192.168.20.15 监控设备\n\n/vi all\n检测设备列表中的全部监控设备并返回巡检表\n\n/vi no\n检测全部监控设备，只返回无法连接的设备\n\n/20 15\n临时检测 192.168.20.15\n\n/110 15\n临时检测 192.168.110.15\n\n15 或 /ping 15\n检测当前默认网段\n\n/net\n查看当前网段\n\n/net 192.168.30\n修改当前网段\n\n/status\n查看客户端状态\n\n/help\n查看命令帮助"
}
