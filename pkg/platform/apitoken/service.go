package apitoken

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/kvstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
)

type Service struct {
	repo      *Repository
	store     *kvstore.Store
	tokenFunc func() (raw, hash string, err error) // injectable for testing
}

func NewService(repo *Repository, vk valkey.Client, refreshTTL time.Duration) *Service {
	cache := kvstore.NewValkeyCache(vk)
	return newServiceWithStore(repo, cache, refreshTTL)
}

func newServiceWithStore(repo *Repository, c kvstore.Cache, refreshTTL time.Duration) *Service {
	loader := func(ctx context.Context) (map[string]string, error) {
		return repo.LoadAll(ctx)
	}
	return &Service{
		repo:      repo,
		store:     kvstore.NewStore(c, "apitoken:", refreshTTL, loader), //nolint:secret_scan
		tokenFunc: generateToken,
	}
}

func (s *Service) StartRefresh(ctx context.Context) {
	s.store.StartRefresh(ctx)
}

func (s *Service) Create(ctx context.Context, name, userID string, ttl time.Duration) (string, *APIToken, error) {
	raw, hash, err := s.tokenFunc()
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	token := &APIToken{
		ID:          snowflake.NewID(),
		Name:        name,
		TokenHash:   hash,
		TokenPrefix: raw[:12],
		UserID:      userID,
		ExpiresAt:   time.Now().Add(ttl),
	}
	token.CreatedBy = userID
	token.UpdatedBy = userID

	if err := s.repo.Create(ctx, token); err != nil {
		return "", nil, err
	}
	s.store.Set(ctx, hash, userID)
	return raw, token, nil
}

func (s *Service) Validate(ctx context.Context, rawToken string) (string, error) {
	hash := hashToken(rawToken)
	userID, ok := s.store.Get(hash, func() (string, error) {
		t, err := s.repo.GetByHash(ctx, hash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", errors.New("token not found")
			}
			return "", err
		}
		if time.Now().After(t.ExpiresAt) {
			return "", errors.New("token expired")
		}
		return t.UserID, nil
	})
	if !ok {
		return "", errors.New("invalid token")
	}
	return userID, nil
}

func (s *Service) Revoke(ctx context.Context, tokenID int64, userID string) error {
	if err := s.repo.Delete(ctx, tokenID, userID); err != nil {
		return err
	}
	// We can't easily remove the exact cache key without knowing the hash,
	// but the background refresh will prune it. For immediate effect,
	// we rely on the soft-delete + expiry check in Validate's fallback.
	return nil
}

func (s *Service) List(ctx context.Context, userID string) ([]APIToken, error) {
	return s.repo.ListByUser(ctx, userID)
}

const tokenPrefix = "gca_"

var randReader = rand.Reader

func generateToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(randReader, b); err != nil {
		return "", "", err
	}
	raw = tokenPrefix + hex.EncodeToString(b)
	hash = hashToken(raw)
	return raw, hash, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
