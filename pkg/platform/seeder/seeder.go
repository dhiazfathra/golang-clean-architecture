package seeder

import (
	"context"
	"fmt"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/rbac"
)

// UserCreator is satisfied by user.Service (wired in M5).
type UserCreator interface {
	CreateUser(ctx context.Context, cmd CreateUserCmd) (string, error) // returns userID
	GetByEmail(ctx context.Context, email string) (*UserRecord, error)
	AssignRole(ctx context.Context, userID, roleID, actor string) error
}

type CreateUserCmd struct {
	Email    string
	Password string
	Actor    string
}

type UserRecord struct {
	ID    string
	Email string
}

// Seed runs all seeders in order. Idempotent — safe to call on every boot.
func Seed(ctx context.Context, rbacSvc *rbac.Service, userSvc UserCreator, superAdminPassword, defaultModulePassword string) error {
	if err := SeedSuperAdminRole(ctx, rbacSvc); err != nil {
		return fmt.Errorf("seed super admin role: %w", err)
	}
	if err := SeedModuleRoles(ctx, rbacSvc); err != nil {
		return fmt.Errorf("seed module roles: %w", err)
	}
	if err := SeedSuperAdminUser(ctx, userSvc, rbacSvc, superAdminPassword); err != nil {
		return fmt.Errorf("seed super admin user: %w", err)
	}
	if err := SeedModuleUsers(ctx, userSvc, rbacSvc, defaultModulePassword); err != nil {
		return fmt.Errorf("seed module users: %w", err)
	}
	return nil
}
