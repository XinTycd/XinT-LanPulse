package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
)

const DefaultPrefix = "192.168.20"

type Config struct {
	App                 AppConfig         `json:"app"`
	Network             NetworkConfig     `json:"network"`
	Feishu              FeishuConfig      `json:"feishu"`
	Platforms           PlatformConfigs   `json:"platforms"`
	Components          []ComponentConfig `json:"components"`
	DeviceTask          DeviceTaskConfig  `json:"deviceTask"`
	CameraAssetSuffixes []int             `json:"cameraAssetSuffixes"`
	Assets              []Asset           `json:"assets"`
	Transport           TransportConfig   `json:"transport"`
}

type AppConfig struct {
	DeviceID        string `json:"deviceId"`
	DeviceName      string `json:"deviceName"`
	AutoStart       bool   `json:"autoStart"`
	StartMinimized  bool   `json:"startMinimized"`
	CloseToTray     bool   `json:"closeToTray"`
	SaveRuntimeLogs bool   `json:"saveRuntimeLogs"`
}

func EnsureDeviceIdentity(c *Config) bool {
	changed := false
	if c.App.DeviceID == "" {
		bytes := make([]byte, 4)
		if _, err := rand.Read(bytes); err == nil {
			c.App.DeviceID = "dev-" + hex.EncodeToString(bytes)
		} else {
			c.App.DeviceID = "dev-local"
		}
		changed = true
	}
	if c.App.DeviceName == "" {
		if hostname, err := os.Hostname(); err == nil && hostname != "" {
			c.App.DeviceName = hostname
		} else {
			c.App.DeviceName = "未命名客户端"
		}
		changed = true
	}
	return changed
}

type NetworkConfig struct {
	Prefix            string `json:"prefix"`
	TimeoutMS         int    `json:"timeoutMs"`
	Attempts          int    `json:"attempts"`
	ConcurrentWorkers int    `json:"concurrentWorkers"`
}
type FeishuConfig struct {
	AppID              string   `json:"appId"`
	AppSecretEncrypted string   `json:"appSecretEncrypted"`
	AllowedUserIDs     []string `json:"allowedUserIds"`
	AdminUserIDs       []string `json:"adminUserIds"`
}
type PlatformConfig struct {
	Enabled         bool   `json:"enabled"`
	AppKey          string `json:"appKey"`
	SecretEncrypted string `json:"secretEncrypted"`
}
type PlatformConfigs struct {
	Active   string         `json:"active"`
	DingTalk PlatformConfig `json:"dingtalk"`
	WeCom    PlatformConfig `json:"wecom"`
}
type TransportConfig struct {
	Mode string `json:"mode"`
}
type Asset struct {
	Suffix int    `json:"suffix"`
	Name   string `json:"name"`
	Area   string `json:"area"`
}
type ComponentConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source"`
}
type DeviceTaskConfig struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"intervalMinutes"`
}

func Default() Config {
	c := Config{App: AppConfig{CloseToTray: true, SaveRuntimeLogs: true}, Network: NetworkConfig{Prefix: DefaultPrefix, TimeoutMS: 1000, Attempts: 2, ConcurrentWorkers: 8}, Transport: TransportConfig{Mode: "feishu_direct"}, Feishu: FeishuConfig{AllowedUserIDs: []string{}, AdminUserIDs: []string{}}, Platforms: PlatformConfigs{Active: "feishu"}, Components: []ComponentConfig{{ID: "device-status", Name: "内网设备状态检测", Description: "设备列表、IP 连通性检测与飞书查询", Enabled: true, Source: "built-in"}, {ID: "ipcamera-addon", Name: "监控设备增量包", Description: "监控管理与监控画面异常检测扩展", Enabled: false, Source: "built-in-addon"}}, DeviceTask: DeviceTaskConfig{IntervalMinutes: 15}, Assets: []Asset{}}
	EnsureDeviceIdentity(&c)
	return c
}
