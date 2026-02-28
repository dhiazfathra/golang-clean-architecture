package main

import (
	"context"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
)

// authUserAdapter bridges user.Service → auth.UserProvider.
type authUserAdapter struct{ svc *user.Service }

func (a *authUserAdapter) GetByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	return a.svc.GetByEmailForAuth(ctx, email)
}

// seederUserAdapter bridges user.Service → seeder.UserCreator.
type seederUserAdapter struct{ svc *user.Service }

func (a *seederUserAdapter) CreateUser(ctx context.Context, cmd seeder.CreateUserCmd) (string, error) {
	return a.svc.CreateUserForSeeder(ctx, cmd)
}

func (a *seederUserAdapter) GetByEmail(ctx context.Context, email string) (*seeder.UserRecord, error) {
	return a.svc.GetByEmailForSeeder(ctx, email)
}

func (a *seederUserAdapter) AssignRole(ctx context.Context, userID, roleID, actor string) error {
	return a.svc.AssignRole(ctx, userID, roleID, actor)
}
