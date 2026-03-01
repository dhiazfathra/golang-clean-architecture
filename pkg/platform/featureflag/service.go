package featureflag

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
)

const valkeyCacheKeyPrefix = "ff:"

type Service struct {
	repo       *Repository
	cache      cache
	local      sync.Map // key → bool (in-process hot cache)
	valkeyTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(repo *Repository, vk valkey.Client, refreshTTL time.Duration) *Service {
	return &Service{
		repo:       repo,
		cache:      newValkeyCache(vk),
		valkeyTTL:  refreshTTL * 2,
		refreshTTL: refreshTTL,
	}
}

// newServiceWithCache is used internally for testing with a mock cache.
func newServiceWithCache(repo *Repository, c cache, refreshTTL time.Duration) *Service {
	return &Service{
		repo:       repo,
		cache:      c,
		valkeyTTL:  refreshTTL * 2,
		refreshTTL: refreshTTL,
	}
}

// StartRefresh launches a background goroutine that reloads all flags
// from Postgres into Valkey and the in-process cache at the configured interval.
func (s *Service) StartRefresh(ctx context.Context) {
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

func (s *Service) reload(ctx context.Context) error {
	flags, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	entries := make(map[string]string, len(flags))
	keys := make([]string, 0, len(flags))
	for _, f := range flags {
		vkKey := valkeyCacheKeyPrefix + f.Key
		val := "0"
		if f.Enabled {
			val = "1"
		}
		entries[vkKey] = val
		keys = append(keys, f.Key)
		s.local.Store(f.Key, f.Enabled)
	}
	if len(entries) > 0 {
		_ = s.cache.SetMulti(ctx, entries, s.valkeyTTL)
	}
	// Prune in-process keys that no longer exist in Postgres
	knownKeys := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		knownKeys[k] = struct{}{}
	}
	s.local.Range(func(key, _ any) bool {
		if _, ok := knownKeys[key.(string)]; !ok {
			s.local.Delete(key)
		}
		return true
	})
	return nil
}

// IsEnabled checks the in-process cache first, then Valkey, then Postgres.
// Returns false for unknown keys.
func (s *Service) IsEnabled(key string) bool {
	// L1: in-process
	if v, ok := s.local.Load(key); ok {
		return v.(bool)
	}
	// L2: Valkey
	vkKey := valkeyCacheKeyPrefix + key
	if val, err := s.cache.Get(context.Background(), vkKey); err == nil {
		enabled := val == "1"
		s.local.Store(key, enabled)
		return enabled
	}
	// L3: Postgres fallback
	f, err := s.repo.GetByKey(context.Background(), key)
	if err != nil {
		return false
	}
	s.local.Store(key, f.Enabled)
	val := "0"
	if f.Enabled {
		val = "1"
	}
	_ = s.cache.Set(context.Background(), vkKey, val, s.valkeyTTL)
	return f.Enabled
}

// Create creates a new feature flag.
func (s *Service) Create(ctx context.Context, key, description string, enabled bool, userID string) (*Flag, error) {
	id := snowflake.NewID()
	f := &Flag{
		ID:          id,
		Key:         key,
		Enabled:     enabled,
		Description: description,
		Metadata:    []byte("{}"),
	}
	f.CreatedBy = userID
	f.UpdatedBy = userID
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	s.setCache(ctx, key, enabled)
	return f, nil
}

// Toggle sets a flag to enabled/disabled by key.
func (s *Service) Toggle(ctx context.Context, key string, enabled bool, userID string) error {
	f, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("feature flag not found")
		}
		return err
	}
	f.Enabled = enabled
	f.UpdatedBy = userID
	if err := s.repo.Update(ctx, f); err != nil {
		return err
	}
	s.setCache(ctx, key, enabled)
	return nil
}

// Delete soft-deletes a feature flag.
func (s *Service) Delete(ctx context.Context, key string, userID string) error {
	f, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("feature flag not found")
		}
		return err
	}
	if err := s.repo.Delete(ctx, f.ID, userID); err != nil {
		return err
	}
	s.deleteCache(ctx, key)
	return nil
}

// List returns all active feature flags from the database.
func (s *Service) List(ctx context.Context) ([]Flag, error) {
	return s.repo.List(ctx)
}

func (s *Service) setCache(ctx context.Context, key string, enabled bool) {
	val := "0"
	if enabled {
		val = "1"
	}
	vkKey := valkeyCacheKeyPrefix + key
	_ = s.cache.Set(ctx, vkKey, val, s.valkeyTTL)
	s.local.Store(key, enabled)
}

func (s *Service) deleteCache(ctx context.Context, key string) {
	vkKey := valkeyCacheKeyPrefix + key
	_ = s.cache.Del(ctx, vkKey)
	s.local.Delete(key)
}
