package envvar

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/kvstore"
)

type mockCache struct {
	mu   sync.Mutex
	data map[string]string
}

var _ kvstore.Cache = (*mockCache)(nil)

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return v, nil
}

func (m *mockCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockCache) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockCache) SetMulti(_ context.Context, entries map[string]string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range entries {
		m.data[k] = v
	}
	return nil
}
