package seeder

import (
	"context"
	"fmt"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func SeedSuperAdminRole(ctx context.Context, rbacSvc *rbac.Service) error {
	existing, _ := rbacSvc.GetRoleByName(ctx, "super_admin")
	if existing != nil {
		return nil // already seeded
	}
	return rbacSvc.CreateRole(ctx, rbac.CreateRoleCmd{
		Name:        "super_admin",
		Description: "Full access to all modules, all actions, all fields",
		Permissions: []rbac.Permission{rbac.SuperAdminPermission()},
		Actor:       "system",
	})
}

func SeedModuleRoles(ctx context.Context, rbacSvc *rbac.Service) error {
	for name, mod := range rbac.Modules() {
		roleName := name + "_admin"
		existing, _ := rbacSvc.GetRoleByName(ctx, roleName)
		if existing != nil {
			continue
		}
		err := rbacSvc.CreateRole(ctx, rbac.CreateRoleCmd{
			Name:        roleName,
			Description: fmt.Sprintf("Full CRUD access to the %s module", name),
			Permissions: mod.DefaultPerms,
			Actor:       "system",
		})
		if err != nil {
			return fmt.Errorf("seed role %s: %w", roleName, err)
		}
	}
	return nil
}
