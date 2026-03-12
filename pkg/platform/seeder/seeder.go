package seeder

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

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
	Password string //nolint:gosec // G117: false positive, this is a command object not a hardcoded secret
	Actor    string
}

type UserRecord struct {
	ID    string
	Email string
}

// ProjectionFlusher flushes pending events into read models synchronously.
type ProjectionFlusher interface {
	RunOnce(ctx context.Context) error
}

// SeedParams holds all dependencies for seeding.
type SeedParams struct {
	RBACService           *rbac.Service
	UserService           UserCreator
	FeatureFlagService    FeatureFlagCreator
	EnvVarService         EnvVarCreator
	APITokenService       APITokenCreator
	OrderService          OrderCreator
	Flusher               ProjectionFlusher
	Logger                zerolog.Logger
	SuperAdminPassword    string
	DefaultModulePassword string
}

// Seed runs all seeders in order. Idempotent — safe to call on every boot.
func Seed(ctx context.Context, params SeedParams) error {
	// Flush any events left from prior runs so idempotency checks see existing data.
	if err := params.Flusher.RunOnce(ctx); err != nil {
		return fmt.Errorf("flush projections (pre-seed): %w", err)
	}
	if err := SeedSuperAdminRole(ctx, params.RBACService); err != nil {
		return fmt.Errorf("seed super admin role: %w", err)
	}
	if err := SeedModuleRoles(ctx, params.RBACService); err != nil {
		return fmt.Errorf("seed module roles: %w", err)
	}
	// Flush newly created role events so they are visible for user seeding.
	if err := params.Flusher.RunOnce(ctx); err != nil {
		return fmt.Errorf("flush projections (post-roles): %w", err)
	}
	if err := SeedSuperAdminUser(ctx, params.UserService, params.RBACService, params.SuperAdminPassword); err != nil {
		return fmt.Errorf("seed super admin user: %w", err)
	}
	if err := SeedModuleUsers(ctx, params.UserService, params.RBACService, params.DefaultModulePassword); err != nil {
		return fmt.Errorf("seed module users: %w", err)
	}
	// Flush user creation events so they are visible for dependent seeders.
	if err := params.Flusher.RunOnce(ctx); err != nil {
		return fmt.Errorf("flush projections (post-users): %w", err)
	}
	if err := SeedFeatureFlags(ctx, params.FeatureFlagService); err != nil {
		return fmt.Errorf("seed feature flags: %w", err)
	}
	if err := SeedEnvVars(ctx, params.EnvVarService); err != nil {
		return fmt.Errorf("seed env vars: %w", err)
	}

	// Get super admin user ID for API token seeding
	superAdmin, err := params.UserService.GetByEmail(ctx, "admin@system.local")
	if err != nil {
		return fmt.Errorf("get super admin user: %w", err)
	}
	if err := SeedAPITokens(ctx, params.Logger, params.APITokenService, superAdmin.ID); err != nil {
		return fmt.Errorf("seed api tokens: %w", err)
	}

	// Collect user IDs for order seeding
	var userIDs []string
	if superAdmin != nil {
		userIDs = append(userIDs, superAdmin.ID)
	}
	for name := range rbac.Modules() {
		email := name + "_admin@system.local"
		if u, err := params.UserService.GetByEmail(ctx, email); err == nil && u != nil {
			userIDs = append(userIDs, u.ID)
		}
	}
	if err := SeedOrders(ctx, params.OrderService, userIDs); err != nil {
		return fmt.Errorf("seed orders: %w", err)
	}

	// Final flush to ensure all seeded data is projected
	if err := params.Flusher.RunOnce(ctx); err != nil {
		return fmt.Errorf("flush projections (final): %w", err)
	}

	return nil
}
