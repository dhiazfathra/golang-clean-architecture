package order_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/order"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

// ---------------------------------------------------------------------------
// Mock EventStore
// ---------------------------------------------------------------------------

type mockEventStore struct{}

func (m *mockEventStore) Append(ctx context.Context, events []eventstore.Event) error {
	return nil
}
func (m *mockEventStore) Load(ctx context.Context, aggregateType, aggregateID string, fromVersion int) ([]eventstore.Event, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock ReadRepository
// ---------------------------------------------------------------------------

type mockReadRepository struct{}

func (m *mockReadRepository) GetRoleByID(ctx context.Context, id string) (*rbac.RoleReadModel, error) {
	return nil, nil
}
func (m *mockReadRepository) GetRoleByName(ctx context.Context, name string) (*rbac.RoleReadModel, error) {
	return nil, nil
}
func (m *mockReadRepository) ListRoles(ctx context.Context) ([]rbac.RoleReadModel, error) {
	return nil, nil
}
func (m *mockReadRepository) GetPermissionsForRole(ctx context.Context, roleID string) ([]rbac.PermissionReadModel, error) {
	return nil, nil
}
func (m *mockReadRepository) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}

func TestRegisterRoutes(t *testing.T) {
	e := echo.New()
	g := e.Group("/api")

	mockHandler := &order.Handler{}                                      // or use a mock if Handler has dependencies
	rbacSvc := rbac.NewService(&mockEventStore{}, &mockReadRepository{}) // or mock it

	order.RegisterRoutes(g, mockHandler, rbacSvc)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int // without auth, expect 401/403
	}{
		{"Create order route exists", http.MethodPost, "/api/orders", http.StatusForbidden},
		{"Get order by ID route exists", http.MethodGet, "/api/orders/123", http.StatusForbidden},
		{"List orders route exists", http.MethodGet, "/api/orders", http.StatusForbidden},
		{"Delete order route exists", http.MethodDelete, "/api/orders/123", http.StatusForbidden},
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
