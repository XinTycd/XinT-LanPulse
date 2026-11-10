package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Store struct{ dir, path string }

func NewStore(appData string) *Store {
	dir := filepath.Join(appData, "XinT-LanPulse")
	return &Store{dir: dir, path: filepath.Join(dir, "config.json")}
}
func (s *Store) Dir() string  { return s.dir }
func (s *Store) Path() string { return s.path }

func (s *Store) Load() (Config, error) {
	if err := os.MkdirAll(filepath.Join(s.dir, "logs"), 0700); err != nil {
		return Config{}, err
	}
	if err := os.MkdirAll(filepath.Join(s.dir, "data"), 0700); err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		c := Default()
		return c, s.Save(c)
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err = json.Unmarshal(b, &c); err != nil || c.Validate() != nil {
		backup := fmt.Sprintf("%s.%s.broken", s.path, time.Now().Format("20060102-150405"))
		_ = os.Rename(s.path, backup)
		c = Default()
		return c, s.Save(c)
	}
	if EnsureDeviceIdentity(&c) {
		if err := s.Save(c); err != nil {
			return Config{}, err
		}
	}
	return c, nil
}
func (s *Store) Save(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, "config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(b); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
