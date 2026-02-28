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

func newMockStore(sess *session.Session) *mockStore {
	store := map[string]*session.Session{}
	if sess != nil {
		store[sess.ID] = sess
	}
	return &mockStore{
		createFn: func(_ context.Context, userID string, ttl time.Duration, meta map[string]string) (*session.Session, error) {
			s := &session.Session{ID: "new-id", UserID: userID, ExpiresAt: time.Now().Add(ttl), Metadata: meta}
			store[s.ID] = s
			return s, nil
		},
		getFn: func(_ context.Context, id string) (*session.Session, error) {
			s, ok := store[id]
			if !ok {
				return nil, nil
			}
			if time.Now().After(s.ExpiresAt) {
				return nil, nil
			}
			return s, nil
		},
		destroyFn: func(_ context.Context, id string) error {
			delete(store, id)
			return nil
		},
		refreshFn: func(_ context.Context, id string, ttl time.Duration) error {
			s, ok := store[id]
			if !ok {
				return nil
			}
			s.ExpiresAt = time.Now().Add(ttl)
			return nil
		},
	}
}

func TestRequireSession_NoCookie_Returns401(t *testing.T) {
	e := echo.New()
	store := newMockStore(nil)
	mw := session.RequireSession(store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw(func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	_ = handler(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireSession_InvalidCookie_Returns401(t *testing.T) {
	e := echo.New()
	store := newMockStore(nil) // empty store
	mw := session.RequireSession(store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad-session-id"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw(func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	_ = handler(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireSession_ValidCookie_InjectsUserID(t *testing.T) {
	e := echo.New()
	sess := &session.Session{
		ID:        "valid-session",
		UserID:    "user-42",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	store := newMockStore(sess)
	mw := session.RequireSession(store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedUserID string
	handler := mw(func(c echo.Context) error {
		capturedUserID = session.UserID(c)
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "user-42", capturedUserID)
}

func TestUserID_NoContextValue_ReturnsEmpty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	assert.Equal(t, "", session.UserID(c))
}
