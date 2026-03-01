package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

// --- hand-rolled mocks ---

type mockSessionStore struct {
	CreateFn  func(ctx context.Context, userID string, ttl time.Duration, meta map[string]string) (*session.Session, error)
	GetFn     func(ctx context.Context, sessionID string) (*session.Session, error)
	DestroyFn func(ctx context.Context, sessionID string) error
	RefreshFn func(ctx context.Context, sessionID string, ttl time.Duration) error
}

func (m *mockSessionStore) Create(ctx context.Context, userID string, ttl time.Duration, meta map[string]string) (*session.Session, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, ttl, meta)
	}
	return &session.Session{ID: "sess_1", UserID: userID}, nil
}
func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, sessionID)
	}
	return nil, nil
}
func (m *mockSessionStore) Destroy(ctx context.Context, sessionID string) error {
	if m.DestroyFn != nil {
		return m.DestroyFn(ctx, sessionID)
	}
	return nil
}
func (m *mockSessionStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	if m.RefreshFn != nil {
		return m.RefreshFn(ctx, sessionID, ttl)
	}
	return nil
}

type mockUserProvider struct {
	GetByEmailFn func(ctx context.Context, email string) (*UserRecord, error)
}

func (m *mockUserProvider) GetByEmail(ctx context.Context, email string) (*UserRecord, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, nil
}

type mockPasswordHasher struct {
	VerifyFn func(password, hash string) bool
}

func (m *mockPasswordHasher) Hash(password string) (string, error) { return "hashed_" + password, nil }
func (m *mockPasswordHasher) Verify(password, hash string) bool {
	if m.VerifyFn != nil {
		return m.VerifyFn(password, hash)
	}
	return true
}

func newTestSvc(sessions session.SessionStore, users UserProvider, hasher PasswordHasher) *Service {
	if sessions == nil {
		sessions = &mockSessionStore{}
	}
	if users == nil {
		users = &mockUserProvider{}
	}
	if hasher == nil {
		hasher = &mockPasswordHasher{}
	}
	return NewService(sessions, users, hasher)
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	users := &mockUserProvider{
		GetByEmailFn: func(_ context.Context, email string) (*UserRecord, error) {
			return &UserRecord{ID: "usr_1", Email: email, PassHash: "hash", Active: true}, nil
		},
	}
	var createdUserID string
	sessions := &mockSessionStore{
		CreateFn: func(_ context.Context, userID string, _ time.Duration, _ map[string]string) (*session.Session, error) {
			createdUserID = userID
			return &session.Session{ID: "sess_1", UserID: userID}, nil
		},
	}
	svc := newTestSvc(sessions, users, nil)

	sess, err := svc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "pass"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "sess_1", sess.ID)
	assert.Equal(t, "usr_1", createdUserID)
}

func TestLogin_UserNotFound(t *testing.T) {
	users := &mockUserProvider{
		GetByEmailFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return nil, nil
		},
	}
	svc := newTestSvc(nil, users, nil)

	_, err := svc.Login(context.Background(), LoginRequest{Email: "x@x.com", Password: "pw"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestLogin_WrongPassword(t *testing.T) {
	users := &mockUserProvider{
		GetByEmailFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return &UserRecord{ID: "u", Email: "a@b.com", PassHash: "hash", Active: true}, nil
		},
	}
	hasher := &mockPasswordHasher{
		VerifyFn: func(_, _ string) bool { return false },
	}
	svc := newTestSvc(nil, users, hasher)

	_, err := svc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "wrong"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestLogin_InactiveUser(t *testing.T) {
	users := &mockUserProvider{
		GetByEmailFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return &UserRecord{ID: "u", Email: "a@b.com", PassHash: "hash", Active: false}, nil
		},
	}
	svc := newTestSvc(nil, users, nil)

	_, err := svc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "pass"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account disabled")
}

// --- Logout ---

func TestLogout_Delegates(t *testing.T) {
	var destroyed string
	sessions := &mockSessionStore{
		DestroyFn: func(_ context.Context, id string) error {
			destroyed = id
			return nil
		},
	}
	svc := newTestSvc(sessions, nil, nil)

	require.NoError(t, svc.Logout(context.Background(), "sess_42"))
	assert.Equal(t, "sess_42", destroyed)
}

func TestLogout_PropagatesError(t *testing.T) {
	sessions := &mockSessionStore{
		DestroyFn: func(_ context.Context, _ string) error {
			return errors.New("destroy failed")
		},
	}
	svc := newTestSvc(sessions, nil, nil)

	assert.Error(t, svc.Logout(context.Background(), "sess_x"))
}

// --- CurrentUser ---

func TestCurrentUser_DelegatesToUserProvider(t *testing.T) {
	want := &UserRecord{ID: "u1", Email: "a@b.com"}
	users := &mockUserProvider{
		GetByEmailFn: func(_ context.Context, id string) (*UserRecord, error) {
			return want, nil
		},
	}
	svc := newTestSvc(nil, users, nil)

	got, err := svc.CurrentUser(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
