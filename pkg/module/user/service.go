package user

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/httputil"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
)

type CreateUserCmd struct {
	Email    string
	Password string
	Actor    string
}

type Service struct {
	store  eventstore.EventStore
	repo   ReadRepository
	hasher interface{ Hash(string) (string, error) }
}

func NewService(store eventstore.EventStore, repo ReadRepository, hasher interface{ Hash(string) (string, error) }) *Service {
	return &Service{store: store, repo: repo, hasher: hasher}
}

func (s *Service) CreateUser(ctx context.Context, cmd CreateUserCmd) (string, error) {
	existing, _ := s.repo.GetByEmail(ctx, cmd.Email)
	if existing != nil {
		return "", fmt.Errorf("email already registered")
	}
	hash, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	id := snowflake.NewStringID()
	agg := NewUserAggregate(id)
	meta := map[string]string{"user_id": cmd.Actor}
	agg.Apply(&UserCreated{
		BaseEvent: eventstore.NewBaseEvent(id, "user", "user.created", 1, meta),
		Email:     cmd.Email,
		PassHash:  hash,
	})
	if err := s.store.Append(ctx, agg.Uncommitted()); err != nil {
		return "", fmt.Errorf("append events: %w", err)
	}
	return id, nil
}

func (s *Service) ChangeEmail(ctx context.Context, id, newEmail, actor string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil || user == nil {
		return httputil.ErrNotFound
	}
	events, _ := s.store.Load(ctx, "user", id, 0)
	agg := NewUserAggregate(id)
	for _, e := range events {
		agg.Rehydrate(e)
	}
	meta := map[string]string{"user_id": actor}
	agg.Apply(&EmailChanged{
		BaseEvent: eventstore.NewBaseEvent(id, "user", "user.email_changed", agg.Version+1, meta),
		OldEmail:  user.Email,
		NewEmail:  newEmail,
	})
	return s.store.Append(ctx, agg.Uncommitted())
}

func (s *Service) DeleteUser(ctx context.Context, id, actor string) error {
	events, _ := s.store.Load(ctx, "user", id, 0)
	agg := NewUserAggregate(id)
	for _, e := range events {
		agg.Rehydrate(e)
	}
	meta := map[string]string{"user_id": actor}
	agg.Apply(&UserDeleted{
		BaseEvent: eventstore.NewBaseEvent(id, "user", "user.deleted", agg.Version+1, meta),
	})
	return s.store.Append(ctx, agg.Uncommitted())
}

func (s *Service) AssignRole(ctx context.Context, userID, roleID, actor string) error {
	events, _ := s.store.Load(ctx, "user", userID, 0)
	agg := NewUserAggregate(userID)
	for _, e := range events {
		agg.Rehydrate(e)
	}
	meta := map[string]string{"user_id": actor}
	agg.Apply(&RoleAssigned{
		BaseEvent: eventstore.NewBaseEvent(userID, "user", "user.role_assigned", agg.Version+1, meta),
		RoleID:    roleID,
	})
	return s.store.Append(ctx, agg.Uncommitted())
}

func (s *Service) GetByID(ctx context.Context, id string) (*UserReadModel, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*UserReadModel, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *Service) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	return s.repo.List(ctx, req)
}

// GetByEmailForAuth satisfies auth.UserProvider.
func (s *Service) GetByEmailForAuth(ctx context.Context, email string) (*auth.UserRecord, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil || u == nil {
		return nil, err
	}
	return &auth.UserRecord{
		ID:       strconv.FormatInt(u.ID, 10),
		Email:    u.Email,
		PassHash: u.PassHash,
		Active:   u.Active,
	}, nil
}

// CreateUserForSeeder satisfies seeder.UserCreator via adapter.
func (s *Service) CreateUserForSeeder(ctx context.Context, cmd seeder.CreateUserCmd) (string, error) {
	return s.CreateUser(ctx, CreateUserCmd{Email: cmd.Email, Password: cmd.Password, Actor: cmd.Actor})
}

// GetByEmailForSeeder satisfies seeder.UserCreator via adapter.
func (s *Service) GetByEmailForSeeder(ctx context.Context, email string) (*seeder.UserRecord, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil || u == nil {
		return nil, err
	}
	return &seeder.UserRecord{ID: strconv.FormatInt(u.ID, 10), Email: u.Email}, nil
}
