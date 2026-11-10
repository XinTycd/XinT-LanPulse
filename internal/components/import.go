// Package components validates declarative component manifests.
// Components are metadata only in this edition: a manifest never contains or executes code.
package components

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xint-tech/xint-lanpulse/internal/config"
)

const maxManifestSize = 256 * 1024

var componentID = regexp.MustCompile(`^[a-z][a-z0-9-]{2,63}$`)

type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func FromFile(path string) (config.ComponentConfig, error) {
	info, err := os.Stat(path)
	if err != nil {
		return config.ComponentConfig{}, err
	}
	if info.Size() > maxManifestSize {
		return config.ComponentConfig{}, fmt.Errorf("组件包不能超过 %d KB", maxManifestSize/1024)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config.ComponentConfig{}, err
	}
	return parse(data, "file:"+filepath.Base(path))
}

func FromURL(rawURL string) (config.ComponentConfig, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return config.ComponentConfig{}, fmt.Errorf("组件 URL 必须是公开可访问的 HTTPS 地址")
	}
	client := &http.Client{Timeout: 12 * time.Second, Transport: &http.Transport{DialContext: dialPublic}, CheckRedirect: func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 || next.URL.Scheme != "https" || next.URL.Hostname() == "" || next.URL.User != nil {
			return fmt.Errorf("重定向地址不安全")
		}
		return nil
	}}
	response, err := client.Get(u.String())
	if err != nil {
		return config.ComponentConfig{}, fmt.Errorf("下载组件清单失败：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return config.ComponentConfig{}, fmt.Errorf("下载组件清单失败：HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxManifestSize+1))
	if err != nil {
		return config.ComponentConfig{}, err
	}
	if len(data) > maxManifestSize {
		return config.ComponentConfig{}, fmt.Errorf("组件清单不能超过 %d KB", maxManifestSize/1024)
	}
	return parse(data, "url:"+u.String())
}

func parse(data []byte, source string) (config.ComponentConfig, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return config.ComponentConfig{}, fmt.Errorf("组件清单不是有效 JSON：%w", err)
	}
	m.ID, m.Name, m.Description = strings.TrimSpace(m.ID), strings.TrimSpace(m.Name), strings.TrimSpace(m.Description)
	if !componentID.MatchString(m.ID) || m.ID == "device-status" {
		return config.ComponentConfig{}, fmt.Errorf("组件 ID 需为 3-64 位小写字母、数字或连字符，且不能使用保留 ID")
	}
	if m.Name == "" || m.Description == "" {
		return config.ComponentConfig{}, fmt.Errorf("组件清单必须包含名称和描述")
	}
	return config.ComponentConfig{ID: m.ID, Name: m.Name, Description: m.Description, Enabled: true, Source: source}, nil
}

var blockedNetworks = func() []netip.Prefix {
	values := []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32"}
	result := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		result = append(result, netip.MustParsePrefix(value))
	}
	return result
}()

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range blockedNetworks {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

// Resolve and validate the exact addresses used to connect, including redirects.
// Disable proxies: a proxy could otherwise resolve the destination independently.
func dialPublic(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return nil, fmt.Errorf("无法解析组件地址")
	}
	for _, ip := range addresses {
		if !publicAddress(ip) {
			return nil, fmt.Errorf("组件地址解析到非公开网络")
		}
	}
	var last error
	for _, ip := range addresses {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		last = err
	}
	return nil, last
}
