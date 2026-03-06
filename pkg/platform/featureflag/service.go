package featureflag

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/kvstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
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
		flags, err := repo.List(ctx)
		if err != nil {
			return nil, err
		}
		m := make(map[string]string, len(flags))
		for _, f := range flags {
			val := "0"
			if f.Enabled {
				val = "1"
			}
			m[f.Key] = val
		}
		return m, nil
	}
	return &Service{
		repo:  repo,
		store: kvstore.NewStore(c, "ff:", refreshTTL, loader),
	}
}

// StartRefresh launches a background goroutine that reloads all flags
// from Postgres into Valkey and the in-process cache at the configured interval.
func (s *Service) StartRefresh(ctx context.Context) {
	s.store.StartRefresh(ctx)
}

// IsEnabled checks the in-process cache first, then Valkey, then Postgres.
// Returns false for unknown keys.
func (s *Service) IsEnabled(key string) bool {
	val, ok := s.store.Get(key, func() (string, error) {
		f, err := s.repo.GetByKey(context.Background(), key)
		if err != nil {
			return "", err
		}
		v := "0"
		if f.Enabled {
			v = "1"
		}
		return v, nil
	})
	if !ok {
		return false
	}
	return val == "1"
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
	s.store.Delete(ctx, key)
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
	s.store.Set(ctx, key, val)
}
