package tests

import (
	"context"
	"github.com/xint-tech/xint-lanpulse/internal/checker"
	"github.com/xint-tech/xint-lanpulse/internal/command"
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"strings"
	"testing"
	"time"
)

type fakeChecker struct{ online bool }

func (f fakeChecker) Ping(context.Context, string, int, int) checker.Result {
	return checker.Result{Online: f.online, LatencyMS: 3, Attempts: 1}
}

type fakeStore struct{ saved config.Config }

func (f *fakeStore) Save(c config.Config) error { f.saved = c; return nil }
func service(online bool) (*command.Service, *config.Config, *fakeStore) {
	c := config.Default()
	c.Feishu.AllowedUserIDs = []string{"ou_user", "ou_admin"}
	c.Feishu.AdminUserIDs = []string{"ou_admin"}
	s := &fakeStore{}
	return &command.Service{Config: &c, Store: s, Checker: fakeChecker{online}, Started: time.Now()}, &c, s
}
func TestCommandsAndAuthorization(t *testing.T) {
	svc, c, s := service(true)
	if got := svc.Execute(context.Background(), "intruder", "15"); !strings.Contains(got, "没有使用") {
		t.Fatal(got)
	}
	got := svc.Execute(context.Background(), "ou_user", "15")
	if !strings.Contains(got, "IP 在线") {
		t.Fatal(got)
	}
	got = svc.Execute(context.Background(), "ou_user", "/net 10.0.0")
	if !strings.Contains(got, "没有修改") {
		t.Fatal(got)
	}
	got = svc.Execute(context.Background(), "ou_admin", "/net 10.0.0")
	if !strings.Contains(got, "已修改") || c.Network.Prefix != "10.0.0" || s.saved.Network.Prefix != "10.0.0" {
		t.Fatalf("%s %+v", got, c)
	}
}
func TestOfflineResponse(t *testing.T) {
	svc, _, _ := service(false)
	got := svc.Execute(context.Background(), "ou_user", "/ping 15")
	if !strings.Contains(got, "暂未响应") {
		t.Fatal(got)
	}
}

func TestFixed20NetworkCommand(t *testing.T) {
	svc, c, _ := service(true)
	c.Network.Prefix = "192.168.30"
	got := svc.Execute(context.Background(), "ou_user", "/20 15")
	if !strings.Contains(got, "192.168.20.15") || !strings.Contains(got, "IP 在线") {
		t.Fatal(got)
	}
}

func TestTemporaryNetworkSegmentCommand(t *testing.T) {
	svc, _, _ := service(true)
	got := svc.Execute(context.Background(), "ou_user", "/110 15")
	if !strings.Contains(got, "192.168.110.15") {
		t.Fatal(got)
	}
}

func TestDeviceCommandIncludesAssetMetadata(t *testing.T) {
	svc, c, _ := service(true)
	c.Assets = []config.Asset{{Suffix: 15, Name: "主控显示器", Area: "一楼中控室"}}
	got := svc.Execute(context.Background(), "ou_user", "/vi 15")
	if !strings.Contains(got, "设备IP：192.168.20.15") || !strings.Contains(got, "设备名称：主控显示器") || !strings.Contains(got, "设备区域：一楼中控室") || !strings.Contains(got, "检测次数：") || !strings.Contains(got, "检测时间：") {
		t.Fatal(got)
	}
}

func TestVideoInspectAllAndNo(t *testing.T) {
	svc, c, _ := service(true)
	c.Assets = []config.Asset{{Suffix: 15, Name: "主控屏", Area: "一楼"}, {Suffix: 16, Name: "副控屏", Area: "二楼"}}
	all := svc.Execute(context.Background(), "ou_user", "/vi all")
	if !strings.Contains(all, "监控设备巡检结果") || !strings.Contains(all, "192.168.20.15") || !strings.Contains(all, "✅ 在线") {
		t.Fatal(all)
	}
	no := svc.Execute(context.Background(), "ou_user", "/vi no")
	if !strings.Contains(no, "所有 2 台监控设备均可连接") {
		t.Fatal(no)
	}
}

func TestVideoInspectNoReturnsOfflineDevices(t *testing.T) {
	svc, c, _ := service(false)
	c.Assets = []config.Asset{{Suffix: 15, Name: "主控屏", Area: "一楼"}}
	no := svc.Execute(context.Background(), "ou_user", "/vi no")
	if !strings.Contains(no, "无法连接的监控设备") || !strings.Contains(no, "主控屏") || !strings.Contains(no, "❌ 无法连接") {
		t.Fatal(no)
	}
}
