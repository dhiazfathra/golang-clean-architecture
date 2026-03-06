package envvar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
	"github.com/valkey-io/valkey-go"
)

const valkeyCacheKeyPrefix = "env:"

type Service struct {
	repo       *Repository
	cache      cache
	local      sync.Map // "platform:key" → value string
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

func newServiceWithCache(repo *Repository, c cache, refreshTTL time.Duration) *Service {
	return &Service{
		repo:       repo,
		cache:      c,
		valkeyTTL:  refreshTTL * 2,
		refreshTTL: refreshTTL,
	}
}

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
	envVars, err := s.repo.ListAll(ctx)
	if err != nil {
		return err
	}
	entries := make(map[string]string, len(envVars))
	keys := make([]string, 0, len(envVars))
	for _, e := range envVars {
		cacheKey := cacheKey(e.Platform, e.Key)
		vkKey := valkeyCacheKeyPrefix + cacheKey
		entries[vkKey] = e.Value
		keys = append(keys, cacheKey)
		s.local.Store(cacheKey, e.Value)
	}
	if len(entries) > 0 {
		_ = s.cache.SetMulti(ctx, entries, s.valkeyTTL)
	}
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

// GetValue checks in-process cache first, then Valkey, then Postgres.
// Returns empty string for unknown keys.
func (s *Service) GetValue(platform, key string) string {
	ck := cacheKey(platform, key)
	if v, ok := s.local.Load(ck); ok {
		return v.(string)
	}
	vkKey := valkeyCacheKeyPrefix + ck
	if val, err := s.cache.Get(context.Background(), vkKey); err == nil {
		s.local.Store(ck, val)
		return val
	}
	e, err := s.repo.GetByPlatformKey(context.Background(), platform, key)
	if err != nil {
		return ""
	}
	s.local.Store(ck, e.Value)
	_ = s.cache.Set(context.Background(), vkKey, e.Value, s.valkeyTTL)
	return e.Value
}

func (s *Service) Get(ctx context.Context, platform, key string) (*EnvVar, error) {
	e, err := s.repo.GetByPlatformKey(ctx, platform, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("env var not found")
		}
		return nil, err
	}
	return e, nil
}

func (s *Service) ListByPlatform(ctx context.Context, platform string, req database.PageRequest) (*database.PageResponse[EnvVar], error) {
	return s.repo.ListByPlatform(ctx, platform, req)
}

func (s *Service) Create(ctx context.Context, platform, key, value, userID string) (*EnvVar, error) {
	id := snowflake.NewID()
	e := &EnvVar{
		ID:       id,
		Platform: platform,
		Key:      key,
		Value:    value,
	}
	e.CreatedBy = userID
	e.UpdatedBy = userID
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	s.setCache(ctx, platform, key, value)
	return e, nil
}

func (s *Service) Update(ctx context.Context, platform, key, value, userID string) (*EnvVar, error) {
	e, err := s.repo.GetByPlatformKey(ctx, platform, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("env var not found")
		}
		return nil, err
	}
	e.Value = value
	e.UpdatedBy = userID
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	s.setCache(ctx, platform, key, value)
	return e, nil
}

func (s *Service) Delete(ctx context.Context, platform, key, userID string) error {
	e, err := s.repo.GetByPlatformKey(ctx, platform, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("env var not found")
		}
		return err
	}
	if err := s.repo.Delete(ctx, e.ID, userID); err != nil {
		return err
	}
	s.deleteCache(ctx, platform, key)
	return nil
}

func (s *Service) setCache(ctx context.Context, platform, key, value string) {
	ck := cacheKey(platform, key)
	vkKey := valkeyCacheKeyPrefix + ck
	_ = s.cache.Set(ctx, vkKey, value, s.valkeyTTL)
	s.local.Store(ck, value)
}

func (s *Service) deleteCache(ctx context.Context, platform, key string) {
	ck := cacheKey(platform, key)
	vkKey := valkeyCacheKeyPrefix + ck
	_ = s.cache.Del(ctx, vkKey)
	s.local.Delete(ck)
}

func cacheKey(platform, key string) string {
	return fmt.Sprintf("%s:%s", platform, key)
}
