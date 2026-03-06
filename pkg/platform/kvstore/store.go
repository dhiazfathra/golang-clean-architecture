package kvstore

import (
	"context"
	"sync"
	"time"
)

// Loader fetches all entries from the database (L3).
// Returns a map of cache-key -> cache-value.
type Loader func(ctx context.Context) (map[string]string, error)

// Store is a generic 3-tier (in-process -> Valkey -> Postgres) cached key-value store.
type Store struct {
	cache      Cache
	local      sync.Map
	prefix     string // Valkey key prefix, e.g. "ff:" or "env:"
	valkeyTTL  time.Duration
	refreshTTL time.Duration
	loader     Loader
}

// NewStore creates a Store.
func NewStore(c Cache, prefix string, refreshTTL time.Duration, loader Loader) *Store {
	return &Store{
		cache:      c,
		prefix:     prefix,
		valkeyTTL:  refreshTTL * 2,
		refreshTTL: refreshTTL,
		loader:     loader,
	}
}

// StartRefresh launches a background goroutine that reloads all entries
// from the database into Valkey and the in-process cache at the configured interval.
func (s *Store) StartRefresh(ctx context.Context) {
	_ = s.reload(ctx)
	go func() {
		ticker := time.NewTicker(s.refreshTTL)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.reload(ctx)
			}
		}
	}()
}

func (s *Store) reload(ctx context.Context) error {
	entries, err := s.loader(ctx)
	if err != nil {
		return err
	}
	valkeyEntries := make(map[string]string, len(entries))
	keys := make(map[string]struct{}, len(entries))
	for k, v := range entries {
		valkeyEntries[s.prefix+k] = v
		keys[k] = struct{}{}
		s.local.Store(k, v)
	}
	if len(valkeyEntries) > 0 {
		_ = s.cache.SetMulti(ctx, valkeyEntries, s.valkeyTTL)
	}
	// Prune in-process keys that no longer exist
	s.local.Range(func(key, _ any) bool {
		if _, ok := keys[key.(string)]; !ok {
			s.local.Delete(key)
		}
		return true
	})
	return nil
}

// Get retrieves a value via L1 -> L2 -> fallback.
// The fallback function is called only if L1 and L2 miss; it typically queries the database
// and returns the cache value string.
func (s *Store) Get(key string, fallback func() (string, error)) (string, bool) {
	// L1: in-process
	if v, ok := s.local.Load(key); ok {
		return v.(string), true
	}
	// L2: Valkey
	vkKey := s.prefix + key
	if val, err := s.cache.Get(context.Background(), vkKey); err == nil {
		s.local.Store(key, val)
		return val, true
	}
	// L3: fallback (Postgres)
	if fallback != nil {
		val, err := fallback()
		if err != nil {
			return "", false
		}
		s.local.Store(key, val)
		_ = s.cache.Set(context.Background(), vkKey, val, s.valkeyTTL)
		return val, true
	}
	return "", false
}

// Set writes a value to both L1 and L2 caches.
func (s *Store) Set(ctx context.Context, key, value string) {
	vkKey := s.prefix + key
	_ = s.cache.Set(ctx, vkKey, value, s.valkeyTTL)
	s.local.Store(key, value)
}

// Delete removes a value from both L1 and L2 caches.
func (s *Store) Delete(ctx context.Context, key string) {
	vkKey := s.prefix + key
	_ = s.cache.Del(ctx, vkKey)
	s.local.Delete(key)
}

// Local returns the value from the in-process cache only.
// Used by consumers that need to check L1 directly.
func (s *Store) Local(key string) (string, bool) {
	v, ok := s.local.Load(key)
	if !ok {
		return "", false
	}
	return v.(string), true
}
