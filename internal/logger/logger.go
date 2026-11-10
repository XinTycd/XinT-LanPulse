package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu        sync.Mutex
	file      *os.File
	entries   []string
	listeners []func(string)
}

func New(dir string, enabled bool) (*Logger, error) {
	l := &Logger{}
	if !enabled {
		return l, nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "app-"+time.Now().Format("2006-01-02")+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	l.file = f
	return l, nil
}
func (l *Logger) Info(message string) {
	line := "[" + time.Now().Format("15:04:05") + "] " + message
	l.mu.Lock()
	l.entries = append(l.entries, line)
	if len(l.entries) > 300 {
		l.entries = l.entries[len(l.entries)-300:]
	}
	if l.file != nil {
		_, _ = fmt.Fprintln(l.file, line)
	}
	listeners := append([]func(string){}, l.listeners...)
	l.mu.Unlock()
	for _, fn := range listeners {
		fn(line)
	}
}
func (l *Logger) Entries() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.entries...)
}
func (l *Logger) Subscribe(fn func(string)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.listeners = append(l.listeners, fn)
}
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
