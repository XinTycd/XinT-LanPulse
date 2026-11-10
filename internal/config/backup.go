package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Backup struct {
	Version   int    `json:"version"`
	CreatedAt string `json:"createdAt"`
	Config    Config `json:"config"`
}

func ExportBackup(path string, c Config) error {
	if filepath.Ext(path) != ".json" {
		return fmt.Errorf("备份文件必须使用 .json 扩展名")
	}
	safe := c
	safe.Feishu.AppSecretEncrypted = ""
	payload := Backup{Version: 1, CreatedAt: time.Now().Format(time.RFC3339), Config: safe}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
func ImportBackup(path string, current Config) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var backup Backup
	if err = json.Unmarshal(b, &backup); err != nil {
		return Config{}, fmt.Errorf("备份文件格式无效：%w", err)
	}
	if backup.Version != 1 {
		return Config{}, fmt.Errorf("不支持的备份版本")
	}
	next := backup.Config
	next.App.DeviceID = current.App.DeviceID
	next.App.DeviceName = current.App.DeviceName
	next.Feishu.AppSecretEncrypted = current.Feishu.AppSecretEncrypted
	if next.Feishu.AppID == "" {
		next.Feishu.AppID = current.Feishu.AppID
	}
	EnsureDeviceIdentity(&next)
	if err = next.Validate(); err != nil {
		return Config{}, err
	}
	return next, nil
}
