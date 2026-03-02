package rbac

import (
	"context"
)

// MockReadRepository is a no-op ReadRepository for use in route/unit tests.
type MockReadRepository struct{}

func (m *MockReadRepository) GetRoleByID(_ context.Context, _ string) (*RoleReadModel, error) {
	return nil, nil
}
func (m *MockReadRepository) GetRoleByName(_ context.Context, _ string) (*RoleReadModel, error) {
	return nil, nil
}
func (m *MockReadRepository) ListRoles(_ context.Context) ([]RoleReadModel, error) {
	return nil, nil
}
func (m *MockReadRepository) GetPermissionsForRole(_ context.Context, _ string) ([]PermissionReadModel, error) {
	return nil, nil
}
func (m *MockReadRepository) GetRolesForUser(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
