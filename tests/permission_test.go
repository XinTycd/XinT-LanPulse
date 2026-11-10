package tests

import (
	"github.com/xint-tech/xint-lanpulse/internal/config"
	"github.com/xint-tech/xint-lanpulse/internal/security"
	"testing"
)

func TestPermissions(t *testing.T) {
	c := config.Default()
	c.Feishu.AllowedUserIDs = []string{"a", "b"}
	c.Feishu.AdminUserIDs = []string{"b"}
	if !security.Allowed(c, "a") || security.Admin(c, "a") || !security.Admin(c, "b") {
		t.Fatal("permission evaluation failed")
	}
	c.Feishu.AdminUserIDs = []string{"z"}
	if err := c.Validate(); err == nil {
		t.Fatal("admin must be subset")
	}
}
