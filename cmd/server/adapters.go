package main

import (
	"context"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
)

// orderUserProvider bridges userService → order.UserProvider.
type orderUserProvider struct{ svc user.UserService }

func (a *orderUserProvider) GetByID(ctx context.Context, id string) (bool, error) {
	u, err := a.svc.GetByID(ctx, id)
	return u != nil, err
}

// authUserAdapter bridges userService → auth.UserProvider.
type authUserAdapter struct{ svc user.UserService }

func (a *authUserAdapter) GetByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	return a.svc.GetByEmailForAuth(ctx, email)
}

// seederUserAdapter bridges userService → seeder.UserCreator.
type seederUserAdapter struct{ svc user.UserService }

func (a *seederUserAdapter) CreateUser(ctx context.Context, cmd seeder.CreateUserCmd) (string, error) {
	return a.svc.CreateUserForSeeder(ctx, cmd)
}

func (a *seederUserAdapter) GetByEmail(ctx context.Context, email string) (*seeder.UserRecord, error) {
	return a.svc.GetByEmailForSeeder(ctx, email)
}

func (a *seederUserAdapter) AssignRole(ctx context.Context, userID, roleID, actor string) error {
	return a.svc.AssignRole(ctx, userID, roleID, actor)
}
