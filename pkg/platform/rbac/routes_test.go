package rbac_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/logging"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func TestRegisterRoutes(t *testing.T) {
	e := echo.New()
	g := e.Group("/admin")

	mockHandler := &rbac.Handler{}
	rbacSvc := rbac.NewService(&eventstore.MockEventStore{}, &rbac.MockReadRepository{})

	rbac.RegisterRoutes(g, mockHandler, rbacSvc, logging.Noop())

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int // without auth, expect 403
	}{
		{"Create role route exists", http.MethodPost, "/admin/roles", http.StatusForbidden},
		{"List roles route exists", http.MethodGet, "/admin/roles", http.StatusForbidden},
		{"Get role by ID route exists", http.MethodGet, "/admin/roles/123", http.StatusForbidden},
		{"Delete role route exists", http.MethodDelete, "/admin/roles/123", http.StatusForbidden},
		{"Grant permission route exists", http.MethodPost, "/admin/roles/123/permissions", http.StatusForbidden},
		{"Revoke permission route exists", http.MethodDelete, "/admin/roles/123/permissions/read", http.StatusForbidden},
		{"List user roles route exists", http.MethodGet, "/admin/users/123/roles", http.StatusForbidden},
		{"Get audit history route exists", http.MethodGet, "/admin/audit/role/123", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Route exists and RBAC middleware fires (not 404)
			assert.NotEqual(t, http.StatusNotFound, rec.Code, "route should be registered")
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
