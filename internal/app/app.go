package app

import (
	"context"
	"fmt"
	"github.com/xint-tech/xint-lanpulse/internal/checker"
	"github.com/xint-tech/xint-lanpulse/internal/command"
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"github.com/xint-tech/xint-lanpulse/internal/feishu"
	"github.com/xint-tech/xint-lanpulse/internal/logger"
	"github.com/xint-tech/xint-lanpulse/internal/security"
	"github.com/xint-tech/xint-lanpulse/internal/transport"
	feishudirect "github.com/xint-tech/xint-lanpulse/internal/transport/feishu_direct"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Application struct {
	Config         config.Config
	Store          *config.Store
	Logger         *logger.Logger
	Checker        checker.Checker
	Transport      transport.Transport
	Commands       *command.Service
	Started        time.Time
	Events         *feishu.EventStore
	mu             sync.RWMutex
	last           checker.Result
	scheduleCancel context.CancelFunc
}

func New() (*Application, error) {
	store := config.DefaultStore()
	c, err := store.Load()
	if err != nil {
		return nil, err
	}
	if ensureComponentDefaults(&c) {
		if err := store.Save(c); err != nil {
			return nil, err
		}
	}
	l, err := logger.New(filepath.Join(store.Dir(), "logs"), c.App.SaveRuntimeLogs)
	if err != nil {
		return nil, err
	}
	events, err := feishu.NewEventStore(filepath.Join(store.Dir(), "data"))
	if err != nil {
		_ = l.Close()
		return nil, err
	}
	a := &Application{Config: c, Store: store, Logger: l, Checker: checker.New(), Started: time.Now(), Events: events}
	hostname, _ := os.Hostname()
	a.Commands = &command.Service{Config: &a.Config, Store: store, Checker: a.Checker, Started: a.Started, ComputerName: hostname, Status: func() string { return a.Transport.Status().Detail }, OnLog: l.Info, OnPingResult: a.RecordPing}
	if err := a.refreshTransport(); err != nil {
		l.Info("飞书凭证无法解密，请重新保存 App Secret")
	}
	a.configureDeviceSchedule()
	l.Info("XinT-LanPulse 已启动")
	return a, nil
}
func DefaultComponents() []config.ComponentConfig {
	return []config.ComponentConfig{{ID: "device-status", Name: "内网设备状态检测", Description: "设备列表、IP 连通性检测与飞书查询", Enabled: true, Source: "built-in"}, {ID: "ipcamera-addon", Name: "监控设备增量包", Description: "监控管理与监控画面异常检测扩展", Enabled: false, Source: "built-in-addon"}}
}
func ensureComponentDefaults(c *config.Config) bool {
	changed := false
	known := map[string]bool{}
	for _, item := range c.Components {
		known[item.ID] = true
	}
	for _, item := range DefaultComponents() {
		if !known[item.ID] {
			c.Components = append(c.Components, item)
			changed = true
		}
	}
	if c.DeviceTask.IntervalMinutes <= 0 {
		c.DeviceTask.IntervalMinutes = 15
		changed = true
	}
	if c.Network.ConcurrentWorkers <= 0 {
		c.Network.ConcurrentWorkers = 8
		changed = true
	}
	return changed
}
func (a *Application) Save(c config.Config) error {
	if err := a.Store.Save(c); err != nil {
		return err
	}
	a.Config = c
	a.Commands.Config = &a.Config
	a.configureDeviceSchedule()
	if err := a.refreshTransport(); err != nil {
		return err
	}
	a.Logger.Info("本地配置已保存")
	return nil
}
func (a *Application) configureDeviceSchedule() {
	if a.scheduleCancel != nil {
		a.scheduleCancel()
		a.scheduleCancel = nil
	}
	if !a.Config.DeviceTask.Enabled || len(a.Config.Assets) == 0 {
		return
	}
	interval := time.Duration(a.Config.DeviceTask.IntervalMinutes) * time.Minute
	if interval < time.Minute {
		interval = time.Minute
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.scheduleCancel = cancel
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.runScheduledAssetChecks(ctx)
			}
		}
	}()
	a.Logger.Info(fmt.Sprintf("设备定时检测已启用：每 %d 分钟", a.Config.DeviceTask.IntervalMinutes))
}
func (a *Application) runScheduledAssetChecks(ctx context.Context) {
	items := a.CheckAssets(ctx)
	online := 0
	for _, item := range items {
		if item.Result.Online {
			online++
		}
	}
	a.Logger.Info(fmt.Sprintf("定时设备检测完成：%d/%d 在线", online, len(items)))
}

