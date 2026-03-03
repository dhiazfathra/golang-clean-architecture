package session_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

// --- mock store (function-field pattern) ---

type mockStore struct {
	createFn  func(ctx context.Context, userID string, ttl time.Duration, meta map[string]string) (*session.Session, error)
	getFn     func(ctx context.Context, sessionID string) (*session.Session, error)
	destroyFn func(ctx context.Context, sessionID string) error
	refreshFn func(ctx context.Context, sessionID string, ttl time.Duration) error
}

func (m *mockStore) Create(ctx context.Context, userID string, ttl time.Duration, meta map[string]string) (*session.Session, error) {
	return m.createFn(ctx, userID, ttl, meta)
}
func (m *mockStore) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	return m.getFn(ctx, sessionID)
}
func (m *mockStore) Destroy(ctx context.Context, sessionID string) error {
	return m.destroyFn(ctx, sessionID)
}
func (m *mockStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	return m.refreshFn(ctx, sessionID, ttl)
}

// --- Valkey integration tests ---

func valkeyURL(t *testing.T) string {
	t.Helper()
	t.Setenv("VALKEY_URL", "localhost:6379")
	url := os.Getenv("VALKEY_URL")
	if url == "" {
		t.Skip("VALKEY_URL not set — skipping Valkey integration tests")
	}
	return url
}

func TestValkeyStore_CreateGetDestroy(t *testing.T) {
	url := valkeyURL(t)
	client := session.MustConnectValkey(url)
	store := session.NewValkeyStore(client)
	ctx := context.Background()

	// Create
	meta := map[string]string{"ip": "127.0.0.1"}
	sess, err := store.Create(ctx, "user-1", time.Minute, meta)
	require.NoError(t, err)
	require.NotEmpty(t, sess.ID)
	assert.Equal(t, "user-1", sess.UserID)
	assert.Equal(t, "127.0.0.1", sess.Metadata["ip"])

	// Get — should return the session
	got, err := store.Get(ctx, sess.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, sess.ID, got.ID)
	assert.Equal(t, "user-1", got.UserID)

	// Destroy
	err = store.Destroy(ctx, sess.ID)
	require.NoError(t, err)

	// Get after destroy — should return nil
	gone, err := store.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Nil(t, gone)
}

func TestValkeyStore_GetUnknown_ReturnsNil(t *testing.T) {
	url := valkeyURL(t)
	client := session.MustConnectValkey(url)
	store := session.NewValkeyStore(client)

	got, err := store.Get(context.Background(), "nonexistent-id")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestValkeyStore_TTLExpiry(t *testing.T) {
	url := valkeyURL(t)
	client := session.MustConnectValkey(url)
	store := session.NewValkeyStore(client)
	ctx := context.Background()

	// Create with 1-second TTL
	sess, err := store.Create(ctx, "user-ttl", time.Second, nil)
	require.NoError(t, err)

	// Immediately readable
	got, err := store.Get(ctx, sess.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Wait for expiry
	time.Sleep(1100 * time.Millisecond)

	expired, err := store.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Nil(t, expired)
}

func TestValkeyStore_Refresh(t *testing.T) {
	url := valkeyURL(t)
	client := session.MustConnectValkey(url)
	store := session.NewValkeyStore(client)
	ctx := context.Background()

	sess, err := store.Create(ctx, "user-refresh", time.Minute, nil)
	require.NoError(t, err)

	err = store.Refresh(ctx, sess.ID, 5*time.Minute)
	require.NoError(t, err)

	got, err := store.Get(ctx, sess.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.ExpiresAt.After(time.Now().Add(4*time.Minute)))
}

func TestValkeyStore_Refresh_NotFound(t *testing.T) {
	url := valkeyURL(t)
	client := session.MustConnectValkey(url)
	store := session.NewValkeyStore(client)

	err := store.Refresh(context.Background(), "no-such-session", time.Minute)
	assert.Error(t, err)
}
