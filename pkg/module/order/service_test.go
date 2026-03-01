package order

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

// --- hand-rolled mocks ---

type mockEventStore struct {
	AppendFn func(ctx context.Context, events []eventstore.Event) error
	LoadFn   func(ctx context.Context, aggType, aggID string, fromVersion int) ([]eventstore.Event, error)
}

func (m *mockEventStore) Append(ctx context.Context, events []eventstore.Event) error {
	if m.AppendFn != nil {
		return m.AppendFn(ctx, events)
	}
	return nil
}

func (m *mockEventStore) Load(ctx context.Context, aggType, aggID string, fromVersion int) ([]eventstore.Event, error) {
	if m.LoadFn != nil {
		return m.LoadFn(ctx, aggType, aggID, fromVersion)
	}
	return nil, nil
}

type mockReadRepo struct {
	GetByIDFn func(ctx context.Context, id string) (*OrderReadModel, error)
	ListFn    func(ctx context.Context, req ListRequest) (*ListResponse, error)
}

func (m *mockReadRepo) GetByID(ctx context.Context, id string) (*OrderReadModel, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReadRepo) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, req)
	}
	return &ListResponse{}, nil
}

type mockUserProvider struct {
	GetByIDFn func(ctx context.Context, id string) (bool, error)
}

func (m *mockUserProvider) GetByID(ctx context.Context, id string) (bool, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return true, nil
}

func newTestSvc(store eventstore.EventStore, repo ReadRepository, userProv UserProvider) *Service {
	if store == nil {
		store = &mockEventStore{}
	}
	if repo == nil {
		repo = &mockReadRepo{}
	}
	if userProv == nil {
		userProv = &mockUserProvider{}
	}
	return NewService(store, repo, userProv)
}

// --- CreateOrder ---

func TestCreateOrder_Success(t *testing.T) {
	var appended []eventstore.Event
	store := &mockEventStore{
		AppendFn: func(_ context.Context, evs []eventstore.Event) error {
			appended = append(appended, evs...)
			return nil
		},
	}
	userProv := &mockUserProvider{
		GetByIDFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := newTestSvc(store, nil, userProv)

	id, err := svc.CreateOrder(context.Background(), CreateOrderCmd{
		UserID: "100", Total: 99.99, Actor: "actor_1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	require.Len(t, appended, 1)
	ev, ok := appended[0].(*OrderCreated)
	require.True(t, ok)
	assert.Equal(t, "100", ev.UserID)
	assert.Equal(t, 99.99, ev.Total)
}

func TestCreateOrder_UserNotFound(t *testing.T) {
	userProv := &mockUserProvider{
		GetByIDFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
	}
	svc := newTestSvc(nil, nil, userProv)

	_, err := svc.CreateOrder(context.Background(), CreateOrderCmd{UserID: "999", Total: 1.0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestCreateOrder_UserProviderError(t *testing.T) {
	userProv := &mockUserProvider{
		GetByIDFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("provider error")
		},
	}
	svc := newTestSvc(nil, nil, userProv)

	_, err := svc.CreateOrder(context.Background(), CreateOrderCmd{UserID: "100", Total: 1.0})
	require.Error(t, err)
}

// --- GetByID ---

func TestGetByID_DelegatesToRepo(t *testing.T) {
	want := &OrderReadModel{ID: 5, Status: "pending"}
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, id string) (*OrderReadModel, error) {
			assert.Equal(t, "5", id)
			return want, nil
		},
	}
	svc := newTestSvc(nil, repo, nil)
	got, err := svc.GetByID(context.Background(), "5")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// --- List ---

func TestList_DelegatesToRepo(t *testing.T) {
	want := &ListResponse{Total: 7}
	repo := &mockReadRepo{
		ListFn: func(_ context.Context, _ ListRequest) (*ListResponse, error) { return want, nil },
	}
	svc := newTestSvc(nil, repo, nil)
	got, err := svc.List(context.Background(), ListRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// --- DeleteOrder ---

func TestDeleteOrder_AppendsDeleteEvent(t *testing.T) {
	var appended []eventstore.Event
	store := &mockEventStore{
		AppendFn: func(_ context.Context, evs []eventstore.Event) error {
			appended = append(appended, evs...)
			return nil
		},
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return []eventstore.Event{
				&OrderCreated{
					BaseEvent: eventstore.NewBaseEvent("ord_1", "order", "order.created", 1, nil),
					UserID:    "100",
					Total:     9.99,
				},
			}, nil
		},
	}
	svc := newTestSvc(store, nil, nil)

	err := svc.DeleteOrder(context.Background(), "ord_1", "actor_1")
	require.NoError(t, err)
	require.Len(t, appended, 1)
	_, ok := appended[0].(*OrderDeleted)
	assert.True(t, ok)
}
