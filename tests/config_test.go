package tests

import (
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrefixNormalization(t *testing.T) {
	got, err := config.NormalizePrefix("192.168.20.")
	if err != nil || got != "192.168.20" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, v := range []string{"192.168", "192.168.20.15", "192.168.300", "abc.168.20"} {
		if _, err := config.NormalizePrefix(v); err == nil {
			t.Fatalf("expected invalid prefix %s", v)
		}
	}
}
func TestSuffixValidation(t *testing.T) {
	for _, v := range []string{"1", "15", "254"} {
		if _, err := config.ValidateSuffix(v); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []string{"0", "255", "-1", "300", "15.20", "abc", "192.168.20.15"} {
		if _, err := config.ValidateSuffix(v); err == nil {
			t.Fatalf("expected invalid suffix %s", v)
		}
	}
}
func TestStoreRoundTripAndBackup(t *testing.T) {
	dir := t.TempDir()
	s := config.NewStore(dir)
	c, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	c.Network.Prefix = "10.0.0"
	if err = s.Save(c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil || got.Network.Prefix != "10.0.0" {
		t.Fatalf("%+v %v", got, err)
	}
	if err = os.WriteFile(s.Path(), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = s.Load()
	if err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(s.Path() + ".*.broken")
	if len(matches) != 1 {
		t.Fatal("broken config was not backed up")
	}
}

func TestAssetValidation(t *testing.T) {
	c := config.Default()
	c.Assets = []config.Asset{{Suffix: 15, Name: "屏幕 A", Area: "中控室"}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Assets = []config.Asset{{Suffix: 15}, {Suffix: 15}}
	if err := c.Validate(); err == nil {
		t.Fatal("duplicate suffix should fail")
	}
}

func TestBackupDoesNotContainSecretAndPreservesTargetIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.json")
	source := config.Default()
	source.Feishu.AppSecretEncrypted = "secret-cipher"
	source.Network.Prefix = "192.168.30"
	source.App.DeviceName = "源客户端"
	if err := config.ExportBackup(path, source); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) == "" || strings.Contains(string(content), "secret-cipher") {
		t.Fatal("backup contains secret")
	}
	target := config.Default()
	targetID, targetName, targetSecret := target.App.DeviceID, target.App.DeviceName, "target-cipher"
	target.Feishu.AppSecretEncrypted = targetSecret
	imported, err := config.ImportBackup(path, target)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Network.Prefix != "192.168.30" || imported.App.DeviceID != targetID || imported.App.DeviceName != targetName || imported.Feishu.AppSecretEncrypted != targetSecret {
		t.Fatalf("unexpected imported config: %+v", imported)
	}
}
