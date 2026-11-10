package security

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	last     map[string]time.Time
	interval time.Duration
}

func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{last: map[string]time.Time{}, interval: interval}
}
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if previous, ok := l.last[key]; ok && now.Sub(previous) < l.interval {
		return false
	}
	l.last[key] = now
	return true
}
