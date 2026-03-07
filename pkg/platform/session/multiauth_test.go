package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

// mockTokenValidator implements session.TokenValidator for tests.
type mockTokenValidator struct {
	validateFn func(ctx context.Context, rawToken string) (string, error)
}

func (m *mockTokenValidator) Validate(ctx context.Context, rawToken string) (string, error) {
	return m.validateFn(ctx, rawToken)
}

func TestRequireMultiAuth_BearerToken_Valid(t *testing.T) {
	e := echo.New()
	store := newMockStore(nil)
	tokenValidator := &mockTokenValidator{
		validateFn: func(_ context.Context, raw string) (string, error) {
			if raw == "valid-token" {
				return "token-user-1", nil
			}
			return "", assert.AnError
		},
	}
	mw := session.RequireMultiAuth(store, tokenValidator)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedUserID, capturedMethod string
	handler := mw(func(c echo.Context) error {
		capturedUserID = session.UserID(c)
		capturedMethod = session.AuthMethod(c)
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "token-user-1", capturedUserID)
	assert.Equal(t, "token", capturedMethod)
}

func TestRequireMultiAuth_BearerToken_Invalid(t *testing.T) {
	e := echo.New()
	store := newMockStore(nil)
	tokenValidator := &mockTokenValidator{
		validateFn: func(_ context.Context, _ string) (string, error) {
			return "", assert.AnError
		},
	}
	mw := session.RequireMultiAuth(store, tokenValidator)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw(func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	_ = handler(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireMultiAuth_SessionFallback_Valid(t *testing.T) {
	e := echo.New()
	sess := &session.Session{
		ID:        "valid-session",
		UserID:    "session-user-42",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	store := newMockStore(sess)
	tokenValidator := &mockTokenValidator{
		validateFn: func(_ context.Context, _ string) (string, error) {
			return "", assert.AnError
		},
	}
	mw := session.RequireMultiAuth(store, tokenValidator)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedUserID, capturedMethod string
	handler := mw(func(c echo.Context) error {
		capturedUserID = session.UserID(c)
		capturedMethod = session.AuthMethod(c)
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "session-user-42", capturedUserID)
	assert.Equal(t, "session", capturedMethod)
}

func TestRequireMultiAuth_SessionFallback_Invalid(t *testing.T) {
	e := echo.New()
	store := newMockStore(nil)
	tokenValidator := &mockTokenValidator{
		validateFn: func(_ context.Context, _ string) (string, error) {
			return "", assert.AnError
		},
	}
	mw := session.RequireMultiAuth(store, tokenValidator)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw(func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	_ = handler(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireMultiAuth_NoCookieNoBearer_Returns401(t *testing.T) {
	e := echo.New()
	store := newMockStore(nil)
	tokenValidator := &mockTokenValidator{
		validateFn: func(_ context.Context, _ string) (string, error) {
			return "", assert.AnError
		},
	}
	mw := session.RequireMultiAuth(store, tokenValidator)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw(func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	_ = handler(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireMultiAuth_BearerTakesPrecedence(t *testing.T) {
	e := echo.New()
	sess := &session.Session{
		ID:        "valid-session",
		UserID:    "session-user",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	store := newMockStore(sess)
	tokenValidator := &mockTokenValidator{
		validateFn: func(_ context.Context, raw string) (string, error) {
			if raw == "valid-token" {
				return "token-user", nil
			}
			return "", assert.AnError
		},
	}
	mw := session.RequireMultiAuth(store, tokenValidator)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedUserID, capturedMethod string
	handler := mw(func(c echo.Context) error {
		capturedUserID = session.UserID(c)
		capturedMethod = session.AuthMethod(c)
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "token-user", capturedUserID)
	assert.Equal(t, "token", capturedMethod)
}

func TestAuthMethod_NoContextValue_ReturnsEmpty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	assert.Equal(t, "", session.AuthMethod(c))
}
