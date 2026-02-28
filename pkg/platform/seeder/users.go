package seeder

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

func SeedSuperAdminUser(ctx context.Context, userSvc UserCreator, rbacSvc *rbac.Service, password string) error {
	existing, _ := userSvc.GetByEmail(ctx, "admin@system.local")
	if existing != nil {
		return nil
	}
	userID, err := userSvc.CreateUser(ctx, CreateUserCmd{
		Email:    "admin@system.local",
		Password: password,
		Actor:    "system",
	})
	if err != nil {
		return fmt.Errorf("create super admin user: %w", err)
	}
	superAdminRole, err := rbacSvc.GetRoleByName(ctx, "super_admin")
	if err != nil || superAdminRole == nil {
		return fmt.Errorf("super_admin role not found")
	}
	return userSvc.AssignRole(ctx, userID, superAdminRole.ID, "system")
}

func SeedModuleUsers(ctx context.Context, userSvc UserCreator, defaultPassword string) error {
	for name := range rbac.Modules() {
		email := name + "_admin@system.local"
		existing, _ := userSvc.GetByEmail(ctx, email)
		if existing != nil {
			continue
		}
		envKey := "SEED_" + strings.ToUpper(name) + "_ADMIN_PASSWORD"
		password := os.Getenv(envKey)
		if password == "" {
			password = defaultPassword
		}
		userID, err := userSvc.CreateUser(ctx, CreateUserCmd{
			Email:    email,
			Password: password,
			Actor:    "system",
		})
		if err != nil {
			return fmt.Errorf("create user for %s: %w", name, err)
		}
		// Role IDs follow convention: "role_" + roleName (see rbac.Service.CreateRole)
		roleID := "role_" + name + "_admin"
		if err := userSvc.AssignRole(ctx, userID, roleID, "system"); err != nil {
			return fmt.Errorf("assign role for %s: %w", name, err)
		}
	}
	return nil
}
