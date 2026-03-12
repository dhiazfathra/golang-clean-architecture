package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func newTestHandler(sessions session.SessionStore, users UserProvider, hasher PasswordHasher) *Handler {
	return NewHandler(newTestSvc(sessions, users, hasher))
}

// --- Login ---

func TestAuthHandler_Login_OK(t *testing.T) {
	t.Parallel()
	users := &mockUserProvider{
		GetByEmailFn: func(_ context.Context, email string) (*UserRecord, error) {
			return &UserRecord{ID: "u1", Email: email, PassHash: "h", Active: true}, nil
		},
	}
	sessions := &mockSessionStore{
		CreateFn: func(_ context.Context, userID string, _ time.Duration, _ map[string]string) (*session.Session, error) {
			return &session.Session{ID: "s1", UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}
	h := newTestHandler(sessions, users, nil)
	c, rec := testutil.EchoCtx(http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"pass"}`)

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify session cookie is set
	cookies := rec.Result().Cookies()
	found := false
	for _, ck := range cookies {
		if ck.Name == "session_id" {
			found = true
			assert.Equal(t, "s1", ck.Value)
			assert.Equal(t, "/", ck.Path, "cookie path must be / so it is sent for all routes")
		}
	}
	assert.True(t, found, "session_id cookie should be set")
}

func TestAuthHandler_Login_BadBody(t *testing.T) {
	t.Parallel()
	h := newTestHandler(nil, nil, nil)
	c, rec := testutil.EchoCtx(http.MethodPost, "/auth/login", "{bad")

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Login_ValidationError(t *testing.T) {
	t.Parallel()
	h := newTestHandler(nil, nil, nil)
	// Missing password
	c, rec := testutil.EchoCtx(http.MethodPost, "/auth/login", `{"email":"a@b.com"}`)

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Login_Unauthorized(t *testing.T) {
	t.Parallel()
	users := &mockUserProvider{
		GetByEmailFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return nil, nil
		},
	}
	h := newTestHandler(nil, users, nil)
	c, rec := testutil.EchoCtx(http.MethodPost, "/auth/login", `{"email":"bad@b.com","password":"wrong"}`)

	require.NoError(t, h.Login(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Logout ---

func TestAuthHandler_Logout_WithCookie(t *testing.T) {
	var destroyed string
	sessions := &mockSessionStore{
		DestroyFn: func(_ context.Context, id string) error {
			destroyed = id
			return nil
		},
	}
	h := newTestHandler(sessions, nil, nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess_42"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.Logout(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "sess_42", destroyed)
}

func TestAuthHandler_Logout_NoCookie(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	c, rec := testutil.EchoCtx(http.MethodPost, "/auth/logout", "")

	require.NoError(t, h.Logout(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Me ---

func TestAuthHandler_Me_OK(t *testing.T) {
	h := newTestHandler(nil, nil, nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "usr_7")

	require.NoError(t, h.Me(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "usr_7")
}
