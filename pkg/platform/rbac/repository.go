package rbac

import "context"

type ReadRepository interface {
	GetRoleByID(ctx context.Context, id string) (*RoleReadModel, error)
	GetRoleByName(ctx context.Context, name string) (*RoleReadModel, error)
	ListRoles(ctx context.Context) ([]RoleReadModel, error)
	GetPermissionsForRole(ctx context.Context, roleID string) ([]PermissionReadModel, error)
	GetRolesForUser(ctx context.Context, userID string) ([]string, error) // returns role IDs
}
