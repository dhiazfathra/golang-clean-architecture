package user

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
	GetByIDFn    func(ctx context.Context, id string) (*UserReadModel, error)
	GetByEmailFn func(ctx context.Context, email string) (*UserReadModel, error)
	ListFn       func(ctx context.Context, req ListRequest) (*ListResponse, error)
}

func (m *mockReadRepo) GetByID(ctx context.Context, id string) (*UserReadModel, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReadRepo) GetByEmail(ctx context.Context, email string) (*UserReadModel, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockReadRepo) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, req)
	}
	return &ListResponse{}, nil
}

type mockHasher struct {
	HashFn func(s string) (string, error)
}

func (m *mockHasher) Hash(s string) (string, error) {
	if m.HashFn != nil {
		return m.HashFn(s)
	}
	return "hashed_" + s, nil
}

func newTestSvc(store eventstore.EventStore, repo ReadRepository, hasher interface{ Hash(string) (string, error) }) *Service {
	if store == nil {
		store = &mockEventStore{}
	}
	if repo == nil {
		repo = &mockReadRepo{}
	}
	if hasher == nil {
		hasher = &mockHasher{}
	}
	return NewService(store, repo, hasher)
}

// --- CreateUser ---

func TestCreateUser_Success(t *testing.T) {
	var appended []eventstore.Event
	store := &mockEventStore{
		AppendFn: func(_ context.Context, evs []eventstore.Event) error {
			appended = append(appended, evs...)
			return nil
		},
	}
	repo := &mockReadRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return nil, nil
		},
	}
	svc := newTestSvc(store, repo, nil)

	id, err := svc.CreateUser(context.Background(), CreateUserCmd{
		Email: "alice@example.com", Password: "pass", Actor: "actor_1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	require.Len(t, appended, 1)
	ev, ok := appended[0].(*UserCreated)
	require.True(t, ok)
	assert.Equal(t, "alice@example.com", ev.Email)
	assert.Equal(t, "hashed_pass", ev.PassHash)
	assert.Equal(t, "actor_1", ev.Metadata()["user_id"])
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := &mockReadRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return &UserReadModel{Email: "alice@example.com"}, nil
		},
	}
	svc := newTestSvc(nil, repo, nil)

	_, err := svc.CreateUser(context.Background(), CreateUserCmd{Email: "alice@example.com", Password: "pass"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email already registered")
}

func TestCreateUser_HashError(t *testing.T) {
	repo := &mockReadRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return nil, nil
		},
	}
	hasher := &mockHasher{
		HashFn: func(_ string) (string, error) { return "", errors.New("hash failed") },
	}
	svc := newTestSvc(nil, repo, hasher)

	_, err := svc.CreateUser(context.Background(), CreateUserCmd{Email: "bob@example.com", Password: "pass"})
	require.Error(t, err)
}

// --- ChangeEmail ---

func TestChangeEmail_Success(t *testing.T) {
	var appended []eventstore.Event
	store := &mockEventStore{
		AppendFn: func(_ context.Context, evs []eventstore.Event) error {
			appended = append(appended, evs...)
			return nil
		},
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return []eventstore.Event{
				&UserCreated{
					BaseEvent: eventstore.NewBaseEvent("usr_1", "user", "user.created", 1, nil),
					Email:     "old@example.com",
				},
			}, nil
		},
	}
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return &UserReadModel{ID: 1, Email: "old@example.com"}, nil
		},
	}
	svc := newTestSvc(store, repo, nil)

	err := svc.ChangeEmail(context.Background(), "usr_1", "new@example.com", "actor_1")
	require.NoError(t, err)
	require.Len(t, appended, 1)
	ev, ok := appended[0].(*EmailChanged)
	require.True(t, ok)
	assert.Equal(t, "new@example.com", ev.NewEmail)
	assert.Equal(t, "old@example.com", ev.OldEmail)
}

func TestChangeEmail_NotFound(t *testing.T) {
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, _ string) (*UserReadModel, error) {
			return nil, nil
		},
	}
	svc := newTestSvc(nil, repo, nil)

	err := svc.ChangeEmail(context.Background(), "nonexistent", "new@example.com", "actor")
	assert.Error(t, err)
}

// --- DeleteUser ---

func TestDeleteUser_Success(t *testing.T) {
	var appended []eventstore.Event
	store := &mockEventStore{
		AppendFn: func(_ context.Context, evs []eventstore.Event) error {
			appended = append(appended, evs...)
			return nil
		},
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return []eventstore.Event{
				&UserCreated{BaseEvent: eventstore.NewBaseEvent("usr_1", "user", "user.created", 1, nil)},
			}, nil
		},
	}
	svc := newTestSvc(store, nil, nil)

	err := svc.DeleteUser(context.Background(), "usr_1", "actor_1")
	require.NoError(t, err)
	require.Len(t, appended, 1)
	_, ok := appended[0].(*UserDeleted)
	assert.True(t, ok)
}

// --- AssignRole ---

func TestAssignRole_Success(t *testing.T) {
	var appended []eventstore.Event
	store := &mockEventStore{
		AppendFn: func(_ context.Context, evs []eventstore.Event) error {
			appended = append(appended, evs...)
			return nil
		},
		LoadFn: func(_ context.Context, _, _ string, _ int) ([]eventstore.Event, error) {
			return nil, nil
		},
	}
	svc := newTestSvc(store, nil, nil)

	err := svc.AssignRole(context.Background(), "usr_1", "role_42", "actor_1")
	require.NoError(t, err)
	require.Len(t, appended, 1)
	ev, ok := appended[0].(*RoleAssigned)
	require.True(t, ok)
	assert.Equal(t, "role_42", ev.RoleID)
}

// --- GetByID / GetByEmail / List ---

func TestGetByID_DelegatesToRepo(t *testing.T) {
	want := &UserReadModel{ID: 7, Email: "x@x.com"}
	repo := &mockReadRepo{
		GetByIDFn: func(_ context.Context, id string) (*UserReadModel, error) {
			assert.Equal(t, "7", id)
			return want, nil
		},
	}
	svc := newTestSvc(nil, repo, nil)
	got, err := svc.GetByID(context.Background(), "7")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetByEmail_DelegatesToRepo(t *testing.T) {
	want := &UserReadModel{Email: "a@b.com"}
	repo := &mockReadRepo{
		GetByEmailFn: func(_ context.Context, email string) (*UserReadModel, error) {
			return want, nil
		},
	}
	svc := newTestSvc(nil, repo, nil)
	got, err := svc.GetByEmail(context.Background(), "a@b.com")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestList_DelegatesToRepo(t *testing.T) {
	want := &ListResponse{Total: 3}
	repo := &mockReadRepo{
		ListFn: func(_ context.Context, req ListRequest) (*ListResponse, error) {
			return want, nil
		},
	}
	svc := newTestSvc(nil, repo, nil)
	got, err := svc.List(context.Background(), ListRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
