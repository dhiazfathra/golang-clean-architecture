package featureflag

import (
	"context"
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
	return newServiceWithStore(repo, mc, 30*time.Second)
}

func newEchoContext() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func runMiddleware(
	t *testing.T,
	svc *Service,
	flag string,
) (handlerCalled bool, status int) {
	t.Helper()

	c, rec := newEchoContext()

	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	}

	mw := RequireFlag(svc, flag)
	err := mw(handler)(c)
	require.NoError(t, err)

	return handlerCalled, rec.Code
}

func TestRequireFlag(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(svc *Service)
		flag           string
		expectCalled   bool
		expectHTTPCode int
	}{
		{
			name: "enabled passes through",
			setup: func(svc *Service) {
				svc.store.Set(context.Background(), "enabled_feature", "1")
			},
			flag:           "enabled_feature",
			expectCalled:   true,
			expectHTTPCode: http.StatusOK,
		},
		{
			name: "disabled returns 404",
			setup: func(svc *Service) {
				svc.store.Set(context.Background(), "disabled_feature", "0")
			},
			flag:           "disabled_feature",
			expectCalled:   false,
			expectHTTPCode: http.StatusNotFound,
		},
		{
			name: "unknown key returns 404",
			setup: func(svc *Service) {
				// Empty functional field used in test case to provide no-op setup for service.
			},
			flag:           "unknown_flag",
			expectCalled:   false,
			expectHTTPCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestServiceForMiddleware(t)
			tt.setup(svc)

			called, code := runMiddleware(t, svc, tt.flag)

			assert.Equal(t, tt.expectCalled, called)
			assert.Equal(t, tt.expectHTTPCode, code)
		})
	}
}
