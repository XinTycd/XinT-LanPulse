package config

import (
	"fmt"
	"strconv"
	"strings"
)

func NormalizePrefix(value string) (string, error) {
	v := strings.TrimSuffix(strings.TrimSpace(value), ".")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("网段必须由三个 IPv4 数字段组成")
	}
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("网段包含空字段")
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 255 {
			return "", fmt.Errorf("网段包含无效 IPv4 数字段")
		}
	}
	return strings.Join(parts, "."), nil
}

func ValidateSuffix(value string) (int, error) {
	v := strings.TrimSpace(value)
	if v == "" || strings.ContainsAny(v, ".+-") {
		return 0, fmt.Errorf("输入无效，请发送 1～254 之间的数字。")
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 254 {
		return 0, fmt.Errorf("输入无效，请发送 1～254 之间的数字。")
	}
	return n, nil
}

func (c Config) Validate() error {
	if _, err := NormalizePrefix(c.Network.Prefix); err != nil {
		return err
	}
	if c.Network.TimeoutMS < 100 || c.Network.TimeoutMS > 60000 {
		return fmt.Errorf("Ping 超时必须在 100～60000 ms")
	}
	if c.Network.Attempts < 1 || c.Network.Attempts > 10 {
		return fmt.Errorf("检测次数必须在 1～10")
	}
	allowed := make(map[string]bool, len(c.Feishu.AllowedUserIDs))
	for _, id := range c.Feishu.AllowedUserIDs {
		if id != "" {
			allowed[id] = true
		}
	}
	for _, id := range c.Feishu.AdminUserIDs {
		if id != "" && !allowed[id] {
			return fmt.Errorf("管理员 %q 不在允许用户列表中", id)
		}
	}
	seenAssets := map[int]bool{}
	for _, asset := range c.Assets {
		if asset.Suffix < 1 || asset.Suffix > 254 {
			return fmt.Errorf("设备列表 IP 后缀必须在 1～254 之间")
		}
		if seenAssets[asset.Suffix] {
			return fmt.Errorf("设备列表存在重复 IP 后缀：%d", asset.Suffix)
		}
		seenAssets[asset.Suffix] = true
	}
	return nil
}
