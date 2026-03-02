package user_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

type routeTestCase struct {
	name           string
	method         string
	path           string
	expectedStatus int
}

func runRouteTests(t *testing.T, e *echo.Echo, tests []routeTestCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.NotEqual(t, http.StatusNotFound, rec.Code, "route should be registered")
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func newTestDeps() (*user.Handler, *rbac.Service) {
	return &user.Handler{}, rbac.NewService(&eventstore.MockEventStore{}, &rbac.MockReadRepository{})
}

func TestRegisterRoutes(t *testing.T) {
	e := echo.New()
	mockHandler, rbacSvc := newTestDeps()
	user.RegisterRoutes(e.Group(""), mockHandler, rbacSvc)

	runRouteTests(t, e, []routeTestCase{
		{"Create user route exists", http.MethodPost, "/users", http.StatusForbidden},
		{"Get user by ID route exists", http.MethodGet, "/users/123", http.StatusForbidden},
		{"List users route exists", http.MethodGet, "/users", http.StatusForbidden},
		{"Delete user route exists", http.MethodDelete, "/users/123", http.StatusForbidden},
	})
}

func TestRegisterAdminRoutes(t *testing.T) {
	e := echo.New()
	mockHandler, rbacSvc := newTestDeps()
	user.RegisterAdminRoutes(e.Group("/admin"), mockHandler, rbacSvc)

	runRouteTests(t, e, []routeTestCase{
		{"Admin get user by ID route exists", http.MethodGet, "/admin/users/123", http.StatusForbidden},
	})
}
