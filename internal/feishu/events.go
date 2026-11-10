package feishu

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventStore persists a bounded recent-event window so repeated deliveries do
// not repeat a ping or apply /net more than once.
type EventStore struct {
	mu    sync.Mutex
	path  string
	items map[string]time.Time
	limit int
}

func NewEventStore(dataDir string) (*EventStore, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	s := &EventStore{path: filepath.Join(dataDir, "events.json"), items: map[string]time.Time{}, limit: 1000}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(b, &s.items) // Corrupt dedup state is non-critical; start a clean window.
	s.pruneLocked(time.Now())
	return s, nil
}

// Seen returns true when ID was already handled. New IDs are synchronously
// persisted before the command runs, making /net idempotent across restarts.
func (s *EventStore) Seen(id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; ok {
		return true, nil
	}
	s.items[id] = time.Now()
	s.pruneLocked(time.Now())
	b, err := json.Marshal(s.items)
	if err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "events-*.tmp")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(b); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return false, err
	}
	return false, os.Rename(tmpName, s.path)
}
func (s *EventStore) pruneLocked(now time.Time) {
	for id, when := range s.items {
		if now.Sub(when) > 24*time.Hour {
			delete(s.items, id)
		}
	}
	for len(s.items) > s.limit {
		var oldest string
		var timeOld time.Time
		for id, when := range s.items {
			if oldest == "" || when.Before(timeOld) {
				oldest, timeOld = id, when
			}
		}
		delete(s.items, oldest)
	}
}
