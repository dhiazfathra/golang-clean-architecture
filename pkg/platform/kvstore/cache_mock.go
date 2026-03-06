package kvstore

import (
	"context"
	"errors"
	"sync"
	"time"
)

type MockCache struct {
	mu   sync.Mutex
	Data map[string]string
}

func NewMockCache() *MockCache {
	return &MockCache{Data: make(map[string]string)}
}

func (m *MockCache) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.Data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return v, nil
}

func (m *MockCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Data[key] = value
	return nil
}

func (m *MockCache) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Data, key)
	return nil
}

func (m *MockCache) SetMulti(_ context.Context, entries map[string]string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range entries {
		m.Data[k] = v
	}
	return nil
}
