package featureflag

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func newTestServiceForMiddleware(t *testing.T) *Service {
	t.Helper()
	db, _ := testutil.NewMockDB(t)
	repo := NewRepository(db)
	mc := newMockCache()
	return newServiceWithCache(repo, mc, 30*time.Second)
}

func TestRequireFlag_Enabled_PassesThrough(t *testing.T) {
	svc := newTestServiceForMiddleware(t)
	svc.local.Store("enabled_feature", true)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	}

	mw := RequireFlag(svc, "enabled_feature")
	err := mw(handler)(c)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireFlag_Disabled_Returns404(t *testing.T) {
	svc := newTestServiceForMiddleware(t)
	svc.local.Store("disabled_feature", false)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	}

	mw := RequireFlag(svc, "disabled_feature")
	err := mw(handler)(c)
	require.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRequireFlag_UnknownKey_Returns404(t *testing.T) {
	svc := newTestServiceForMiddleware(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	}

	// Unknown key — not in any cache layer, and mock Valkey/DB won't find it
	mw := RequireFlag(svc, "unknown_flag")
	err := mw(handler)(c)
	require.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
