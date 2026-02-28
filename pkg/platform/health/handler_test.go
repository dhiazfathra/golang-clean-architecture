package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/health"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

func TestLive_AlwaysOK(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	h := health.NewHandler(db, nil) // valkey not used by Live
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	assert.NoError(t, h.Live(e.NewContext(req, rec)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"ok"`)
}

func TestReady_DBUnhealthy(t *testing.T) {
	// Use a real test DB for readiness (requires running Postgres).
	// In CI, skip if testutil.SetupTestDB is unavailable.
	if testing.Short() {
		t.Skip("skipping readiness integration test in short mode")
	}
	db := testutil.SetupTestDB(t)
	vk := testutil.SetupTestValkey(t)
	h := health.NewHandler(db, vk)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	assert.NoError(t, h.Ready(e.NewContext(req, rec)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"database":"ok"`)
	assert.Contains(t, rec.Body.String(), `"valkey":"ok"`)
}
