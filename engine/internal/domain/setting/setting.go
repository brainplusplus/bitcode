package setting

import (
	"context"
	"fmt"
	"sync"
)

type Setting struct {
	Key    string `json:"key" gorm:"primaryKey;size:100"`
	Value  string `json:"value" gorm:"type:text"`
	Module string `json:"module" gorm:"size:100;index"`
}

type Store struct {
	settings map[string]string
	mu       sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		settings: make(map[string]string),
	}
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.settings[key]
	if !ok {
		return "", fmt.Errorf("setting %q not found", key)
	}
	return val, nil
}

func (s *Store) GetWithDefault(key string, defaultVal string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.settings[key]; ok {
		return val
	}
	return defaultVal
}

func (s *Store) Set(ctx context.Context, key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[key] = value
}

func (s *Store) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string, len(s.settings))
	for k, v := range s.settings {
		result[k] = v
	}
	return result
}

func (s *Store) LoadDefaults(module string, defaults map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, val := range defaults {
		fullKey := module + "." + key
		if _, exists := s.settings[fullKey]; !exists {
			s.settings[fullKey] = fmt.Sprintf("%v", val)
		}
	}
}
