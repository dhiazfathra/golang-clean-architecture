package order

import (
	"context"
	"fmt"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/snowflake"
)

// UserProvider avoids a direct import of the user package.
type UserProvider interface {
	GetByID(ctx context.Context, id string) (userExists bool, err error)
}

type CreateOrderCmd struct {
	UserID string
	Total  float64
	Actor  string
}

type Service struct {
	store    eventstore.EventStore
	repo     ReadRepository
	userProv UserProvider
}

func NewService(store eventstore.EventStore, repo ReadRepository, userProv UserProvider) *Service {
	return &Service{store: store, repo: repo, userProv: userProv}
}

func (s *Service) CreateOrder(ctx context.Context, cmd CreateOrderCmd) (string, error) {
	exists, err := s.userProv.GetByID(ctx, cmd.UserID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("user not found")
	}
	id := snowflake.NewStringID()
	agg := NewOrderAggregate(id)
	meta := map[string]string{"user_id": cmd.Actor}
	agg.Apply(&OrderCreated{
		BaseEvent: eventstore.NewBaseEvent(id, "order", "order.created", 1, meta),
		UserID:    cmd.UserID,
		Total:     cmd.Total,
	})
	if err := s.store.Append(ctx, agg.Uncommitted()); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*OrderReadModel, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *Service) DeleteOrder(ctx context.Context, id, actor string) error {
	events, _ := s.store.Load(ctx, "order", id, 0)
	agg := NewOrderAggregate(id)
	for _, e := range events {
		agg.Rehydrate(e)
	}
	meta := map[string]string{"user_id": actor}
	agg.Apply(&OrderDeleted{
		BaseEvent: eventstore.NewBaseEvent(id, "order", "order.deleted", agg.Version+1, meta),
	})
	return s.store.Append(ctx, agg.Uncommitted())
}
