package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	core "github.com/xint-tech/xint-lanpulse/internal/app"
	"github.com/xint-tech/xint-lanpulse/internal/assets"
	"github.com/xint-tech/xint-lanpulse/internal/components"
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"github.com/xint-tech/xint-lanpulse/internal/filepicker"
	"github.com/xint-tech/xint-lanpulse/internal/security"
)

type DesktopAPI struct{ app *core.Application }

type DesktopState struct {
	Connection string                   `json:"connection"`
	StartedAt  string                   `json:"startedAt"`
	LatestPing string                   `json:"latestPing"`
	Config     PublicConfig             `json:"config"`
	Assets     []config.Asset           `json:"assets"`
	Logs       []string                 `json:"logs"`
	Platforms  []PlatformState          `json:"platforms"`
	Components []config.ComponentConfig `json:"components"`
	DeviceTask config.DeviceTaskConfig  `json:"deviceTask"`
}
type PlatformState struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
type PublicConfig struct {
	DeviceID            string                  `json:"deviceID"`
	DeviceName          string                  `json:"deviceName"`
	AppID               string                  `json:"appID"`
	DingTalkAppKey      string                  `json:"dingTalkAppKey"`
	WeComCorpID         string                  `json:"weComCorpID"`
	ActivePlatform      string                  `json:"activePlatform"`
	Prefix              string                  `json:"prefix"`
	TimeoutMS           int                     `json:"timeoutMS"`
	Attempts            int                     `json:"attempts"`
	ConcurrentWorkers   int                     `json:"concurrentWorkers"`
	AllowedUserIDs      []string                `json:"allowedUserIDs"`
	AdminUserIDs        []string                `json:"adminUserIDs"`
	AutoStart           bool                    `json:"autoStart"`
	StartMinimized      bool                    `json:"startMinimized"`
	CloseToTray         bool                    `json:"closeToTray"`
	SaveRuntimeLogs     bool                    `json:"saveRuntimeLogs"`
	DeviceTask          config.DeviceTaskConfig `json:"deviceTask"`
	CameraAssetSuffixes []int                   `json:"cameraAssetSuffixes"`
}
type SettingsInput struct {
	DeviceName        string   `json:"deviceName"`
	AppID             string   `json:"appID"`
	AppSecret         string   `json:"appSecret"`
	Prefix            string   `json:"prefix"`
	TimeoutMS         int      `json:"timeoutMS"`
	Attempts          int      `json:"attempts"`
	ConcurrentWorkers int      `json:"concurrentWorkers"`
	AllowedUserIDs    []string `json:"allowedUserIDs"`
	AdminUserIDs      []string `json:"adminUserIDs"`
	AutoStart         bool     `json:"autoStart"`
	StartMinimized    bool     `json:"startMinimized"`
	CloseToTray       bool     `json:"closeToTray"`
	SaveRuntimeLogs   bool     `json:"saveRuntimeLogs"`
}
type PlatformSettingsInput struct {
	Platform string `json:"platform"`
	AppKey   string `json:"appKey"`
	Secret   string `json:"secret"`
}
type DeviceTaskInput struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"intervalMinutes"`
}
type AssetInput struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Area    string `json:"area"`
}
type AssetCheckResult struct {
	Suffix    int    `json:"suffix"`
	Name      string `json:"name"`
	Area      string `json:"area"`
	Address   string `json:"address"`
	Online    bool   `json:"online"`
	LatencyMS int    `json:"latencyMS"`
	Attempts  int    `json:"attempts"`
	CheckedAt string `json:"checkedAt"`
}

func NewDesktopAPI(app *core.Application) *DesktopAPI { return &DesktopAPI{app: app} }
func (d *DesktopAPI) startup(context.Context) {
	d.app.Logger.Info("Wails 界面已初始化")
	if d.app.Config.Feishu.AppID != "" && d.app.Config.Feishu.AppSecretEncrypted != "" {
		go func() { _ = d.app.StartConnection(context.Background()) }()
	}
}
func (d *DesktopAPI) shutdown(context.Context) { d.app.Close() }

func (d *DesktopAPI) GetState() DesktopState {
	c := d.app.Config
	latest := d.app.LatestPing()
	latestText := "暂无"
	if latest.Address != "" {
		latestText = fmt.Sprintf("%s · %s", latest.Address, map[bool]string{true: "在线", false: "暂未响应"}[latest.Online])
	}
	return DesktopState{
		Connection: d.connectionDetail(),
		StartedAt:  d.app.Started.Format(time.DateTime), LatestPing: latestText,
		Config: PublicConfig{DeviceID: c.App.DeviceID, DeviceName: c.App.DeviceName, AppID: c.Feishu.AppID, DingTalkAppKey: c.Platforms.DingTalk.AppKey, WeComCorpID: c.Platforms.WeCom.AppKey, ActivePlatform: activePlatform(c.Platforms.Active), Prefix: c.Network.Prefix, TimeoutMS: c.Network.TimeoutMS, Attempts: c.Network.Attempts, ConcurrentWorkers: c.Network.ConcurrentWorkers, AllowedUserIDs: c.Feishu.AllowedUserIDs, AdminUserIDs: c.Feishu.AdminUserIDs, AutoStart: c.App.AutoStart, StartMinimized: c.App.StartMinimized, CloseToTray: c.App.CloseToTray, SaveRuntimeLogs: c.App.SaveRuntimeLogs, DeviceTask: c.DeviceTask, CameraAssetSuffixes: append([]int(nil), c.CameraAssetSuffixes...)},
		Assets: append([]config.Asset{}, c.Assets...), Logs: d.app.Logger.Entries(),
		Platforms:  []PlatformState{{Name: "飞书", Status: cleanConnectionStatus(d.connectionDetail())}, {Name: "钉钉", Status: platformStatus(c.Platforms.DingTalk.Enabled)}, {Name: "企业微信", Status: platformStatus(c.Platforms.WeCom.Enabled)}},
		Components: append([]config.ComponentConfig{}, c.Components...), DeviceTask: c.DeviceTask,
	}
}
func (d *DesktopAPI) connectionDetail() string {
	if d.app.Transport == nil {
		return "未配置"
	}
	return d.app.Transport.Status().Detail
}
func activePlatform(value string) string {
	if value == "dingtalk" || value == "wecom" || value == "feishu" {
		return value
	}
	return "feishu"
}
func (d *DesktopAPI) SetComponentEnabled(id string, enabled bool) error {
	c := d.app.Config
	for i := range c.Components {
		if c.Components[i].ID == id {
			c.Components[i].Enabled = enabled
			return d.app.Save(c)
		}
	}
	return fmt.Errorf("未找到组件")
}
func (d *DesktopAPI) DeleteComponent(id string) error {
	if id == "device-status" {
		return fmt.Errorf("内置组件不可删除")
	}
	c := d.app.Config
	for i := range c.Components {
		if c.Components[i].ID == id {
			c.Components = append(c.Components[:i], c.Components[i+1:]...)
			return d.app.Save(c)
		}
	}
	return fmt.Errorf("未找到组件")
}
func (d *DesktopAPI) SelectPlatform(platform string) error {
	platform = activePlatform(platform)
	c := d.app.Config
	c.Platforms.Active = platform
	return d.app.Save(c)
}
func (d *DesktopAPI) SavePlatformSettings(input PlatformSettingsInput) error {
	platform := activePlatform(input.Platform)
	c := d.app.Config
	key := strings.TrimSpace(input.AppKey)
	if platform == "feishu" {
		c.Platforms.Active = platform
		if key != "" {
			c.Feishu.AppID = key
		}
		if input.Secret != "" {
			cipher, err := security.Encrypt(input.Secret)
			if err != nil {
				return err
			}
			c.Feishu.AppSecretEncrypted = cipher
		}
		return d.app.Save(c)
	}
	var item *config.PlatformConfig
	if platform == "dingtalk" {
		item = &c.Platforms.DingTalk
	} else {
		item = &c.Platforms.WeCom
	}
	if key == "" && item.AppKey == "" {
		return fmt.Errorf("请填写接入标识")
	}
	if key != "" {
		item.AppKey = key
	}
	if input.Secret != "" {
		cipher, err := security.Encrypt(input.Secret)
		if err != nil {
			return err
		}
		item.SecretEncrypted = cipher
	}
	item.Enabled = item.AppKey != "" && item.SecretEncrypted != ""
	c.Platforms.Active = platform
	return d.app.Save(c)
}
func platformStatus(enabled bool) string {
	if enabled {
		return "待接入"
	}
	return "未启用"
}

func (d *DesktopAPI) ImportComponentPackage() (config.ComponentConfig, error) {
	path, ok, err := filepicker.OpenComponentPackage()
	if err != nil || !ok {
		return config.ComponentConfig{}, err
	}
	item, err := components.FromFile(path)
	if err != nil {
		return config.ComponentConfig{}, err
	}
	return d.addComponent(item)
}
func (d *DesktopAPI) AddComponentURL(rawURL string) (config.ComponentConfig, error) {
	item, err := components.FromURL(rawURL)
	if err != nil {
		return config.ComponentConfig{}, err
	}
	return d.addComponent(item)
}
func (d *DesktopAPI) addComponent(item config.ComponentConfig) (config.ComponentConfig, error) {
	c := d.app.Config
	for _, existing := range c.Components {
		if existing.ID == item.ID {
			return config.ComponentConfig{}, fmt.Errorf("组件 ID 已存在：%s", item.ID)
		}
	}
	c.Components = append(c.Components, item)
	if err := d.app.Save(c); err != nil {
		return config.ComponentConfig{}, err
	}
	return item, nil
}

func (d *DesktopAPI) SaveSettings(input SettingsInput) error {
	if strings.TrimSpace(input.DeviceName) == "" {
		return fmt.Errorf("设备名称不能为空")
	}
	prefix, err := config.NormalizePrefix(input.Prefix)
	if err != nil {
		return err
	}
	if input.TimeoutMS <= 0 || input.Attempts <= 0 || input.ConcurrentWorkers < 1 || input.ConcurrentWorkers > 64 {
		return fmt.Errorf("Ping 超时、检测次数必须大于 0，同时检测线程数应为 1 到 64")
	}
	c := d.app.Config
	c.App.DeviceName, c.Feishu.AppID = strings.TrimSpace(input.DeviceName), strings.TrimSpace(input.AppID)
	c.Network.Prefix, c.Network.TimeoutMS, c.Network.Attempts, c.Network.ConcurrentWorkers = prefix, input.TimeoutMS, input.Attempts, input.ConcurrentWorkers
	c.Feishu.AllowedUserIDs, c.Feishu.AdminUserIDs = cleanIDs(input.AllowedUserIDs), cleanIDs(input.AdminUserIDs)
	c.App.AutoStart, c.App.StartMinimized, c.App.CloseToTray, c.App.SaveRuntimeLogs = input.AutoStart, input.StartMinimized, input.CloseToTray, input.SaveRuntimeLogs
	if input.AppSecret != "" {
		cipher, err := security.Encrypt(input.AppSecret)
		if err != nil {
			return err
		}
		c.Feishu.AppSecretEncrypted = cipher
	}
	if err := core.SetAutoStart(c.App.AutoStart); err != nil {
		return err
	}
	return d.app.Save(c)
}
func (d *DesktopAPI) SaveDeviceTask(input DeviceTaskInput) error {
	if input.IntervalMinutes < 1 || input.IntervalMinutes > 1440 {
		return fmt.Errorf("检测间隔应为 1 到 1440 分钟")
	}
	c := d.app.Config
	c.DeviceTask = config.DeviceTaskConfig{Enabled: input.Enabled, IntervalMinutes: input.IntervalMinutes}
	return d.app.Save(c)
}
func (d *DesktopAPI) BulkCheckAssets() ([]AssetCheckResult, error) {
	if !componentEnabled(d.app.Config.Components, "device-status") {
		return nil, fmt.Errorf("请先启用内网设备状态检测组件")
	}
	checks := d.app.CheckAssets(context.Background())
	items := make([]AssetCheckResult, 0, len(checks))
	for _, item := range checks {
		items = append(items, AssetCheckResult{Suffix: item.Asset.Suffix, Name: item.Asset.Name, Area: item.Asset.Area, Address: item.Result.Address, Online: item.Result.Online, LatencyMS: item.Result.LatencyMS, Attempts: item.Result.Attempts, CheckedAt: item.CheckedAt.Format(time.DateTime)})
	}
	return items, nil
}
func (d *DesktopAPI) SaveCameraAssets(suffixes []int) error {
	known := map[int]bool{}
	for _, asset := range d.app.Config.Assets {
		known[asset.Suffix] = true
	}
	seen := map[int]bool{}
	clean := make([]int, 0, len(suffixes))
	for _, suffix := range suffixes {
		if !known[suffix] {
			return fmt.Errorf("监控设备必须来自设备列表")
		}
		if !seen[suffix] {
			clean = append(clean, suffix)
			seen[suffix] = true
		}
	}
	c := d.app.Config
	c.CameraAssetSuffixes = clean
	return d.app.Save(c)
}
func componentEnabled(items []config.ComponentConfig, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return item.Enabled
		}
	}
	return false
}

func (d *DesktopAPI) TestPing(suffix string) (string, error) {
	return d.app.TestPing(context.Background(), suffix)
}

type LiveStatus struct {
	Connection string `json:"connection"`
	LatestPing string `json:"latestPing"`
}

func (d *DesktopAPI) TestPingTarget(prefix, suffix string) (string, error) {
	prefix, err := config.NormalizePrefix(prefix)
	if err != nil {
		return "", err
	}
	n, err := config.ValidateSuffix(suffix)
	if err != nil {
		return "", err
	}
	r := d.app.Checker.Ping(context.Background(), fmt.Sprintf("%s.%d", prefix, n), d.app.Config.Network.TimeoutMS, d.app.Config.Network.Attempts)
	d.app.RecordPing(r)
	if r.Online {
		return fmt.Sprintf("%s 在线，响应时间 %d ms", r.Address, r.LatencyMS), nil
	}
	return fmt.Sprintf("%s 暂未响应", r.Address), nil
}
func (d *DesktopAPI) GetLiveStatus() LiveStatus {
	r := d.app.LatestPing()
	latest := "暂无"
	if r.Address != "" {
		latest = fmt.Sprintf("%s · %s", r.Address, map[bool]string{true: "在线", false: "暂未响应"}[r.Online])
	}
	return LiveStatus{Connection: cleanConnectionStatus(d.connectionDetail()), LatestPing: latest}
}
func cleanConnectionStatus(status string) string {
	return strings.TrimSpace(strings.TrimPrefix(status, "飞书连接："))
}
func (d *DesktopAPI) StartConnection() error {
	if d.app.Transport == nil {
		return fmt.Errorf("请先保存飞书 App ID 与 App Secret")
	}
	return d.app.StartConnection(context.Background())
}
func (d *DesktopAPI) StopConnection() error {
	if d.app.Transport == nil {
		return nil
	}
	return d.app.StopConnection(context.Background())
}
func (d *DesktopAPI) SaveAsset(asset config.Asset) error {
	suffix, err := config.ValidateSuffix(fmt.Sprint(asset.Suffix))
	if err != nil {
		return err
	}
	c := d.app.Config
	asset.Suffix, asset.Name, asset.Area = suffix, strings.TrimSpace(asset.Name), strings.TrimSpace(asset.Area)
	for i := range c.Assets {
		if c.Assets[i].Suffix == suffix {
			c.Assets[i] = asset
			return d.app.Save(c)
		}
	}
	c.Assets = append(c.Assets, asset)
	return d.app.Save(c)
}
func (d *DesktopAPI) AddAsset(input AssetInput) error {
	asset, err := d.assetFromInput(input)
	if err != nil {
		return err
	}
	for _, item := range d.app.Config.Assets {
		if item.Suffix == asset.Suffix {
			return fmt.Errorf("该设备 IP 已存在，请使用设备编辑")
		}
	}
	return d.SaveAsset(asset)
}
func (d *DesktopAPI) UpdateAsset(oldSuffix int, input AssetInput) error {
	asset, err := d.assetFromInput(input)
	if err != nil {
		return err
	}
	c := d.app.Config
	index := -1
	for i, item := range c.Assets {
		if item.Suffix == oldSuffix {
			index = i
		}
		if item.Suffix == asset.Suffix && item.Suffix != oldSuffix {
			return fmt.Errorf("该设备 IP 已存在")
		}
	}
	if index < 0 {
		return fmt.Errorf("未找到设备")
	}
	c.Assets[index] = asset
	if oldSuffix != asset.Suffix {
		for i, suffix := range c.CameraAssetSuffixes {
			if suffix == oldSuffix {
				c.CameraAssetSuffixes[i] = asset.Suffix
			}
		}
	}
	return d.app.Save(c)
}
func (d *DesktopAPI) assetFromInput(input AssetInput) (config.Asset, error) {
	parts := strings.Split(strings.TrimSpace(input.Address), ".")
	if len(parts) != 4 {
		return config.Asset{}, fmt.Errorf("设备 IP 格式不正确")
	}
	prefix, err := config.NormalizePrefix(strings.Join(parts[:3], "."))
	if err != nil {
		return config.Asset{}, err
	}
	if prefix != d.app.Config.Network.Prefix {
		return config.Asset{}, fmt.Errorf("设备 IP 必须位于当前检测网段 %s", d.app.Config.Network.Prefix)
	}
	suffix, err := config.ValidateSuffix(parts[3])
	if err != nil {
		return config.Asset{}, err
	}
	return config.Asset{Suffix: suffix, Name: strings.TrimSpace(input.Name), Area: strings.TrimSpace(input.Area)}, nil
}
func (d *DesktopAPI) DeleteAsset(suffix int) error {
	c := d.app.Config
	for i, asset := range c.Assets {
		if asset.Suffix == suffix {
			c.Assets = append(c.Assets[:i], c.Assets[i+1:]...)
			filtered := c.CameraAssetSuffixes[:0]
			for _, cameraSuffix := range c.CameraAssetSuffixes {
				if cameraSuffix != suffix {
					filtered = append(filtered, cameraSuffix)
				}
			}
			c.CameraAssetSuffixes = filtered
			return d.app.Save(c)
		}
	}
	return fmt.Errorf("未找到设备")
}
func (d *DesktopAPI) DeleteAssets(suffixes []int) error {
	if len(suffixes) == 0 {
		return fmt.Errorf("请至少选择一台设备")
	}
	selected := map[int]bool{}
	for _, suffix := range suffixes {
		selected[suffix] = true
	}
	c := d.app.Config
	assets := c.Assets[:0]
	for _, asset := range c.Assets {
		if !selected[asset.Suffix] {
			assets = append(assets, asset)
		}
	}
	if len(assets) == len(c.Assets) {
		return fmt.Errorf("未找到已选择的设备")
	}
	c.Assets = assets
	cameras := c.CameraAssetSuffixes[:0]
	for _, suffix := range c.CameraAssetSuffixes {
		if !selected[suffix] {
			cameras = append(cameras, suffix)
		}
	}
	c.CameraAssetSuffixes = cameras
	return d.app.Save(c)
}
func (d *DesktopAPI) ImportAssets() (int, error) {
	path, ok, err := filepicker.OpenXLSX()
	if err != nil || !ok {
		return 0, err
	}
	items, err := assets.Import(path)
	if err != nil {
		return 0, err
	}
	c := d.app.Config
	c.Assets = items
	return len(items), d.app.Save(c)
}
func (d *DesktopAPI) ExportAssetTemplate() error {
	path, ok, err := filepicker.SaveXLSX()
	if err != nil || !ok {
		return err
	}
	return assets.ExportTemplate(path)
}
func (d *DesktopAPI) ExportBackup() error {
	path, ok, err := filepicker.SaveJSON()
	if err != nil || !ok {
		return err
	}
	return config.ExportBackup(path, d.app.Config)
}
func (d *DesktopAPI) ImportBackup() error {
	path, ok, err := filepicker.OpenJSON()
	if err != nil || !ok {
		return err
	}
	c, err := config.ImportBackup(path, d.app.Config)
	if err != nil {
		return err
	}
	return d.app.Save(c)
}
func cleanIDs(ids []string) []string {
	var out []string
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}
