package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/health"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/testutil"
)

// newPingMockDB creates a sqlmock DB with ping monitoring enabled.
func newPingMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp),
		sqlmock.MonitorPingsOption(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	return sqlx.NewDb(rawDB, "sqlmock"), mock
}

func TestNewHandler(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	h := health.NewHandler(db, nil)
	assert.NotNil(t, h)
}

func TestLiveAlwaysOK(t *testing.T) {
	db, _ := testutil.NewMockDB(t)
	h := health.NewHandler(db, nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	assert.NoError(t, h.Live(e.NewContext(req, rec)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"ok"`)
}

func TestReadyAllHealthy(t *testing.T) {
	db, mock := newPingMockDB(t)
	mock.ExpectPing()

	vk := testutil.SetupTestValkey(t)
	h := health.NewHandler(db, vk)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	err := h.Ready(e.NewContext(req, rec))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"database":"ok"`)
	assert.Contains(t, rec.Body.String(), `"valkey":"ok"`)
}

func TestReadyDBUnhealthy(t *testing.T) {
	db, _ := newPingMockDB(t) // no ExpectPing → ping fails

	vk := testutil.SetupTestValkey(t)
	h := health.NewHandler(db, vk)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	err := h.Ready(e.NewContext(req, rec))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), `"unhealthy"`)
}

func TestReadyIntegration(t *testing.T) {
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
