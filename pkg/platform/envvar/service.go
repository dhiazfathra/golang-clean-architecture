package envvar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/kvstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
	"github.com/valkey-io/valkey-go"
)

type Service struct {
	repo  *Repository
	store *kvstore.Store
}

func NewService(repo *Repository, vk valkey.Client, refreshTTL time.Duration) *Service {
	cache := kvstore.NewValkeyCache(vk)
	return newServiceWithStore(repo, cache, refreshTTL)
}

func newServiceWithStore(repo *Repository, c kvstore.Cache, refreshTTL time.Duration) *Service {
	loader := func(ctx context.Context) (map[string]string, error) {
		envVars, err := repo.ListAll(ctx)
		if err != nil {
			return nil, err
		}
		m := make(map[string]string, len(envVars))
		for _, e := range envVars {
			m[cacheKey(e.Platform, e.Key)] = e.Value
		}
		return m, nil
	}
	return &Service{
		repo:  repo,
		store: kvstore.NewStore(c, "env:", refreshTTL, loader),
	}
}

func (s *Service) StartRefresh(ctx context.Context) {
	s.store.StartRefresh(ctx)
}

// GetValue checks in-process cache first, then Valkey, then Postgres.
// Returns empty string for unknown keys.
func (s *Service) GetValue(platform, key string) string {
	ck := cacheKey(platform, key)
	val, ok := s.store.Get(ck, func() (string, error) {
		e, err := s.repo.GetByPlatformKey(context.Background(), platform, key)
		if err != nil {
			return "", err
		}
		return e.Value, nil
	})
	if !ok {
		return ""
	}
	return val
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
	s.store.Set(ctx, cacheKey(platform, key), value)
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
	s.store.Set(ctx, cacheKey(platform, key), value)
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
	s.store.Delete(ctx, cacheKey(platform, key))
	return nil
}

func cacheKey(platform, key string) string {
	return fmt.Sprintf("%s:%s", platform, key)
}
