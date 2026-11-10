package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	core "github.com/xint-tech/xint-lanpulse/internal/app"
	"github.com/xint-tech/xint-lanpulse/internal/security"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

// A tiny opaque ICO keeps the notification-area entry visible without relying
// on an external file next to the executable.
var trayIcon = []byte{
	0, 0, 1, 0, 1, 0, 1, 1, 0, 0, 1, 0, 32, 0, 48, 0, 0, 0, 22, 0, 0, 0,
	40, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 1, 0, 32, 0, 0, 0, 0, 0,
	4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	192, 204, 88, 255, 0, 0, 0, 0,
}

func loadTrayIcon() []byte {
	exe, err := os.Executable()
	if err == nil {
		if icon, readErr := os.ReadFile(filepath.Join(filepath.Dir(exe), "icon.ico")); readErr == nil && len(icon) > 0 {
			return icon
		}
	}
	return trayIcon
}

func main() {
	debug.SetMemoryLimit(180 << 20)
	debug.SetGCPercent(80)
	application, err := core.New()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) == 2 && os.Args[1] == "--configure-from-env" {
		configureFromEnv(application)
		application.Close()
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "--grant-query-open-id" {
		grantQuery(application, os.Args[2])
		application.Close()
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "--grant-admin-open-id" {
		grantAdmin(application, os.Args[2])
		application.Close()
		return
	}
	desktop := NewDesktopAPI(application)
	var exitRequested atomic.Bool
	var trayOnce sync.Once
	startTray := func(ctx context.Context) {
		trayOnce.Do(func() {
			go systray.Run(func() {
				systray.SetIcon(loadTrayIcon())
				systray.SetTooltip("XinT-LanPulse")
				show := systray.AddMenuItem("显示主窗口", "显示 XinT-LanPulse")
				systray.AddSeparator()
				quit := systray.AddMenuItem("退出", "退出 XinT-LanPulse")
				go func() {
					for {
						select {
						case <-show.ClickedCh:
							runtime.WindowShow(ctx)
							runtime.WindowUnminimise(ctx)
						case <-quit.ClickedCh:
							exitRequested.Store(true)
							runtime.Quit(ctx)
							return
						}
					}
				}()
			}, func() {})
		})
	}
	if err := wails.Run(&options.App{
		Title:            "XinT-LanPulse",
		Width:            1040,
		Height:           760,
		MinWidth:         860,
		MinHeight:        620,
		DisableResize:    false,
		AssetServer:      &assetserver.Options{Assets: frontendAssets},
		BackgroundColour: &options.RGBA{R: 18, G: 21, B: 26, A: 255},
		OnStartup: func(ctx context.Context) {
			desktop.startup(ctx)
			startTray(ctx)
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if application.Config.App.CloseToTray && !exitRequested.Load() {
				runtime.WindowHide(ctx)
				return true
			}
			return false
		},
		OnShutdown: func(ctx context.Context) {
			systray.Quit()
			desktop.shutdown(ctx)
		},
		Bind:    []interface{}{desktop},
		Windows: &windows.Options{Theme: windows.Dark},
	}); err != nil {
		application.Close()
		log.Fatal(err)
	}
}

func configureFromEnv(application *core.Application) {
	appID, secret := os.Getenv("LANTPULSE_APP_ID"), os.Getenv("LANTPULSE_APP_SECRET")
	if appID == "" || secret == "" {
		log.Fatal("LANTPULSE_APP_ID 和 LANTPULSE_APP_SECRET 必须同时设置")
	}
	cipher, err := security.Encrypt(secret)
	if err != nil {
		log.Fatal(err)
	}
	c := application.Config
	c.Feishu.AppID, c.Feishu.AppSecretEncrypted = appID, cipher
	if err := application.Save(c); err != nil {
		log.Fatal(err)
	}
	fmt.Println("飞书凭证已写入当前 Windows 用户的加密配置。")
}

func grantQuery(application *core.Application, openID string) {
	if len(openID) < 4 || openID[:3] != "ou_" {
		log.Fatal("Open ID 格式无效")
	}
	c := application.Config
	for _, existing := range c.Feishu.AllowedUserIDs {
		if existing == openID {
			return
		}
	}
	c.Feishu.AllowedUserIDs = append(c.Feishu.AllowedUserIDs, openID)
	if err := application.Save(c); err != nil {
		log.Fatal(err)
	}
	fmt.Println("已授予查询权限。")
}

func grantAdmin(application *core.Application, openID string) {
	grantQuery(application, openID)
	c := application.Config
	for _, existing := range c.Feishu.AdminUserIDs {
		if existing == openID {
			return
		}
	}
	c.Feishu.AdminUserIDs = append(c.Feishu.AdminUserIDs, openID)
	if err := application.Save(c); err != nil {
		log.Fatal(err)
	}
	fmt.Println("已授予管理员权限。")
}

var _ = context.Background