type AssetCheck struct {
	Asset     config.Asset
	Result    checker.Result
	CheckedAt time.Time
}

func (a *Application) CheckAssets(ctx context.Context) []AssetCheck {
	c := a.Config
	items := make([]AssetCheck, len(c.Assets))
	workers := c.Network.ConcurrentWorkers
	if workers < 1 {
		workers = 8
	}
	if workers > len(c.Assets) {
		workers = len(c.Assets)
	}
	if workers == 0 {
		return items
	}
	jobs := make(chan int)
	var workersDone sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		workersDone.Add(1)
		go func() {
			defer workersDone.Done()
			for index := range jobs {
				asset := c.Assets[index]
				address := fmt.Sprintf("%s.%d", c.Network.Prefix, asset.Suffix)
				result := a.Checker.Ping(ctx, address, c.Network.TimeoutMS, c.Network.Attempts)
				a.RecordPing(result)
				items[index] = AssetCheck{Asset: asset, Result: result, CheckedAt: time.Now()}
			}
		}()
	}
	for index := range c.Assets {
		jobs <- index
	}
	close(jobs)
	workersDone.Wait()
	return items
}
func (a *Application) refreshTransport() error {
	if a.Transport != nil {
		_ = a.Transport.Stop(context.Background())
	}
	secret := ""
	var decryptErr error
	if a.Config.Feishu.AppSecretEncrypted != "" {
		secret, decryptErr = security.Decrypt(a.Config.Feishu.AppSecretEncrypted)
	}
	t := feishudirect.New(a.Config.Feishu.AppID, secret)
	t.SetStatusListener(func(status transport.Status) { a.Logger.Info(status.Detail) })
	h := feishu.NewHandler(a.Commands, a.Events, func(ctx context.Context, chatID, text string) error {
		err := t.SendReply(ctx, transport.ReplyMessage{ChatID: chatID, Text: text})
		if err != nil {
			a.Logger.Info("飞书消息回复失败：" + err.Error())
		} else {
			a.Logger.Info("飞书消息回复成功")
		}
		return err
	}, a.Logger.Info)
	t.SetHandler(func(ctx context.Context, m feishudirect.IncomingMessage) error {
		err := h.Handle(ctx, feishu.IncomingMessage{EventID: m.EventID, UserID: m.UserID, ChatID: m.ChatID, Text: m.Text})
		if err != nil {
			a.Logger.Info("飞书消息处理失败")
		}
		return err
	})
	a.Transport = t
	return decryptErr
}
func (a *Application) TestPing(ctx context.Context, suffix string) (string, error) {
	n, err := config.ValidateSuffix(suffix)
	if err != nil {
		return "", err
	}
	target := fmt.Sprintf("%s.%d", a.Config.Network.Prefix, n)
	r := a.Checker.Ping(ctx, target, a.Config.Network.TimeoutMS, a.Config.Network.Attempts)
	a.RecordPing(r)
	if r.Online {
		return fmt.Sprintf("%s 在线，响应时间 %d ms", target, r.LatencyMS), nil
	}
	return fmt.Sprintf("%s 暂未响应（检测次数：%d，单次超时：%d ms）", target, r.Attempts, r.TimeoutMS), nil
}
func (a *Application) RecordPing(result checker.Result) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.last = result
}
func (a *Application) LatestPing() checker.Result { a.mu.RLock(); defer a.mu.RUnlock(); return a.last }
func (a *Application) StartConnection(ctx context.Context) error {
	err := a.Transport.Start(ctx)
	if err != nil {
		a.Logger.Info("飞书连接启动失败")
		return err
	}
	a.Logger.Info("飞书长连接启动请求已发送")
	return nil
}
func (a *Application) StopConnection(ctx context.Context) error {
	err := a.Transport.Stop(ctx)
	if err == nil {
		a.Logger.Info("飞书长连接已停止")
	}
	return err
}
func (a *Application) Close() {
	if a.scheduleCancel != nil {
		a.scheduleCancel()
	}
	a.Logger.Info("XinT-LanPulse 正在退出")
	_ = a.StopConnection(context.Background())
	_ = a.Logger.Close()
}
