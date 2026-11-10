package security

import "github.com/xint-tech/xint-lanpulse/internal/config"

func Allowed(c config.Config, userID string) bool {
	for _, id := range c.Feishu.AllowedUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}
func Admin(c config.Config, userID string) bool {
	for _, id := range c.Feishu.AdminUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}
